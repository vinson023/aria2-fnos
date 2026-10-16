#!/bin/bash
set -x
target=/vol1/@appshare/aria2/downloads/cleanup-dir
mkdir -p "$target"
dd if=/dev/zero of="$target/part.bin" bs=1M count=2 status=none
chown -R aria2:AppUsers "$target" || true
printf '{"paths":["%s"]}\n' "$target" > /tmp/cleanup-request.json
cat /tmp/cleanup-request.json
curl -s -o /tmp/cleanup.out -w 'HTTP=%{http_code}\n' --unix-socket /run/aria2/aria2.sock -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -X POST --data-binary @/tmp/cleanup-request.json http://localhost/api/v1/cleanup
cat /tmp/cleanup.out
echo
ls -l /vol1/@appshare/aria2/downloads