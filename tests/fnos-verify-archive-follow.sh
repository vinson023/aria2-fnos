#!/bin/bash
# 0.1.90 设备实测：界面修改下载目录时默认归档目录必须跟随；复原后回到原值；随后真实下载验证。
# 该脚本会临时改写测试机应用配置并在结束时复原，仅用于授权测试机。
set -u
SOCK=/run/aria2/aria2.sock
CONF=/vol1/@appconf/aria2/config.json
CFG=/tmp/follow_cfg.json
ALT=/vol1/1000/download
SHARED=/vol1/@appshare/aria2/downloads
fail=0
ok() { echo "ok   $*"; }
bad() { echo "FAIL $*"; fail=$((fail + 1)); }

getcfg() {
  curl -s --max-time 10 --unix-socket "$SOCK" -H "X-Trim-Username: admin" -H "X-Trim-Isadmin: 1" http://localhost/api/v1/config > "$CFG"
}
flat() { tr -d '\n\r\t ' < "$CFG"; }
field() { flat | tr ',{}[]' '\n\n\n\n\n' | sed -n "s/^\"$1\"://p" | head -1 | tr -d '"'; }
putcfg() {
  curl -s --max-time 40 -o /tmp/put.out -w "%{http_code}" --unix-socket "$SOCK" -H "X-Trim-Username: admin" -H "X-Trim-Isadmin: 1" -H "Content-Type: application/json" -X PUT -d "$1" http://localhost/api/v1/config
}
body() {
  printf %s '{"rpc_port":'$1',"listen_port":'$2',"dht_listen_port":'$3',"dht_message_timeout":"'$4'","enable_dht":true,"enable_dht6":true,"gateway_user":"","gateway_role":"","download_root":"'$5'","archive_enabled":true,"archive_root":"'$6'","special_mode":"move","authorized_roots":'$7',"tracker_source":"'$8'","tracker_auto_update":true}'
}
rpc() {
  printf %s "$1" > /tmp/rc.json
  curl -s --max-time 20 --unix-socket "$SOCK" -H "X-Trim-Username: admin" -H "X-Trim-Isadmin: 1" -H "Content-Type: application/json" -d @/tmp/rc.json http://localhost/jsonrpc
}

getcfg
RPCP=$(field rpc_port); LP=$(field listen_port); DHP=$(field dht_listen_port)
TMO=$(field dht_message_timeout); TCS=$(field tracker_source)
ORIGDL=$(field download_root); ORIGAR=$(field archive_root)
ORIGROOTS=$(flat | sed -n 's/.*"authorized_roots":\(\[[^]]*\]\).*/\1/p')
if [ -z "$ORIGROOTS" ]; then ORIGROOTS='[]'; fi
if [ "$ORIGDL" = "$ALT" ]; then
  NEW=$SHARED; NEWROOTS='[]'
else
  NEW=$ALT; NEWROOTS='["/vol1/1000/download"]'
fi
echo "--- 起始配置"
echo "    download_root=$ORIGDL"
echo "    archive_root=$ORIGAR"
echo "    authorized_roots=$ORIGROOTS"
echo "    端口 rpc=$RPCP bt=$LP dht=$DHP timeout=$TMO"

echo "--- 步骤1 把下载目录改为 $NEW（归档目录回显旧值，模拟界面保存）"
CODE=$(putcfg "$(body "$RPCP" "$LP" "$DHP" "$TMO" "$NEW" "$ORIGAR" "$NEWROOTS" "$TCS")")
echo "    PUT http=$CODE $(head -c 200 /tmp/put.out)"
[ "$CODE" = 204 ] || bad "保存下载目录变更失败"
getcfg
NDL=$(field download_root); NAR=$(field archive_root)
echo "    保存后 download_root=$NDL archive_root=$NAR"
[ "$NDL" = "$NEW" ] || bad "下载目录未更新"
[ "$NAR" = "$NEW/completed" ] && ok "归档目录自动跟随到新下载目录下的 completed" || bad "归档目录未跟随: $NAR"
if [ -d "$NEW/completed" ]; then ok "归档目录已自动创建 $(stat -c '%U:%G %a' "$NEW/completed")"; else bad "归档目录未自动创建"; fi

echo "--- 步骤2 复原为测试前的配置"
CODE=$(putcfg "$(body "$RPCP" "$LP" "$DHP" "$TMO" "$ORIGDL" "$ORIGAR" "$ORIGROOTS" "$TCS")")
echo "    PUT http=$CODE $(head -c 200 /tmp/put.out)"
[ "$CODE" = 204 ] || bad "复原失败"
getcfg
RDL=$(field download_root); RAR=$(field archive_root)
RROOTS=$(flat | sed -n 's/.*"authorized_roots":\(\[[^]]*\]\).*/\1/p')
echo "    复原后 download_root=$RDL archive_root=$RAR authorized_roots=$RROOTS"
[ "$RDL" = "$ORIGDL" ] || bad "下载目录未复原"
[ "$RAR" = "$ORIGAR" ] && ok "归档目录随复原回到原值" || bad "复原后归档目录不对: $RAR"

echo "--- 步骤3 复原后做一次真实下载，确认归档链路仍然可用"
rm -rf /tmp/a2b; mkdir -p /tmp/a2b; head -c 1200000 /dev/urandom > /tmp/a2b/reverify.bin
busybox httpd -p 127.0.0.1:18082 -h /tmp/a2b -c /dev/null; sleep 1
GID=$(rpc '{"jsonrpc":"2.0","id":11,"method":"aria2.addUri","params":[["http://127.0.0.1:18082/reverify.bin"],{}]}' | tr -d '\n' | sed -n 's/.*"result":"\([0-9a-f]*\)".*/\1/p')
echo "    gid=$GID 归档目录=$ORIGAR"
i=0; while [ $i -lt 30 ]; do [ -f "$ORIGAR/reverify.bin" ] && break; i=$((i + 1)); sleep 2; done
if [ -f "$ORIGAR/reverify.bin" ]; then ok "任务落入归档目录: $(stat -c '%s 字节 %U:%G' "$ORIGAR/reverify.bin")"; else bad "未见归档文件"; ls -l "$ORIGDL" | head -5; fi
[ -f "$ORIGDL/reverify.bin" ] && bad "下载目录仍残留已完成文件" || ok "下载目录已清空"
rpc "{\"jsonrpc\":\"2.0\",\"id\":12,\"method\":\"aria2.forceRemove\",\"params\":[\"$GID\"]}" > /dev/null
kill $(ss -ltnp 2>/dev/null | sed -n 's/.*127.0.0.1:18082.*pid=\([0-9]*\).*/\1/p' | head -1) 2>/dev/null
rm -f "$ORIGAR/reverify.bin"; rm -rf /tmp/a2b
grep -i 'archive done' /vol1/@appdata/aria2/aria2.log 2>/dev/null | tail -2 | sed 's/^/    /'

echo "--- 运行期参数核对（getGlobalOption 一次性取全部选项后筛选）"
rpc '{"jsonrpc":"2.0","id":13,"method":"aria2.getGlobalOption","params":[]}' | tr ',' '\n' > /tmp/opt.txt
for k in dir max-overall-upload-limit bt-request-peer-speed-limit listen-port rpc-listen-port dht-listen-port dht-message-timeout seed-time; do
  printf '    %-28s ' "$k"
  grep "\"$k\"" /tmp/opt.txt | head -1
done
echo "--- 结果: $fail 处失败"
exit $((fail > 0))
