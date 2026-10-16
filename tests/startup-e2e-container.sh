#!/bin/bash
# 用解包后的真实发布产物跑一遍 fpk/cmd/main 的 start 分支，验证启动自愈、startup.log 与
# 服务用户边界在"生命周期脚本 + 管理器 + aria2c"三者串联时确实生效。
#
# 仅在 debian:12-slim 容器内执行。目录布局与 TRIM_ 变量按设备上的实际安装结构复刻。
# 该镜像没有 procps，因此进程与属主断言一律走 pid 文件 + /proc，而不是 pgrep/ps。
set -u
PKG=${PKG:-/hosttmp/pkg094}
fail=0
ok() { echo "  [ok] $*"; }
bad() { echo "  [FAIL] $*"; fail=$((fail + 1)); }

prepare_tree() {
    rm -rf /var/apps/aria2 /vol1 /run/aria2
    mkdir -p /var/apps/aria2/cmd /var/apps/aria2/target
    cp -R "$PKG/app/." /var/apps/aria2/target/
    cp -R "$PKG/cmd/." /var/apps/aria2/cmd/
    [ -f "$PKG/manifest" ] && cp "$PKG/manifest" /var/apps/aria2/manifest
    # 归档里 cmd/* 是 666，真实安装由应用中心补上可执行位，这里照做以免误判成回调没跑。
    chmod 755 /var/apps/aria2/cmd/*
    chmod 755 /var/apps/aria2/target/server/aria2c /var/apps/aria2/target/server/aria2-manager
    mkdir -p /vol1/@appconf/aria2 /vol1/@appdata/aria2 /vol1/@appshare/aria2/downloads
    mkdir -p /vol1/1000/download /vol1/@appcenter/aria2
    id aria2 > /dev/null 2>&1 || adduser --system --group --uid 981 --no-create-home aria2 > /dev/null 2>&1
    chown -R aria2:aria2 /vol1/@appconf/aria2 /vol1/@appdata/aria2
    chown -R aria2:aria2 /vol1/@appshare/aria2 /vol1/1000/download
    export TRIM_APPDEST=/vol1/@appcenter/aria2
    export TRIM_PKGETC=/vol1/@appconf/aria2
    export TRIM_PKGVAR=/vol1/@appdata/aria2
    export TRIM_DATA_SHARE_PATHS=/vol1/1000/download
}

write_secrets() {
    printf '%s\n' "$1" > /vol1/@appconf/aria2/config.json
    # 管理器要求 RPC 密钥至少 32 字符，这里给一个足够长的占位值，不是真实凭据。
    printf 'e2e-placeholder-secret-value-0123456789abcdef\n' > /vol1/@appconf/aria2/rpc.secret
    chown aria2:aria2 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret
    chmod 600 /vol1/@appconf/aria2/config.json /vol1/@appconf/aria2/rpc.secret
}

HEALTHY='{"rpc_port":6801,"listen_port":6888,"dht_listen_port":6888,"dht_message_timeout":"10","enable_dht":true,"enable_dht6":true,"download_root":"/vol1/1000/download","archive_enabled":true,"archive_root":"/vol1/1000/download/completed","special_mode":"move","authorized_roots":["/vol1/1000/download"],"tracker_source":"https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt","tracker_auto_update":true,"config_version":1}'

run_start() {
    rm -f /vol1/@appdata/aria2/startup.log
    /var/apps/aria2/cmd/main start > /tmp/start.out 2> /tmp/start.err
    rc=$?
    echo "  start 退出码: $rc"
    [ -s /tmp/start.err ] && sed 's/^/  stderr: /' /tmp/start.err
    echo "  --- startup.log ---"
    if [ -s /vol1/@appdata/aria2/startup.log ]; then
        sed 's/^/  | /' /vol1/@appdata/aria2/startup.log
    else
        echo "  | (不存在)"
    fi
    return $rc
}

# running_as_app_user 用 pid 文件与 /proc/<pid>/status 判定，容器无 procps 也照样可靠。
running_as_app_user() {
    name="$1"
    pid=$(cat "/vol1/@appdata/aria2/$name.pid" 2> /dev/null)
    [ -n "$pid" ] || { echo "nopid"; return 1; }
    kill -0 "$pid" 2> /dev/null || { echo "dead"; return 1; }
    uid=$(awk '/^Uid:/{print $2; exit}' "/proc/$pid/status" 2> /dev/null)
    want=$(id -u aria2 2> /dev/null)
    if [ -n "$uid" ] && [ "$uid" = "$want" ]; then
        echo "$pid"
        return 0
    fi
    echo "wronguid:$uid"
    return 1
}

check_processes() {
    label="$1"
    for n in aria2 manager; do
        r=$(running_as_app_user "$n")
        if [ $? -eq 0 ]; then
            ok "$label：$n 由 aria2 用户运行（pid $r）"
        else
            bad "$label：$n 未按预期运行（$r）"
        fi
    done
    [ -S /run/aria2/aria2.sock ] && ok "$label：网关 Socket 就绪" || bad "$label：网关 Socket 缺失"
}

echo "### 用例 1：健康配置应正常启动并记录启动完成"
prepare_tree
write_secrets "$HEALTHY"
run_start || bad "用例 1 start 非零退出"
S=/vol1/@appdata/aria2/startup.log
grep -q '启动完成' "$S" 2> /dev/null && ok "startup.log 记录了启动完成" || bad "startup.log 没有成功记录"
grep -q '自动修复' "$S" 2> /dev/null && bad "健康配置被自愈改写" || ok "健康配置未触发自愈"
# 启动锁只应在启动过程内存在。残留会让应用数据目录里永久留下 root 属主的目录，
# 而 stop 分支无权替 start 清理它，见 fpk/cmd/main 的 trap。
[ -e /vol1/@appdata/aria2/.start.lock ] && bad "启动成功后残留 .start.lock" || ok "启动成功后已释放启动锁"
# 两份 pid 文件由 start-stop-daemon 以 root 建立，cmd/main 需交回专属用户，
# 否则应用数据目录里会混进 root 属主文件，「清空应用配置」的删除范围不再单一。
for pf in aria2 manager; do
    owner=$(stat -c '%U' "/vol1/@appdata/aria2/$pf.pid" 2> /dev/null || echo 缺失)
    [ "$owner" = "aria2" ] && ok "$pf.pid 属主为 aria2" || bad "$pf.pid 属主是 $owner，未交回专属用户"
done
grep -q '"rpc_port": 6801' /vol1/@appconf/aria2/config.json && ok "用户端口 6801 原样保留" || bad "端口被改写（见 ADR-006）"
check_processes "用例 1"
head -n 20 /vol1/@appdata/aria2/aria2.log | grep -qi 'error while loading shared libraries' && bad "aria2c 缺共享库" || ok "aria2c 依赖完整"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true
sleep 1
[ -e /vol1/@appdata/aria2/aria2.pid ] && bad "stop 后仍残留 aria2.pid" || ok "stop 清理干净"
grep -q '"rpc_port": 6801' /vol1/@appconf/aria2/config.json && ok "stop 回写保留修复结果" || bad "stop 把配置写坏了"

echo "### 用例 2：下载目录与授权目录双双失效（换机复现）应自愈后照常启动"
prepare_tree
write_secrets '{"rpc_port":6801,"listen_port":6888,"dht_listen_port":6888,"dht_message_timeout":"10","enable_dht":true,"enable_dht6":true,"download_root":"/vol1/1000/no-such-share","archive_enabled":true,"archive_root":"/vol1/1000/no-such-share/completed","special_mode":"move","authorized_roots":["/tmp/h1/locked","/vol1/1000/download"],"tracker_source":"http://insecure/trackers.txt","tracker_auto_update":true,"config_version":1}'
run_start || bad "坏配置仍未能自愈启动，死锁没有解除"
grep -q '在当前机器上不存在' "$S" 2> /dev/null && ok "记录了下载目录失效" || bad "缺少下载目录失效说明"
grep -q '移出授权列表' "$S" 2> /dev/null && ok "记录了越界授权根被移出" || bad "缺少授权根说明"
grep -q '不合法' "$S" 2> /dev/null && ok "记录了非 https Tracker 源" || bad "缺少 Tracker 源说明"
grep -q '启动完成' "$S" 2> /dev/null && ok "坏配置仍然完成启动，界面可进（死锁已解除）" || bad "仍然没能启动"
grep -q '/tmp/h1/locked' /vol1/@appconf/aria2/config.json && bad "越界目录仍留在持久配置里" || ok "修复结果已持久化"
grep -q '"rpc_port": 6801' /vol1/@appconf/aria2/config.json && ok "自愈没有顺带改端口" || bad "端口被自愈改写"
grep -q '"archive_root": "/vol1/1000/download/completed"' /vol1/@appconf/aria2/config.json && ok "归档目录随新下载目录重新派生" || bad "归档目录未跟随（会被静默关闭）"
grep -q '"config_version": 1' /vol1/@appconf/aria2/config.json && ok "config_version 未被抹掉" || bad "config_version 丢失（见 ADR-011）"
check_processes "用例 2"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true

echo "### 用例 3：持久配置缺失时应带原因中止，而不是无声失败"
prepare_tree
write_secrets "$HEALTHY"
rm -f /vol1/@appconf/aria2/config.json
/var/apps/aria2/cmd/main start > /dev/null 2>&1
echo "  start 退出码: $?"
grep -q '不可读' "$S" 2> /dev/null && ok "startup.log 给出了缺失原因" || bad "缺失持久配置时无任何说明"

echo "### 用例 4：损坏的配置文本应被自愈成出厂默认并照常启动"
prepare_tree
write_secrets '{"rpc_port":6801,"listen_port":'
run_start || bad "半截 JSON 未能自愈"
grep -q '无法解析' "$S" 2> /dev/null && ok "记录了配置不可解析" || bad "缺少配置解析失败说明"
grep -q '启动完成' "$S" 2> /dev/null && ok "半截 JSON 也能启动" || bad "未能启动"
check_processes "用例 4"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true

echo "### 用例 5：RPC 与 BT 端口同为 6888 时应解除冲突并启动"
prepare_tree
write_secrets '{"rpc_port":6888,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"move","authorized_roots":[],"config_version":1}'
run_start || bad "端口自冲突未能自愈，仍卡在无法启动"
grep -q '冲突' "$S" 2> /dev/null && ok "记录了端口冲突修复" || bad "缺少端口冲突说明"
grep -q '启动完成' "$S" 2> /dev/null && ok "端口自冲突也能启动" || bad "未能启动"
check_processes "用例 5"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true

echo "### 用例 6：整机无可用共享目录时应落到私有目录且归档仍开启"
prepare_tree
export TRIM_DATA_SHARE_PATHS=/vol1/9999/gone
write_secrets '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"move","authorized_roots":[],"config_version":1}'
run_start || bad "无共享目录时未能兜底启动"
grep -q '没有可用的飞牛数据共享目录' "$S" 2> /dev/null && ok "记录了兜底降级" || bad "缺少兜底说明"
grep -q '"download_root": "/vol1/@appdata/aria2/downloads"' /vol1/@appconf/aria2/config.json && ok "下载目录落到应用私有目录" || bad "兜底目录位置不对"
grep -q '"special_mode": "move"' /vol1/@appconf/aria2/config.json && ok "自动归档没有被静默关闭" || bad "兜底启动关掉了自动归档"
[ -d /vol1/@appdata/aria2/downloads ] && ok "私有下载目录已创建" || bad "私有下载目录缺失"
check_processes "用例 6"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true

echo "### 用例 7：startup.log 轮转上限"
prepare_tree
write_secrets "$HEALTHY"
: > /vol1/@appdata/aria2/startup.log
i=0
while [ $i -lt 320 ]; do
    echo "filler line $i" >> /vol1/@appdata/aria2/startup.log
    i=$((i + 1))
done
/var/apps/aria2/cmd/main start > /dev/null 2>&1
n=$(wc -l < /vol1/@appdata/aria2/startup.log 2> /dev/null || echo 0)
echo "  startup.log 行数: $n"
[ "$n" -le 210 ] && ok "startup.log 被限制在 200 行附近" || bad "startup.log 未被截断（$n 行）"
[ -e /vol1/@appdata/aria2/startup.log.tmp ] && bad "残留 startup.log.tmp" || ok "无临时文件残留"
/var/apps/aria2/cmd/main stop > /dev/null 2>&1 || true

echo "### 用例 8：卸载「清空应用配置」应删掉 startup.log"
prepare_tree
write_secrets "$HEALTHY"
/var/apps/aria2/cmd/main start > /dev/null 2>&1
[ -s "$S" ] && ok "startup.log 已生成" || bad "startup.log 未生成"
wizard_keep_config=no /var/apps/aria2/cmd/uninstall_callback > /dev/null 2>&1
[ -e "$S" ] && bad "清空配置后 startup.log 仍存在" || ok "startup.log 已随清空配置删除"
[ -e /vol1/@appconf/aria2/config.json ] && bad "清空配置后仍有残留" || ok "持久配置已清除"
[ -d /vol1/1000/download ] && ok "用户下载目录未被删除" || bad "误删用户数据目录"

echo
if [ "$fail" -eq 0 ]; then
    echo "真实产物启动链路验证全部通过。"
else
    echo "共 $fail 处失败"
fi
exit $fail
