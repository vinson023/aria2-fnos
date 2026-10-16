#!/bin/bash
# 发货产物全链路验证：只在 debian:12-slim 容器里跑，用解包后的真实发布件走完
# 生命周期脚本 -> 管理器网关 -> 静态 Core -> 完成回调 -> 归档搬移 -> 删除清理 一整条链路。
#
# 与 tests/startup-e2e-container.sh 的分工：那个脚本考察"起不起来"（退出码、自愈、日志、属主），
# 本脚本考察"起来之后功能对不对"（真实下载、自动归档、部分文件清理）。
# 容器里没有 curl 与 python，HTTP 源、JSON-RPC 驱动和轮询都由 tests/btfixture 提供。
set -u
PKG=${PKG:-/hosttmp/pkg0195}
FIX=${FIX:-/hosttmp/btfixture-linux}
APP=/vol1/@appshare/aria2/downloads
S=/vol1/@appdata/aria2/startup.log
fail=0
ok() { echo "  [ok] $*"; }
bad() { echo "  [FAIL] $*"; fail=$((fail + 1)); }

prepare() {
    rm -rf /var/apps/aria2 /vol1 /run/aria2 /tmp/srv
    mkdir -p /var/apps/aria2/cmd /var/apps/aria2/target /tmp/srv
    cp -R "$PKG/app/." /var/apps/aria2/target/
    cp -R "$PKG/cmd/." /var/apps/aria2/cmd/
    [ -f "$PKG/manifest" ] && cp "$PKG/manifest" /var/apps/aria2/manifest
    # 归档内 cmd/* 与 server/* 是 666，真实安装由应用中心补可执行位，这里照做。
    chmod 755 /var/apps/aria2/cmd/* /var/apps/aria2/target/server/*
    mkdir -p /vol1/@appconf/aria2 /vol1/@appdata/aria2 "$APP" /vol1/@appcenter/aria2
    id aria2 > /dev/null 2>&1 || adduser --system --group --uid 981 --no-create-home aria2 > /dev/null 2>&1
    chown -R aria2:aria2 /vol1/@appconf/aria2 /vol1/@appdata/aria2 "$APP"
    export TRIM_APPDEST=/vol1/@appcenter/aria2
    export TRIM_PKGETC=/vol1/@appconf/aria2
    export TRIM_PKGVAR=/vol1/@appdata/aria2
    export TRIM_DATA_SHARE_PATHS="$APP"
    mkdir -p /vol1/@appcenter/aria2/target
    ln -sfn /var/apps/aria2/target /vol1/@appcenter/aria2/target/real 2>/dev/null || true
    conf='{"rpc_port":6801,"listen_port":6888,"dht_listen_port":6888,"dht_message_timeout":"10","enable_dht":true,"enable_dht6":true,"download_root":"'"$APP"'","archive_enabled":true,"archive_root":"'"$APP"'/completed","special_mode":"move","authorized_roots":["'"$APP"'"],"tracker_source":"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt","tracker_auto_update":false,"config_version":1}'
    printf '%s' "$conf" > /vol1/@appconf/aria2/config.json
    printf 'chain-placeholder-secret-value-0123456789abcdef\n' > /vol1/@appconf/aria2/rpc.secret
    chown aria2:aria2 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret
    chmod 600 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret
    # 本地 HTTP 源：一个 4 MiB 随机文件用于完整下载，/stall.bin 用于制造部分下载。
    dd if=/dev/urandom of=/tmp/srv/good.bin bs=1M count=4 status=none
    GOOD_SHA=$(sha256sum /tmp/srv/good.bin | cut -d" " -f1)
    "$FIX" fileserver -dir /tmp/srv -listen 127.0.0.1:6977 -port-file /tmp/srv/port > /tmp/srv/fs.log 2>&1 &
    FS_PID=$!
    sleep 1
    /var/apps/aria2/cmd/main start > /tmp/start.out 2> /tmp/start.err
    echo "  start 退出码: $?"
}

gw() { "$FIX" gateway -method "$1" -params "$2" 2>&1; }

echo "### 准备发货产物与本地 HTTP 源"
prepare
[ -s "$S" ] && grep -q '启动完成' "$S" && ok "服务已启动，startup.log 记录启动完成" || bad "服务未正常启动：$(tail -3 "$S" 2>/dev/null)"
[ -S /run/aria2/aria2.sock ] && ok "网关 Socket 就绪" || bad "网关 Socket 缺失"
grep -q '静态链接\|Core 自检通过' "$S" 2>/dev/null && ok "Core 预检通过" || bad "Core 预检未留痕"

echo "### 用例 A：管理员身份经网关提交真实 HTTP 下载"
# addUri 的 params 是「URL 数组的数组」：[[ "url" ]]。
GID=$(gw aria2.addUri '[["http://127.0.0.1:6977/good.bin"]]' | tr -d '"')
if [ -n "$GID" ] && [ "${#GID}" -ge 8 ]; then ok "任务已提交，gid=$GID"; else bad "提交失败：$GID"; fi
sleep 6
st=$(gw aria2.tellStatus "[\"$GID\"]")
echo "  tellStatus: $(echo "$st" | head -c 160)"

echo "### 用例 B：完成后自动归档到 completed/（回调链）"
if "$FIX" waitfile -path "$APP/completed/good.bin" -timeout 45 > /tmp/wait.log 2>&1; then
    got=$(sha256sum "$APP/completed/good.bin" | cut -d" " -f1)
    if [ "$got" = "$GOOD_SHA" ]; then ok "归档文件逐字节一致：$got"; else bad "归档文件内容不符：$got != $GOOD_SHA"; fi
    [ -e "$APP/good.bin" ] && bad "归档后原位置仍残留文件" || ok "原位置已清空（move 语义正确）"
else
    bad "45s 内未出现 $APP/completed/good.bin（$(cat /tmp/wait.log)）"
    echo "  下载目录: $(ls -la "$APP" 2>/dev/null | tail -n +2 | tr '\n' ' ')"
    echo "  aria2.log 尾部: $(tail -5 /vol1/@appdata/aria2/aria2.log 2>/dev/null | tr '\n' ' ')"
fi

echo "### 用例 C：删除未完成任务应连带清理部分文件与 .aria2"
GID2=$(gw aria2.addUri '[["http://127.0.0.1:6977/stall.bin"]]' | tr -d '"')
if [ -n "$GID2" ] && [ "${#GID2}" -ge 8 ]; then ok "卡住的任务已提交，gid=$GID2"; else bad "提交失败：$GID2"; fi
"$FIX" waitfile -path "$APP/stall.bin" -timeout 30 > /tmp/wait2.log 2>&1 || bad "未能制造部分下载文件"
ls "$APP"/stall.bin* > /tmp/parts.txt 2>&1
echo "  删除前残留: $(tr '\n' ' ' < /tmp/parts.txt)"
grep -q stall.bin /tmp/parts.txt || bad "删除前未见部分文件"
rm=$(gw aria2.remove "[\"$GID2\"]")
echo "  remove 返回: $(echo "$rm" | head -c 80)"
sleep 6
left=$(ls "$APP" 2>/dev/null | grep -c 'stall.bin' || true)
if [ "$left" = "0" ]; then ok "部分文件与 .aria2 控制文件均已清理"; else bad "仍有 $left 个 stall.bin 相关残留: $(ls "$APP" | grep stall.bin | tr '\n' ' ')"; fi
[ -e "$APP/stall.bin.aria2" ] && bad ".aria2 控制文件残留" || true

echo "### 用例 D：清理不该误伤已归档文件"
[ -e "$APP/completed/good.bin" ] && ok "已归档文件未被清理误删" || bad "已归档文件消失"

echo "### 用例 E：AriaNg 的批量查询必须可用，且不得成为越权通道"
# 任务详情页与重试按钮走 system.multicall；Core 对嵌套调用逐个鉴权，
# 因此批量查询是否真能用，只有对着真实 Core 跑一遍才算数。
MC=$(gw system.multicall '[[{"methodName":"aria2.getGlobalStat","params":[]},{"methodName":"aria2.tellActive","params":[["gid","status"]]}]]')
echo "  multicall 返回: $(echo "$MC" | head -c 140)"
case "$MC" in
    *downloadSpeed*) ok "批量查询取到全局统计，网关注入的嵌套密钥生效" ;;
    *) bad "批量查询未取到数据: $MC" ;;
esac
case "$MC" in
    *Unauthorized*) bad "嵌套子调用未通过鉴权，token 注入位置不对" ;;
esac
TD=$(gw system.multicall '[[{"methodName":"aria2.tellStatus","params":["0123456789abcdef0123456789abcdef",["gid","status"]]},{"methodName":"aria2.getPeers","params":["0123456789abcdef0123456789abcdef"]}]]')
echo "  详情页形状返回: $(echo "$TD" | head -c 140)"
case "$TD" in
    *"Invalid GID"*) ok "详情页批量调用已过鉴权，仅 gid 不存在（预期结果）" ;;
    *) bad "详情页批量调用异常: $TD" ;;
esac
# 真机实测 Core 会接受嵌套的 changeGlobalOption dir=/etc，网关必须拦在最前面。
ESC=$(gw system.multicall '[[{"methodName":"aria2.changeGlobalOption","params":[{"dir":"/etc"}]}]]')
echo "  嵌套越权返回: $(echo "$ESC" | head -c 140)"
case "$ESC" in
    *OK*) bad "嵌套 changeGlobalOption 越权成功，目录守卫被绕过" ;;
    *) ok "嵌套写方法被网关拒绝" ;;
esac
REMOVES=$(gw system.multicall '[[{"methodName":"aria2.remove","params":["0123456789abcdef0123456789abcdef"]}]]')
case "$REMOVES" in
    *OK*) bad "嵌套 remove 被执行，种子台账会被绕过" ;;
    *) ok "嵌套 remove 被网关拒绝" ;;
esac
DIRNOW=$(gw aria2.getGlobalOption '[]')
case "$DIRNOW" in
    *"/etc"*) bad "下载目录已被批量调用越权改写" ;;
    *) ok "下载目录未受越权尝试影响" ;;
esac
LM=$(gw system.listMethods '[]')
case "$LM" in
    *"system.multicall"*) ok "system.listMethods 已放行（AriaNg 调试页需要）" ;;
    *) bad "system.listMethods 未放行: $(echo "$LM" | head -c 100)" ;;
esac

/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true
kill "$FS_PID" 2>/dev/null || true

echo
if [ "$fail" -eq 0 ]; then echo "发货产物全链路验证全部通过。"; else echo "共 $fail 处失败"; fi
exit $fail
