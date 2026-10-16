#!/bin/bash
set -x
secret=$(cat /run/aria2/rpc.secret)
dir=$(awk -F= '/^dir=/{print $2; exit}' /vol1/@appdata/aria2/aria2.conf)
rpc_port=$(awk -F= '/^rpc-listen-port=/{print $2; exit}' /vol1/@appdata/aria2/aria2.conf)
nohup python3 /tmp/slowserver.py >/tmp/slowserver.log 2>&1 &
server_pid=$!
sleep 1
url="http://127.0.0.1:18086/slow.bin"
add_response=$(curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"aria2.addUri\",\"params\":[\"token:$secret\",[\"$url\"]]}")
echo "ADD=$add_response"
gid=$(printf '%s' "$add_response" | python3 -c 'import sys,json; print(json.load(sys.stdin)["result"])' 2>/dev/null || true)
echo "GID=$gid"
sleep 12
status_response=$(curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"aria2.tellStatus\",\"params\":[\"token:$secret\",\"$gid\",[\"status\",\"files\",\"bittorrent\",\"dir\",\"infoHash\",\"errorCode\",\"errorMessage\"]]}")
echo "STATUS=$status_response"
echo "FILES:"
ls -la "$dir"
kill "$server_pid" 2>/dev/null || true