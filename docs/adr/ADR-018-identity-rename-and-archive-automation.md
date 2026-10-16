# ADR-018：应用标识改名、归档目录自动化与并发调参

## 状态

已接受，2026-10-16 起生效，对应发布版本 `0.2.0`。本版**尚未在设备验收**。

## 背景

`0.1.96` 之后用户提出五项要求：

1. 应用账号由 `aria2pro` 改为 `aria2`，开发者与发布者改为 `vinson023`；
2. 归档目录不要再让用户手工设置，而是下载完成后自动在下载目录下创建 `completed`
   并把文件移入；
3. 同时下载几十个文件时会卡死；
4. 有时文件已下载完成，但任务不会自动进入「已完成/已停止」；
5. 把项目里所有 `aria2pro` 相关处全部改为 `aria2`。

第 1、5 项是同一件事的两面：应用标识要整体改名。第 2 项是对 ADR-010 的收口：
ADR-010 已把归档目录默认值定为 `<下载目录>/completed`，但仍允许用户在前台改成
任意路径，用户希望彻底去掉这个手工入口。第 3、4 项是运行期稳定性问题。

## 决策

1. **应用标识与专属账号统一改为 `aria2`。** `fpk/manifest` 的 `appname` 与
   `desktop_applaunchname`、`fpk/config/privilege` 的 `username`/`groupname`、
   `fpk/config/resource` 的数据共享目录名、`fpk/app/ui/config` 的网关前缀与
   Socket 名，以及两个 Go 命令的目录名与 `go.mod` 模块名，全部由 `aria2pro`
   改为 `aria2`。`maintainer` 与 `distributor` 改为 `vinson023`。

   这是一次**破坏性变更**：飞牛应用中心按 `appname` 识别应用，改名后它会被当作
   新应用，旧 `aria2pro` 应用不会被覆盖升级。升级路径是先卸载旧应用
   （按需选择是否清空配置）再安装 `aria2.fpk`；旧数据在
   `/vol1/@appdata/aria2pro`，新应用使用 `/vol1/@appdata/aria2`，不做自动迁移。
   决策理由：用户明确要求统一命名，且旧标识与产品实际名称（Aria2）不一致，
   长期维护成本高于一次性重装成本。

2. **运行期路径与环境变量同步改名。** 运行目录 `/run/aria2`、Socket
   `/run/aria2/aria2.sock`、回调二进制 `aria2-hook`/`aria2-hook-complete|error|removed`、
   管理器 `aria2-manager`、环境变量 `ARIA2_CONFIG_JSON`/`ARIA2_RPC_SECRET`
   全部对齐新标识，避免新旧命名混用。

3. **归档目录改为纯派生值，取消手工设置。** `normalizeConfig` 无论历史配置或前台
   写过什么，都强制把 `archive_root` 置为 `<download_root>/completed`；下载目录为空时
   归档目录一并清空。前台适配层移除「选择归档目录」菜单项，AriaNg 侧若仍提交
   `archive_root` 会被管理器覆盖，因此不依赖前端重建。`startup_repair` 中
   「用户自定义归档子目录」分支随之删除。本决策收口 ADR-010 第 1、2 条的
   「缺省时派生」，升级为「恒派生」。

4. **并发卡死按两条线索处理。**
   - 网关侧：`proxyRPC` 此前使用默认 `http.Transport`，其 `MaxIdleConnsPerHost=2`。
     任务多时 AriaNg 每秒轮询 `tellActive`/`tellWaiting`/`tellStopped`，空闲连接不够
     会不断重建连接；叠加较大的响应体，表现为界面卡死。改为自定义 Transport
     （`MaxIdleConns=MaxIdleConnsPerHost=128`，空闲 90s），并把代理超时与网关
     `ReadTimeout`/`WriteTimeout` 由 30s 放宽到 60s，避免大响应中途被截断后
     前台拿到残缺 JSON 反复重试。
   - 下载侧：`disk-cache` 64M→128M，`max-concurrent-downloads` 5→10，
     `max-connection-per-server` 16→8，`split` 64→16，`min-split-size` 4M→8M，
     `rlimit-nofile` 8192→65535，`bt-max-open-files` 1024→4096。方向是**降低单服务器
     并发连接、提高缓存与文件描述符上限**，避免大量任务把连接表与磁盘队列打满。

5. **完成后任务不移动按结果留存处理。** `max-download-result` 300→2000。该选项决定
   `tellStopped` 能返回多少条历史结果，取值过小会让大量任务完成后旧记录被挤出列表，
   与「任务不移动」的现象一致。BT 侧保持 `seed-time=0`（立即停止做种）与
   `bt-detach-seed-only=true`，`seed-ratio=1.0` 与 `seed-time=0` 并存时任一条件满足即
   结束做种，因此任务完成即离开活动列表。

6. **版本号定为 `0.2.0`。** 标识改名是破坏性变更，主版本段进位以区别于 0.1.x
   的增量修复。

## 影响

- 前端 `www/js/aria-ng-*.min.js` 与 `build-output/ariang/js/aria-ng-*.min.js` 中硬编码的
  `/app/aria2pro/jsonrpc` 已就地替换为 `/app/aria2/jsonrpc`（字符串替换，未重建 AriaNg，
  两份文件保持逐字节一致）。
- 回归验证：`gofmt -l`、`go vet`、`go test -count=1 ./...` 全部通过（Windows 宿主，
  POSIX 专属用例跳过）；两个 Go 产物以 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 -trimpath`
  交叉编译并校验 ELF 魔数；`fnpack build` 成功。
- 未验证项：真机安装、启动、归档自动创建与搬移、几十任务并发、完成后任务入列、
  改名后的桌面入口与网关路径。需在测试机（192.168.x.x）完成设备验收。
