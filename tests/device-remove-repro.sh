#!/bin/bash
set -x
aria2_conf=/vol1/@appdata/aria2/aria2.conf
secret=$(cat /run/aria2/rpc.secret)
dir=$(awk -F= '/^dir=/{print $2; exit}' "$aria2_conf")
rpc_port=$(awk -F= '/^rpc-listen-port=/{print $2; exit}' "$aria2_conf")
rm -f "$dir/serve.bin" "$dir/serve.bin.aria2" "$dir/serve.bin.torrent"
rm -rf /tmp/http-tmp
mkdir -p /tmp/http-tmp
dd if=/dev/zero of=/tmp/http-tmp/serve.bin bs=1M count=512 status=none
chmod -R a+rX /tmp/http-tmp
python3 -m http.server 18083 --directory /tmp/http-tmp >/tmp/http-tmp.log 2>&1 &
server_pid=$!
sleep 1
add_url="http://127.0.0.1:18083/serve.bin"
add_response=$(curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"aria2.addUri\",\"params\":[\"token:$secret\",[\"$add_url\"]]}")
echo "ADD=$add_response"
gid=$(printf '%s' "$add_response" | python3 -c 'import sys,json; print(json.load(sys.stdin)["result"])' 2>/dev/null || true)
echo "GID=$gid"
remove_response=$(curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"aria2.forceRemove\",\"params\":[\"token:$secret\",\"$gid\"]}")
echo "REMOVE=$remove_response"
sleep 1
echo "AFTER:"
ls -l "$dir"
if [ -e "$dir/serve.bin" ] || [ -e "$dir/serve.bin.aria2" ] || [ -e "$dir/serve.bin.torrent" ]; then
  echo "RESULT=REMAIN"
else
  echo "RESULT=CLEAN"
fi
kill "$server_pid" 2>/dev/null || true