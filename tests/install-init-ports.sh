#!/bin/sh
set -eu

install_init=$1
install_callback=$2
work=$(mktemp -d)
trap 'rm -rf "$work"' EXIT

prepare_app() {
    mkdir -p "$work/app/server"
    : > "$work/app/server/aria2c"
    : > "$work/app/server/aria2-manager"
    : > "$work/app/server/aria2-hook"
}

run_init() {
    env -i PATH="$PATH" wizard_rpc_port="$wizard_rpc_port" wizard_listen_port="$wizard_listen_port" "$install_init"
}

run_callback() {
    env -i PATH="$PATH" TRIM_APPDEST="$work/app" TRIM_PKGETC="$work/etc" TRIM_PKGVAR="$work/var" wizard_rpc_port="$wizard_rpc_port" wizard_listen_port="$wizard_listen_port" "$install_callback"
}

wizard_rpc_port=6801
wizard_listen_port=6889
run_init
prepare_app
run_callback
grep -q '"rpc_port": 6801' "$work/etc/config.json"
grep -q '"listen_port": 6889' "$work/etc/config.json"

rm -rf "$work/etc" "$work/var"
wizard_rpc_port=1023
wizard_listen_port=6888
if run_init; then
    echo "expected invalid low port to fail" >&2
    exit 1
fi

wizard_rpc_port=6800
wizard_listen_port=6800
if run_init; then
    echo "expected duplicate ports to fail" >&2
    exit 1
fi
