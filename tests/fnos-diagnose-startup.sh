#!/bin/bash
# Aria2（飞牛 fnOS FPK）启动失败一次性诊断脚本。
#
# 用途：某些机器安装后只提示“无法启动”，而应用中心不显示原因。本脚本把启动自愈日志、
# 配置里的机器相关取值、目录权限、端口占用、进程与网关 Socket 状态一次性收齐，
# 把完整输出发回开发侧即可定位。
#
# 严格只读：不创建、不修改、不删除任何文件，不重启应用。在设备上以 root 直接执行。
set -u

APP=aria2
ok=0
warn=0
bad=0
say_ok() { echo "[正常] $*"; ok=$((ok + 1)); }
say_warn() { echo "[注意] $*"; warn=$((warn + 1)); }
say_bad() { echo "[异常] $*"; bad=$((bad + 1)); }
section() { echo ""; echo "===== $* ====="; }

section "0. 机器上下文"
echo "主机名: $(uname -n)"
echo "内核: $(uname -sr)"
echo "架构: $(uname -m)"
echo "glibc: $(ldd --version 2>/dev/null | head -n 1)"
if [ -r /etc/os-release ]; then
    grep -E '^(NAME|VERSION)=' /etc/os-release | sed 's/^/系统: /'
fi

# 卷号因机器而异，必须探测，禁止硬编码 /vol1。
confd=""
vard=""
sharepath=""
for v in /vol[0-9]*; do
    if [ -d "$v/@appconf/$APP" ]; then confd="$v/@appconf/$APP"; fi
    if [ -d "$v/@appdata/$APP" ]; then vard="$v/@appdata/$APP"; fi
    if [ -d "$v/@appshare/$APP" ]; then sharepath="$v/@appshare/$APP"; fi
done
echo "TRIM_PKGETC: ${confd:-未找到}"
echo "TRIM_PKGVAR: ${vard:-未找到}"
echo "@appshare:   ${sharepath:-未找到}"
if [ -z "$confd" ] || [ -z "$vard" ]; then
    say_bad "应用配置或数据目录缺失，说明安装未完成或数据卷未挂载"
fi

target="/var/apps/$APP/target"
server="$target/server"

section "1. 安装内容与版本"
# manifest 用 "key = value" 形式，等号两侧有空格，正则必须容忍空白。
if [ -r "/var/apps/$APP/manifest" ]; then
    grep -E '^(appname|version|platform|display_name)[[:space:]]*=' "/var/apps/$APP/manifest" | tr -s ' ' | sed 's/^/manifest: /'
else
    say_warn "读不到 /var/apps/$APP/manifest"
fi
for f in aria2c aria2-manager aria2-hook; do
    if [ -x "$server/$f" ]; then
        say_ok "$f 可执行，大小 $(stat -c %s "$server/$f" 2>/dev/null) 字节"
    else
        say_bad "$f 缺失或不可执行：$server/$f"
    fi
done
# 生命周期脚本混入 CRLF 会让 bash 报 bad interpreter，换机场景下出现过。
cr=$(printf '\r')
for f in main start stop status install_callback uninstall_callback; do
    if [ -f "$target/cmd/$f" ] && grep -q "$cr" "$target/cmd/$f"; then
        say_bad "cmd/$f 含 CRLF 换行，bash 无法解释该脚本"
    fi
done

section "2. 专属应用用户"
if id "$APP" > /dev/null 2>&1; then
    say_ok "$(id "$APP")"
else
    say_bad "应用用户 $APP 不存在，安装阶段建用户失败，必然无法启动"
fi

section "3. 本机 CPU 与运行时兼容性自检"
if [ -x "$server/aria2c" ]; then
    # 判定只用设备上确实存在的工具：fnOS 没有 readelf/objdump/nm，只有 file 与 ldd。
    # 早期版本在这里用 readelf -h / readelf -d，命令不存在时输出为空，
    # 于是把动态 Core 误报成「静态链接、不依赖宿主库」，是假阳性，必须避免。
    file_out=$(file "$server/aria2c" 2>/dev/null)
    echo "aria2c 镜像: ${file_out:-file 命令不可用}"
    link_kind=$(echo "$file_out" | grep -oE 'statically linked|dynamically linked' | head -n 1)
    arch=$(echo "$file_out" | grep -oE 'ELF 32-bit|ELF 64-bit' | head -n 1)
    machine=$(echo "$file_out" | grep -oE 'x86-64|ARM aarch64|AArch64|80386' | head -n 1)
    if [ -n "$arch" ] && [ -n "$machine" ]; then
        echo "架构: $arch $machine / 本机 CPU: $(uname -m)"
    fi
    if [ "$link_kind" = "statically linked" ]; then
        # 自 0.1.95 起 Core 应为静态链接（ADR-016）：不依赖宿主任何共享库。
        say_ok "aria2c 为静态链接，不依赖宿主共享库"
    elif [ "$link_kind" = "dynamically linked" ]; then
        echo "aria2c 为动态链接，逐个检查其共享库在本机是否可得"
        ldd_out=$(ldd "$server/aria2c" 2>&1)
        echo "$ldd_out" | sed 's/^/  /'
        miss=$(echo "$ldd_out" | grep -i 'not found' | tr -s ' \t' ' ' | tr '\n' ';')
        if [ -n "$miss" ]; then
            say_bad "aria2c 缺失共享库:$miss —— 本机系统库集合不满足动态 Core，需升级到静态 Core 的版本"
        else
            say_ok "aria2c 所需共享库本机全部具备"
        fi
    else
        # file 不可用或不认时退回 ldd：静态产物会报 not a dynamic executable。
        ldd_probe=$(ldd "$server/aria2c" 2>&1)
        if echo "$ldd_probe" | grep -qi 'not a dynamic executable'; then
            say_ok "aria2c 为静态链接（依 ldd 判定），不依赖宿主共享库"
        elif echo "$ldd_probe" | grep -qi 'not found'; then
            echo "$ldd_probe" | sed 's/^/  /'
            say_bad "aria2c 存在缺失的共享库，见上方 not found 行"
        else
            say_bad "无法判定 aria2c 的链接方式（file 与 ldd 均无有效结论）"
        fi
    fi
    out=$(su -s /bin/bash "$APP" -c "$server/aria2c --version" 2>&1 | head -n 1)
    if [ -n "$out" ]; then
        say_ok "aria2c 自检：$out"
    else
        err=$(su -s /bin/bash "$APP" -c "$server/aria2c --version" 2>&1 | head -n 3 | tr '\n' ' ')
        say_bad "aria2c 无法在本机执行：${err:-无输出} —— 多为 CPU 指令集、架构或缺失共享库"
    fi
fi
out=$(su -s /bin/bash "$APP" -c "$server/aria2-manager --help" 2>&1 | head -n 1)
if [ -n "$out" ]; then
    say_ok "manager 自检：$out"
else
    say_bad "aria2-manager 无法在本机执行"
fi

section "4. 启动自愈与失败原因（最关键的一节）"
if [ -n "$vard" ] && [ -s "$vard/startup.log" ]; then
    echo "--- startup.log 最近 40 行 ---"
    tail -n 40 "$vard/startup.log"
    echo "--- 其中的修复与中止记录 ---"
    grep -E '自动修复|中止|失败|无法' "$vard/startup.log" | tail -n 15
    if grep -q '启动完成' "$vard/startup.log"; then
        say_ok "startup.log 里存在成功启动记录"
    else
        say_bad "startup.log 中没有任何成功启动记录，请把第 4 节全文发回"
    fi
else
    say_warn "没有 $vard/startup.log（早于 0.1.94 的版本不落这份日志，或从未进入启动流程）"
fi

section "5. 运行日志尾部"
for f in manager.log aria2.log; do
    echo "--- $f 最近 20 行 ---"
    if [ -n "$vard" ] && [ -s "$vard/$f" ]; then
        tail -n 20 "$vard/$f" | sed 's/\x1b\[[0-9;]*m//g'
    else
        echo "（空或不存在）"
    fi
done

section "6. 配置取值与目录可达性"
cfg="$confd/config.json"
# config.json 是缩进过的 JSON，键与值之间有空格，因此先取 "键" 到下一个逗号/大括号，
# 再统一去掉引号与空白，按第一个冒号切值。
field() {
    if [ ! -r "$cfg" ]; then
        return 0
    fi
    grep -o "\"$1\"[[:space:]]*:[^,}]*" "$cfg" | head -n 1 | tr -d ' \t"' | cut -d: -f2-
}
if [ -r "$cfg" ]; then
    say_ok "持久配置可读：$cfg"
    for key in rpc_port listen_port dht_listen_port dht_message_timeout enable_dht enable_dht6 special_mode download_root archive_root tracker_source tracker_auto_update config_version; do
        echo "config: $key = $(field "$key")"
    done
    grep -o '"authorized_roots"[[:space:]]*:[[:space:]]*\[[^]]*\]' "$cfg" | tr -s ' ' | sed 's/^/config: /'
else
    say_bad "持久配置不可读：$cfg"
fi
dl=$(field download_root)
if [ -n "$dl" ]; then
    if [ -d "$dl" ]; then
        echo "下载目录 $dl 的 $(stat -c '属主/权限: %U:%G %a' "$dl" 2>/dev/null)"
        if su -s /bin/bash "$APP" -c "test -w '$dl' && echo writable" 2>/dev/null | grep -q writable; then
            say_ok "专属用户 $APP 对下载目录可写"
        else
            say_bad "专属用户 $APP 对下载目录不可写，这是最常见的无法启动原因"
        fi
        df -P "$dl" 2>/dev/null | tail -n 1 | tr -s ' ' | sed 's/^/磁盘用量: /'
    else
        say_bad "配置里的下载目录不存在：$dl（0.1.94 起会自动改用默认目录并记录原因）"
    fi
fi
ar=$(field archive_root)
if [ -n "$ar" ]; then
    if [ -d "$ar" ]; then
        say_ok "归档目录存在：$ar"
    else
        say_warn "归档目录尚不存在，启动时应自动创建：$ar"
    fi
fi

section "7. 飞牛数据共享目录上下文"
mpid=$(cat "$vard/manager.pid" 2>/dev/null)
apid=$(cat "$vard/aria2.pid" 2>/dev/null)
if [ -n "${mpid:-}" ] && [ -r "/proc/$mpid/environ" ]; then
    tr '\0' '\n' < "/proc/$mpid/environ" | grep -E '^TRIM_(DATA_SHARE_PATHS|DATA_ACCESSIBLE_PATHS|PKGVAR|PKGETC|APPDEST|APPVER|APP_STATUS|USERNAME|UID)=' | sed 's/^/environ: /'
    # API token 属于凭据，只确认存在，绝不输出取值。
    tr '\0' '\n' < "/proc/$mpid/environ" | grep -q '^TRIM_API_TOKEN=' && echo "environ: TRIM_API_TOKEN 已注入（值略）"
else
    say_warn "管理器未在运行，取不到生命周期注入的 TRIM_ 上下文"
fi
if mount | grep -qE ' on /vol[0-9]+'; then
    mount | grep -E ' on /vol[0-9]+' | sed 's/^/挂载: /'
else
    say_bad "没有已挂载的数据卷，换盘或硬盘离线会直接导致无法启动"
fi
ls -d /vol[0-9]*/[0-9]* 2>/dev/null | head -n 20 | sed 's/^/用户数据目录: /'

section "8. 端口占用（区分本应用自身与真正的冲突）"
rpc=$(field rpc_port)
bt=$(field listen_port)
dht=$(field dht_listen_port)
[ -n "$rpc" ] || rpc=6800
[ -n "$bt" ] || bt=6888
[ -n "$dht" ] || dht="$bt"
if ! command -v ss > /dev/null 2>&1; then
    say_warn "设备无 ss 命令，跳过端口检查"
else
    dedup=""
    for p in "$rpc" "$bt" "$dht"; do
        case " $dedup " in *" $p "*) continue ;; esac
        dedup="$dedup $p"
        # 用 sport 过滤本机监听端口：直接 grep ":$p" 会误配对端地址列，也可能撞上别的端口号。
        rows=$(ss -lntupH sport = :"$p" 2>/dev/null)
        [ -n "$rows" ] || continue
        foreign=""
        for x in $(echo "$rows" | grep -o 'pid=[0-9]*' | cut -d= -f2 | sort -u); do
            if [ "$x" != "${mpid:-}" ] && [ "$x" != "${apid:-}" ]; then
                foreign="$foreign $x"
            fi
        done
        echo "$rows" | tr -s ' ' | cut -c1-140 | sed "s/^/端口 $p: /"
        if [ -n "$foreign" ]; then
            say_warn "端口 $p 被本应用之外的进程占用（pid$foreign）。0.1.94 起启动时会自动改用其他端口"
            continue
        fi
        say_ok "端口 $p 由本应用自身监听"
        # RPC 只允许绑定回环是安全边界（ADR-007）。这里只匹配"本地地址:本端口"这一个整词，
        # 避免把对端地址列的 0.0.0.0:* 误判成对外开放。
        if echo "$rows" | grep -qE "(^|[[:space:]])(\*|0\.0\.0\.0|\[::\]):$p([[:space:]]|$)"; then
            if [ "$p" = "$rpc" ]; then
                say_bad "RPC 端口 $p 绑在了通配地址，违反只监听回环的安全边界"
            else
                echo "说明: BT/DHT 端口 $p 监听所有网卡属预期行为"
            fi
        fi
    done
fi

section "9. 进程与网关 Socket"
sock="/run/$APP/$APP.sock"
if [ -S "$sock" ]; then
    say_ok "网关 Socket 就绪：$sock"
else
    say_bad "网关 Socket 不存在：$sock"
fi
for name in manager aria2; do
    pid=$(cat "$vard/$name.pid" 2>/dev/null)
    if [ -n "${pid:-}" ] && kill -0 "$pid" 2>/dev/null; then
        say_ok "$name 正在运行，pid $pid"
    else
        say_bad "$name 未运行，pidfile 内容：${pid:-无}"
    fi
done

section "10. 结论"
echo "统计：正常 $ok 项，注意 $warn 项，异常 $bad 项"
if [ "$bad" -gt 0 ]; then
    echo "请把本脚本的完整输出发回开发侧，重点是第 4 节 startup.log 与第 6 节目录可达性。"
else
    echo "未发现问题。若界面仍打不开，请另外补发应用中心的应用详情页与界面截图。"
fi
exit 0
