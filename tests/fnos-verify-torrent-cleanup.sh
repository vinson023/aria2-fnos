#!/bin/bash
# 设备实测：RPC 上传种子后，删除任务必须连带清理 Aria2 保存的 .torrent 元数据文件。
# 除守卫中的 tr 转义外不出现反斜杠，避免跨语言转义被破坏；仅在授权测试机执行。
set -u
SOCK=/run/aria2/aria2.sock

# 前置守卫：本用例会往下载目录留下 .torrent，只有带种子台账的版本才能安全执行。
MGR_PID=$(cat /vol1/@appdata/aria2/manager.pid 2>/dev/null || true)
if [ -z "$MGR_PID" ] || ! tr "\0" "\n" < "/proc/$MGR_PID/cmdline" 2>/dev/null | grep -q -- "--torrent-ledger"; then
  echo "FAIL 运行中的管理器没有 --torrent-ledger，设备仍是修复前的版本，测试中止（避免制造无法追踪的残留种子）"
  exit 1
fi
DL=$(grep download_root /vol1/@appconf/aria2/config.json | tr -d ' ",' | cut -d: -f2)
fail=0
ok() { echo "ok   $*"; }
bad() { echo "FAIL $*"; fail=$((fail + 1)); }
rpcfile() { curl -s --max-time 25 --unix-socket "$SOCK" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d @/tmp/tr.json http://localhost/jsonrpc; }
list() { find "$DL" -maxdepth 1 -name "$1" -print 2>/tmp/find_t.err; }

echo "下载目录: $DL"
echo "预期 info_hash: $(cat /tmp/aria2verify/probe.hash)"
[ -d "$DL" ] || { echo "FAIL 下载目录不可达，测试中止"; exit 1; }

echo "--- 1. 添加种子（pause=true，不产生真实流量）"
base64 -w 0 /tmp/aria2verify/probe.torrent > /tmp/torrent.b64
printf '{"jsonrpc":"2.0","id":61,"method":"aria2.addTorrent","params":["%s",[],{"pause":"true"}]}' "$(cat /tmp/torrent.b64)" > /tmp/tr.json
OUT=$(rpcfile)
echo "    $OUT"
GID=$(echo "$OUT" | grep -oE '"result":"[0-9a-f]+"' | cut -d'"' -f4)
[ -n "$GID" ] || { echo "FAIL 种子添加失败，测试中止"; exit 1; }
echo "    gid=$GID"
sleep 3

echo "--- 2. Aria2 保存的种子文件（rpc-save-upload-metadata=true）"
TOR=$(list '*.torrent') || { bad "无法读取下载目录"; exit 1; }
echo "$TOR" | sed '/^$/d; s/^/    /'
[ -n "$TOR" ] || bad "未见 .torrent，测试前提不成立"

echo "--- 3. forceRemove 删除任务"
printf '{"jsonrpc":"2.0","id":62,"method":"aria2.forceRemove","params":["%s"]}' "$GID" > /tmp/tr.json
rpcfile > /dev/null
sleep 5

echo "--- 4. 残留检查"
LEFT=$(list '*.torrent') || { bad "复检种子时目录不可读"; exit 1; }
if [ -n "$LEFT" ]; then bad "删除任务后残留种子: $LEFT"; else ok "删除任务后无 .torrent 残留"; fi
RES=$(list 'torrentprobe.bin*') || { bad "复检数据文件时目录不可读"; exit 1; }
if [ -n "$RES" ]; then bad "残留数据文件: $RES"; else ok "无数据文件残留"; fi

echo "--- 结果: $fail 处失败"
exit $((fail > 0))
