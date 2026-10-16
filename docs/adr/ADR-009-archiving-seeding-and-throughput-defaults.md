# ADR-009：归档语义、做种终止与吞吐默认值

## 状态

已接受，2026-09-04 起生效，对应发布版本 `0.1.88`。

## 背景

用户反馈三个问题：下载完成后归档不生效；BT 任务完成后长期停留在“分享”；
多任务下载时应用很卡。设备取证与源码核对确认三处独立根因：

1. 归档开关写的是 `special_mode=move`，但回调只认 `archive_enabled`，
   而 `archive_enabled` 只在已经存在 `archive_root` 时才会置真，导致没有选择
   归档目录时配置自相矛盾且回调静默不执行。
2. `aria2.conf` 从未写入 `seed-time`。Aria2 的 `seed-time` 没有默认值，只有
   `seed-ratio=1.0` 生效；在全局上传限速 2M 下分享比例几乎永远达不到，任务
   因此永不结束做种。`bt-detach-seed-only=false` 又让做种任务占满
   `max-concurrent-downloads` 槽位。
3. 默认参数沿用 Aria2 上游保守值（`split=5`、`max-connection-per-server=1`、
   `disk-cache=16M`、`bt-max-open-files=100`、`bt-max-peers=55`、
   `summary-interval=60`、`show-console-readout=true`、soft `ulimit -n 1024`），
   并且网关 `proxyRPC` 把响应静默截断到 8 MiB：任务较多时 `tellStopped` 返回
   残缺 JSON，前端反复重试，界面表现为明显卡顿。

## 决策

1. 单一事实源：`special_mode` 表示用户意图，`archive_enabled` 由管理器派生，
   规则为 `archive_root` 为空时强制 `special_mode=off`，
   `archive_enabled = (special_mode == "move")`。归一化在读取消毒后的持久配置、
   运行态配置和写入配置三条路径上统一执行，历史遗留的不一致配置在安装或重启后
   自愈，无需用户手动修正。
2. 归档开关与归档目录变更不触发受控重启，只改变回调行为；`aria2.conf` 始终注册
   `on-download-complete`，是否移动由回调读取当前配置决定。
3. 回调只在授权数据根内工作：多文件/BT 任务移动整个任务目录，目标同名时按
   `名称 (1)` 递增避让，跨文件系统时回退为复制后删除，遇到符号链接一律拒绝，
   完成事件后等待 2 秒再操作以让 Aria2 释放 `.aria2` 控制文件；成功时记录
   `archive done: 源 -> 目标` 供设备侧核验。
4. 做种策略默认“完成即停止”：`seed-time=0`、`seed-ratio=1.0`、
   `bt-detach-seed-only=true`。Aria2 中 `seed-time` 与 `seed-ratio` 是任一满足
   即结束的关系，`seed-time` 单位为分钟。需要长期分享的用户可在前台 BT 设置中
   调大 `seed-time`，不再需要修改应用内部实现。
5. 吞吐默认值对齐上游 Aria2 Pro 并按 NAS 硬件收敛：`split=64`、
   `max-connection-per-server=16`、`min-split-size=4M`、`piece-length=1M`、
   `disk-cache=64M`、`bt-max-open-files=1024`、`bt-max-peers=128`、
   `bt-prioritize-piece=head=32M,tail=32M`、`rlimit-nofile=8192`；同时
   `summary-interval=0`、`show-console-readout=false`、`max-download-result=300`
   降低输出与内存压力。刻意不采用上游的 `max-tries=0`：无限重试会掩盖失败任务并
   长期占用下载槽位。
6. 网关 RPC 代理请求上限提升到 10 MiB，响应整体读取而非截断，超过 32 MiB 时
   返回明确的 502 并回填 `Content-Length`，让前端能看到失败原因而不是解析坏数据。
7. AriaNg 适配层新增“归档目录”配置项：打开“下载完成后归档”但未选择目录时，
   自动弹出飞牛目录选择器；保存失败时回显管理器返回的中文纯文本原因，不再静默。

## 后果

- 默认安装后 BT 任务完成即离开活动列表并释放槽位，不再无限占用上传带宽；
  代价是默认不做长期分享，需要用户显式调大 `seed-time`。
- 归档依赖“先选归档目录”，未选目录时开关会被归一化为关闭，界面上通过弹窗引导，
  避免出现“显示已开启但实际不生效”的配置。
- `rlimit-nofile=8192` 需要宿主 soft limit 允许；若被限制，Aria2 会在日志中告警，
  不影响启动。
- AriaNg 前台修改的运行期参数仍会在重启后被 `aria2.conf` 覆盖，这是已知遗留项，
  后续需以“运行期参数持久化”单独决策。

## 设备验收证据（2026-09-04，0.1.88，测试机 192.168.x.x）

- 安装状态：应用中心 `aria2 / Aria2 / 0.1.88 / running`；`aria2c` 与
  `aria2-manager` 均以专属用户 `aria2` 运行，无 root 进程；RPC 仅监听
  `127.0.0.1:6801` 与 `[::1]:6801`（`6800` 被第三方应用占用后自动切换）。
- 运行期生效：通过应用网关读取 `aria2.getGlobalOption`，确认
  `seed-time=0`、`seed-ratio=1.0`、`bt-detach-seed-only=true`、`split=64`、
  `max-connection-per-server=16`、`disk-cache=67108864`、`bt-max-peers=128`、
  `bt-max-open-files=1024`、`summary-interval=0`、`show-console-readout=false`、
  `max-download-result=300`。
- 归档实测：HTTP 任务完成后文件从 `/vol1/1000/download` 移动到
  `/vol1/1000/download/completed`，`aria2.log` 记录 `archive done:`，源目录清空；
  12 个并发 8 MiB 任务全部归档成功。用户自己的 134 MiB BT 任务同样在完成后被归档。
- 做种实测：该 BT 任务完成后 Aria2 立即记录
  `Your share ratio was 0.0, uploaded/downloaded=0B/134MiB` 并结束做种，
  任务不再停留在“正在分享”，活动列表随后为空。
- 界面实测：简体中文界面、左侧菜单点击跳转正常；“下载完成后归档”与“归档目录”
  正确回显，切换“否/是”并刷新后持久配置同步为
  `special_mode=off / archive_enabled=false` 与
  `special_mode=move / archive_enabled=true`，截图见 `tests/ui-evidence/`。
- 清理回归：真实“error”任务（服务端提前断开）在界面删除后，
  `part.bin` 与 `part.bin.aria2` 一并从下载目录消失。
- 网关实测：`aria2.tellActive` / `tellStopped` 经网关往返约 13-16 ms，
  响应完整可解析，无截断。
- 前端链路实测：在真实浏览器中通过飞牛网关相对路径 `jsonrpc` 调用
  `aria2.tellActive`、`aria2.tellWaiting`、`aria2.tellStopped(0..1000)` 与
  `aria2.getVersion`，单次往返 4-5 ms 且全部返回可解析 JSON，不存在截断或重试风暴。
