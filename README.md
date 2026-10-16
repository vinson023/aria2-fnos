# 飞牛版 Aria2

原生飞牛 fnOS FPK：集成 Aria2 Core、Go 管理器和 AriaNg。当前 x86_64 候选包为
`0.1.96`，运行时不依赖 Docker、Node.js、Python 或数据库，也不再依赖宿主的任何共享库。

## 项目理解

本项目将使用飞牛原生打包、生命周期与权限机制替代 P3TERX Aria2 Pro 的 Docker 运行层，在保留下载体验的同时，安全接入应用中心、飞牛权限体系和桌面 Web 界面。

## 已选技术方向

- **FPK 打包：** fnpack 1.2.3 原生应用模板。
- **下载引擎：** 可复现构建、按 CPU 架构分发的 Aria2 Core。
- **管理与网关：** 小型静态 Go 服务，负责 Unix Socket、Web 界面、RPC 代理、管理员校验和 Tracker 调度。
- **Web 界面：** 固定版本的 AriaNg 加轻量飞牛适配层；适配层使用 `@trimjs/web-app`，AriaNg 负责下载任务控制。
- **配置与状态：** 使用飞牛应用目录及文本配置文件；单实例首版不需要数据库。
- **公开入口：** `/app/aria2` 统一网关，不开放 Aria2 RPC 端口；RPC 默认 `6800`、BT/DHT 默认 `6888`，均可在应用设置中修改。

选择 Go 管理服务而不是 CGI，是因为统一网关能保持飞牛登录态、适配长期服务，并避免公开 RPC；它也比为静态文件和反向代理额外引入 Node.js 更小、更稳定。

## 高层架构

```text
飞牛桌面 iframe
  -> /app/aria2（飞牛统一网关，已认证）
  -> aria2-manager Unix Socket
     -> 提供 AriaNg 与飞牛适配层静态资源
     -> 校验 X-Trim-Isadmin
     -> 向 HTTP JSON-RPC 代理请求注入 RPC 密钥
     -> 按受控计划刷新 BT Tracker
  -> aria2c，监听 127.0.0.1:<rpc_port>
     -> 下载与 BT/DHT 端口监听 NAS 主机的 <listen_port>（TCP+UDP）
```

持久配置位于 `$TRIM_PKGETC`；aria2 会话、DHT 数据、PID 和日志等运行状态位于 `$TRIM_PKGVAR`。

## 规划目录

```text
.
├── AGENTS.md
├── PROJECT_CONTEXT.md
├── README.md
├── docs/
│   ├── architecture.md
│   ├── requirements.md
│   ├── development-plan.md
│   └── adr/
├── fpk/
├── manager/
├── ui-adapter/
├── scripts/
├── tests/
├── tools/
└── reference-snapshot/
```

当前 `fpk/` 已包含 x86_64 Core、静态 Go 管理器、完成回调和适配后的 AriaNg；`manager/`

## 开发阶段

1. 确认范围、默认值、发布架构和安全策略。
2. 创建 FPK 骨架和包元数据。
3. 构建并校验固定版本的原生 Aria2 Pro Core。
4. 实现生命周期、配置迁移与进程管理。
5. 实现 Unix Socket 网关和 RPC 保护。
6. 集成 AriaNg 与飞牛 Web SDK 适配层。
7. 实现 Tracker、清理、归档和任务恢复行为。
8. 构建 FPK 并完成测试机验证。
9. 准备发布说明、上游署名、许可证和更新流程。

## 文档

- [需求分析](docs/requirements.md)
- [需求追踪与验收标准](docs/requirements-traceability.md)
- [阶段一决策表](docs/phase1-decisions.md)
- [阶段一审查清单](docs/phase1-review.md)
- [参考资料基线](docs/reference-baseline.md)
- [安全模型](docs/security-model.md)
- [验收矩阵](docs/acceptance-matrix-0.1.6.md)
- [候选发布物](docs/release-candidate.md)
- [0.1.96 设备验收报告](docs/acceptance-report-0.1.96.md)
- [飞牛集成能力矩阵](docs/fnos-integration-matrix.md)
- [架构设计](docs/architecture.md)
- [开发计划](docs/development-plan.md)
- [架构决策记录](docs/adr/)
- [离线参考资料](reference-snapshot/README.md)

## 当前状态

### `0.2.2`：AriaNg 基本设置页移除归档目录字段（待设备验收）

0.2.1 已在后端和飞牛适配层移除归档目录设置，但 AriaNg 原生「基本设置」页仍显示「归档目录」和
「下载完成后归档」两个字段。本版直接在 AriaNg 压缩包的 `basicOptions` 数组中删除 `special-mode`
和 `fnos-archive-dir` 两项，使它们不再出现在基本设置页面上。后端 `normalizeConfig` 仍恒定派生
`archive_root = <download_root>/completed`，归档行为不依赖前端这两项设置。

发布物：`aria2.fpk`，版本 `0.2.2`，11,160,512 B，外层 SHA-256
`17c0e8756cbd94ea5baeaaee87a9ff69ab4cef5b6ecc0cb9388a271bba5a0cab`（`fnpack` 元数据不稳定，
仅作本次指纹）；溯源以 `build-output/fpk-0.2.2.inner.SHA256SUMS`（49 个文件）为准。
静态 Core `aria2c` 逐字节未变（`a482fb24…015b`）。本版尚未在设备验收。

### `0.2.1`：归档失败降级 + 前端移除归档设置（待设备验收）

修复用户截图中的红字错误：当用户选了一个对 aria2 服务用户不可写的下载目录
（如 `/vol2/1000/downloads`，属主 `1000:1000`），后端 `ensureConfiguredDirectories`
在该目录下 `mkdir completed` 时 `permission denied`，之前会阻塞整个下载目录保存。
本版改为归档目录创建失败时静默降级（关闭归档、清空归档目录、记日志），不再阻塞
下载目录等基本配置的保存。

同时从前端 UI 移除归档目录相关设置：`showSettingsPanel` 移除"特殊模式" select，
菜单移除"设置特殊模式"项，`window.fnosArchiveDirectory.pick()` 不再弹目录选择器
（AriaNg bundle 仍会调它，返回空走"用户取消"分支，不保存开启状态）。

发布物：`aria2.fpk`，版本 `0.2.1`，11,160,587 B，外层 SHA-256
`c955873ff93c5ed352c6d2aa00919be36348ec0a35254005ca036e296b72f0df`（`fnpack` 元数据不稳定，
仅作本次指纹）；溯源以 `build-output/fpk-0.2.1.inner.SHA256SUMS`（49 个文件）为准。
静态 Core `aria2c` 逐字节未变（`a482fb24…015b`）。本版尚未在设备验收。

### `0.2.0`：应用标识与账号改名、归档自动化、并发与完成态修复（待设备验收）

本版是**应用标识的破坏性变更**：`appname` 由 `aria2pro` 改为 `aria2`，专属账号与组由
`aria2pro` 改为 `aria2`，开发者与发布者改为 `vinson023`。飞牛应用中心会把它当作**新应用**，
旧 `aria2pro` 应用不会被覆盖升级；升级路径是先在应用中心卸载旧应用（按需选择是否清空配置），
再安装新的 `aria2.fpk`。旧数据位于 `/vol1/@appdata/aria2pro`，新应用使用 `/vol1/@appdata/aria2`。

四项行为修复：

1. **归档目录不再需要手工指定。** 归档目录恒定为 `<下载目录>/completed`，由管理器在归一化时
   派生、在启动与保存时自动创建，下载完成后由完成回调把文件搬入其中。前台不再提供「选择归档目录」
   入口，历史配置里写过的自定义归档路径会被自动收敛回 `<下载目录>/completed`。
2. **并发卡死。** 网关到回环 Aria2 的 RPC 连接改为复用长连接（此前沿用默认
   `MaxIdleConnsPerHost=2`，任务多时不断重建连接）；网关读写超时从 30s 放宽到 60s，
   避免大响应被截断后前台反复重试。下载侧调参：`disk-cache` 64M→128M、
   `max-concurrent-downloads` 5→10、`max-connection-per-server` 16→8、`split` 64→16、
   `min-split-size` 4M→8M、`rlimit-nofile` 8192→65535、`bt-max-open-files` 1024→4096。
3. **完成后任务不移动。** `max-download-result` 300→2000：该值决定「已完成/已停止」列表能记住
   多少条结果，过小会让大量任务完成后旧记录被挤掉；BT 侧保持 `seed-time=0` 立即停止做种。
4. **全局改名。** 源码、配置、生命周期脚本、回调脚本、前端适配层、测试脚本与文档中的
   `aria2pro` 全部改为 `aria2`（含 `aria2-manager`/`aria2-hook`/`aria2.sock`/`/run/aria2`/
   `/app/aria2` 与 `ARIA2_CONFIG_JSON`/`ARIA2_RPC_SECRET` 环境变量）。

发布物：`aria2.fpk`，版本 `0.2.0`，13,868,442 B，外层 SHA-256
`eb0afd52fe9236298cdf7b9a96a3ce2a8b267825c1a1a20019e0ddd66aad32c4`（`fnpack` 元数据不稳定，
仅作本次指纹）；溯源以 `build-output/fpk-0.2.0.inner.SHA256SUMS`（49 个文件）为准。
静态 Core `aria2c` 逐字节未变（`a482fb24…015b`）。本版尚未在设备验收。

### `0.1.x` 历史

「部分机器安装后无法启动」有两类根因，`0.1.94` 与 `0.1.95` 各消一类。

第一类是配置与机器取值耦合：下载目录、归档目录和授权目录都是机器相关的，换机、卷号不同、
共享目录被改名或删除、配置跨 NAS 拷贝，都会让启动前的校验失败，而生命周期脚本随即静默中止 ——
界面也没起来，用户没有任何入口改回配置。`0.1.94` 把启动路径改为尽力自愈：逐条修复失效取值、
保住端口与用户设置不动、每条改动都写进新的持久日志 `$TRIM_PKGVAR/startup.log`，
实在没有可用共享目录时退到应用私有下载目录并留痕；前台保存仍然严格，改错了照样明确报错。
决策与实测见 `docs/adr/ADR-015-startup-self-repair.md`。

第二类是二进制依赖：Core 原本是动态链接的，除 glibc 外还要 `libz`、`libexpat`、`libsqlite3`、
`libssl3`、`libcrypto3`、`libssh2`、`libstdc++`，任一缺失就只有 `aria2.log` 里一行动态链接器报错，
前台同样是一句没有信息的「无法启动」。`0.1.95` 把 Core 换成完全静态链接（musl 加全部依赖静态库），
产物零动态依赖，这一类故障从根上消除，安装包同时从 46,003,943 B 降到 13,853,128 B。
构建配方、许可证分析与真机功能验收见 `docs/adr/ADR-016-static-core-linking.md`。
另附只读排查脚本 `tests/fnos-diagnose-startup.sh`，在出问题的机器上跑一次即可收齐全部病因线索。

`0.1.93` 补上「删除任务连带清理 RPC 上传的种子元数据」：Aria2 的 `rpc-save-upload-metadata`
默认开启，会把上传的种子按内容 SHA-1 改名存进任务目录，而 `on-download-*` 回调拿不到这个文件名，
所以由网关按 gid 记台账（`$TRIM_PKGVAR/aria2-torrents.tsv`，0600、原子重写），删除与清空时精确删除，
重启后与 `aria2.getSessionInfo` 对账；卸载选择「清空应用配置」时台账一并清除。决策见
`docs/adr/ADR-014-rpc-uploaded-metadata-ledger.md`。该版本的行为规则（含删除未完成任务时连带清理已上传 `.torrent`）已于 2026-10-16 在设备实测通过
（`tests/fnos-verify-torrent-cleanup.sh`，`0 处失败`）。

`0.1.92` 的两项修复已于 2026-09-05 在设备实测通过（`tests/fnos-verify-runtime-options.sh`，`0 处失败`）。

`0.1.92` 修掉了 `0.1.91` 设备验收当场抓出的两个真实缺陷，二者都会让「界面看到的取值」与「重启后生效的取值」分叉：

1. `aria2.changeGlobalOption` 的选项对象只认 `params[0]`。AriaNg 不带密钥时发的是 `params:[选项对象]`，
   旧逻辑却在全部参数里找第一个非空对象，于是 `[{}, {选项}]` 会被 Aria2 应用 `{}` 并返回 `OK`，
   网关却把从未生效的取值落了盘，重启后凭空出现。设备上已复现：内存值停在 9，重启后变成 11。
2. 网关专属项改为**转发侧**也拦截。`0.1.91` 只挡住了落盘，`dir` 仍被 Aria2 运行期接受，
   一次保存即可把下载目录移出 `authorized_roots`，绕过目录授权与归档边界；`on-*` 回调同理。
   决策见 `docs/adr/ADR-013-gateway-owned-option-forwarding-filter.md`。

运行期选项持久化与配置版本化迁移仍按 ADR-011、ADR-012 工作：AriaNg 里改过且被 Aria2 接受的全局选项
会落盘到 `aria2-overrides.conf`，重启时合并进重新生成的 `aria2.conf`；端口、目录、密钥、会话与回调脚本属受管项，
前台既改不动也留不住。`0.1.91` 已完成设备实测（`config_version` 迁移、覆盖文件读写、重启存活、清理复位全部通过）。

`0.1.96` 已在测试机（192.168.x.x）完成全部设备验收（2026-10-16）：multicall 16/16 通过、
启动诊断 15/15 正常、种子台账清理 0 处失败、PID 属主验证通过、前端 AriaNg 正常打开。
完整报告见 `docs/acceptance-report-0.1.96.md`。

`0.1.96` 做了一轮运行期体检，修掉一处前端功能缺失与两处生命周期卫生问题：

1. 网关此前只放行 `aria2.` 前缀，把 AriaNg 的 `system.multicall` 与 `system.listMethods` 判为非法请求，
   任务详情页（`tellStatus` + `getPeers`）、重试按钮与调试页因此一直不可用，失败还会反复重试。
   现在批量调用按**只读白名单**受控转发，密钥逐个注入嵌套调用；实测过的两条绕过路径
   （嵌套 `changeGlobalOption` 改 `dir`、嵌套 `remove` 绕开种子台账）在网关就被拒掉。
   决策与证据见 `docs/adr/ADR-017-multicall-controlled-forwarding.md`。
2. 启动锁 `.start.lock` 改为随启动过程创建即释放（含各失败出口），`stop` 不再代 `start` 删锁，
   「清空应用配置」兜底删除残留；`aria2.pid` 属主交回专属用户。见 `ADR-015` 第 14 条。

阶段一决策已确认。`0.1.96` 发布物已通过 `fnpack` 构建，版本 `0.1.96`，13,861,296 B，SHA-256 为
`F9F22A52F0E73F880FEAE9F29074BB22232F892C57D73A69C4E0C4D52B0353C9`；发布溯源以解包后的逐文件清单
`build-output/fpk-0.1.96.inner.SHA256SUMS`（63 个文件）为准，相对 `0.1.95` 仅 `server/aria2-manager`
（multicall 受控转发）、`cmd/main` 与 `cmd/uninstall_callback`（启动锁与 pid 属主）及 `manifest` 四项变化，
其余 59 项（含静态 Core、全部 34 个 `www` 静态资源、回调与向导）逐字节未变。
静态 Core：10,739,352 B / `a482fb24722b87a0a719acee330e329d6dfa21075808c66202e451782812015b`。
测试机已验证安装、启动、停止、卸载保留、Tracker 刷新、完成回调、端口监听、会话/DHT 保留、五类协议下载、
桌面 iframe 目标、AriaNg Socket 页面、删除/清空未完成任务文件清理和管理员 RPC 安全边界。


`0.1.90` 已在设备完成本轮需求的实测验收：默认下载目录与自动创建的 `completed` 归档目录、「下载完成后归档」默认为「是」且完成后自动搬移（含 rename 快速路径与用户真实 BT 任务）、界面修改下载目录时归档目录同步跟随、删除未完成任务连带清理部分文件与 `.aria2` 控制文件、简体中文界面与设置项可保存。验收入口为 `tests/fnos-verify-archive-follow.sh`、`tests/fnos-verify-cleanup.sh` 与 `tests/fnos-verify-ui.mjs`，截图证据在 `tests/ui-evidence/`。

`0.1.90` 起固化默认值：下载目录为飞牛授权的数据共享目录（`/vol1/@appshare/aria2/downloads`），归档目录默认为其下的 `completed` 且缺失时自动创建，「下载完成后归档」默认为「是」；在界面修改下载目录时，默认归档目录会跟随到新目录下的 `completed`。决策依据见 `docs/adr/ADR-010-default-download-and-archive-policy.md`。

`0.1.88` 已在测试机完成实测验收：下载完成后归档真实生效（含 12 个并发任务与用户自己的
BT 任务）、BT 任务完成即停止做种并离开活动列表、简体中文界面与“下载完成后归档/归档目录”
设置项可正常保存，网关 RPC 往返约 13-16 ms 且响应不再截断。验收入口为
`tests/fnos-verify-0188.sh`（设备侧检查）与 `tests/fnos-verify-ui.mjs`（真实浏览器界面验收），
截图证据在 `tests/ui-evidence/`，决策与证据记录见 `docs/adr/ADR-009-archiving-seeding-and-throughput-defaults.md`。
