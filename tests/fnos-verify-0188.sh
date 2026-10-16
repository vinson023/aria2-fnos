#!/bin/bash
# Aria2 (aria2) device acceptance. Read-only unless ARCHIVE_TEST=1 or RUNTIME_TEST=1.
set -u
CONF=/vol1/@appconf/aria2/config.json
ACONF=/vol1/@appdata/aria2/aria2.conf
ALOG=/vol1/@appdata/aria2/aria2.log
SOCK=/run/aria2/aria2.sock
OVR=/vol1/@appdata/aria2/aria2-overrides.conf
rpc() { curl -s --max-time 25 --unix-socket "$SOCK" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d "$1" http://localhost/jsonrpc; }
api() { curl -s --max-time 15 --unix-socket "$SOCK" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' "http://localhost$1"; }
fail=0
say() { printf '%s\n' "$*"; }
bad() { printf 'FAIL %s\n' "$*"; fail=$((fail+1)); }
json() { tr -d '\n\r\t' < "$CONF" | sed -n "s/.*\"$1\":\s*\"\{0,1\}\([^,\"}]*\)\"\{0,1\}.*/\1/p" | head -1; }
opt() { sed -n "s/.*\"$1\":\"\{0,1\}\([^,\"}]*\)\"\{0,1\}.*/\1/p" /tmp/a2opt.txt | head -1; }

say "=== 1. processes (must all run as aria2)"
ps -eo user,pid,args | grep -E '/aria2/(target/server|aria2)' | grep -v grep | awk '{print "   ", $1, $2, $3}' | head -5
if ps -eo user,args | grep -v grep | grep -qE '^root.*/var/apps/aria2/target/server'; then bad "root-owned app process"; else say "ok   no root-owned app process"; fi

say "=== 2. runtime options actually applied (RPC through the app gateway)"
rpc '{"jsonrpc":"2.0","id":1,"method":"aria2.getVersion","params":[]}' | head -c 120; echo
# aria2.getGlobalOption 只接受密钥参数并返回全部选项。早期写法把选项名当成第一个参数
# 传进去，会被当作错误的 rpc-secret，导致整段核对恒为 <none>。
rpc '{"jsonrpc":"2.0","id":2,"method":"aria2.getGlobalOption","params":[]}' | tr -d '\n' | tr ',' '\n' > /tmp/a2opt.txt
if grep -q '"result"' /tmp/a2opt.txt; then
  say "ok   getGlobalOption returned the full option set"
else
  bad "getGlobalOption 未返回选项集"
fi
for k in seed-time seed-ratio bt-detach-seed-only split max-connection-per-server disk-cache bt-max-peers summary-interval; do
  printf "   %-26s %s\n" "$k" "$(opt "$k")"
  [ -n "$(opt "$k")" ] || bad "运行期选项 $k 未生效"
done

say "=== 3. config consistency"
RPCP=$(json rpc_port); DR=$(json download_root); AR=$(json archive_root); SM=$(json special_mode); AE=$(json archive_enabled)
say "   rpc=$RPCP download_root=$DR archive_root=$AR special_mode=$SM archive_enabled=$AE"
if [ -n "$AR" ] && [ "$SM" = move ] && [ "$AE" = true ]; then say "ok   archive configured and enabled";
elif [ -z "$AR" ] && [ "$SM" = off ] && [ "$AE" = false ]; then say "ok   archive disabled consistently";
else bad "archive state inconsistent"; fi

say "=== 4. RPC binding"
ss -lntp 2>/dev/null | grep ":${RPCP} " | sed 's/^/   /' | cut -c1-90
if ss -lnt 2>/dev/null | grep ":${RPCP} " | grep -qvE '127.0.0.1|\[::1\]'; then bad "RPC exposed publicly"; else say "ok   RPC loopback only on ${RPCP}"; fi

say "=== 5. archive dirs"
for d in "$DR" "$AR"; do
  if [ -d "$d" ]; then say "   exists $d ($(stat -c '%U:%G %a' "$d"))"; else bad "missing dir $d"; fi
done

if [ "${RUNTIME_TEST:-0}" = 1 ]; then
  say "=== 6. AriaNg 运行期选项持久化"
  # 网关把成功的 aria2.changeGlobalOption 写入 aria2-overrides.conf，生命周期脚本生成
  # aria2.conf 时合并；本用例跑完自行收尾，不把测试取值留在设备上。
  had_before=no
  [ -f "$OVR" ] && had_before=yes
  say "   变更前的运行期选项: $(api /api/v1/runtime-options)"
  rpc '{"jsonrpc":"2.0","id":5,"method":"aria2.changeGlobalOption","params":[{},{"max-concurrent-downloads":"3"}]}' | head -c 120; echo
  if [ -f "$OVR" ]; then
    say "   $OVR: $(tr '\n' ' ' < "$OVR")"
    grep -q '^max-concurrent-downloads=3$' "$OVR" || bad "改动未写入 $OVR"
  else
    bad "缺少 $OVR，运行期选项未落盘"
  fi
  api /api/v1/runtime-options > /tmp/a2rt.json
  grep -q 'max-concurrent-downloads' /tmp/a2rt.json || bad "GET /api/v1/runtime-options 未回显改动"
  rpc '{"jsonrpc":"2.0","id":6,"method":"aria2.changeGlobalOption","params":[{},{"rpc-listen-port":"1"}]}' | head -c 120; echo
  grep -q '^rpc-listen-port=' "$OVR" 2>/dev/null && bad "独占项 rpc-listen-port 被写入运行期文件"
  say "   --- 被拒绝的独占项"
  grep -c 'rpc-listen-port' "$OVR" 2>/dev/null | sed 's/^/      0=已拒绝 /'
  if [ "$had_before" = yes ]; then
    rpc '{"jsonrpc":"2.0","id":7,"method":"aria2.changeGlobalOption","params":[{},{"max-concurrent-downloads":"5"}]}' >/dev/null
    say "   保留了原有运行期选项，只回滚本次测试取值"
  else
    curl -s -X DELETE --max-time 15 --unix-socket "$SOCK" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' http://localhost/api/v1/runtime-options > /dev/null
    if [ -f "$OVR" ]; then bad "DELETE 之后 $OVR 仍存在"; else say "ok   已清空测试写入的运行期选项"; fi
  fi
fi

if [ "${ARCHIVE_TEST:-0}" = 1 ]; then
  say "=== 7. archive end-to-end"
  BEFORE=$(ls -1 "$AR" 2>/dev/null | wc -l)
  rm -rf /tmp/a2srv; mkdir -p /tmp/a2srv; head -c 2000000 /dev/urandom > /tmp/a2srv/verify-a.bin
  busybox httpd -p 127.0.0.1:18081 -h /tmp/a2srv -c /dev/null
  sleep 1
  GID=$(rpc '{"jsonrpc":"2.0","id":3,"method":"aria2.addUri","params":[["http://127.0.0.1:18081/verify-a.bin"],{}]}' | tr -d '\n' | sed -n 's/.*"result":"\([0-9a-f]*\)".*/\1/p')
  say "   gid=$GID"
  i=0; while [ $i -lt 45 ]; do ls "$AR"/verify-a*.bin >/dev/null 2>&1 && break; i=$((i+1)); sleep 2; done
  AFTER=$(ls -1 "$AR" 2>/dev/null | wc -l)
  if [ "$AFTER" -gt "$BEFORE" ]; then
    say "ok   file landed in archive: $(ls -1 "$AR" | grep verify-a | head -2 | tr '\n' ' ')"
  else
    bad "no archived file; archive dir: $(ls -1 "$AR" 2>/dev/null | head -3 | tr '\n' ' ')"
    bad "download dir: $(ls -1 "$DR" 2>/dev/null | head -6 | tr '\n' ' ')"
  fi
  if [ -e "$DR/verify-a.bin" ] || [ -d "$DR/verify-a.bin" ]; then bad "source still in download dir"; else say "ok   download dir cleaned"; fi
  rpc "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"aria2.forceRemove\",\"params\":[\"$GID\"]}" >/dev/null
  kill $(ss -ltnp 2>/dev/null | sed -n 's/.*127.0.0.1:18081.*pid=\([0-9]*\).*/\1/p' | head -1) 2>/dev/null
  rm -rf /tmp/a2srv
  say "   --- last archive log lines"
  grep -a 'archive' "$ALOG" 2>/dev/null | tail -3 | cut -c1-160
fi

say "=== result: $fail failure(s)"
exit $((fail>0))