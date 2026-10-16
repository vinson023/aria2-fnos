#!/bin/bash
# 真实 BitTorrent 传输验收：在同一台 glibc 设备上用同一份待测 Core 分别做种子端与下载端，
# 通过本地最小 Tracker 交换对等体信息，传完比对 SHA-256。
#
# 这一步补的是"数据面"证据：此前的真机验收只证明了 DHT/BT 监听端口能建立、
# 磁力任务能提交，并没有证明分片校验与对等体传输真的跑得通。Tracker、种子与下载文件
# 全部在本地临时目录，不联网，不触碰现役实例。
set -u

SRC_BIN=${1:-/hosttmp/shipped/aria2c}
FIXTURE=${2:-/hosttmp/btfixture-linux}
WORK=/hosttmp/bttest
TRK_PORT=6910
SEED_BT=6911
LEECH_BT=6912
SEED_RPC=6913
FILE_MB=8
TIMEOUT_S=120

fail() { echo "FAIL: $*"; exit 1; }
ok() { echo "OK: $*"; }
cleanup() {
    for p in $SEED_PID $LEECH_PID $TRK_PID; do
        [ -n "${p:-}" ] && kill "$p" 2>/dev/null || true
    done
    wait 2>/dev/null || true
}
trap cleanup EXIT INT TERM

[ -x "$SRC_BIN" ] || fail "找不到待测二进制 $SRC_BIN"
[ -x "$FIXTURE" ] || fail "找不到测试夹具 $FIXTURE（先用 GOOS=linux 交叉编译 tests/btfixture）"

pkill -f "$WORK/" 2>/dev/null || true
sleep 1
rm -rf "$WORK"
mkdir -p "$WORK/seed" "$WORK/dl" "$WORK/data" || fail "无法创建临时目录"

# 造一个内容随机、大小固定的文件，避免稀疏文件让校验失去意义。
dd if=/dev/urandom of="$WORK/seed/btfile.bin" bs=1M count="$FILE_MB" status=none || fail "无法生成测试文件"
want=$(sha256sum "$WORK/seed/btfile.bin" | cut -d' ' -f1)
ok "测试文件就绪：$FILE_MB MiB，SHA-256 ${want:0:16}…"

"$FIXTURE" mktorrent -file "$WORK/seed/btfile.bin" -out "$WORK/bt.torrent" \
    -announce "http://127.0.0.1:$TRK_PORT/announce" || fail "生成 .torrent 失败"
ok ".torrent 已生成"

"$FIXTURE" tracker -listen "127.0.0.1:$TRK_PORT" > "$WORK/tracker.log" 2>&1 &
TRK_PID=$!
sleep 1
kill -0 "$TRK_PID" 2>/dev/null || { cat "$WORK/tracker.log"; fail "Tracker 未能启动"; }
ok "本地 Tracker 已监听 127.0.0.1:$TRK_PORT"

common=(--enable-dht=false --enable-dht6=false --bt-enable-lpd=false
        --enable-peer-exchange=false --summary-interval=0 --show-console-readout=false
        --max-overall-download-limit=0 --max-overall-upload-limit=0
        --file-allocation=none --log-level=notice)

# 种子端：文件已在目录里，必须 --check-integrity=true 才会按种子校验并认领既有文件，
# 否则 aria2 会因为"文件存在但没有 .aria2 控制文件"直接取消，防止把用户文件截断。
# seed-ratio 与 seed-time 都置 0 表示无限做种；DHT 已关闭，不再传无关参数。
# 做种开关：aria2 只在 seed-ratio 或 seed-time 至少一个为正时才做种，
# 两者都是 0 等于"完成即退出"（实测种子端会立刻打 "Seeding is over"）。
"$SRC_BIN" "${common[@]}" --dir="$WORK/seed" --check-integrity=true \
    --listen-port="$SEED_BT" --enable-rpc=false --seed-ratio=1.0 --seed-time=1800 \
    "$WORK/bt.torrent" > "$WORK/seed.log" 2>&1 &
SEED_PID=$!

# 下载端：独立目录与独立 BT 端口，RPC 仅用于查询状态，密钥关掉以免干扰。
"$SRC_BIN" "${common[@]}" --dir="$WORK/dl" \
    --listen-port="$LEECH_BT" --enable-rpc=true --rpc-listen-port="$SEED_RPC" \
    --seed-time=0 --seed-ratio=0 \
    "$WORK/bt.torrent" > "$WORK/leech.log" 2>&1 &
LEECH_PID=$!

echo "等待真实 BT 传输（最长 ${TIMEOUT_S}s）…"
i=0
while [ "$i" -lt "$TIMEOUT_S" ]; do
    if [ -f "$WORK/dl/btfile.bin" ]; then
        got=$(sha256sum "$WORK/dl/btfile.bin" 2>/dev/null | cut -d' ' -f1)
        if [ "$got" = "$want" ]; then
            ok "BT 传输完成且分片校验一致：SHA-256 ${got:0:16}…"
            break
        fi
    fi
    if ! kill -0 "$SEED_PID" 2>/dev/null; then sed 's/^/  种子端: /' "$WORK/seed.log"; fail "种子端提前退出"; fi
    if ! kill -0 "$LEECH_PID" 2>/dev/null; then sed 's/^/  下载端: /' "$WORK/leech.log"; fail "下载端提前退出"; fi
    i=$((i + 1))
    sleep 1
done

if [ "${got:-}" != "$want" ]; then
    echo "--- 种子端日志 ---"; tail -n 20 "$WORK/seed.log"
    echo "--- 下载端日志 ---"; tail -n 20 "$WORK/leech.log"
    echo "--- Tracker 日志 ---"; tail -n 10 "$WORK/tracker.log"
    fail "${TIMEOUT_S}s 内未完成 BT 传输（下载目录状态：$(ls -la "$WORK/dl" | tail -n +2 | tr '\n' ' ')）"
fi

# 只看真正的异常前缀，不用宽泛的 Error 关键字：aria2 的正常输出里也会出现该词。
if grep -qE '\[C[0-9]+\]' "$WORK/leech.log" "$WORK/seed.log"; then
    grep -E '\[C[0-9]+\]' "$WORK/leech.log" "$WORK/seed.log"
    fail "BT 两端日志出现异常"
fi
peers=$(grep -c 'Taking in from' "$WORK/seed.log" || true)
echo "种子端接受的对等体连接: $peers 条"
ok "静态 Core 真实 BT 传输验收通过（对等体发现、握手、分片校验、落盘全链路）"
