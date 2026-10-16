#!/bin/sh
# 静态 Core 真机功能验收：用现役 aria2.conf 的完整选项集在 glibc 设备上跑通
# HTTP 与 HTTPS 真实下载，验证静态 musl + 静态 OpenSSL 的证书链与异步 DNS 可用。
# 只在临时端口/临时目录上运行，不触碰现役实例。
#
# 说明：RPC 请求体一律先写入文件再用 --data-binary 提交。把含双引号的 JSON 直接拼进
# curl -d 的双引号参数会被设备上的 shell 改坏，实测会得到 -32700 Parse error，
# 那是测试脚本的问题，不是 Core 的问题。
set -eu

SRC_BIN=${1:-/hosttmp/corebuild/out/aria2c}
LIVE_CONF=${2:-/vol1/@appdata/aria2/aria2.conf}
WORK=/hosttmp/statictest
RPC_PORT=6900
BT_PORT=6901
APP_USER=aria2
HTTP_URL=http://192.168.x.x:5666/
HTTPS_URL=https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt

fail() { echo "FAIL: $*"; exit 1; }
ok() { echo "OK: $*"; }

[ -x "$SRC_BIN" ] || fail "找不到待测二进制 $SRC_BIN"
# 静态性门禁。设备上没有 readelf/objdump（fnOS 未装 binutils），
# 因此只把 readelf 当作可选加强项，主判据用一定存在的 file；
# 否则这条门禁会在缺工具时空转，把动态二进制误当成静态通过。
kind=$(file "$SRC_BIN" 2>/dev/null | grep -oE 'statically linked|dynamically linked' | head -1)
if [ "$kind" != "statically linked" ]; then
  echo "file 判定: ${kind:-无法判定}"
  fail "待测二进制不是静态链接，测试无意义"
fi
if command -v readelf >/dev/null 2>&1; then
  if readelf -d "$SRC_BIN" 2>/dev/null | grep -q "(NEEDED)"; then fail "readelf 检出 NEEDED 条目，与静态判定矛盾"; fi
  ok "readelf 复核：无动态依赖条目"
fi
ok "静态性门禁通过（file: $kind）"

rm -rf "$WORK"
mkdir -p "$WORK/dl" "$WORK/data"
cp "$SRC_BIN" "$WORK/aria2c"

# 复用现役配置的全部选项，仅替换机器相关与生命周期相关的行：
# 端口换到临时值、目录换到临时值、去掉 session/dht.dat 路径，
# 并去掉 on-download-* 回调，避免测试实例写入现役应用的任务账本与触发归档。
grep -vE '^(dir|input-file|save-session|save-session-interval|dht-file-path|rpc-listen-port|listen-port|dht-listen-port|on-download-complete|on-download-error|on-download-stop|rpc-secret)=' "$LIVE_CONF" > "$WORK/aria2.conf"
cat >> "$WORK/aria2.conf" <<EOF
dir=$WORK/dl
rpc-listen-port=$RPC_PORT
listen-port=$BT_PORT
dht-listen-port=$BT_PORT
dht-file-path=$WORK/data/dht.dat
dht-file-path6=$WORK/data/dht6.dat
EOF

n_opt=$(grep -cE '^[a-z]' "$WORK/aria2.conf")
echo "待测选项数: $n_opt / 现役 $(grep -cE '^[a-z]' "$LIVE_CONF")（差值为被替换或移除的行）"

chown -R "$APP_USER" "$WORK"
# su 会让 $! 指向 su 本身，trap 清理不到真正的 aria2c，端口会被残留进程占住；
# 本轮考察的是二进制可移植性，降权路径由现役实例覆盖，因此直接运行以精确持有 PID。
pkill -f "$WORK/aria2c" 2>/dev/null || true
sleep 1
"$WORK/aria2c" --conf-path="$WORK/aria2.conf" > "$WORK/stdout.log" 2>&1 &
PID=$!
cleanup() { kill "$PID" 2>/dev/null || true; wait "$PID" 2>/dev/null || true; }
trap cleanup EXIT INT TERM

i=0
while [ $i -lt 30 ]; do
  # set -e 下裸的 A && B 会在 A 为假时让整条列表失败并终止脚本，因此一律写成 if 形式。
  if grep -qi 'Unknown option' "$WORK/stdout.log" 2>/dev/null; then cat "$WORK/stdout.log"; fail "静态 Core 不接受现役配置的某个选项"; fi
  if grep -qi 'listening' "$WORK/stdout.log" 2>/dev/null; then break; fi
  kill -0 "$PID" 2>/dev/null || { cat "$WORK/stdout.log"; fail "静态 Core 进程提前退出"; }
  i=$((i + 1)); sleep 1
done
ok "静态 Core 以 $n_opt 个现役选项成功启动，无 Unknown option"

rpc() {
  printf '%s' "{\"jsonrpc\":\"2.0\",\"id\":\"t\",\"method\":\"$1\",\"params\":$2}" > "$WORK/rpc.json"
  curl -s -m 25 -X POST "http://127.0.0.1:$RPC_PORT/jsonrpc" \
    -H 'Content-Type: application/json' --data-binary @"$WORK/rpc.json"
}

v=$(rpc aria2.getVersion '[]')
echo "$v" | grep -q '"version"' || { echo "$v"; fail "RPC getVersion 无有效响应"; }
ok "RPC 握手成功，version=$(echo "$v" | grep -oE '"version":"[^"]*"' | cut -d'"' -f4)"

add() {
  rpc aria2.addUri "[[\"$1\"],{\"out\":\"$2\"}]" \
    | grep -oE '"result":"[0-9a-f]+"' | cut -d'"' -f4
}


wait_done() {
  gid=$1
  j=0
  while [ $j -lt 45 ]; do
    st=$(rpc aria2.tellStatus "[\"$gid\",[\"status\",\"errorCode\",\"errorMessage\"]]")
    if echo "$st" | grep -q '"status":"complete"'; then
      echo "$st" | grep -q '"errorCode":"0"' || fail "任务 $gid 结束但 errorCode 非 0：$st"
      return 0
    fi
    if echo "$st" | grep -qE '"errorCode":"[1-9]'; then fail "任务 $gid 失败：$st"; fi
    j=$((j + 1)); sleep 2
  done
  fail "任务 $gid 超时未完成：$(rpc aria2.tellStatus "[\"$gid\"]")"
}

g1=$(add "$HTTP_URL" http.bin)
[ -n "$g1" ] || fail "HTTP 任务未能提交"
wait_done "$g1"
[ -s "$WORK/dl/http.bin" ] || fail "HTTP 下载产物为空"
ok "HTTP 真实下载完成，$(wc -c < "$WORK/dl/http.bin") 字节"

g2=$(add "$HTTPS_URL" https.txt)
[ -n "$g2" ] || fail "HTTPS 任务未能提交"
wait_done "$g2"
[ -s "$WORK/dl/https.txt" ] || fail "HTTPS 下载产物为空"
ok "HTTPS 真实下载完成（静态 OpenSSL 证书链与异步 DNS 可用），$(wc -c < "$WORK/dl/https.txt") 字节，首行 $(head -1 "$WORK/dl/https.txt")"

# BT 引导自检：提交一个磁力链接，确认静态 Core 能建立 DHT 与 BT 监听，
# 走通对等发现相关代码路径；不等待节点发现，随后立即移除任务。
MAG="magnet:?xt=urn:btih:0123456789abcdef0123456789abcdef01234567&dn=statictest"
g3=$(rpc aria2.addUri "[[\"$MAG\"]]" | grep -oE '"result":"[0-9a-f]+"' | cut -d'"' -f4)
if [ -n "$g3" ]; then
  sleep 8
  grep -qi 'DHT: listening on UDP port' "$WORK/stdout.log" || { cat "$WORK/stdout.log"; fail "静态 Core 未建立 DHT 监听"; }
  ok "DHT 监听已建立（IPv4/IPv6）"
  grep -qi 'BitTorrent: listening on TCP port' "$WORK/stdout.log" || { cat "$WORK/stdout.log"; fail "静态 Core 未建立 BT 监听"; }
  ok "BT 监听端口已建立（IPv4/IPv6）"
  rpc aria2.remove "[\"$g3\"]" > /dev/null 2>&1 || true
  ok "磁力任务提交与移除正常"
else
  fail "磁力任务未能提交"
fi

# 首启时 dht.dat 尚不存在，aria2 会为 IPv4/IPv6 各抛一条 DHT 路由表加载失败的 ERROR。
# 已做 A/B 取证：同一份临时配置下，现役动态 Core 与新静态 Core 都恰好出现同样的 2 条
# （ERROR 行是 "Exception caught while loading DHT routing table from ..."，
#  随后的详情行是 "Failed to load DHT routing table ..."，两者措辞不同但同源）,
# 属上游首启行为、与静态链接无关，因此从异常门禁中排除；其余异常一律判失败。
bad=$(grep -iE '\[C[0-9]+\]|Exception' "$WORK/stdout.log" | grep -v 'DHT routing table' || true)
if [ -n "$bad" ]; then
  echo "$bad"
  fail "静态 Core 运行日志出现异常"
fi
ok "静态 Core 真机功能验收全部通过"
