#!/bin/bash
set -x
secret=$(cat /run/aria2/rpc.secret)
aria2_conf=/vol1/@appdata/aria2/aria2.conf
dir=$(awk -F= '/^dir=/{print $2; exit}' "$aria2_conf")
rpc_port=$(awk -F= '/^rpc-listen-port=/{print $2; exit}' "$aria2_conf")
listen_port=$(awk -F= '/^listen-port=/{print $2; exit}' "$aria2_conf")
magnet="magnet:?xt=urn:btih:08ada5a7a6183aae1e09d831df6748d566095a10&dn=Sintel&tr=udp://tracker.opentrackr.org:1337/announce&tr=https://tracker.opentrackr.org:443/announce"
add=$(curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\"aria2.addUri\",\"params\":[\"token:$secret\",[\"$magnet\"]]}")
echo "ADD=$add"
gid=$(printf '%s' "$add" | python3 -c 'import sys,json; print(json.load(sys.stdin)["result"])' 2>/dev/null || true)
sleep 5
echo "LISTEN_CHECK:"
ss -tulnp 2>/dev/null | grep ":$listen_port" || true
echo "STATUS:"
curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":2,\"method\":\"aria2.tellStatus\",\"params\":[\"token:$secret\",\"$gid\",[\"status\",\"bittorrent\",\"files\"]]}" || true
if [ -n "$gid" ]; then
  curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":3,\"method\":\"aria2.forceRemove\",\"params\":[\"token:$secret\",\"$gid\"]}" >/dev/null || true
  curl -s "http://127.0.0.1:$rpc_port/jsonrpc" -d "{\"jsonrpc\":\"2.0\",\"id\":4,\"method\":\"aria2.removeDownloadResult\",\"params\":[\"token:$secret\",\"$gid\"]}" >/dev/null || true
fi
echo "FILES:"
ls -la "$dir"
