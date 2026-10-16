#!/bin/bash
# 0.1.90 设备实测：删除进行中的任务必须连带清理部分文件与 .aria2 控制文件。
# 依赖 busybox httpd 与 curl，仅在授权测试机执行。
# 限速制造"真正进行中"的任务，删除后必须连带清理部分文件与 .aria2 控制文件。
set -u
SOCK=/run/aria2/aria2.sock
DL=$(grep download_root /vol1/@appconf/aria2/config.json | tr -d ' ",' | cut -d: -f2)
fail=0
rpc() { printf %s "$1" > /tmp/dr.json; curl -s --max-time 20 --unix-socket "$SOCK" -H "X-Trim-Username: admin" -H "X-Trim-Isadmin: 1" -H "Content-Type: application/json" -d @/tmp/dr.json http://localhost/jsonrpc; }
echo "当前下载目录: $DL"
rm -rf /tmp/a2f; mkdir -p /tmp/a2f; head -c 60000000 /dev/urandom > /tmp/a2f/slow.bin
busybox httpd -p 127.0.0.1:18085 -h /tmp/a2f -c /dev/null; sleep 1
GID=$(rpc '{"jsonrpc":"2.0","id":41,"method":"aria2.addUri","params":[["http://127.0.0.1:18085/slow.bin"],{"out":"slowcheck.bin","max-download-limit":"128K"}]}' | tr -d '\n' | sed -n 's/.*"result":"\([0-9a-f]*\)".*/\1/p')
echo "任务 gid=$GID 限速 128K/s"
sleep 8
echo "--- 进行中状态"
rpc "{\"jsonrpc\":\"2.0\",\"id\":42,\"method\":\"aria2.tellStatus\",\"params\":[\"$GID\",[\"status\",\"totalLength\",\"completedLength\"]]}"
echo
ls -l "$DL" | sed 's/^/    /'
if [ ! -f "$DL/slowcheck.bin.aria2" ]; then echo "FAIL 未产生 .aria2 控制文件，测试前提不成立"; fail=1; fi
rpc "{\"jsonrpc\":\"2.0\",\"id\":43,\"method\":\"aria2.forceRemove\",\"params\":[\"$GID\"]}" > /dev/null
sleep 4
gone=1
[ -f "$DL/slowcheck.bin" ] && { echo "FAIL 删除后残留部分数据文件 ($(stat -c %s "$DL/slowcheck.bin") 字节)"; gone=0; }
[ -f "$DL/slowcheck.bin.aria2" ] && { echo "FAIL 删除后残留 .aria2 控制文件"; gone=0; }
[ $gone = 1 ] && echo "ok   删除未完成任务已连带清理部分文件与 .aria2 控制文件" || fail=1
echo "--- 清理后目录"
ls -la "$DL" | sed 's/^/    /'
kill $(ss -ltnp 2>/dev/null | sed -n 's/.*127.0.0.1:18085.*pid=\([0-9]*\).*/\1/p' | head -1) 2>/dev/null
rm -rf /tmp/a2f
echo "--- 结果: $fail 处失败"
exit $fail
