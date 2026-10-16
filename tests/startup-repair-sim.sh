#!/bin/bash
# 启动自愈的容器内行为验证：用真机复现出的几份坏配置，确认 aria2-manager 在
# --ensure-ports 探针路径下必定成功返回可用配置，并把每一条改动打到 stderr。
# 仅在 debian:12-slim 容器里执行，不接触设备上的真实应用目录。
set -u
MGR=/hosttmp/aria2-manager
Pkgvar=/sim/pkgvar
Pcgetc=/sim/pkgetc
fail=0
ok() { echo "  [ok] $*"; }
bad() { echo "  [FAIL] $*"; fail=$((fail + 1)); }

setup() {
    rm -rf /sim /vol1
    mkdir -p "$Pkgvar" "$Pcgetc" /vol1/1000/download
    export TRIM_PKGVAR="$Pkgvar" TRIM_PKGETC="$Pcgetc"
    export TRIM_DATA_SHARE_PATHS="$1"
}

run_case() {
    name="$1"; cfg="$2"
    printf '%s\n' "$cfg" > /sim/config.json
    echo "===== $name ====="
    out=$(ARIA2PRO_CONFIG_JSON="$cfg" "$MGR" --config /sim/config.json --config-env ARIA2PRO_CONFIG_JSON --ensure-ports 2>/sim/err)
    rc=$?
    echo "  退出码: $rc"
    sed 's/^/  修复: /' /sim/err
    # 断言一律针对 stdout：--ensure-ports 只在端口真正变化时回写文件，落盘的 JSON 未必是最新值。
    outc=$(echo "$out" | tr -d ' \t\n')
    echo "  结果: $(echo "$outc" | grep -o '"download_root":"[^"]*"\|"rpc_port":[0-9]\+\|"listen_port":[0-9]\+\|"special_mode":"[^"]*"\|"authorized_roots":\[[^]]*\]' | tr '\n' ' ')"
    [ "$rc" -eq 0 ] || bad "$name 退出码非 0，应用仍然无法启动"
    [ -n "$out" ] || bad "$name stdout 为空，脚本会拿到空配置"
    # 返回值必须是合法 JSON：交给同一个管理器再解析一次作为校验。
    echo "$out" | "$MGR" --config-stdin --print-download-root > /dev/null 2>&1 \
        && ok "返回的配置可被再次解析" || bad "返回的不是合法配置"
    echo
}

echo "### 用例 A：健康配置不得产生任何自愈记录"
setup /vol1/1000/download
run_case "A 健康配置" '{"rpc_port":6801,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"move","authorized_roots":["/vol1/1000/download"],"config_version":1}'
grep -q '修复' /sim/err && bad "健康配置被改动了" || ok "健康配置原样返回"
echo "$outc" | grep -q '"rpc_port":6801' && ok "用户自定义端口 6801 被保留" || bad "端口被改掉了（见 ADR-006）"

echo "### 用例 B：真机复现——下载目录指向已不存在的共享目录"
setup /vol1/1000/download
run_case "B 目录消失" '{"rpc_port":6801,"listen_port":6888,"download_root":"/vol1/1000/no-such-share","special_mode":"move","authorized_roots":[],"config_version":1}'
grep -q '在当前机器上不存在' /sim/err && ok "给出了目录失效说明" || bad "缺少目录失效说明"
echo "$outc" | grep -q '"download_root":"/vol1/1000/download"' && ok "已回落到可用共享目录并落盘" || bad "未回落到共享目录"

echo "### 用例 C：真机复现——授权根不在数据卷上"
setup /vol1/1000/download
run_case "C 授权根越界" '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"move","authorized_roots":["/tmp/h1/locked"],"config_version":1}'
grep -q '移出授权列表' /sim/err && ok "逐条丢弃越界授权根而非整体拒绝" || bad "未处理越界授权根"
echo "$outc" | grep -q '/tmp/h1/locked' && bad "越界目录仍留在返回配置里" || ok "越界目录已清除"

echo "### 用例 D：配置半截/不是 JSON"
setup /vol1/1000/download
run_case "D1 空配置" ''
run_case "D2 半截 JSON" '{"rpc_port":6800,"listen_port":'
run_case "D3 端口自冲突" '{"rpc_port":6888,"listen_port":6888,"download_root":"/vol1/1000/download","config_version":1}'
run_case "D4 端口越界" '{"rpc_port":80,"listen_port":70000,"download_root":"/vol1/1000/download","config_version":1}'
run_case "D5 非法归档模式" '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"nonsense","config_version":1}'
run_case "D6 Tracker 非 https" '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/download","tracker_source":"http://evil/tracker.txt","config_version":1}'

echo "### 用例 E：整机没有可用数据共享目录（换盘/卷未挂载）"
setup /vol1/9999/gone
run_case "E 无共享目录" '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/1000/download","special_mode":"move","authorized_roots":[],"config_version":1}'
grep -q '没有可用的飞牛数据共享目录' /sim/err && ok "给出了兜底说明" || bad "缺少兜底说明"
fb=$(echo "$outc" | grep -o '"download_root":"[^"]*"' | cut -d'"' -f4)
case "$fb" in
    "$Pkgvar/downloads")
        [ -d "$fb" ] && ok "兜底目录已创建：$fb" || bad "兜底目录未创建"
        echo "$outc" | grep -q '"special_mode":"move"' && ok "兜底启动后自动归档仍保持开启" || bad "兜底启动把自动归档静默关掉了"
        ;;
    *) bad "兜底目录位置不对：$fb" ;;
esac

echo "### 用例 F：兜底目录已存在时的第二次启动（幂等 + 仍要留痕）"
export TRIM_PKGVAR="$Pkgvar" TRIM_PKGETC="$Pcgetc" TRIM_DATA_SHARE_PATHS=/vol1/8888/gone2
printf '%s\n' '{"rpc_port":6800,"listen_port":6888,"download_root":"/vol1/8888/gone2","special_mode":"move","authorized_roots":[],"config_version":1}' > /sim/config.json
out=$(ARIA2PRO_CONFIG_JSON="$(cat /sim/config.json)" "$MGR" --config /sim/config.json --config-env ARIA2PRO_CONFIG_JSON --ensure-ports 2>/sim/err); rc=$?
echo "  退出码: $rc"
sed 's/^/  修复: /' /sim/err
[ "$rc" -eq 0 ] || bad "F 退出码非 0"
grep -q '没有可用的飞牛数据共享目录' /sim/err && ok "沿用私有目录时同样留下说明" || bad "私有目录降级是静默的，日志无法解释"

echo
if [ "$fail" -eq 0 ]; then
    echo "全部用例通过：任何机器相关的坏配置都不会再让应用卡在无法启动。"
else
    echo "共 $fail 处失败"
fi
exit $fail