#!/bin/bash
# system.multicall 与 system.listMethods 的网关验收（0.1.96 起）。
# 覆盖两件事：AriaNg 任务详情页/重试按钮所依赖的批量查询必须可用；
# 批量调用不得成为写方法与目录越权的 bypass。真机实测过 Core 会接受嵌套的
# changeGlobalOption dir=/etc，因此第二组断言是安全门禁，不是可选检查。
set -u
S=/run/aria2/aria2.sock
rpc() { curl -s --max-time 25 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d "$1" http://localhost/jsonrpc; }
# code() 只回状态码，用于区分 400 与 200
code() { curl -s -o /dev/null -w '%{http_code}' --max-time 25 --unix-socket "$S" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d "$1" http://localhost/jsonrpc; }
mc() { printf '{"jsonrpc":"2.0","id":1,"method":"system.multicall","params":[[%s]]}' "$1"; }
fail=0
bad() { echo "FAIL $*"; fail=$((fail+1)); }
good() { echo "  ok   $*"; }

DIR_BEFORE=$(rpc '{"jsonrpc":"2.0","id":1,"method":"aria2.getGlobalOption","params":[]}' | tr -d '\n' | sed 's|\\/|/|g' | sed -n 's/.*"dir":"\([^,"]*\)".*/\1/p')
echo "=== 起点 ==="
echo "  当前下载目录 = ${DIR_BEFORE:-未取到}"

echo "=== 1. system.listMethods 必须放行（AriaNg 调试页需要） ==="
LM=$(rpc '{"jsonrpc":"2.0","id":2,"method":"system.listMethods"}')
printf '%s' "$LM" | grep -q 'system.multicall' && good "枚举到 system.multicall" || bad "listMethods 未返回方法列表: $LM"
printf '%s' "$LM" | grep -q 'Unauthorized' && bad "listMethods 被要求鉴权，网关注入位置不对" || true

echo "=== 2. 只读批量调用必须可用，且嵌套调用要带上网关注入的密钥 ==="
RS=$(rpc "$(mc '{"methodName":"aria2.getGlobalStat","params":[]},{"methodName":"aria2.tellActive","params":[["gid","status"]]}')")
printf '%s' "$RS" | grep -q 'downloadSpeed' && good "getGlobalStat 子调用取到数据" || bad "批量查询未返回数据: $RS"
printf '%s' "$RS" | grep -q 'Unauthorized' && bad "嵌套子调用未通过鉴权，token 注入位置错误" || true
printf '%s' "$RS" | grep -q 'wrong type' && bad "调用数组形状不对，Core 直接拒收" || true

echo "=== 3. 任务详情页形状（tellStatus + getPeers） ==="
TD=$(rpc "$(mc '{"methodName":"aria2.tellStatus","params":["0123456789abcdef0123456789abcdef",["gid","status","totalLength"]]},{"methodName":"aria2.getPeers","params":["0123456789abcdef0123456789abcdef"]}')")
printf '%s' "$TD" | grep -q 'Invalid GID' && good "子调用已通过鉴权，仅 gid 不存在（预期结果）" || bad "详情页批量调用异常: $TD"
printf '%s' "$TD" | grep -q 'Unauthorized' && bad "详情页批量调用被拒鉴权" || true

echo "=== 4. 嵌套写方法必须被拒，且不得落到 Core ==="
for entry in \
    'aria2.changeGlobalOption|{"dir":"/etc"}' \
    'aria2.changeOption|{"0123456789abcdef0123456789abcdef":{"dir":"/etc"}}' \
    'aria2.addUri|[["http://evil.invalid/a.bin"]]' \
    'aria2.addTorrent|["aGVsbG8="]' \
    'aria2.remove|["0123456789abcdef0123456789abcdef"]' \
    'aria2.forceRemove|["0123456789abcdef0123456789abcdef"]' \
    'aria2.purgeDownloadResult|[]' \
    'aria2.saveSession|[]' \
    'aria2.shutdown|[]' \
    'system.multicall|[]'; do
    method=${entry%%|*}
    params=${entry#*|}
    body=$(mc "{\"methodName\":\"$method\",\"params\":$params}")
    c=$(code "$body")
    [ "$c" = "400" ] && good "拒绝嵌套的 $method（HTTP 400）" || bad "嵌套的 $method 未被拦截，HTTP $c"
done

echo "=== 5. 目录守卫在批量调用下依然有效 ==="
DIR_AFTER=$(rpc '{"jsonrpc":"2.0","id":3,"method":"aria2.getGlobalOption","params":[]}' | tr -d '\n' | sed 's|\\/|/|g' | sed -n 's/.*"dir":"\([^,"]*\)".*/\1/p')
echo "  尝试越权后的下载目录 = ${DIR_AFTER:-未取到}"
[ "$DIR_BEFORE" = "$DIR_AFTER" ] && good "下载目录未被批量调用改写" || bad "下载目录被嵌套调用改成 $DIR_AFTER"

echo "=== 6. 畸形与超大批量调用 ==="
c=$(code '{"jsonrpc":"2.0","id":4,"method":"system.multicall","params":[[]]}')
[ "$c" = "400" ] && good "拒绝空调用列表" || bad "空列表未被拒绝，HTTP $c"
big='{"jsonrpc":"2.0","id":5,"method":"system.multicall","params":[['
for i in $(seq 1 129); do
    [ "$i" -gt 1 ] && big="$big,"
    big="$big{\"methodName\":\"aria2.getVersion\",\"params\":[]}"
done
big="$big]]}"
c=$(code "$big")
[ "$c" = "400" ] && good "拒绝超过 128 项的批量调用" || bad "超大批量调用未被拒绝，HTTP $c"

echo
if [ "$fail" -eq 0 ]; then
    echo "system.multicall 网关验收全部通过"
else
    echo "system.multicall 网关验收失败 $fail 项"
fi
exit $fail