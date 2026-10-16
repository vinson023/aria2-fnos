#!/bin/bash
# 就绪门禁失败时的病因归集验证（只在 debian:12-slim 容器里跑）。
#
# 为什么需要专门一个脚本：`aria2.log` 与 `manager.log` 每次启动都被删除重建，应用中心又不收集它们，
# 所以"进程起来了但立刻挂掉"这类故障在跨重启之后线索全部消失，前台只剩一句「无法启动」。
# `cmd/main` 因此在就绪失败时把两个日志的尾部归集进持久的 `startup.log`。
# 本脚本用一个"预检能过、真正拉起立刻失败"的假 Core 把这条路径逼出来，并断言病因确实落盘。
#
# 关键构造：假 Core 对 `--version` 返回成功（否则会被 ADR-015 第 11 条预检挡住，走不到就绪门禁），
# 对其他参数打一行动态链接器风格的错误并以 127 退出。
set -u
PKG=${PKG:-/hosttmp/pkg0195}
fail=0
ok() { echo "  [ok] $*"; }
bad() { echo "  [FAIL] $*"; fail=$((fail + 1)); }
S=/vol1/@appdata/aria2/startup.log
APP=/vol1/@appshare/aria2/downloads

rm -rf /var/apps/aria2 /vol1 /run/aria2
mkdir -p /var/apps/aria2/cmd /var/apps/aria2/target
cp -R "$PKG/app/." /var/apps/aria2/target/
cp -R "$PKG/cmd/." /var/apps/aria2/cmd/
[ -f "$PKG/manifest" ] && cp "$PKG/manifest" /var/apps/aria2/manifest
chmod 755 /var/apps/aria2/cmd/* /var/apps/aria2/target/server/*

cat > /var/apps/aria2/target/server/aria2c <<'FAKE'
#!/bin/sh
if [ "$1" = "--version" ]; then echo "aria2 version 1.36.0"; exit 0; fi
echo "error while loading shared libraries: libsomething.so.9: cannot open shared object file" >&2
exit 127
FAKE
chmod 755 /var/apps/aria2/target/server/aria2c

mkdir -p /vol1/@appconf/aria2 /vol1/@appdata/aria2 "$APP" /vol1/@appcenter/aria2
id aria2 > /dev/null 2>&1 || adduser --system --group --uid 981 --no-create-home aria2 > /dev/null 2>&1
chown -R aria2:aria2 /vol1/@appconf/aria2 /vol1/@appdata/aria2 "$APP"
export TRIM_APPDEST=/vol1/@appcenter/aria2
export TRIM_PKGETC=/vol1/@appconf/aria2
export TRIM_PKGVAR=/vol1/@appdata/aria2
export TRIM_DATA_SHARE_PATHS="$APP"
printf '%s' '{"rpc_port":6801,"listen_port":6888,"dht_listen_port":6888,"dht_message_timeout":"10","enable_dht":true,"enable_dht6":true,"download_root":"'"$APP"'","archive_enabled":true,"archive_root":"'"$APP"'/completed","special_mode":"move","authorized_roots":["'"$APP"'"],"tracker_source":"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt","tracker_auto_update":false,"config_version":1}' > /vol1/@appconf/aria2/config.json
printf 'attribution-placeholder-secret-0123456789abc\n' > /vol1/@appconf/aria2/rpc.secret
chown aria2:aria2 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret
chmod 600 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret

echo "### 用例：Core 立刻挂掉时，病因应被归集进 startup.log"
/var/apps/aria2/cmd/main start > /tmp/start.out 2> /tmp/start.err
rc=$?
echo "  start 退出码: $rc"
[ "$rc" -ne 0 ] && ok "就绪门禁失败时按预期中止，而不是假装启动成功" || bad "启动竟然成功了，测试构造失效"
[ -s "$S" ] && ok "startup.log 已生成" || bad "startup.log 缺失"
grep -q 'Core 自检通过' "$S" 2>/dev/null && ok "预检通过留痕（说明假 Core 确实绕过了预检）" || bad "缺少预检留痕"
grep -q '未能就绪' "$S" 2>/dev/null && ok "记录了就绪门禁失败" || bad "未记录就绪门禁失败"
grep -q 'aria2.log 尾部' "$S" 2>/dev/null && ok "归集了 aria2.log 尾部" || bad "未归集 aria2.log 尾部"
grep -q 'libsomething.so.9' "$S" 2>/dev/null && ok "真实病因文字进入 startup.log，可直接据此定位" \
    || bad "病因没有被归集，用户仍然只看到「无法启动」"
grep -q 'manager.log 尾部' "$S" 2>/dev/null && ok "同时归集了 manager.log 尾部" || bad "未归集 manager.log 尾部"
echo "  --- startup.log 全文 ---"
sed 's/^/  | /' "$S" 2>/dev/null
[ -e /vol1/@appdata/aria2/aria2.pid ] && bad "回滚后仍残留 aria2.pid" || ok "回滚清理干净"
[ -e /vol1/@appdata/aria2/.start.lock ] && bad "启动失败后残留 .start.lock" || ok "启动失败同样释放了启动锁"

echo
if [ "$fail" -eq 0 ]; then echo "启动失败病因归集验证通过。"; else echo "共 $fail 处失败"; fi
exit $fail
