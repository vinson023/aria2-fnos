#!/bin/bash
# 校验 fpk/cmd/main 的 merge_runtime_options：覆盖生效、不产生重复键、独占项与损坏行被拒绝。
set -u

src_main="${1:?用法: shell-merge-runtime-options.sh <fpk/cmd/main 路径>}"
src_dir=$(CDPATH= cd -- "$(dirname -- "$src_main")" && pwd)
src_main="$src_dir/$(basename -- "$src_main")"
[ -f "$src_main" ] || { echo "FAIL 找不到 $src_main"; exit 1; }

work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT
export TRIM_PKGVAR="$work"

# 直接提取生命周期脚本里的真实函数，避免测试与实现各写一份逻辑。
awk '/^merge_runtime_options\(\) \{/,/^}$/' "$src_main" > "$work/mro.sh"
if [ ! -s "$work/mro.sh" ]; then
    echo "FAIL 未能从生命周期脚本提取 merge_runtime_options"
    exit 1
fi
. "$work/mro.sh"

fail=0
key_count() { sed -n "s/^\($1\)=.*/\1/p" "$work/aria2.conf" | wc -l | tr -d ' '; }

# 用例一：普通覆盖项替换同名默认值，其余行与注释保持不变。
cat > "$work/aria2.conf" <<CONF
dir=/vol1/@appshare/aria2/downloads
rpc-listen-port=6800
rpc-secret=supersecretvalue
max-overall-upload-limit=2M
max-concurrent-downloads=5
split=64
# 注释行必须原样保留
bt-request-peer-speed-limit=10M
CONF
printf 'max-concurrent-downloads=12\nsplit=8\n' > "$work/aria2-overrides.conf"
merge_runtime_options
for key in max-concurrent-downloads split; do
    n=$(key_count "$key")
    if [ "$n" != 1 ]; then echo "FAIL $key 出现 $n 次，期望 1 次"; fail=1; fi
done
grep -qx 'max-concurrent-downloads=12' "$work/aria2.conf" || { echo 'FAIL 未采用覆盖取值 12'; fail=1; }
grep -qx 'split=8' "$work/aria2.conf" || { echo 'FAIL 未采用覆盖取值 8'; fail=1; }
grep -qx 'dir=/vol1/@appshare/aria2/downloads' "$work/aria2.conf" || { echo 'FAIL 误删未覆盖的 dir'; fail=1; }
grep -qx 'rpc-secret=supersecretvalue' "$work/aria2.conf" || { echo 'FAIL 误删未覆盖的 rpc-secret'; fail=1; }
grep -qx 'max-overall-upload-limit=2M' "$work/aria2.conf" || { echo 'FAIL 误删未覆盖的限速默认值'; fail=1; }
grep -qx '# 注释行必须原样保留' "$work/aria2.conf" || { echo 'FAIL 注释未保留'; fail=1; }
if [ "$(key_count rpc-secret)" != 1 ]; then echo 'FAIL rpc-secret 行数异常'; fail=1; fi

# 用例二：独占项、注入行和损坏行一律拒绝，且绝不删除对应的默认值。
cat > "$work/aria2.conf" <<CONF
dir=/vol1/1000/download
rpc-listen-port=6800
rpc-secret=supersecretvalue
on-download-complete=/vol1/@appdata/aria2/server/aria2-hook-complete
save-session=/vol1/@appdata/aria2/aria2.session
log=/vol1/@appdata/aria2/aria2.log
enable-dht=true
dht-listen-port=6888
bt-tracker=http://before.invalid/announce
CONF
cp "$work/aria2.conf" "$work/aria2.conf.expected"
printf 'dir=/evil\nrpc-secret=attacker\nrpc-listen-port=1\non-download-complete=/bin/sh\nsave-session=/tmp/x\nlog=/tmp/x\nenable-dht=false\ndht-listen-port=1\nbt-tracker=http://evil/announce\nnonkey\n=missing-key\n' > "$work/aria2-overrides.conf"
merge_runtime_options
if ! cmp -s "$work/aria2.conf" "$work/aria2.conf.expected"; then
    echo 'FAIL 独占项或损坏行未被完全拒绝，aria2.conf 被改写'
    diff -u "$work/aria2.conf.expected" "$work/aria2.conf" | sed 's/^/      /'
    fail=1
fi
if [ -e "$work/aria2.conf.merged" ] || [ -e "$work/aria2.conf.overrides" ]; then
    echo 'FAIL 合并后残留临时文件'
    fail=1
fi

# 用例三：覆盖文件为空或不存在时，配置文件保持原样。
printf 'max-overall-upload-limit=2M\n' > "$work/aria2.conf"
: > "$work/aria2-overrides.conf"
merge_runtime_options
grep -qx 'max-overall-upload-limit=2M' "$work/aria2.conf" || { echo 'FAIL 空覆盖文件改动了配置'; fail=1; }
rm -f "$work/aria2-overrides.conf"
merge_runtime_options
grep -qx 'max-overall-upload-limit=2M' "$work/aria2.conf" || { echo 'FAIL 覆盖文件缺失时改动了配置'; fail=1; }

# 用例四：取值里的等号与逗号原样保留，bt-prioritize-piece 这类选项依赖该行为。
printf 'disk-cache=32M\n' > "$work/aria2.conf"
printf 'bt-prioritize-piece=head=32M,tail=32M\n' > "$work/aria2-overrides.conf"
merge_runtime_options
grep -qx 'bt-prioritize-piece=head=32M,tail=32M' "$work/aria2.conf" || { echo 'FAIL 含等号的取值被破坏'; fail=1; }
grep -qx 'disk-cache=32M' "$work/aria2.conf" || { echo 'FAIL 误删未覆盖项'; fail=1; }

if [ "$fail" = 0 ]; then
    echo 'PASS merge_runtime_options 全部用例通过'
fi
exit "$fail"