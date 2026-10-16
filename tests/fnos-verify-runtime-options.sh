#!/bin/bash
# 运行期选项跨重启持久化验收：AriaNg 走的就是这条链路。
set -u
S=/run/aria2/aria2.sock
OVR=/vol1/@appdata/aria2/aria2-overrides.conf
ACONF=/vol1/@appdata/aria2/aria2.conf
# 设备实际取值可用环境变量覆盖，避免写死路径与端口
EXPECT_DIR=${EXPECT_DIR:-/vol1/1000/download}
EXPECT_RPC_PORT=${EXPECT_RPC_PORT:-6801}
rpc() { curl -s --max-time 25 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d "$1" http://localhost/jsonrpc; }
api() { local method=$1 path=$2; shift 2; curl -s -X "$method" --max-time 90 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' "$@" "http://localhost$path"; }
getopt_raw() { rpc '{"jsonrpc":"2.0","id":1,"method":"aria2.getGlobalOption","params":[]}'; }
# Aria2 回读把 / 转义成 \/ ，先还原再比较，否则明文断言必然误报
getopt_all() { getopt_raw | tr -d '\n' | sed 's|\\/|/|g' | tr ',' '\n'; }
val() { getopt_all | sed -n "s/.*\"$1\":\"\{0,1\}\([^,\"}]*\)\"\{0,1\}.*/\1/p" | head -1; }
fail=0
bad() { echo "FAIL $*"; fail=$((fail+1)); }

echo "=== 起点 ==="
echo "  max-concurrent-downloads = $(val max-concurrent-downloads)"
echo "  split                    = $(val split)"
echo "  overrides 文件存在        = $([ -f "$OVR" ] && echo yes || echo no)"

echo "=== 1. 前台改一个普通全局选项（并发数 -> 9） ==="
rpc '{"jsonrpc":"2.0","id":2,"method":"aria2.changeGlobalOption","params":[{"max-concurrent-downloads":"9","split":"16"}]}'
echo
echo "  内存即时值: max-concurrent-downloads=$(val max-concurrent-downloads) split=$(val split)"
echo "  $OVR:"
sed 's/^/      /' "$OVR" 2>/dev/null || echo "      (文件缺失)"
grep -q '^max-concurrent-downloads=9$' "$OVR" || bad "并发数未落盘"
grep -q '^split=16$' "$OVR" || bad "split 未落盘"
echo "  GET /api/v1/runtime-options: $(api GET /api/v1/runtime-options)"

echo "=== 2. 独占项必须被拒绝（尝试改 dir / rpc-listen-port / bt-tracker） ==="
rpc '{"jsonrpc":"2.0","id":3,"method":"aria2.changeGlobalOption","params":[{"dir":"/etc","rpc-listen-port":"9999","bt-tracker":"http://evil.invalid:6969/announce"}]}'
echo
grep -Eq '^(dir|rpc-listen-port|bt-tracker)=' "$OVR" && bad "独占项被写入覆盖文件" || echo "  ok   独占项未进入覆盖文件"
RAW=$(getopt_raw | tr -d '\n')
DIRNOW=$(val dir)
echo "  运行期实际取值: dir=$DIRNOW rpc-listen-port=$(val rpc-listen-port) on-download-complete=$(val on-download-complete)"
[ "$DIRNOW" = "$EXPECT_DIR" ] || bad "dir 被运行期改写为 $DIRNOW，网关未拦截专属选项"
[ "$(val rpc-listen-port)" = "$EXPECT_RPC_PORT" ] || bad "rpc-listen-port 被运行期改写"
printf '%s' "$RAW" | grep -q 'evil.invalid' && bad "bt-tracker 被注入非法 Tracker" || echo "  ok   bt-tracker 未被注入"
printf '%s' "$RAW" | grep -q '/bin/sh' && bad "on-* 回调被劫持" || echo "  ok   on-* 回调未被劫持"

echo "=== 2a. 专属项与普通项混发时，普通项必须仍然生效 ==="
rpc '{"jsonrpc":"2.0","id":9,"method":"aria2.changeGlobalOption","params":[{"dir":"/etc","retry-wait":"5"}]}'
echo
[ "$(val dir)" = "$EXPECT_DIR" ] || bad "混发时 dir 仍被改写"
[ "$(val retry-wait)" = "5" ] || bad "混发时普通选项 retry-wait 未生效（过滤过度）"
grep -q "^retry-wait=5$" "$OVR" || bad "混发时普通选项未落盘"
grep -Eq "^dir=" "$OVR" && bad "混发时 dir 被落盘"

echo "=== 2b. 空对象在前时不得持久化（Aria2 只读 params[0]） ==="
rpc '{"jsonrpc":"2.0","id":8,"method":"aria2.changeGlobalOption","params":[{},{"max-concurrent-downloads":"11"}]}'
echo
grep -q '^max-concurrent-downloads=11$' "$OVR" && bad "网关把 Aria2 从未应用的 params[1] 落盘了" || echo "  ok   params[1] 未被持久化"
echo "  内存值 = $(val max-concurrent-downloads)（期望仍为 9，空对象不应改动 Aria2）"
[ "$(val max-concurrent-downloads)" = "9" ] || bad "params[0] 为空对象时 Aria2 取值被改动"

echo "=== 3. 重启应用，看取值是否存活 ==="
echo "  POST /api/v1/restart -> $(api POST /api/v1/restart)"
i=0
while [ $i -lt 40 ]; do
  if [ "$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' http://localhost/healthz)" = "204" ]; then break; fi
  sleep 1; i=$((i+1))
done
sleep 3
echo "  重启耗时 ${i}s，healthz=$(curl -s -o /dev/null -w '%{http_code}' --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' http://localhost/healthz)"
A=$(val max-concurrent-downloads); B=$(val split)
echo "  重启后: max-concurrent-downloads=$A split=$B"
[ "$A" = "9" ] || bad "重启后并发数回退为 $A（期望 9）"
[ "$B" = "16" ] || bad "重启后 split 回退为 $B（期望 16）"
echo "  aria2.conf 合并结果:"
grep -nE '^(max-concurrent-downloads|split|dir|rpc-listen-port|bt-tracker)=' "$ACONF" | sed 's/^/      /'
n_dup=$(grep -c '^split=' "$ACONF"); [ "$n_dup" = "1" ] || bad "aria2.conf 出现 $n_dup 行 split，合并产生重复键"
n_dup2=$(grep -c '^max-concurrent-downloads=' "$ACONF"); [ "$n_dup2" = "1" ] || bad "aria2.conf 出现 $n_dup2 行并发数"
grep -q '^rpc-secret=' "$ACONF" || bad "合并后 rpc-secret 丢失"

echo "=== 4. 清空运行期选项并复原 ==="
api DELETE /api/v1/runtime-options
echo
[ -f "$OVR" ] && bad "DELETE 之后文件仍存在" || echo "  ok   覆盖文件已删除"
echo "  POST /api/v1/restart -> $(api POST /api/v1/restart)"
i=0; while [ $i -lt 40 ]; do [ "$(curl -s -o /dev/null -w '%{http_code}' --max-time 3 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' http://localhost/healthz)" = "204" ] && break; sleep 1; i=$((i+1)); done
sleep 3
echo "  复原后: max-concurrent-downloads=$(val max-concurrent-downloads) split=$(val split)（期望回到随包默认 5 / 64）"

echo "=== 结果: $fail 处失败 ==="
exit $((fail > 0))