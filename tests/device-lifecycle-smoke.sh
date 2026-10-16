#!/bin/sh
set -eu

test -n "$TRIM_APPDEST"
test -n "$TRIM_PKGETC"
test -n "$TRIM_PKGVAR"
test -n "$APP_CMD"
test -n "$INSTALL_CMD"

"$INSTALL_CMD"
chmod 755 "$TRIM_APPDEST/server/aria2c" "$TRIM_APPDEST/server/aria2-manager"
"$APP_CMD" start
sleep 2
test -S "$TRIM_APPDEST/aria2.sock"
curl --fail --silent --show-error --unix-socket "$TRIM_APPDEST/aria2.sock" http://localhost/healthz -o /dev/null
curl --fail --silent --show-error --unix-socket "$TRIM_APPDEST/aria2.sock" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' http://localhost/ -o /dev/null
curl --fail --silent --show-error --unix-socket "$TRIM_APPDEST/aria2.sock" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":1,"method":"aria2.getVersion","params":[]}' http://localhost/jsonrpc | grep -q 'version'
curl --fail --silent --show-error --unix-socket "$TRIM_APPDEST/aria2.sock" -X PUT -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d '{"rpc_port":6801,"listen_port":6889,"gateway_user":"","gateway_role":"","download_root":"","archive_enabled":false,"archive_root":"","authorized_roots":[]}' http://localhost/api/v1/config -o /dev/null
"$APP_CMD" stop
"$APP_CMD" start
curl --fail --silent --show-error --unix-socket "$TRIM_APPDEST/aria2.sock" -H 'X-Trim-Username: admin' -H 'X-Trim-Isadmin: 1' -H 'Content-Type: application/json' -d '{"jsonrpc":"2.0","id":2,"method":"aria2.getVersion","params":[]}' http://localhost/jsonrpc | grep -q 'version'
grep -q '6801' "$TRIM_PKGETC/config.json"
grep -q '6889' "$TRIM_PKGETC/config.json"
"$APP_CMD" stop
