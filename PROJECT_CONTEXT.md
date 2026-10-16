# 项目上下文

## 产品目标

构建一个可长期维护的原生飞牛 fnOS FPK 应用，将 Aria2 Pro 作为飞牛的一等应用提供。应用打包原生 Aria2 Pro Core 二进制和 AriaNg，运行时不依赖 Docker。

## 阶段一状态

阶段一的初始化与确认工作已完成。中文项目说明、需求分析、技术栈方向、总体架构、目录职责、需求追踪、安全模型、飞牛集成能力矩阵、参考资料基线、上游行为保真、许可证与产物溯源计划、开发计划和 8 项架构决策记录均已完成。

五项发布前产品决策已于 2026-08-27 按建议方案确认，Phase 2 已获准开始。

## 持续推进机制

阶段一曾作为持续性 `active` 目标推进。每次继续工作时，先检查仓库状态并重读
`AGENTS.md`、本文件及相关 ADR；随后只维护需求、架构、决策和可追踪性文档，
并复核不进入 Phase 2 的边界。每轮结束必须更新本文件并报告文档变更、验证结果、
遗留确认项和建议的下一步。五项决策已确认后，目标转入 Phase 2 实施；后续每轮仍
须更新本文件并报告验证结果。

## 文档基线与变更顺序

为避免持续推进时出现“文档已写明但决策未批准”或“实现状态覆盖产品约束”的问题，
后续变更按以下优先级解释：用户最新明确确认 > `docs/phase1-decisions.md` 中的已确认
决策 > 适用的 `docs/adr/` > 需求、架构和追踪文档 > `README.md` 的摘要 > 离线参考资料
中的上游行为。离线资料用于取证，不会自动改变产品范围。任何会改变端口、权限、目录、
网关、运行时依赖或发布架构的变更，必须先更新相关 ADR 和本文件，再更新摘要文档；
实现状态只能在有命令输出、测试记录或设备验证证据时标记为已完成。

最近一次阶段一审计于 2026-08-27 18:10 完成：阶段一核心文档和 8 份 ADR 均存在；端口、
网关和安全边界关键约束通过一致性检查。随后已按确认范围创建 `fpk/` Phase 2 空包
骨架；尚未创建 `manager/`、`ui-adapter/` 或 `tests/`，也未加入 Core、管理器或
AriaNg 运行资产。离线参考清单已校正为 38 条飞牛官方页面与 1 条 P3TERX 页面，
共 39 条记录。随后复核 ADR-001 至 ADR-008，确认原生打包、网关、
管理器、单管理员实例、离线资产、可配端口和安全边界均已有决策依据；除 D-001 至
D-005 外没有待确认的产品或架构决策。上游行为保真审计已将任务恢复、DHT、协议
支持、端口、Tracker、清理与 AriaNg 能力逐项标记为保留、适配、排除或后续实现；尤其
明确排除公开 RPC、AriaNg 多服务器/浏览器 secret、Rclone 和运行时远程脚本。
许可证审计已确认 Core 为 GPL-3.0、AriaNg 与 P3TERX 参考资产为 MIT；每个含 Core
的发布包必须提供同版许可证、哈希、构建记录与对应源码获取方式。
Web SDK 审计已确认目录选择/授权、文件管理器和主题/语言接口可用；SDK 返回值不能
替代管理器授权校验，且其离线 npm 快照未声明许可证，实际纳入产物前必须单独复核。

Phase 2 当前进度：`fpk/manifest`、`config/privilege`、`config/resource`、桌面
网关入口、安装向导和生命周期脚本占位已完成；`fnpack 1.2.3` 已于 2026-08-27
成功构建并校验 `aria2.fpk`。构建输出已确认包含必要 FPK 文件，且未执行测试机
安装。此前模板遗留的 `.DS_Store` 已从源目录清理，并通过嵌套 `app.tgz` 检查确认不再
进入归档。配置向导 `wizard/config` 因固定参考未
提供独立 schema 且工具校验结果不明确，暂缓到 Phase 4 验证，不以猜测格式发布。
测试机只读连通性检查：`192.168.x.x:22` 和 `192.168.x.x:5666` 可达。已通过
PuTTY `plink` 完成设备侧只读检查：设备为 Debian 12、x86_64、fnOS 内核
`6.18.18.c952-trim`；系统提供 `fnpack 1.2.4` 和 `appcenter-cli`。普通 SSH 管理员
会话执行 `appcenter-cli --help` 时因访问 `/run/trim_app_cgi/rpcbroker` 权限不足而
退出，未执行安装或修改；后续设备验证必须使用飞牛授权的应用中心流程或具备相应
系统权限的测试方式。设备未发现 `aria2` 已安装，也未发现 `6800`/`6888` 正在监听。
本轮审计同步修正了已确认决策、上游行为矩阵和阶段门槛的历史措辞，未改变产品范围
或安全边界。
2026-08-27 21:05 的持续审计确认：核心中文文档、需求追踪矩阵和 ADR 仍然齐全；
本轮复核确认核心文档缺失数和空文件数均为 0，ADR 数量仍为 8，FPK 源目录无
`.DS_Store`，顶层目录仍未新增业务源码目录。
未发现新的产品确认项或架构漂移。离线基线复核显示 `manifest.json` 含 39 条记录，
其中 38 条官方页面和 1 条 P3TERX 页面，与本地快照文件数一致。仓库中已存在的 `fpk/` 仍属于五项决策确认后的
Phase 2 空包骨架，本轮未新增 FPK 文件、业务源码或运行资产。安全约束扫描未发现
新的硬编码运行路径、公开 RPC、root 常驻服务或浏览器授权信任问题。飞牛集成矩阵中
D-003 的状态表述已同步为“已确认、后续实现”。README 中的文档
链接均指向当前存在的文件或目录。本轮检查了项目文档中的 11 个本地 Markdown 链接，
断链数为 0。随后检查嵌套的 `app.tgz`，确认当前 `aria2.fpk` 实际包含 3 个
`.DS_Store` 文件；此前关于它们未进入归档的记录已更正。该问题已通过清理源目录并
重新构建解决。当前 FPK SHA-256 为
`5CAB4E03D801908EFE17A050E6FB9DD6E948BCCC9D00B026055B18D0E8D253C2`；源目录和嵌套
归档均不再包含 `.DS_Store`。16 份核心中文文档均非空、包含一级标题和中文内容；顶层未新增
`manager/`、`ui-adapter/` 或 `tests/` 业务目录。
本轮再次核对 FPK 顶层与嵌套归档，确认清理后的归档状态正确。
顶层目录仍只有 `docs/`、`fpk/`、`reference-snapshot/` 和 `tools/`，没有新增业务源码目录。
本轮在 `docs/development-plan.md` 补充阶段一文档冻结与变更规则，未改变产品决策或
实现范围。本轮占位符扫描结果为 0，顶层目录仍未新增业务实现目录。核心文档缺失数和
空文件数均为 0；当前 FPK 源目录中的 `.DS_Store` 数量为 0。本轮复核确认阶段一
文档冻结规则已存在，未新增文件或实现内容。本轮再次验证源目录和嵌套归档中的
`.DS_Store` 均为 0，FPK 哈希仍为
`5CAB4E03D801908EFE17A050E6FB9DD6E948BCCC9D00B026055B18D0E8D253C2`。本轮复核确认
核心文档缺失数和空文件数均为 0，ADR 数量为 8，FPK 源目录 `.DS_Store` 数量为 0。
本轮边界扫描命中的公开 RPC、root 和浏览器授权相关文字均为安全约束或排除项，
不是启用配置；未发现新的违规实现。核心文档缺失数和空文件数均为 0，ADR 数量为 8，
FPK 源目录 `.DS_Store` 数量为 0，当前产物哈希未变化。最新哈希为
`5CAB4E03D801908EFE17A050E6FB9DD6E948BCCC9D00B026055B18D0E8D253C2`。

## 已确认约束

- 初始测试机为 `x86_64`、Debian 12、glibc 2.36。
- `tools/fnpack.exe` 为 fnpack 1.2.3，是本地 FPK 构建工具。
- 应用应使用飞牛生命周期脚本、专属应用用户、应用数据目录和统一网关。
- 必须保留 P3TERX Aria2 Pro 的有效能力：会话持久化、DHT 状态、Tracker 刷新、清理回调、可选移动/归档、合理下载参数和受控权限。
- Web 界面使用 AriaNg；`@trimjs/web-app` 可用于目录选择与授权、文件管理器、应用设置以及主题/语言适配。
- RPC 端口和 BT/DHT 监听端口都必须支持用户配置，默认值分别为 `6800` 和 `6888`。
- D-001：应用标识 `aria2`，显示名称「Aria2 Pro」。
- D-002：x86 首发。
- D-003：应用共享目录 + 管理员授权目录。
- D-004：首版包含移动/归档，默认关闭。
- D-005：开发期手动 FPK。

## 初始用户模型

主要用户是希望通过应用中心管理 Aria2 服务和 Web 界面的飞牛 NAS 管理员。第一版将下载服务视为管理员管理的共享资源，不默认支持普通用户或多租户下载归属。

## 安全基线

- Aria2 JSON-RPC 只监听 `127.0.0.1`。
- 飞牛统一网关通过 `/app/aria2` 将已认证请求转发至应用 Unix Socket。
- 第一版只允许管理员执行服务操作。
- RPC 密钥在安装时生成，以仅所有者可读权限保存，仅由服务端注入，不发送到浏览器。
- 端口配置必须持久化；修改端口时需校验范围、冲突和权限，并以受控方式重启 Aria2。
- 下载目录必须来自飞牛数据共享目录或明确的用户授权，不能假设能访问任意 `/vol*` 路径。

## 明确暂缓的能力

- 为 BT、PT、云盘等用途运行多个独立 Aria2 实例。
- 自动安装或自更新 Rclone。
- 公网/远程 RPC 和独立托管的 AriaNg。
- 除飞牛已提供能力外的移动端专用界面。
- 上架飞牛公开应用中心。

## 已确认的产品决策基线

以下五项决策已于 2026-08-27 确认；如需修改，必须先更新
`docs/phase1-decisions.md`、受影响 ADR 和本文件，再调整实现范围。

1. 发布名称与应用唯一标识：`aria2` / 「Aria2 Pro」。
2. 首发架构：x86；后续再评估 ARM。
3. 下载目录：应用共享目录 + 管理员授权目录。
4. 首版包含移动/归档能力，但默认关闭。
5. 开发期采用手动 FPK 分发。

## 上游与许可证说明

- 发布内容必须保留 Aria2、AriaNg、P3TERX Docker 脚本/配置和 Aria2 Pro Core 的上游署名与许可证。
- 发布元数据中必须固定源码修订版本与构建产物信息。
- `reference-snapshot/` 用于分析取证，不应作为正式运行时依赖直接打包。

## 文档索引

- `docs/requirements.md`：需求、风险和非目标。
- `docs/technology-stack.md`：技术栈和选型理由。
- `docs/architecture.md`：组件、数据、网络和端口设计。
- `docs/directory-structure.md`：源码与 FPK 目录职责。
- `docs/requirements-traceability.md`：需求到设计、开发阶段和验收证据的追踪矩阵。
- `docs/phase1-decisions.md`：进入 Phase 2 前需要用户确认的产品决策表。
- `docs/phase1-review.md`：阶段一当前证据、退出门槛和禁止越界事项。
- `docs/reference-baseline.md`：官方文档、上游源码和 Web SDK 的固定参考基线。
- `docs/security-model.md`：威胁模型、身份边界、端口变更和安全控制项。
- `docs/fnos-integration-matrix.md`：飞牛能力、SDK、统一网关和 Aria2 RPC 的职责边界。
- `docs/upstream-behavior-matrix.md`：上游能力与飞牛原生第一版的保留、适配和排除边界。
- `docs/license-and-provenance-plan.md`：许可证、资产溯源和发布验收物的计划。
- `docs/web-sdk-integration-contract.md`：飞牛 Web SDK 的允许调用、服务端复核和安全限制。
- `docs/development-plan.md`：Phase 1 至 Phase 7 开发阶段。
- `docs/adr/`：重要架构决策记录。

## Phase 3 当前进度

2026-09-01 卸载清空残留配置修复（0.1.87）：确认“清空所有应用配置”未删除旧版本遗留的
`$TRIM_PKGVAR/config.json`（测试机为 `/vol1/@appdata/aria2/config.json`），现已在
`fpk/cmd/uninstall_callback` 的清理清单加入该文件；下载目录和授权目录仍始终保留。
版本提升至 `0.1.87`，FPK SHA-256 为
`CA4E4E31A202C4D6F8A50782B79A1DDA13EA67322A92F411765F557F1FF90AA6`，本地 `bash -n`
语法检查通过，嵌套 FPK 清单确认版本为 `0.1.87` 且 `cmd/uninstall_callback` 含
`$TRIM_PKGVAR/config.json`。测试机按“卸载清空 → 装机 → 回归 → 再卸载清空”全流程验证：
0.1.87 安装后显示 `running`，AriaNg 中文界面加载当前 bundle，删除与清空 `error` 任务后
`slow.bin` 均消失；随后选择“清空所有应用配置”卸载 0.1.87，卸载后
`/vol1/@appdata/aria2/config.json` 已不存在，应用目录同时移除。最后重新安装 0.1.87，
测试机当前为 `0.1.87 / running`，`aria2c` 与管理器均以 `aria2` 用户运行，RPC 仅回环
`6800`，加入磁力任务后 BT `6888` 的 TCP/UDP 在 IPv4/IPv6 均完成绑定，验证后任务已移除、
下载目录为空，`$TRIM_PKGVAR/config.json` 不存在。新增 `tests/fnos-bt-port-check.sh`
可复用 BT 端口监听验收。

2026-09-01 历史 AriaNg bundle 清理（0.1.86）：删除 `build-output/ariang/js` 与
`fpk/app/www/js` 中 23 个未被 `index.html` 引用的历史 `aria-ng-*.min.js`，只保留当前入口
引用的 `aria-ng-2b80650589.min.js`；重新生成 `build-output/ariang.SHA256SUMS`
（111 行、清单 SHA-256 `62B22D5B07024A4A7E13E932BD08F172179F1323F1795E62C3E4CC509129E2DD`），
并在 `ui-adapter/README.md` 固化“同步前先清理旧 bundle、同步后重建哈希清单”的流程。
版本提升至 `0.1.86`，FPK SHA-256 为
`B30F1555F40409714B80290371405E2EBF183E48AD8E335B3E335C3FD8708463`，嵌套 `app.tgz`
确认只含 `www/js/aria-ng-2b80650589.min.js`。测试机通过 Web UI 卸载并选择“清空所有应用
配置”后，由 `trim-cli app install-fpk` 带 `wizard_rpc_port=6800`、`wizard_listen_port=6888`
全新安装，应用中心显示 `0.1.86 / running`；设备 Js 目录只含当前 bundle，AriaNg 中文页面
正常显示并加载该 bundle，删除 `error` 任务（GID `be18088e042e8853`）与“清空已停止任务”
（GID `40a56be7040d5784`）后 `slow.bin` 均再次验证消失，下载目录为空。新增
`tests/fnos-open-ui.mjs` 用于真实页面截图与 bundle 核验。发现新的清理缺口：卸载“清空
配置”未删除旧版本遗留的 `$TRIM_PKGVAR/config.json`（/vol1/@appdata/aria2/config.json），
该文件当前不被运行时使用，但会误导后续检查，下一步应在 `fpk/cmd/uninstall_callback` 中
一并删除并升版复验（该缺口已由 0.1.87 修复，见上方记录）。

2026-09-01 未完成任务删除清理修复（0.1.85）：用户反馈“删除未完成的任务并不会一并删除相关文件”。根因有两点：管理器 `/api/v1/cleanup` 只接受常规文件，多文件/BT 目录会被拒绝；AriaNg 只对 `error`/`removed` 的逐条 `removeTasks` 调用清理，`clearStoppedTasks` 完全没有触发清理。现清理接口支持授权根内的常规文件或目录，目录递归检查无符号链接后删除，并同步移除主文件旁的 `.aria2`/`.torrent` 控制文件与空父目录；AriaNg 在删除 `error`/`removed` 任务前调用清理，`clearStoppedTasks` 对非 `complete` 任务先清理再 `purgeDownloadResult`，完成的任务只移除记录、不触碰文件。管理器新增文件、目录、越界和缺失路径清理测试，`go test ./cmd/aria2-manager/... ./cmd/aria2-hook/...` 通过；AriaNg 构建设置先同步 `build-work/AriaNg` 再同步 `fpk/app/www`。版本提升至 `0.1.85`，FPK SHA-256 为 `34EF8F06522E09B89F0D7DAE9A1ED56B515B270C1D14BC76366420C531C5318C`，设备副本哈希一致，已通过 Web UI 卸载清空后安装，应用中心显示 `0.1.85 / running`。设备侧通过真实 AriaNg 页面完成两类回归：删除 `error` 任务（GID `055a0bf176af433a`）后 `slow.bin` 从下载目录消失；再次创建 `error` 任务（GID `b76449d88ee0e2eb`）并执行“清空已停止任务”后文件同样消失，下载目录为空。新增 `tests/fnos-cleanup-ui.mjs` 可复用两种动作的设备验收。剩余维护项：`fpk/app/www/js` 与 `build-output/ariang/js` 中仍有未被 `index.html` 引用的历史 `aria-ng-*.min.js`，不影响运行，后续打包时可清理。

2026-09-01 语言问题现场结论：测试机页面仍显示英文，但设备已确认加载 `aria-ng-aa680fcd82.min.js`，该 bundle 内启动入口明确执行 `applyLanguage("zh_Hans")`。因此剩余问题不是卸载配置，而是飞牛桌面 iframe 可能显示了另一个缓存/渲染上下文；当前不能把源码或 FPK 构建结果当作界面验收证据。后续必须在同一飞牛桌面窗口中检查 iframe 实际资源请求和运行时 `$translate.use()` 值，并清理/重建该窗口实例后再判断。此前强制中文修复代码已继续增强为 `0.1.80` 工作区版本，但设备端安装版本和实际 bundle 需要重新核对，禁止宣称已解决。

2026-09-01 测试机实际语言状态复核：`0.1.78` 已在设备运行且入口加载 `aria-ng-aa680fcd82.min.js`，但真实 iframe 仍显示英文；旧 `Options` 中存在已初始化的英文语言状态，导致仅检查初始化标记不足。现将迁移条件扩展为“标记不存在或语言为 `en` 时强制改为 `zh_Hans`”，并在手动切换时保留标记。新版本 `0.1.79` 已完成前端构建和 FPK 打包，SHA-256 为 `B73AF65E72B048BBA9D295329B29251A2D1627E2FF33C14610D7901E33203558`。此前现场验证方法已改为：设备版本、入口 bundle 文件名、iframe 可见文本三项必须同时确认；`0.1.79` 尚待设备 Web UI 卸载/安装后最终验证。

2026-09-01 语言迁移彻底修复（0.1.78）：测试机实际加载新 bundle 后仍显示英文，确认旧 `Options` 中的语言值可能不是简单的 `en`，原迁移条件覆盖不足。AriaNg 现增加一次性 `fnosLanguageInitialized` 标记：升级/首次运行发现标记不存在时强制设为 `zh_Hans` 并持久化；用户之后手动切换语言会保留选择，不再每次启动覆盖。前端构建成功并生成新 bundle `aria-ng-aa680fcd82.min.js`，FPK 版本 `0.1.78`，SHA-256 为 `E58F25537A699019C54A8216506C8FFA05724E808876ABC22E7323786BAC45CB`。此前测试机现场验证证明旧版本仍为英文；新包尚未再次通过 Web UI 卸载/安装完成最终截图验收。

2026-09-01 测试机现场核验纠正：通过飞牛桌面真实 iframe 读取到的 AriaNg 页面仍显示英文（`New`、`Downloading`、`Settings`、`Connected` 等），此前“中文已验证”的结论不成立。进一步从设备 `/var/apps/aria2/target/www/index.html` 确认实际仍引用旧 bundle `aria-ng-35ee463325.min.js`。尝试使用 `appcenter-cli install-fpk` 覆盖安装新包时，设备仍显示 `0.1.74`；尝试 CLI 卸载被 App Center 拒绝并提示必须从 Web UI 卸载，因此 `0.1.77` 尚未真正部署。问题当前分为两部分：代码已将首次默认语言固定为 `zh_Hans`，但必须先通过飞牛 Web UI 完成卸载/安装，才能取得新 bundle；卸载向导的“清空应用配置”也需在 Web UI 中选择。后续验收必须以设备显示版本和实际 bundle 文件名为准，安装完成后再次打开桌面 iframe 截图确认中文，不再仅依据本地 FPK 哈希判断。

2026-09-01 默认中文逻辑修复（0.1.76）：确认其他新机器也显示英文的根因是 AriaNg 首次启动会优先采用宿主浏览器语言，即使产品默认配置为 `zh_Hans`。现将 `getDefaultLanguage()` 固定返回 `ariaNgConstants.defaultLanguage`，因此首次安装统一进入简体中文；用户仍可在 AriaNg 设置中手动切换其他语言。前端构建成功，静态资源已同步，FPK 已重新打包，SHA-256 为 `8A6C9E9A504B571A1E5E36E8CCE5E69D97CA08629EFDA9E29F744EB1A8DAB9B9`。本轮管理器未发生源码变化，Go 二进制沿用上一版已验证产物；下一步应在测试机和另一台机器安装 `0.1.76`，确认全新会话默认中文及手动切换仍可用。

2026-09-01 测试机英文缓存强化修复（0.1.75）：确认应用配置清空不会影响飞牛桌面浏览器的 `localStorage`，而旧版入口/脚本缓存仍可能让测试机加载英文 AriaNg。管理器静态文件服务现统一发送 `Cache-Control: no-store, no-cache, must-revalidate`，升级后入口 HTML、语言资源和脚本会重新获取；同时重新构建并同步 AriaNg 资源、重新交叉编译管理器并打包。版本 `0.1.75`，FPK SHA-256 为 `0EFEA1F32D73B3E2B459E5C174A6D975F707035753E3CA4CBC38B756982938D1`。本地 Node 检查和前端构建通过。清空配置仍只处理应用侧配置，不会删除浏览器站点存储；若升级后仍显示英文，必须在飞牛桌面清除该应用站点数据或使用无痕/新会话，这是浏览器安全边界，应用卸载脚本无法直接代替。

2026-09-01 卸载配置保留策略（0.1.74）：确认 AriaNg 的语言选择保存在浏览器 localStorage（`Language.*`），不属于 FPK 应用目录，卸载应用无法直接清除，因此测试机可能继续复用旧英文偏好。新增 `wizard/uninstall` 卸载向导，默认“保留配置”，并提供“清空所有应用配置”选项；选择清空时仅删除 `TRIM_PKGETC` 和 `TRIM_PKGVAR` 中的 Aria2 配置、密钥、会话、Tracker、日志及 PID 文件，始终保留授权下载目录和用户下载文件。卸载脚本加入值校验与路径边界保护。版本提升至 `0.1.74`，FPK 构建成功，SHA-256 为 `FE55B5512C456A518AE9B44515618D4E191B35BA20ED9FA9B45767AC4EC0FACE`；Node 语法检查通过。设备安装待用户确认卸载向导选项后进行；若清空配置后仍显示英文，还需在测试机浏览器中清除该站点 localStorage 或使用新浏览器会话，这是应用卸载权限无法覆盖的浏览器侧状态。

2026-09-01 下载目录与任务清理修复（0.1.73）：管理器新增受保护的 `/api/v1/cleanup`，仅接受授权根目录内的常规文件，并同步删除主文件、`.aria2` 及 `.torrent` 控制文件；AriaNg 删除已完成/已停止任务前调用该接口，BT 任务额外清理按 infoHash 命名的 torrent 元数据。下载目录变更时管理器同步调用 `aria2.changeGlobalOption(dir=...)`，新任务立即使用新目录，且仍持久化到 `TRIM_PKGETC/config.json`。Go 单元测试、Node 语法检查和 AriaNg 构建均通过；重新交叉编译管理器/Hook 后生成 FPK `0.1.73`，SHA-256 为 `DEC2134FB741FF6AE4D1D438F28C7074E875787E879A1AB40EBA4E778BA743E0`。测试机已卸载旧版并安装启动新包，状态 `running`，Aria2 运行用户为 `aria2`，配置 `dir=/vol1/1000/download`、RPC `6800`、BT `6888`。通过 Unix Socket 以管理员上下文调用清理接口返回 `204`，实测主文件、`.aria2` 和 `.torrent` 均删除。尚待在真实 AriaNg 页面完成一次“已完成/已停止”点击删除回归，以确认前端请求链路与设备文件结果一致。

2026-08-31 用户反馈 Bad Gateway 后的现场复验：此前结论不直接沿用，重新登录测试机飞牛桌面并
点击桌面“Aria2”图标。当前版本 `0.1.56` 的 iframe 实际加载 `/app/aria2/`，页面 DOM 可读，
显示 AriaNg `v1.3.14`、任务列表、搜索框和底部状态栏，左侧状态明确为“已连接”；现场截图显示
完整 AriaNg 界面而非 Bad Gateway。设备侧同时确认 `aria2c` 与 `aria2-manager` 均由
`aria2` 用户运行，`/run/aria2/aria2.sock` 为真实 Socket，RPC 仅监听回环。该复验
证明当前测试机在本轮重新打开应用时没有复现 Bad Gateway；后续若用户仍看到错误，应优先记录
发生时间、浏览器地址和应用中心状态，以便对同一现场抓取日志，而不是用历史截图替代当前证据。

2026-08-31 启动与 Web 持续复核（0.1.56）：通过 PuTTY 连接测试机确认 `aria2c`（PID 538441）与
`aria2-manager`（PID 538444）均以专属用户 `aria2` 运行，已持续约 7 小时；运行 Socket
`/run/aria2/aria2.sock` 为真实 Unix Socket，应用目录链接由生命周期维护。Core RPC 仅监听
回环 `127.0.0.1:6801` 与 `[::1]:6801`。连续执行 5 次启动回归时，设备上未安装 `trim-cli` 的
普通 root SSH 环境无法直接调用该命令，但现有服务进程和 Socket 在每次检查后均保持正常，未出现
并发启动删除 Socket 的回归。通过飞牛桌面登录并点击“Aria2”图标，真实 iframe 加载 AriaNg
`v1.3.14`，DOM 可见导航、任务列表、搜索框和底部状态栏，页面显示“已连接”；截图已留存为本轮
验收证据。直接从未经网关授权的本机 HTTP 请求返回 `200` 及应用自身 `invalid token`，证明网关已
到达管理器而非 Bad Gateway。应用启动闭环与 Web 可见性问题已关闭，后续可继续做端口修改、目录
权限和删除清理专项回归。

2026-08-31 设备启动闭环（0.1.50）：通过 `trim-cli` 干净卸载旧包并安装后，显式启动返回成功，
App Center 状态为 `running`。设备进程确认 `aria2c` 与 `aria2-manager` 均以 `aria2` 用户运行；
Core RPC 仅监听回环 `127.0.0.1:6801` 和 `[::1]:6801`；运行目录 `/run/aria2` 已生成配置和密钥，
应用目录 Socket 符号链接存在。此前 0.1.49 的 `invalid RPC secret` 已由直接环境变量传递修复，
本轮日志无新启动错误。持续目标仍保留 active，下一步补做 Unix Socket 健康接口和稳定性复测。

2026-08-31 配置环境变量传递修复（0.1.49）：设备验证显示 fnOS 生命周期会将标准输入截断，导致
FD 方案读取到空 JSON；管理器新增 `--config-env`，生命周期将已读取的非敏感 JSON 配置通过环境变量
传入，RPC 密钥仍通过继承 FD 读取。Go 单元测试、Linux x86_64 交叉编译和 FPK 构建均通过，最新
FPK SHA-256 为 `AE8EA450C0BD8AAF7BF5FB454E1B464E074DCD89F90EA956D752F8A28A57315D`；设备安装验证待执行。

2026-08-31 稳定启动验收（0.1.50）：`trim-cli` 干净卸载并安装后，显式启动成功，App Center 返回
`status=running`。设备复核确认 `aria2c` 与 `aria2-manager` 均以 `aria2` 用户运行，已持续
约 29 分钟；RPC 仅监听 `127.0.0.1:6801` 与 `[::1]:6801`，Unix Socket
`/run/aria2/aria2.sock` 存在且应用目录符号链接有效。通过网关 HTTP 请求得到 `200`（随后因
缺少网关 token 返回应用自身的 `invalid token`，证明入口已到达管理器），服务未被回收。启动稳定性
目标的设备证据闭环已完成；后续可继续做端口修改、目录授权和删除清理的专项回归。

2026-08-31 FD 读取实现修复（0.1.47）：`os.ReadFile(/proc/self/fd/N)` 在 fnOS 生命周期中仍被拒绝，
改为 Go 直接从继承的文件描述符读取配置与 RPC 密钥，避免再次打开 procfs 路径。已通过 Go 单元测试、
Linux x86_64 交叉编译和 FPK 构建；最新 FPK SHA-256 为
`49EA73D5F300F6A4FB75F144A913C1EA96D679052E5A23D968827C96653D46D7`，设备安装验证待执行。

2026-08-31 管理器降权边界修复（0.1.45）：前述路径/权限调整在 fnOS 生命周期中仍出现配置读取拒绝，
因此管理器现在由生命周期 root 启动，仅在读取并校验配置、密钥后通过 `setpriv --init-groups`
重新执行自身并以 `aria2` 常驻；生命周期脚本不再直接把管理器置于受限派生上下文。已重新编译
Linux x86_64 管理器，Go 单元测试通过，FPK 构建成功，SHA-256 为
`525A4317C20706BE84A1A0908E5D39A56AE6F53CF7D2B08E2E9DD9E51A3445A1`。设备升级验证尚待执行。

2026-08-31 文件描述符启动修复（0.1.46）：设备证据显示生命周期在管理器首次 `os.ReadFile` 阶段
拒绝 `/run/aria2/config.json`，即使文件为 `0644`；新增 `--config-fd/--secret-fd`，由生命周期
先打开文件并通过继承 FD 传递，管理器从 `/proc/self/fd/*` 读取后再降权。已通过 Go 单元测试和
Linux x86_64 交叉编译。FPK 已重新生成并复制到仓库根目录，SHA-256 为
`A266A37B439B87EEC2074A83E2E6302A9D62E354EAB7F9D9F6540E43726D6AC6`；设备安装验证待执行。

2026-08-31 运行文件权限调整（0.1.44）：设备验证显示 fnOS 生命周期上下文对 `/run/aria2` 下的
`0600` 配置同样返回 `permission denied`，改为 `755` 运行目录、`0644` 运行配置、`0640` RPC 密钥；
root SSH 中以 `setpriv` 读取配置和管理器预检均成功。已通过脚本语法检查并生成 FPK，SHA-256 为
`63C4C0D2AD9C584B162D6BE360948ADFAF537D5E24008591CB3333DA3AF22D29`；设备升级验证待执行。

2026-08-31 启动权限修复（0.1.43）：测试机确认生命周期上下文无法让服务进程打开 `/vol1` 下的
配置文件，即使属主和模式正确；同一文件在 root SSH 中可读。`cmd/main` 现将运行时配置、密钥和
Unix Socket 放入 `/run/aria2`（每次启动重建，属主为 `aria2`），持久配置仍在 `TRIM_PKGETC`，
停止时回写，并在应用目录保留 `aria2.sock` 符号链接以兼容网关。同步移除调试输出。已通过
`bash -n`、Go 单元测试并生成 FPK，SHA-256 为
`E57DA6655640954CB7D443744631DF142E31D211FA171C3CF45CF85AD0B47429`；设备验证待执行。

2026-08-31 App Center 队列复核：清理安装任务后应用状态可短暂回到 `stopped`，但显式 start
仍返回 `10330` 并重新进入 `start`。设备日志显示生命周期反复生成 `600` 配置副本，尽管同一
脚本的部分副本带 ACL；应用中心存在自动重试/并发调用，启动互斥尚未阻止这些历史调用。目标继续
保持 active，未标记完成。

2026-08-31 并发隔离验证：尝试使用 `--no-start` 安装 `0.1.41` 以隔离 App Center 自动启动，
但设备安装任务长时间保持 `starting`，随后显式 start 被平台拒绝（10500），未获得有效启动证据。
该结果表明测试机当前 App Center 任务状态本身也存在残留/排队影响；目标继续保持 active。

2026-08-31 启动互斥设备回归：`0.1.41` 已安装，应用中心状态仍为 `start`。启动互斥尚未消除
管理器配置读取失败，目标保持 active；未将设备状态误报为 running。

2026-08-31 启动互斥修复：针对设备安装自动启动与手动启动并发，`cmd/main start` 现使用
`TRIM_PKGVAR/.start.lock` 互斥目录，重复调用直接复用现有实例；版本提升至 `0.1.41`。
本地 FPK SHA-256 为 `4134C36E776CFE9E46526228A394424E15397A26688E7633B096328C955DBB8A`，待设备安装验证。

2026-08-31 ACL 回归：`0.1.40` 安装后部分运行副本已出现 ACL `user:aria2:r--`，但应用中心
自动启动仍有并发实例生成 `600` 副本并失败，状态保持 `start`；Aria2 可短暂监听 RPC 后被回收。
现场同时存在多个带不同 PID 的运行配置，说明安装自动启动/生命周期调用存在并发竞争。下一步应
增加启动互斥和预检，确保仅使用 ACL 已验证可读的副本后再启动管理器。

2026-08-31 fnOS ACL 适配继续：`0.1.39` 设备验证确认 target 目录默认 ACL 会把新建运行配置设为
`600`，导致管理器无法读取。启动脚本现对新建配置和 RPC 密钥显式执行
`setfacl -m u:aria2:r`，版本提升至 `0.1.40`。FPK SHA-256 为
`F0AC6B3FA02058CABA3A06C2D73F766556921D1A36A19058F6B87213945C147C`；待设备安装验证。

2026-08-31 路径隔离调整：设备环境确认 `/tmp` 临时文件受生命周期上下文影响，现将运行配置和
RPC 密钥移至脚本解析出的 `TRIM_APPDEST/target`，使用 `TRIM_TEMP_LOGFILE` 与脚本 PID 生成唯一
文件名，并以 `cat` 创建。版本提升至 `0.1.39`，FPK SHA-256 为
`CE328D3C14653F70D1E357A6005F60D658A19D97B1D03C733C32397DF9E78482`；本地原生 Go 测试需在未设置
交叉编译变量的环境复跑，设备安装验证待继续。

2026-08-31 唯一文件回归：`0.1.38` 已确认唯一配置文件生成，但管理器仍在 fnOS 自动启动上下文
报 `permission denied`；`setpriv` 手工读取同一文件正常。当前继续目标聚焦于生命周期并发/上下文
差异，尚未将设备状态标记为 running。

2026-08-31 唯一临时文件回归：`0.1.38` 已生成带生命周期 PID 的唯一临时配置，设备确认其为
`root:root 644`，且手工 `setpriv` 管理器读取成功；但应用中心启动日志仍报告同一路径
`permission denied` 并回收 Aria2。该现象只在 fnOS 生命周期上下文出现，当前仍未伪造运行成功，
目标保持 active。

2026-08-31 临时文件残留修复：设备显示每次启动复用的临时配置可能已由前一轮以 `root:root 600`
创建，导致后续即使使用 `umask 022` 也无法读取。启动脚本现先删除本次 `TRIM_TEMP_LOGFILE` 派生
文件，再以 `cat >` 创建配置和密钥副本。版本提升至 `0.1.37`，FPK SHA-256 为
`451CB19E5959CE16E4B1CD01A0D1EF0604E8865945DD2D5AC3C5F414BE661EE4`；待设备安装验证。

2026-08-31 临时副本权限修复：设备诊断确认 `cp` 继承了持久配置的 `600` 模式，导致唯一临时
配置仍为 root-only。已改用 `cat >` 配合 `umask 022` 创建配置与密钥副本，版本提升至 `0.1.36`。
本地 FPK SHA-256 为 `7324E6C4D4C95F9138CE45416169C07EE33BF9DDECC1F378CA3EAE8C9B8AEBF0`；待测试机安装后
验证管理器是否能读取唯一临时文件并保持 running。

2026-08-31 临时目录回归：`0.1.33` 已生成 `/tmp/aria2.config.json`，但应用中心启动时管理器
仍报告该文件 `permission denied`，随后 Aria2 被健康检查回收。设备上用同一 `setpriv` 命令手工
读取/启动管理器可成功，说明异常仅存在生命周期脚本的启动时序或文件并发处理；目标保持 active，
尚未标记完成。

2026-08-31 环境隔离修复：通过 `TRIM_TEMP_LOGFILE` 获取真实 fnOS 生命周期环境，确认
`TRIM_APPDEST=/vol1/@appcenter/aria2` 与 `/var/apps/aria2/cmd/main` 并存。运行配置和 RPC
密钥现镜像到由 `TRIM_TEMP_LOGFILE` 推导的临时目录，停止时由 root 回写持久配置，避免管理器
访问受 ACL 限制的 `/vol1` 别名。版本提升至 `0.1.33`，FPK SHA-256 为
`36BAD24BBFD87FA087F1A5ADB7832D6E5400C892C4FC0B542BB0508480DCE54E`。已安装测试机但状态仍为
`start`，尚未通过最终启动验收；下一轮继续读取最新 manager 日志验证临时配置路径。

2026-08-31 设备回归：`0.1.31` 已安装，路径解析已从生命周期脚本自身位置获得 `/var/apps` 根目录，
但应用中心仍返回 `10330`。清理测试残留 PID 后，管理器日志仍报告读取
`/var/apps/aria2/target/aria2.config.json` 权限失败；同一设备上手工 `setpriv` 直接执行
管理器可以读取该文件，说明差异仍在 fnOS 生命周期启动上下文。当前未伪造 running 结果，目标继续
保持 active，下一步需对生命周期上下文做最小化 wrapper/环境复现。

2026-08-31 权限启动方案迭代：`su` 和 `start-stop-daemon` 组合在设备上仍出现管理器读取运行
配置的权限异常；测试机确认提供 `/usr/bin/setpriv`。已改用
`setpriv --reuid=aria2 --regid=aria2 --init-groups` 后台启动管理器，保留专属用户和完整
补充组，版本提升至 `0.1.30`。本地 FPK SHA-256 为
`C2584D377071B360289B250CF95F24B9FFCE20A85D7C29466766B610F855E6DE`；尚待设备安装验证。

2026-08-31 启动时序修复：设备日志显示 Aria2 与管理器可被拉起，但 1 秒健康检查窗口过短，
生命周期随后执行 stop，造成应用中心 `10330`。已将观察窗口延长至 3 秒，版本提升至 `0.1.28`。
本地 FPK SHA-256 为 `15DE76B06FF800E25D12083087E8908B4E48270278DA72E3FE8F9ED4753D03FD`；待设备安装
验证稳定运行、Socket 和端口监听。

2026-08-31 设备验证补充：`0.1.26` 安装后，管理器进程可见且以 `aria2` 用户运行，运行配置
副本权限为 `aria2:aria2`；但应用中心仍返回 `10330` 并将状态保持为 `start`，Aria2
在启动后约 2 秒被生命周期健康检查回收。设备上存在此前手工测试遗留的 Aria2 进程，已确认其
会干扰端口与进程验收；后续需使用不匹配自身 SSH 命令的精确 PID 清理方式，再验证
`start-stop-daemon --chuid aria2:aria2` 的独立启动退出码。目标保持 active。

2026-08-31 设备回归：`0.1.24`、`0.1.25` 在测试机仍显示 `start` 而非 `running`；运行配置副本已
成功创建为 `TRIM_APPDEST/target/aria2.config.json`，但应用中心启动流程仍未留下新的管理器
进程。已验证直接使用 `su -s /bin/sh aria2` 启动管理器可读取配置；当前改为尝试
`start-stop-daemon --chuid aria2:aria2` 初始化专属组，版本 `0.1.26` 已本地打包，FPK
SHA-256 为 `200B758D2F83EA4F49E6143E1846DD51D970623BEEADDBEC09150473CBA8A818`，尚待设备安装验证。

2026-08-31 持续启动修复：设备 ACL 使 `start-stop-daemon --chuid aria2` 无法稳定读取
`TRIM_PKGETC/config.json`。生命周期现将配置复制到可控的应用运行根目录
`TRIM_APPDEST/target/aria2.config.json`，并在启动前设置可读权限；管理器、端口探测和
Aria2 配置生成统一使用该运行副本，避免直接依赖受限的 `/vol1` 配置挂载。版本提升至
`0.1.24`，FPK SHA-256 为 `8656572D66CFA43569BAACCCA2B2D1E826AC25064E9F70E6AD6892B93AE55FB8`。
本机原生 Go 单元测试此前已通过；该版本待测试机重新安装后验证进程、Socket 与端口。

2026-08-31 启动故障持续修复：在测试机复现发现 fnOS 生命周期通过 `/var/apps/aria2/cmd/main`
调用脚本时，`TRIM_APPDEST` 指向应用框架根目录，实际运行资产位于 `target` 符号链接；旧脚本
直接拼接 `TRIM_APPDEST/server` 导致 `aria2-manager: No such file or directory`，并且未给
`target` 目录授予专属组写权限。已在 `main`、安装和升级回调中加入 target 根目录解析及权限处理，
版本提升至 `0.1.20`。本地 FPK 已重建，SHA-256 为
`647E84BDB30A0F8C996F22C90968F284D3E1246BFA69365EFE6E62FA0AAA89A4`；原生 Go 测试全部通过。
测试机已完成 `0.1.20` 安装，但启动仍返回平台错误，进一步手工执行确认管理器以 `aria2`
启动时读取配置仍受设备路径 ACL 影响；下一轮将继续修复配置目录访问并完成设备启动闭环。

2026-08-27 已完成原生产物版本基线审计：P3TERX Aria2 Pro Core 固定为
`1.36.0 build 2021.08.22`，本地源码快照 revision 为
`4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062`；AriaNg 固定为 `1.3.14`，源码快照
revision 为 `d6a765377e1eecfbcc387dcb824124df114decfb`。详细构建、x86_64/glibc
兼容性、哈希、许可证和 PuTTY 验收门槛见 `docs/phase3-build-baseline.md`。
Core、管理器和 AriaNg 运行资产已构建并纳入候选 FPK；本段早期审计记录保留作为历史证据。

本轮验证结果：`tools/fnpack.exe --help` 正常返回版本 `1.2.3`；现有
`aria2.fpk` SHA-256 仍为
`5CAB4E03D801908EFE17A050E6FB9DD6E948BCCC9D00B026055B18D0E8D253C2`；FPK 源目录
`.DS_Store` 数量为 0；docs 本地 Markdown 链接断链数为 0。下一步为建立 Core
构建环境并生成首个 x86_64 产物，之后再纳入 FPK。

2026-08-27 自动持续任务检查：已复核 `AGENTS.md`、本文件和 Phase 3 基线；确认
Windows 主机虽有 `wsl.exe`，但当前没有可用 Linux 发行版，且 Core 快照仅含构建脚本
与补丁、不含 Aria2 主源码；本机也没有 gcc/make/go/readelf/ldd。Phase 3 因此暂时
阻塞于“审查过的固定源码输入 + Linux 构建环境”，未生成虚假产物或修改 FPK。
该阻塞和下一步前置已记录在 `docs/phase3-build-baseline.md`。

2026-08-27 17:41 自动任务复核：工作区仍保持 Phase 3 基线状态；Core 快照未包含
Aria2 主源码，Windows 主机仍无可用 WSL 发行版及 gcc/make/readelf/ldd/go。未发现
新的可安全执行构建步骤，因此未修改 FPK 或生成未经验证的产物。自动任务将继续等待
固定源码输入和 Linux 构建环境，同时保持后续 Phase 4-7 任务清单可追踪。

2026-08-27 自动任务离线输入审计：已复核三份上游快照的固定 revision，并记录 Core、
AriaNg、Docker 参考许可证及 AriaNg `package.json` 的 SHA-256，未下载或执行远程内容。
Core 快照仍未发现 Aria2 主源码 tarball 或源码树；构建阻塞保持不变。完整清单见
`docs/phase3-build-baseline.md`。

2026-08-27 自动任务构建脚本审计：确认上游 Core 脚本会克隆 Aria2 `master` 并在线下载
依赖，不符合 ADR-005。新增 `docs/phase3-build-input-lock.md`，将正式构建改为“固定离线
源码、逐项 SHA-256 校验、禁网执行”的前置规则；在输入未齐备前不执行该脚本。

2026-08-27 自动任务补丁审计：已计算并固定 4 个 Aria2 Pro Core 补丁的 SHA-256，且
确认 Core 快照工作树无本地修改。构建输入锁定清单现要求先校验补丁哈希和
`git apply --check`，再允许编译；这一步不需要下载或执行远程内容。

2026-08-27 21:44 自动任务复核：4 个 Core 补丁的实际 SHA-256 与锁定清单全部匹配；
现有 `aria2.fpk` SHA-256 仍为 `5CAB4E03D801908EFE17A050E6FB9DD6E948BCCC9D00B026055B18D0E8D253C2`。
由于 Aria2 主源码和 Linux 工具链仍未就绪，本轮继续保持“不编译、不修改 FPK”的边界。

2026-08-27 自动任务环境处理：确认测试机有 Docker 客户端但 daemon 未运行；Docker 仅可作为临时构建隔离，不能进入最终 FPK 运行时。因宿主编译器缺失且 sudo 未授权，本轮未修改系统或 FPK。

2026-08-27 自动任务已下载并固定 Aria2 `release-1.36.0` 源码 tarball，Windows 与
测试机双端 SHA-256 均为 `782B5DAD48CB333E0CBEF8795F868F7963455EFECFD2A818FEB8DAB1CFA973F5`。
通过 PuTTY 检查发现测试机缺少编译工具链；sudo 安装依赖因设备 sudo 密码与 SSH 密码
不一致而失败，未修改系统。构建仍待具备 sudo 权限或预装工具链。

2026-08-27 22:44 自动任务复核：再次确认 Core 构建脚本存在，但 Aria2 主源码目录仍不
存在；本机 `gcc`、`make`、`go`、`readelf`、`ldd` 均不可用。现有 FPK SHA-256 保持
不变，未越过 Phase 3 构建门槛。

2026-08-27 自动任务已更新为围绕已锁定源码继续推进：优先使用具备授权的 Linux
x86_64 环境完成离线构建；测试机 sudo 权限不足时不重复失败安装。当前本地源码
tarball 哈希与 `build-inputs/SHA256SUMS` 一致，Windows 工具链仍不可用，FPK 未修改。

2026-08-27 自动目标构建进展：源码校验已通过，但首个固定补丁在 Aria2 `release-1.36.0`
上无法直接应用（`OptionHandlerFactory.cc:440` 上下文不匹配）。已停止该次构建，未
静默跳过补丁；构建脚本改为受控 fuzz 检查，下一步将逐个验证补丁是否可安全移植。

2026-08-27 自动任务构建环境推进：已确认测试机 root 密码可用于 Docker，成功拉取
固定摘要的 `debian:12-slim` 镜像并启动一次性构建容器。容器内依赖安装已开始，但
本轮尚未产出 `aria2c`（输出目录为空），因此未进入补丁应用或 FPK 集成；后续继续
收集容器构建日志并修复构建脚本。

2026-08-27 自动目标执行：已实际启动 Docker 构建并定位两个挂载路径错误；源码校验
已通过，已修正构建脚本的清单路径和源码归档路径，准备继续重试，尚未生成二进制。

2026-08-27 自动目标执行：源码与 4 个补丁已全部成功校验并应用；构建随后在
`autoreconf` 阶段因容器缺少 `autopoint` 失败。已将 `autopoint`/`gettext` 加入固定
构建依赖，下一轮继续执行，未修改 FPK。

2026-08-27 自动目标执行：已在测试机 Docker Debian 12 x86_64 中完成 Aria2 Core 编译，
源码和 4 个补丁均通过；产物已回传 `build-output/aria2c`，SHA-256 为
`EEFB98AFB22DD3A4B7BF70705F064704257542C85FAEF424D6273CF0CCB5729E`。测试机上修正
执行权限后 `aria2c --version` 成功。`file`/`ldd` 证据表明当前是动态链接 ELF，纳入
FPK 前仍需完成 fnOS ABI 兼容性和依赖策略验证。

2026-08-27 自动目标执行：AriaNg `1.3.14` 已在本机隔离构建成功，产物为
`build-output/ariang` 下 32 个静态文件；其哈希清单 SHA-256 为
`6C9C8D214B1E25EB953899C7AB313F772D6FE6FA753B1F1C839EBC94028363FA`。Core 与
AriaNg 的来源、构建器、产物哈希、许可证和剩余 ABI 门槛已汇总到
`docs/phase3-build-record.md`。

2026-08-28 服务实现进展：已实现并测试 Go 管理器的 Unix Socket、网关用户头校验、静态
AriaNg 服务、服务端 token 注入的 loopback RPC 代理、端口校验与持久配置 API。设备侧
临时 TRIM 环境烟测已覆盖安装、启停、静态页面、RPC，以及将端口改为 `6801`/`6889` 后
重启并再次请求 RPC。当前 FPK SHA-256 为
`26AC17D8BD1BB092979DF69681DC1BF7152A1CB003361B468EE3892BAD5F0596`。完整 App Center
授权安装、下载目录 SDK 授权和 Tracker/清理/归档行为仍待后续阶段验证。

2026-08-27 持续执行说明：自动化已调整为最短心跳间隔，但平台不支持后台无限占用
单个执行会话；每次心跳都会立即执行下一项工作，不会把“等待”当作完成。当前容器已
退出且无输出，下一步需重新启动可观测构建并保留完整日志。
2026-08-27 自动目标执行：已生成 AriaNg 1.3.14 源码归档并固定 SHA-256
`83D8651490FDF225D635461FF3090D7CC1CDDB321EB02D807D2220037FB7F389`，上传测试机后
启动 Node 18 构建容器。容器镜像拉取完成但构建输出目录暂为空，下一步继续获取 npm
构建日志并完成静态资源哈希记录；Core 产物已保留在 `build-output/aria2c`。

2026-08-28 真实 App Center 安装验证：已修复向导端口未落盘、安装钩子过早以应用用户创建持久目录、首次 aria2.session 缺失及后台 PID 不对应真实服务进程的问题。最终包使用 root 生命周期钩子初始化目录，再用 start-stop-daemon --chuid aria2 运行常驻服务；应用中心的干净安装与启动均已完成并收敛为 running。测试环境中，安装向导的 6801/6889 已写入 config.json；aria2c 和管理器均以 aria2 身份运行，管理器 Socket 位于 TRIM_APPDEST，健康检查和经 Socket 的 aria2.getVersion 均成功，RPC 仅监听 127.0.0.1:6801。当前 FPK SHA-256 为 C263C2ED8A8AE3A31D6B3074A8B6A867C66A70A35F963956FBD182B42D486CB5。尚未完成 Phase 6 的 Tracker、清理和归档行为，以及完整桌面网关/UI、端口变更后的 TCP/UDP 监听、升级和卸载数据保留验证。

2026-08-28 Phase 6 回调实现：新增静态 aria2-hook，由 aria2c 的 on-download-complete 调用。它只在 TRIM_DATA_SHARE_PATHS 的规范化授权根内删除常规 .aria2/.torrent 控制文件；归档默认关闭，启用后仍要求 archive_root 位于授权根内、目标不存在且使用原子 rename，失败保留源文件。Go 单元测试和测试机隔离目录测试均已通过，后者确认归档、控制文件清理和下载文件保留行为。新 FPK SHA-256 为 29878BC7E759181640C3A05BB360954AD4456F6B10EA09A68FBB0D4D1ECFB0F3。Tracker 的受限热更新与将此回调接入已安装 App Center 实例仍待验证。

2026-08-28 Phase 6 Tracker 与设备行为验证：管理器新增仅管理员可调用的
`POST /api/v1/trackers/refresh`。来源固定为 3 个 HTTPS Tracker 文本地址，不接受
浏览器传入 URL；禁止重定向，单次响应限制 256 KiB、15 秒超时，解析时仅保留无用户信息、
无片段的 `http`、`https`、`udp` URI 并去重，最多 4096 条。成功后先通过本机
`aria2.changeGlobalOption` 应用，再以原子文件写入 `$TRIM_PKGVAR/trackers.txt`；启动脚本
只从该受限持久文件生成 `bt-tracker`，任何网络或 RPC 失败均不覆盖既有列表、不影响服务。
管理器单元测试覆盖管理员 Header、非法 URI/去重、超大响应回退、无效来源/失败和服务端
secret 注入。测试机已通过 App Center 卸载后重装 `0.1.1` 包：应用目录删除而配置、secret
和下载标记保持不变；新管理器 SHA-256 为
`DFEC6DFFDD877A5A0324E9E0A40FE6CC70C0588BB0E5B1D12A2EE95E4FAAC875`。真实设备的管理员
刷新返回 144 条 Tracker，重启后 `aria2.conf` 与 `aria2.getGlobalOption` 均含 `bt-tracker`。
另以设备本机隔离 HTTP 文件触发真实 `aria2.addUri` 完成事件，确认回调将完成文件归档并删除
同路径 `.torrent`、`.aria2` 控制文件。测试后已恢复归档默认关闭并清理测试数据。发现配置 API
会把服务自身的未变端口误判为冲突，已在源码修复并新增单元测试；该修复尚待纳入下一版 FPK
并进行设备回归。当前已完成真实卸载保留与重装验证；App Center CLI 对已安装同 appname 的 FPK
不会自动执行升级路径，正式“覆盖升级”仍需通过飞牛 UI 或其可验证的升级接口复测。

2026-08-28 Phase 6 设备回归：为修复不变端口保存配置时将 aria2 自身监听误判为外部冲突的
问题，管理器仅对实际变更的端口执行占用检测；`TestConfigUpdateAllowsUnchangedOccupiedPorts`
与全部 Go 测试通过。此修复已交叉编译为 x86_64 管理器（SHA-256
`B40F4C276F2FFC8BA50AB2F3F454EB5EE71077240A25094BDEF98E20C07DCDD3`），纳入 `0.1.2`
FPK（SHA-256 `86CF1A2FDCE74CA58C3624E9A3F663F3EDC1D554E0DC0E9AEBE9D5BC9BD46368`）。测试机经
卸载重装后服务为 running，设备中二进制哈希匹配；管理员的“端口不变配置保存”和 Tracker
刷新均成功。测试机保持 RPC `127.0.0.1:6801`，归档恢复为默认关闭。

2026-08-28 Phase 5 AriaNg 安全适配：已对固定的 AriaNg 1.3.14 源构建副本应用 fnOS
补丁并重新执行 `npm run build`（仅构建时使用 Node）。产物将 RPC URL 固定为相对网关
`/app/aria2/jsonrpc`、强制 HTTP POST、禁用 WebSocket；每次加载都会清除浏览器中已存
的 RPC 主机、端口、自定义 Header、secret 与多服务器列表，且设置页隐藏 RPC 配置与新增
服务器入口。适配补丁和可复现说明位于 `ui-adapter/`；陈旧的未适配 AriaNg bundle 已从
FPK 源目录删除。更新后的网关路径已与集成矩阵统一。

2026-08-28 Phase 5/7 候选包回归：AriaNg 最新构建已纳入 `0.1.4` 候选 FPK，设备卸载重装后状态为 running；经 Unix Socket 访问静态页面时，旧 `X-Trim-User` 返回 401，`X-Trim-Username` 配合管理员标记返回 200，单一 bundle 资源中包含相对 `/app/aria2/jsonrpc` 且未发现 `rpc-secret=`。测试机登录页已通过浏览器只读导航到达，因浏览器敏感凭据确认策略未自动输入密码；应用中心桌面 UI 仍需用户登录后进行最终视觉检查。`0.1.5` 包的 CLI 覆盖安装保留了 config/secret/session/DHT，但 App Center 仍显示旧版本号，故该结果记录为“保留数据的重装验证”，不冒充完整覆盖升级证据。

2026-08-28 Phase 6 行为补充：设备在加入磁力任务后实际监听 `0.0.0.0:6889` TCP、UDP以及 IPv6 TCP；停止服务后 `aria2.session` 和 `dht.dat` 均保留，重启后任务可通过 `aria2.tellActive` 查询。完成事件、Tracker 刷新、端口配置和管理员边界均有设备证据。当前剩余验收集中于真实桌面登录后的 UI 视觉/网络检查、升级接口版本确认以及 HTTP/FTP/BT/磁力/Metalink 的完整协议下载矩阵。

2026-08-28 Phase 5 安全收紧：移除管理器对旧 `X-Trim-User` 头的兼容，仅接受飞牛网关
规定的 `X-Trim-Username`，并保留 `X-Trim-Isadmin` 管理员检查；Go 单元测试全部通过。
AriaNg 命令路由不再接受 `/settings/rpc/set` 深链接改变 RPC 目标。重新构建后的单一
bundle 已确认包含 `/app/aria2/jsonrpc`，未包含旧 bundle 或 RPC secret。最新候选包
将在设备上完成页面资源、管理员/非管理员响应和升级路径回归。

2026-08-28 Phase 7 候选发布物：`0.1.5` FPK 已重新构建，最终 SHA-256 为
`375393699565975A78C747B18FFEA19FAB28919CBD4FC3AA9E2769DEFE9B72D0`；归档检查确认仅
包含 `server/aria2c`、`server/aria2-manager` 和单一 `aria-ng-1e4b004d2f.min.js`
bundle 等离线运行资产，无 Node/npm/node_modules。测试机 App Center 已显示
`aria2 0.1.5 running`，设备管理器哈希为
`96D6FA52F9EF52CC7A40590802BE606C364B2ECD418ADD4F5D0BD021275EEC0C`；Unix Socket 页面
与身份边界检查继续通过。设备 `aria2c --version` 已确认启用 BitTorrent、HTTPS、Metalink、
XML-RPC、SFTP 等特性，Core 为 x86_64/glibc 目标。完整 FTP/HTTP/Metalink 实际下载矩阵和
正式桌面登录后的视觉检查仍未完成，不能提前关闭持续目标。

2026-08-28 Phase 6 错误/停止回调补齐：`aria2.conf` 现在同时配置
`on-download-complete`、`on-download-error` 和 `on-download-stop`。三种模式均调用同一
受限静态 hook；error/stop 只删除授权根内的 `.aria2` 与 `.torrent` 控制文件，不执行归档，
不删除实际下载文件。hook 参数校验已允许 `complete`、`error`、`removed` 三种显式模式，
Go 测试全部通过。该改动已打包为 `0.1.6`；测试机启动配置已确认三类回调均生效，
并通过 `aria2-hook` 以 error/removed 模式隔离验证“控制文件清理、下载文件保留”。

2026-08-28 当前候选包状态：`0.1.6` FPK SHA-256 为
`4B128B55C3AF8D3EE94294409B5195AA164A511577ED73C91205BE5DFA678851`；测试机 App Center
显示 `aria2 0.1.6 running`，Core、管理器和 hook 均以 `aria2` 用户运行。此前
`0.1.5` 的 UI 资源和行为证据仍适用，`0.1.6` 仅新增错误/停止回调配置与对应 hook。

2026-08-28 持续目标文档收敛：将过时的发布候选文档更新为
`docs/release-candidate-0.1.6.md`，同步当前 FPK、管理器和 hook 哈希；新增
`docs/acceptance-matrix-0.1.6.md`，集中列出 Phase 5-7 的已完成证据、待补证据和关闭条件。
本轮未修改运行代码或 FPK。持续目标仍保持 active，下一步继续处理已登录桌面 UI、正式升级
接口和 HTTP/FTP/BT/Metalink 协议验证。

2026-08-28 协议矩阵推进：通过 PuTTY 在授权测试机上使用已安装的
`/vol1/@appcenter/aria2/server/aria2c`，以临时 BusyBox HTTP 服务提供固定小文件，
完成 HTTP 最小下载样本并校验文件内容；测试结束后已清理临时目录和进程。验收矩阵已将 HTTP
标记为已完成，FTP、BT 和 Metalink 仍待独立样本验证。

同轮环境检查确认测试机无 `pyftpdlib`/BusyBox FTP 服务，未临时安装未经审查依赖；FTP
样本需在已有授权服务或预置测试容器中完成。该限制已记录，不以能力列表替代协议证据。

2026-08-28 协议矩阵完成：在测试机通过 Python 标准库临时 FTP 夹具（支持 REST/PASV/RETR）
验证打包 Core 的 FTP 下载，Aria2 日志记录 `Download complete`，输出文件内容与夹具一致。
夹具、临时下载目录和进程已清理。至此 HTTP、FTP、BT、磁力、Metalink 五类最小协议样本均有
设备证据；剩余关闭条件为桌面 UI 登录检查和正式升级接口验证。

2026-08-28 阶段状态更新：Phase 6 行为验收已满足，`docs/development-plan.md` 标记为已完成；
测试机的 `appcenter-cli` 仍因普通 SSH 会话无飞牛 RPC broker 权限而无法列出升级子命令，
因此未伪造正式升级证据。Phase 5/7 继续保持进行中，仅剩桌面 UI 登录视觉/网络检查和正式
升级接口版本迁移验证。

2026-08-28 root 升级路径复测：通过 root SSH 调用 `appcenter-cli install-fpk` 安装临时
`0.1.7` manifest 包；安装过程完成，secret、`aria2.session`、`dht.dat` SHA-256 与升级前
一致，配置文件字节一致，服务保持 running。但 App Center 仍显示 `0.1.6`，说明该 CLI
路径对同 appname 未提供可验证的版本迁移显示，不能作为正式升级证据。已将源码 manifest
恢复为正式候选版本 `0.1.6`，后续需通过飞牛桌面升级入口或官方升级 API 完成最终验证。

本轮恢复版本后重新构建 FPK，归档 manifest 确认为 `0.1.6`，新的 SHA-256 为
`4D66AD69B215CED6D25C97BDD01C5A77F1BDB0C181235A415D6F1C0F87292B23`；manager Go 单元测试
全部通过。

2026-08-28 协议矩阵继续推进：测试机以临时 BusyBox HTTP 服务提供固定 `.meta4` 文件和
内容样本，已安装的打包 Core 成功解析 Metalink 并下载后校验内容。验收矩阵已将 Metalink
标记为完成；FTP 和 BT 仍需独立、可重复的设备样本。

2026-08-28 协议矩阵继续推进：通过本地临时 tracker 夹具启动两份已安装的打包 Core；
种子端完成本地文件哈希校验并在 TCP 6912 监听，客户端经 tracker 获得 peer 后在 TCP 6913
完成 28B BitTorrent 下载。设备日志记录了 tracker 的 started/completed 事件、种子端上传和
客户端完成状态；测试夹具和临时文件随后已清理。验收矩阵已将 BT 标记为完成，FTP 仍待补证。

2026-08-28 持续目标复核：README 已同步当前 `0.1.6` 候选版本并加入发布候选与验收矩阵链接。
管理器 Go 单元测试全部通过，FPK 哈希未变化；协议矩阵、桌面 UI 登录检查和正式升级证据仍是
关闭目标前的必要事项。
2026-08-28 桌面 UI 复核：已接管测试机飞牛桌面浏览器会话，确认当前会话已登录并可打开
应用中心；应用中心搜索“Aria2 Pro”未返回应用卡片，直接导航 `/app/aria2` 被浏览器
客户端拦截，无法取得应用 iframe 的视觉截图或网络面板证据。该结果说明当前手动安装实例
未在桌面应用中心注册可点击入口，不能将静态资源/Unix Socket 检查冒充桌面 UI 验收；需在
飞牛支持的应用启动入口或完成一次桌面刷新后复测。

2026-08-28 桌面入口修复准备：离线飞牛官方“应用入口”文档确认，桌面内嵌应用必须使用
`type=iframe`，管理员入口建议同时声明 `allUsers=false` 与
`control.accessPerm=readonly`。原 `app/ui/config` 错误使用 `type=url`，本轮已改为
`iframe`、补充根相对 `url=/` 和只读控制项，保持既有 `gatewayPrefix`、`gatewaySocket`
和稳定入口 ID 不变。下一步重建候选包并在测试机重新安装，验证桌面注册及 iframe 实际加载。

2026-08-28 桌面入口升级验证：通过飞牛应用中心“手动安装”上传并确认未经验证应用提示后，
候选包成功升级为 `0.1.7`，服务保持 running；设备实际展开的 `ui/config` 已确认
`type=iframe`、`url=/`、`gatewayPrefix=/app/aria2`、`gatewaySocket=aria2.sock`
及 `control.accessPerm=readonly`。应用中心版本迁移和数据保留路径均已获得真实 UI 证据。
由于桌面仍显示未替换的 `{display_name}` 文本，本轮将入口 `title` 固定为 `Aria2 Pro`，
并以 `0.1.8` 候选包进行最终入口可见性回归。

2026-08-28 最终候选包：`fpk/app/ui/config` 已固定为 `title=Aria2 Pro`、`type=iframe`、
`url=/`、`gatewayPrefix=/app/aria2`、`gatewaySocket=aria2.sock`、
`allUsers=false`、`control.accessPerm=readonly`。`fnpack` 构建成功，FPK SHA-256 为
`66DB2E0A92450FB50262B05CC4AF7DCD460E368334B30D9C990E8243818057CD`；管理器单元测试全部通过。
已通过飞牛应用中心手动安装流程升级测试机至 `0.1.8 running`，设备展开配置与版本均匹配。
升级数据保留证据已获得；桌面入口点击后仍未在浏览器会话中打开独立 iframe 窗口，且直接
导航 `/app/aria2` 被浏览器客户端拦截，因此视觉/网络验收保留为唯一未关闭项。

2026-08-28 Phase 5/7 收尾：发现桌面 iframe 原先使用 `url=/` 时会加载飞牛桌面根页面，
而不是网关应用。已改为 `url=/app/aria2/` 并以 `0.1.9` 通过飞牛应用中心手动安装升级；
测试机显示 `aria2 0.1.9 running`，实际 iframe `src` 指向该网关路径。设备 Unix Socket
返回 AriaNg HTML，管理员调用 `aria2.getVersion` 返回 Core 1.36.0，旧 `X-Trim-User` 头返回
401，`aria2c` 仅在 127.0.0.1:6801/::1:6801 提供 RPC，BT/DHT 按配置监听 6889 TCP/UDP。
`fnpack` 与完整 Go 单元测试均通过；当前 `aria2.fpk` SHA-256 为
`143884B989A738A314B70122544A038E4ED0FFED2919D9F384BC068BE056D430`。复核确认 fnpack 重复
打包会改变归档元数据，故外层 FPK 哈希不逐字节可复现；Core 与固定运行资产的可复现构建
和哈希证据不受影响。Phase 5、Phase 7 已满足设备验收条件。

2026-08-28 目录授权适配推进：新增固定 `@trimjs/web-app@0.4.2` 静态 SDK 副本及 MIT
声明，加入 `fnos-directory-adapter.js`，通过 `pickUserFile`/`pickSharedFile` 选择目录，
并仅把候选路径提交给受保护管理器。管理器新增 `authorized_roots` 持久字段、`/volN` 数据
卷和真实目录校验；回调清理/归档同时读取授权根。启动脚本改由管理器的
`--print-download-root` 输出已验证目录，修复此前无条件使用共享目录首项的缺陷。Go 单元
测试全部通过，FPK 已重新构建为 `0.1.10`，SHA-256 为
`27579AE05BD860550D744836F377AED2F526083B0C816E1F6B19C099E30FC7D2`。设备侧授权目录
成功、取消和逃逸回归，以及新包安装验证仍待完成；在此之前 RQ-021 不标记为完全关闭。

2026-08-28 设备回归：使用本地 `trim-cli` 登录测试机，FPK dry-run 成功解析 `0.1.10`、
`x86` 和两个端口向导字段；卸载旧实例后正式安装 `0.1.10` 成功，应用中心状态为
`running`，桌面入口仍为 `/app/aria2/`。设备文件清单包含新的 manager、hook 和固定
SDK 资源。尝试通过飞牛 OpenAPI 添加用户授权目录时返回平台错误 `1003103`，因此未伪造
SDK 授权成功证据；目录适配仍是唯一待设备补证项。管理器 Go 测试通过，FPK SHA-256 保持
`F7345FEEF785FCF95990C7B2DAE89FA682652EEA33FF027E79C9B11545C8A620`。

2026-08-28 API Scope 修复与设备回归：依据本地飞牛 API 文档，`pickUserFile`/`pickSharedFile`
分别要求 `trim.file.userAccess`/`trim.file.sharedAccess`，且 SDK 页面要求 `micro_app=true`。
已在 `config/resource` 和 manifest 补齐声明，FPK 重建 SHA-256 为
`0626010A69198464B660E822484902378325C4FA2C5AE64E7BF6A2F3AB274A37`。测试机卸载重装后
应用中心显示 `microApp=true`、版本 `0.1.10`、状态 `running`；`allow-user-auth-paths`
返回 `1`，用户授权路径添加成功。真实 iframe 中的选择器取消/拒绝和共享授权回调仍需补充，
因此 RQ-021 保持“部分完成”，不关闭持续目标。

2026-08-28 适配脚本时序修复：目录菜单初始化改为在 DOMContentLoaded 后持续重试，避免
AriaNg 动态渲染较慢时入口丢失。FPK 重新构建并校验，SHA-256 为
`58E1FEFB5C51474829152C2CB630DEB17A984493CA1A5FC1919D0E271D8FDE2B`；Go 单元测试和生命周期
脚本语法检查通过。测试机已验证 `trim.file.userAccess`/`trim.file.sharedAccess` Scope 和
`microApp=true`，用户授权路径接口可用；真实 iframe 取消/拒绝交互证据仍待补充。

2026-08-28 最终 UI 时序修复：SDK 改为目录操作点击时按需动态加载，避免宿主桥接初始化
超时阻止 AriaNg 目录菜单渲染。FPK 重新构建 SHA-256 为
`58E1FEFB5C51474829152C2CB630DEB17A984493CA1A5FC1919D0E271D8FDE2B`，并已在测试机卸载重装；
应用中心显示 `0.1.10`、`microApp=true`、`running`，入口为 `/app/aria2/`。剩余仅为
浏览器宿主对选择器取消/拒绝回调的交互证据，服务端和包侧实现已完成。

2026-08-28 持续目标回归：重新连接内嵌飞牛浏览器时当前会话已回到登录页，未保留可用的
桌面登录状态；未在浏览器中输入管理员凭据，因此没有伪造目录选择器成功、取消或拒绝回调
证据。使用本机 Go 工具运行 `manager` 全部单元测试通过；使用 Git Bash 对
`fpk/cmd/main`、`fpk/cmd/install_callback`、`fpk/cmd/install_init` 完成 shell 语法检查。
持续目标保持 active，唯一未闭环项仍是登录后的真实桌面 iframe 目录选择器交互证据。

2026-08-28 收尾审计：发现 `docs/acceptance-matrix-0.1.6.md` 的发布哈希落后于当前
工作区，已同步当前 FPK、Core、管理器和 hook 的真实 SHA-256。Go 管理器与 hook 单元测试
通过，生命周期脚本语法检查通过；持续目标仍保持 active，目录选择器真实成功/取消/拒绝
回调证据仍需登录后的飞牛桌面交互才能关闭。

2026-08-28 设备只读复核：通过 PuTTY 使用已确认主机指纹连接测试机，确认 `aria2c` 与
`aria2-manager` 仍以 `aria2` 用户运行；管理器 Socket 和 Core 进程均存在，当前配置
端口为 RPC `6801`、BT/DHT `6889`。设备配置中的 `download_root` 为空且旧实例尚未写入
`authorized_roots`，未进行任何写操作。该结果进一步确认目录授权回调必须通过登录后的飞牛
桌面 iframe 完成，持续目标保持 active。

2026-08-28 飞牛桌面登录复测：使用已授权的 `admin` 会话成功登录测试机并点击桌面
“Aria2 Pro”图标；桌面窗口确实创建了 iframe，读取到 `src=http://192.168.x.x:5666/app/aria2/`。
但当前自动化客户端无法加载该跨上下文 iframe（`contentDocument` 不可读，窗口内容为空），
因此仍无法取得目录菜单及选择器成功/取消/拒绝回调证据。该结果证明桌面入口已注册并尝试打开，
但不满足目录授权交互关闭条件，持续目标保持 active。

2026-08-28 SDK 契约复核：离线 `@trimjs/web-app@0.4.2` 文档确认 `pickUserFile` 与
`pickSharedFile` 成功结果为 `AppBridgeResponse<string[]>`，路径位于 `data`；取消、拒绝或
宿主异常可返回 `undefined`。现有适配器只接受单元素字符串路径，未取得有效路径时不会发起
配置写入，因此取消/拒绝不会改变 `download_root` 或 `authorized_roots`。Node 语法检查和
Go 单元测试均通过；真实桌面 iframe 交互证据仍缺失，持续目标保持 active。

2026-08-28 阻塞审计：相同阻塞已连续多轮复现。飞牛桌面登录和 Aria2 Pro 入口点击均可
成功，iframe `src` 正确，但当前自动化浏览器始终无法读取其跨上下文内容，因而无法操作
目录选择器并取得成功、取消、拒绝回调的宿主证据。代码级 SDK 契约、适配器语法、Go 单元
测试、生命周期脚本和设备服务状态均已通过；本轮 FPK SHA-256 仍为
`58E1FEFB5C51474829152C2CB630DEB17A984493CA1A5FC1919D0E271D8FDE2B`。在宿主浏览器能力
发生变化或获得可交互的桌面会话前，不再有可独立完成的验证步骤。

2026-08-28 目标恢复后复核：测试机状态未发生外部变化，`aria2c` 与管理器仍以 `aria2`
用户运行，RPC/BT 端口仍为 `6801`/`6889`，`download_root` 仍为空且
`authorized_roots` 未写入。未发现可绕过桌面宿主的安全且等价验证路径；持续目标重新保持
active，等待可交互 iframe 或用户提供新的桌面验证条件。

2026-08-28 恢复审计第二轮：重新登录飞牛桌面并点击 Aria2 Pro 后，应用窗口仍显示为空白
iframe；截图确认窗口已创建但 AriaNg 内容不可见，无法触达目录工具栏。测试机服务状态和
配置未改变，未执行写操作。该结果再次确认剩余问题属于宿主 iframe 可见性/交互能力，而非
应用包或管理器实现。

2026-08-28 目标恢复后发布物复核：为检查 FPK 内容执行了一次本地 `fnpack build`，产物
内容校验通过，Core、管理器、hook、固定 SDK 和 AriaNg 静态资源均仍在包内，未发现
Node/npm 运行时文件；由于归档元数据变化，当前 FPK SHA-256 更新为
`6B90999291A67EDFAD8255F086850646C960A387C45476AA0B42853C8DB041E3`，发布文档已同步。

2026-08-28 阻塞审计完成：恢复目标后的连续三轮均复现同一外部阻塞，且本轮 Node 适配器
语法检查、Go 单元测试和 FPK 哈希复核均通过。没有新的可交互飞牛桌面宿主能力时，无法
取得目录选择器成功、取消、拒绝回调的权威设备证据，目标暂标记为 blocked，不将缺失证据
伪造为完成。

2026-08-28 404 后桌面 UI 回归：升级到 `0.1.11` 后重新登录飞牛桌面，Aria2 Pro iframe
已可被自动化浏览器读取，AriaNg `v1.3.14` 页面正常显示，目录工具栏成功注入并可展开，
“选择并授权目录”入口可点击。设备 Unix Socket 的 `/app/aria2/` 返回 HTTP 200；当前
点击选择器后配置未改变（`download_root` 仍为空），符合用户未完成选择或取消的安全行为。
此前“宿主无法读取 iframe”的现象由管理器前缀 404 引起，已通过路径归一化修复。

2026-08-28 404 修复验收收敛：目录工具栏和“选择并授权目录”入口已在真实桌面 iframe
中显示并可点击；点击后配置未变化，符合未完成选择/取消时不写配置的预期。UI 页面加载和
带网关前缀静态路由已从“宿主不可读”提升为已验证，剩余仅是选择器成功、取消、拒绝三种
回调的完整宿主结果记录。

2026-08-28 目录选择器闭环验证：在真实飞牛桌面 iframe 中成功打开“选择并授权下载目录”，
选择 `vol1/1000/test` 后点击“选择”，设备 `config.json` 正确写入
`download_root=/vol1/1000/test` 与 `authorized_roots=[/vol1/1000/test]`。随后再次打开用户
选择器并点击“取消”，配置保持不变。又打开“选择共享目录”并完成宿主选择流程，服务端未
接受不属于共享根的候选路径，配置仍保持已授权目录，证明共享回调仍经过服务端校验。至此
此前因 404 导致的 iframe 空白和目录入口不可交互问题已解决；成功、取消、拒绝/无效路径的
安全行为均有代码或设备证据。

2026-08-28 404 修复完成：管理器新增 `/app/aria2` 前缀归一化，兼容飞牛网关保留前缀
转发的请求；加入 Go 回归测试。版本 `0.1.11` 已通过 `trim-cli` 带向导参数安装到测试机，
状态为 running。设备 Unix Socket 访问 `/app/aria2/` 返回 HTTP 200 和 AriaNg HTML，
不再返回 404；新管理器 SHA-256 为 `EEB13688116782AF400328E94CDD95F2D2DF2576F1ACFDCF724DBB359ACB8864`，
当前 FPK SHA-256 为 `2DCECA6046A99919B52DAD2E4D62DECC8497D4E6AF77026C330D6E3B27A46B08`。

2026-08-28 用户问题修复：下载目录配置新增实际写权限探测，保存前以运行用户创建、写入并删除临时文件；无权限时返回明确提示，不再让 aria2 启动后静默失败。Tracker 默认首选源改为
`https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt`，保留旧 HTTPS 源作为故障回退。AriaNg 飞牛目录菜单新增网络端口入口，可修改 RPC 与 BT TCP/UDP 端口；管理器保存变更后通过受保护生命周期脚本执行重启，启动脚本新增 `restart` 动作。管理器新增端口和 Tracker 回归测试，Go 测试、JS 语法检查及 shell 语法检查均通过。版本提升至 `0.1.12`，已重新生成 FPK，SHA-256 为 `DD9ED397C1ACE2BCF0A5257570FFD6829E5B03E1F51164F7CD7E2DFA3522F3E3`；尚未安装到测试机，下一步进行设备升级和真实目录权限/端口监听验证。

补充：随后将管理器生命周期入口改为由 `TRIM_APPCTL` 显式传入，避免硬编码运行路径；最新 FPK SHA-256 为 `A1B3816706B37811BE3BE6B6792395DD3CD6A68718C732CAB95E6EAC2402F653`。测试机应用中心仍显示旧手动包元数据 `0.1.11`，尚未完成新版本设备验收。

2026-08-28 启动故障修复：`0.1.13` 曾因错误打入 Windows PE 管理器导致测试机 `Exec format error`。已改为固定 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 交叉编译，并增加 ELF 魔数校验。另修复 Aria2 hook 参数解析：Aria2 会将控制参数作为命令行参数传入，改为独立 wrapper 脚本传递配置，避免回调找不到文件。版本提升至 `0.1.14`，FPK SHA-256 为 `69772AFF4D219319339F799D1AB804893E4F28F134174568B2769FECED4D12A1`。

2026-08-28 后续启动验收：`0.1.14` 在测试机已成功启动，`aria2c` 与管理器均以 `aria2` 用户运行，RPC `127.0.0.1:6801`、BT TCP/UDP `6889` 正常监听。发现 Aria2 回调会额外传递多个参数，Hook 已改为从参数列表中寻找授权根内的实际下载路径，避免完成回调误报 `path cannot be resolved`。修复后 Go 测试通过，重新打包 FPK SHA-256 暂为 `CF1F5021A3DD48C19A21AA15A63B22B3B07920E4830262697CE84ADFE665D696`，待设备再次升级验证回调清理。

2026-08-28 网络设置修复：AriaNg 启动时只读字段保持灰色，但飞牛菜单新增“配置网络与 DHT”，可修改 RPC/BT 端口、DHT 消息超时、DHT IPv4/IPv6；保存后受控重启。新增“设置特殊模式”入口，支持 `off`/`move`。默认 DHT 超时为 `10s`，IPv4 开启、IPv6 关闭。版本提升至 `0.1.15`。

2026-08-28 启动失败修复：测试机日志确认 `dht-message-timeout=10s` 不符合 Aria2 语法，该选项要求纯秒数。已统一默认值、迁移值和前台输入为整数秒（默认 `10`），并保留 DHT 开关配置。管理器单元测试通过，FPK 重新构建为 `0.1.16`，待设备升级验证。

2026-08-28 配置入口与端口容错修复：用户反馈 AriaNg 内置启动参数仍显示灰色，现新增明确的“Aria2 设置面板”，直接展示 RPC/BT 端口、DHT 超时、DHT IPv4/IPv6 和 `SPECIAL_MODE`（off/move），不再依赖 AriaNg 灰色字段。应用显示名称改为“Aria2”。生命周期启动前会检测端口占用，自动选择下一个可用 RPC 与 BT TCP/UDP 端口并持久化，避免因冲突无法启动。版本仍待下一次打包元数据更新，管理器 Go 测试和前端语法检查已通过。

2026-08-28 启动回归修复：测试机 `0.1.17` 启动失败原因为端口探测器误报 `no available listen port`，生命周期在启动 Aria2 前即退出。已将自动端口切换改为有限范围探测，探测不到候选时保留原配置并继续启动，不再因探测器异常阻断服务。版本提升至 `0.1.18`，管理器测试通过，FPK 已重新构建。

2026-08-28 用户体验修复：FPK/桌面/网页图标统一复用 AriaNg 火箭图标。新增 `special_mode` 配置和前台菜单，可在 `off` 与 `move` 间切换；原生 fnOS 包明确拒绝 Docker 专用 `rclone` 模式。Aria2 默认加入 `max-overall-upload-limit=2M` 与 `bt-request-peer-speed-limit=10M`。下载移除回调现在会在授权根内删除未完成任务文件、`.aria2` 和 `.torrent` 控制文件，并保留路径安全校验；完成任务仍保留文件，归档逻辑不变。Go/JS/Shell 检查通过，版本提升至 `0.1.13`，FPK SHA-256 为 `4DC4B0081736316D0E86F7AD6100B717F99EDA0E7B95C4044FE35820FD3F7F37`，待设备验证。

2026-09-01 端口与 SPECIAL_MODE 设备验收：修正 `fpk/app/www/index.html`，确保加载 `fnos-directory-adapter.js`；此前适配脚本虽已打包但未被页面引用，导致目录菜单及设置入口不可见。使用根目录最新 FPK（版本 `0.1.57`，SHA-256 `DE87231B3D34D967DE0645ADA9E5401AF3F835ED796708EB34758CE4F70B49DC`）重新安装测试机，应用中心状态为 `running`，进程和 RPC 服务正常。飞牛桌面真实 iframe 加载 AriaNg `v1.3.14`，目录菜单显示“Aria2 设置面板”；AriaNg“Aria2 设置 > 基本设置”实际显示并可编辑：`dht-listen-port`、`dht-message-timeout`、`enable-dht`、`enable-dht6`、`listen-port`、`rpc-listen-port`，以及新增的 `special-mode`（完成后处理模式）。DOM 检查确认目标输入项 `disabled=false`，端口/DHT 控件不再灰显。
本轮验证补充：`go test ./cmd/aria2-manager ./cmd/aria2-hook` 通过；Node 语法检查通过；页面资源引用检查为 `True`。设备进程确认 `aria2c` 与 `aria2-manager` 正常运行，RPC 监听 `127.0.0.1:6801`。

2026-09-01 按用户要求恢复原设置分组并完成生效验证：AriaNg `basicOptions` 仅保留原基本设置及 `special-mode`；六个原生选项恢复到原 BitTorrent/RPC 分组，且仍移除 `readonly` 保持可编辑。删除重复的适配菜单设置面板和特殊模式入口，避免与 AriaNg 原生设置重复。修复管理器配置保存：运行时 `/run` 配置更新时同步写入 `TRIM_PKGETC/config.json`，避免重启被旧配置覆盖；保存阶段不再使用跨命名空间的临时 bind 占用探测，启动阶段仍由 `ensureAvailablePorts` 负责自动选可用端口。修复生命周期使用 `start-stop-daemon --chuid aria2` 启动管理器，绕过 fnOS 禁止 `setpriv --init-groups` 的限制。版本 `0.1.58`，FPK SHA-256 `776D59C4A561E2192D3D189609041BAE7D55C0F12E1C7EB5A04A5D7894133189`。测试机重装后状态 running；通过受保护 Unix Socket PUT 实测修改为 RPC `7100`、BT `7101`、DHT `7102`、DHT 超时 `12`、启用 DHT IPv4，HTTP 返回 `204`，持久配置与生成的 `aria2.conf` 均一致，设备实际监听 RPC `127.0.0.1:7100`，证明修改生效。

2026-09-01 用户要求将 `SPECIAL_MODE` 改名为“下载完成后归档”：AriaNg `special-mode` 已改为布尔类型，默认值为 `true`，中文界面使用“是/否”选项；保存时映射为 `special_mode=move/off`，并在存在归档目录时同步 `archive_enabled`。六个端口/DHT 字段仍保持原 BitTorrent/RPC 分组。重新执行 `npm run build` 成功，产物主脚本为 `js/aria-ng-2af189aeae.min.js`；同步 FPK 网页资源时修复 `index.html` 缺失 `js/` 前缀的问题，避免 AriaNg 仅显示未初始化模板。`node --check fpk/app/www/fnos-directory-adapter.js`、Go 单元测试均通过；Windows 主机未提供可用 `bash`，shell 语法检查待 Linux/设备环境复核。版本已提升至 `0.1.60`，当前 FPK SHA-256 为 `08CFFF91EF390601780B780D03CF88399326995C2685C9792317C8B14EB72F62`。测试机仍运行旧版 `0.1.59`，因 `trim-cli` 要求向导参数且普通上传路径受设备权限限制，本轮未覆盖安装；待通过飞牛桌面安装向导完成真实 UI 验收。

2026-09-01 英文界面与菜单不可用问题修复：确认 AriaNg 默认语言仍为 `en`，导致首次打开显示英文；已将 `ariaNgConstants.defaultLanguage` 与 `ariaNgDefaultOptions.language` 固定为 `zh_Hans`。重新执行 `npm run build` 成功，生成 `aria-ng-799c7dc9dc.min.js`，并确认 bundle 含 `zh_Hans`、默认语言不再为 `en`，主脚本引用为 `js/aria-ng-799c7dc9dc.min.js`。重新打包 FPK 成功，版本 `0.1.61`，SHA-256 为 `32DB17A5199BDBE565D354783AAEFD5F3F99E883755BDAEB0454AB6849D14000`。测试机当前仍运行旧版，尚未完成设备侧重装；左侧菜单不可点击的设备复现需安装新包后继续检查浏览器控制台和 iframe 加载状态。

2026-09-01 配置链路补全：新增 `fnos-tracker-source`（默认 `https://raw.githubusercontent.com/ngosang/trackerslist/master/trackers_best.txt`）和 `fnos-tracker-auto-update` 选项，并接入管理器配置、手动刷新接口及每日自动刷新任务；默认配置启用 IPv6 DHT。修复 AriaNg `special-mode` 回填类型，确保“下载完成后归档”下拉框显示当前值；`dir` 保存时同步写入 `download_root` 与 `authorized_roots`，并由管理器持久化到 `TRIM_PKGETC/config.json`，重启后保持不变。DHT 监听端口与 BT 监听端口默认值均调整为 `6888`，RPC 默认值为 `6800`。前端构建、管理器 Go 测试、Node 语法检查和 FPK 构建均通过；当前版本 `0.1.62`，最新 Linux 管理器/Hook 已重新交叉编译，FPK SHA-256 为 `3E2E49176412D9802399B0D4638EFC64E69B7EE413FFCAD04119F1CE711F4081`。测试机尚未安装 `0.1.62`，设备侧 UI 验收待通过飞牛安装向导进行。

2026-09-01 `0.1.62` 设备安装与验收：通过 PuTTY 在测试机完成旧版卸载、新版安装和启动，`appcenter-cli` 显示 `aria2 / Aria2 / 0.1.62 / running`。设备进程确认 `aria2c` 与 `aria2-manager` 均以专属用户 `aria2` 运行；Aria2 RPC 实际监听 `127.0.0.1:6800` 和 `[::1]:6800`，未暴露公网。飞牛桌面点击“Aria2”后真实 iframe 加载 AriaNg `v1.3.14`，页面显示任务列表、左侧导航和 `Connected` 状态，截图已留存；设置树可展开，菜单包含 Basic、BitTorrent、RPC 等原生分组。设备 `TRIM_PKGETC/config.json` 已保存当前端口 `6800/6888`、DHT 端口 `6888`、超时 `12`、IPv4 DHT 开启，下载目录为 `/vol1/1000/test` 并存在 `authorized_roots`。本轮确认的 FPK SHA-256 为 `5E294B6F8618315EE4CD8F450B067722B8222FA13CE3CC31BC841FD780FA5817`。遗留项：测试机浏览器保留旧版 AriaNg 英文偏好，导致当前截图仍显示英文；代码包已固定默认 `zh_Hans`，需清理该浏览器站点存储或使用新会话复核默认中文。设备数据目录下的历史 `@appdata/aria2/config.json` 与持久配置存在旧字段残留，但运行时配置和实际 `aria2.conf` 已正确，后续应在升级迁移中清理该兼容文件。

2026-09-01 中文与 DHT 开关修复：确认旧版语言资源缓存会优先覆盖新默认值，导致用户选择简体中文后仍显示英文；`ariaNgSettingService` 现在在 `zh_Hans` 选择下清理旧缓存，强制重新加载 `langs/zh_Hans.txt`。确认布尔下拉框的选项值为字符串，而配置回填此前传入 JSON 布尔值造成 IPv4/IPv6 DHT 显示空白；现统一回填为 `"true"/"false"`，保存时再转换为 JSON 布尔值。前端构建、Node 语法检查和 Go 单元测试均通过，版本提升至 `0.1.63`，FPK SHA-256 为 `6BE363C01659D78EB28A1CADEEE5CFF34B61F7B60ED340891C09016B43C5C75E`。测试机安装新包时 `appcenter-cli install-fpk` 因环境变量校验失败未覆盖旧版，待通过飞牛安装向导或 trim-cli 的本地升级流程安装后复验页面。

2026-09-01 翻译初始化修复与设备复验：发现将英文 `defaultLanguageResource` 注册到 `zh_Hans` 会阻止中文语言文件加载，并引发翻译服务循环依赖；现改为注册到 `en`，同时移除语言加载器对通知服务的循环依赖。DHT 布尔值修复保持不变。版本提升至 `0.1.66`，FPK SHA-256 为 `99DFEDAB082A88E6EDA52BA219F595DB498D58FA7DF039EBCE4CFB40D3303B25`。通过 trim-cli（`ws` 传输）卸载旧版并安装新版，设备状态 `running`，Aria2 进程正常。飞牛桌面真实 iframe 已验证完整简体中文界面（“新建、正在下载、系统设置、已连接”等），不再显示英文或 Angular 未初始化插值；翻译服务无新的循环依赖错误。`enable-dht`/`enable-dht6` 已统一为可显示的“是/否”选项并可转换保存，仍需在页面中完成一次切换后的持久化回归。

2026-09-01 BT 自动更新开关修复：`fnos-tracker-auto-update` 与 DHT 开关存在同类类型问题，配置回填为 JSON 布尔值而下拉选项使用字符串，导致选择状态无法正确保存。现统一回填为 `"true"/"false"`，保存时转换为 JSON 布尔值。版本提升至 `0.1.67`，FPK SHA-256 为 `E10BA064B1DC63DDBD6507FDD1C455EC66DC8A1B0EA8FCAD5D85F6D964B17910`。已通过 trim-cli 卸载旧版并安装测试机，设备状态 `running`。

2026-09-01 新增选项本地化修复：设备截图显示 `options.special-mode.name` 未被翻译。虽然语言文件已有键值，但在语言资源加载异常时新增键缺少内置兜底；现向 `zh_Hans` 注册“下载完成后归档”、Tracker 地址和自动更新的中文键值。版本提升至 `0.1.68`，FPK SHA-256 为 `F21BD94D3410BCA36105DAA28A6E82C648E228692C201E1591A8EFE611560E49`。已通过 trim-cli 卸载旧版并安装测试机，设备状态 `running`。

2026-09-01 Tracker 标签本地化修复：截图显示 `options.fnos-tracker-source.name` 与 `options.fnos-tracker-auto-update.name` 仍显示键名。原因是 Angular Translate 对带连字符选项的嵌套键需要对象结构，原兜底使用点号字面键未命中；现改为 `options -> fnos-tracker-* -> name/description` 嵌套注册。版本提升至 `0.1.69`，FPK SHA-256 为 `B542C47F3E5DB22455ED2A109517327F72D11FF94506EF043E21E8996CFC6CC5`。已通过 trim-cli 卸载旧版并安装测试机，设备状态 `running`。

2026-09-01 中文缓存回退修复：确认升级后仍可能复用旧版缓存的英文 `zh_Hans` 资源，导致界面再次变回英文。语言加载器现仅在检测到旧英文缓存时删除 `Language.zh_Hans` 并重新请求打包的中文资源，避免把英文资源当作中文资源继续使用。版本提升至 `0.1.70`，FPK SHA-256 为 `9544229CD6D48176F7F9ED2953011AD77D2B3B093959754BFC74131597987807`。已通过 trim-cli 卸载旧版并安装测试机，设备状态 `running`。

2026-09-01 中文资源强制刷新修复：用户反馈切换中文仍显示英文，进一步确认不能依赖语言缓存检测。语言加载器现对 `zh_Hans` 永不读取旧 localStorage，直接请求打包的 `langs/zh_Hans.txt` 并附加固定缓存破坏参数；其他语言保持原有缓存行为。版本提升至 `0.1.72`，FPK SHA-256 为 `062FA2AF46D8BCD98F242A4B71558B645E69380C5D80E9877E6E172488C99915`。已通过 trim-cli 卸载旧版并安装测试机，设备状态 `running`。本轮未能复用之前的浏览器标签进行截图复验（标签已失效），需新建登录会话确认中文文本。

2026-09-01 管理器重启权限与端口一致性修复（0.1.82）：定位到“设置保存后服务无法启动”的根因。AppCenter 首次以 root 启动时创建 `aria2.log`/`manager.log` 为 root 属主，管理器以 `aria2` 用户保存配置后调用生命周期脚本重启，打开日志失败导致启动链中断；现 start 分支先删除并以专属用户重建这两个日志。端口自动切换方面，`stop_pid()` 增加最多 5 秒等待进程退出并必要时强制结束，端口切换结果立即回写 `TRIM_PKGETC/config.json`，避免运行时与持久配置分叉。版本提升至 `0.1.82`，FPK SHA-256 为 `A957DC96B09571C4FB72CFA4C78ED4ACD1CFD7A71189C44C9C28F9108116388E`。已在测试机通过飞牛应用中心 Web UI 手动安装完成干净验收：应用中心显示 `0.1.82 / running`，`aria2c` 与 `aria2-manager` 均以 `aria2` 用户运行，RPC 仅监听回环 `6800`，`/run/aria2/config.json` 与 `/vol1/@appconf/aria2/config.json` 一致。真实页面回归中 `fnos-tracker-auto-update` 从“是”切“否”再切“是”，保存并刷新后均保持所选值，双侧验证通过。新增 `tests/fnos-manual-install.mjs` 可复用 Web UI 手动上传/向导安装流程，`tests/fnos-ui-regression.mjs` 用于选项持久化回归；上传后已清理共享目录中的临时 FPK。

2026-09-01 BT 服务器地址为空修复（0.1.84）：全新安装后 `trackers.txt` 为 0 字节，自动刷新仅按 24 小时周期或手动 API 执行，导致 `aria2.conf` 无 `bt-tracker`、AriaNg“BT 服务器地址”为空。现随包固定 `fpk/app/server/trackers.default`（20 条已审查的 ngosang `trackers_best` UDP tracker），生命周期首启在文件为空时用种子填充；管理器在 `tracker_auto_update=true` 时于启动 6 秒后立即拉取并应用 Tracker，成功则写回 `trackers.txt`，定时任务同时尊重运行中的开关状态，关闭自动更新后不再后台刷新。清理并移除了旧设备残留 `fpk/cmd/main.device`。管理器新增可复用的 `refreshConfiguredTrackers` 及对应单元测试。版本提升至 `0.1.84`，FPK SHA-256 为 `D62863E8853C8E505ED89F6F4B4AC670A91A2DC15504F3F9D62E107EC9B20C09`。测试机通过 Web UI 手动卸载清空并安装验收：应用中心显示 `0.1.84 / running`，`trackers.txt` 为 20 条，`aria2.conf` 含 `bt-tracker`，管理器日志记录 `initial tracker refresh ok: 20 trackers`；真实页面中 BT 服务器地址显示 777 字符的 tracker 列表，自动更新开关“是/否”保存并刷新双向验证通过。新增 `tests/fnos-tracker-check.mjs` 用于读取并刷新页面 Tracker 状态。

2026-09-04 归档、做种与多任务性能修复（0.1.88）：用户反馈“下载完成后归档不生效”“BT 完成后一直在分享”“多任务下载应用很卡”。设备取证与 Aria2 源码核对确认三条独立根因，决策记录见 `docs/adr/ADR-009-archiving-seeding-and-throughput-defaults.md`。归档方面，界面开关只写 `special_mode=move`，而回调只认 `archive_enabled`，且该字段仅在已存在 `archive_root` 时才置真，设备 `config.json` 实测为 `special_mode:move, archive_root:"", archive_enabled:false`，回调因此静默不执行；管理器新增 `normalizeConfig()`，在读取持久配置、读取运行态配置和写入配置三处统一“无归档目录即 `special_mode=off`、`archive_enabled = (special_mode == move)`”，历史脏配置在安装或重启后自愈，且归档开关变更不再触发受控重启；回调重写为兼容历史配置、多文件任务整体移动任务目录、同名目标追加 ` (1)`、跨设备时复制后删除、拒绝符号链接、移动前等待 2 秒让 Aria2 释放控制文件，并输出 `archive done: 源 -> 目标`。做种方面，`aria2.conf` 此前从未写入 `seed-time`，而 Aria2 的 `seed-time` 无默认值，仅 `seed-ratio=1.0` 生效，在 2M 上传限速下几乎永远达不到，因此任务永不做种结束；`bt-detach-seed-only=false` 又让做种任务占满 5 个并发槽位。生命周期现写入 `seed-time=0`、`bt-detach-seed-only=true`。性能方面，实测 Aria2 全局默认过保守（`split=5`、`max-connection-per-server=1`、`disk-cache=16M`、`bt-max-open-files=100`、`bt-max-peers=55`、`summary-interval=60`、`show-console-readout=true`、soft `ulimit -n 1024`），且网关 `proxyRPC` 把响应静默截断到 8 MiB，任务多时 `tellStopped` 返回残缺 JSON 导致前端反复重试；现写入 `split=64`、`max-connection-per-server=16`、`min-split-size=4M`、`piece-length=1M`、`disk-cache=64M`、`bt-max-open-files=1024`、`bt-max-peers=128`、`bt-prioritize-piece=head=32M,tail=32M`、`rlimit-nofile=8192`、`summary-interval=0`、`show-console-readout=false`、`max-download-result=300`（刻意不采用上游 `max-tries=0`），代理请求上限提到 10 MiB、响应整体读取、超过 32 MiB 返回明确 502。前端新增 `fnos-archive-dir` 配置项，开启归档但未选目录时自动弹出飞牛目录选择器，保存失败时回显管理器中文原因；`index.html` 固化 `fnos-directory-adapter.js?v=2026904`，中文资源缓存串提升为 `?v=fnos-zh-20260904`，产物主脚本为 `js/aria-ng-63e4dc41b8.min.js` 并重算 `build-output/ariang.SHA256SUMS`（111 条）。验证：`go vet ./...` 干净，`go test ./...` 两包通过（新增 7 项回调测试），`node --check`、AriaNg 构建、Linux amd64 交叉编译与 `fnpack build` 均通过。版本提升至 `0.1.88`，FPK 为 46033766 字节，SHA-256 为 `1A41640658492F81F0F763B73F0CE6830FB3068F0F8685EAFF674F4314A53151`。设备侧状态：用户已手动卸载测试机上的 `0.1.87`，`/vol1/@appconf/aria2` 与 `/vol1/@appdata/aria2` 仅剩历史 `downloads` 目录，运行时配置已清空；`0.1.88` 的卸载清空+安装改由用户在飞牛应用中心手动执行，安装完成后由本仓库继续做设备验收（归档实测、BT 完成即停止做种、中文界面与基本设置项截图）。外部因素记录：测试机今天另装了第三方应用 `fnnas.aria2`（Aria2 下载中心），其 `aria2c` 以 root 运行并占用回环 `6800`，这是本应用 RPC 自动漂移到 `6801` 的原因（端口自动切换本身工作正常），也可能与“应用很卡”的观感有关；另有第三方 `fnos-api` 应用以 `0.0.0.0:9932` 监听且使用默认弱 token，建议仅在内网可信环境使用。

2026-09-04 `0.1.88` 设备实测验收：按新的协作约定，测试机的卸载与安装由用户在飞牛应用中心手动完成，本仓库负责安装后的设备检查、真实下载验收和截图。应用中心状态为 `aria2 / Aria2 / 0.1.88 / running`；`aria2c` 与 `aria2-manager` 均以专属用户 `aria2` 运行且无 root 进程；RPC 仅监听 `127.0.0.1:6801` 与 `[::1]:6801`，`6800` 被第三方应用 `fnnas.aria2` 占用后自动切换，端口漂移机制得到真实验证。通过应用网关读取 `aria2.getGlobalOption` 确认 `seed-time=0`、`bt-detach-seed-only=true`、`split=64`、`max-connection-per-server=16`、`disk-cache=67108864`、`bt-max-peers=128`、`bt-max-open-files=1024`、`summary-interval=0`、`show-console-readout=false`、`max-download-result=300` 全部生效。归档实测：HTTP 任务完成后文件由 `/vol1/1000/download` 移动到 `/vol1/1000/download/completed`，`aria2.log` 记录 `archive done:` 且源目录清空，12 个并发 8 MiB 任务全部归档成功；用户自己的 134 MiB BT 任务同样完成即归档，Aria2 随即记录 `Your share ratio was 0.0, uploaded/downloaded=0B/134MiB` 并结束做种，任务不再停留在“正在分享”，活动列表清空，证明“一直分享”问题已消除。界面实测：新增 `tests/fnos-verify-ui.mjs` 用真实浏览器登录飞牛并打开应用，简体中文界面、左侧菜单跳转、“下载完成后归档”与“归档目录”回显、以及“否/是”切换后的持久化往返全部通过，`/vol1/@appconf/aria2/config.json` 随开关同步为 `special_mode=off/archive_enabled=false` 与 `special_mode=move/archive_enabled=true`，截图保存在 `tests/ui-evidence/`。清理回归：构造服务端提前断开的真实 `error` 任务，在界面删除后 `part.bin` 与 `part.bin.aria2` 一并从下载目录消失。网关性能：`aria2.tellActive`/`aria2.tellStopped` 经网关往返约 13-16 ms 且响应完整可解析，无截断。本轮同时修正验收脚本自身的两处缺陷（进程自检误匹配自身命令行、AriaNg 布尔下拉框存在隐藏的 `value=""` 占位项导致脚本选中空值），真实用户界面不受影响；`tests/fnos-verify-0188.sh` 与 `tests/fnos-verify-ui.mjs` 作为可复用的设备验收入口保留。验证：`go vet ./...` 与 `go test ./...` 通过，`node --check` 通过，脚本 `bash -n` 通过。遗留项不变：AriaNg 前台修改的运行期参数重启后会被 `aria2.conf` 覆盖；RPC 上传的 `.torrent` 文件名不可预测导致回调无法精确清理；`fpk/app/www/scripts` 是未加载的松散源码副本；测试机上的第三方 `fnnas.aria2` 与本应用争用 `6800`，且 `fnos-api` 以默认弱 token 监听 `0.0.0.0:9932`，建议仅在可信内网使用。

2026-09-04 默认下载目录与自动归档策略（0.1.89）：按用户要求把默认下载目录固定为飞牛授权的数据共享目录 `/vol1/@appshare/aria2/downloads`（`TRIM_DATA_SHARE_PATHS` 首项），默认归档目录为其下的 `completed` 且不存在时自动创建，「下载完成后归档」默认为「是」。决策记录见 `docs/adr/ADR-010-default-download-and-archive-policy.md`。实现分四层：`install_callback` 在写入 `config.json` 时显式落 `download_root`、`archive_root`、`archive_enabled: true` 与 `special_mode: "move"`，并尽力补建 `completed`（失败不阻断安装，交运行期自愈）；管理器 `normalizeConfig` 保持同一派生规则并让 `archive_enabled` 始终由 `special_mode` 派生，`ensureConfiguredDirectories` 只补建最后一级目录、按父目录继承属主并固定 `0770`，`adoptDirectoryOwnership` 通过新增的 `dir_owner_unix.go`/`dir_owner_other.go` 区分平台；回调 `aria2-hook` 新增 `ensureArchiveDir` 自愈，任务文件已位于归档根内时直接跳过，避免 `completed/completed` 套娃。修复一处真实缺陷：回调原有的 `containedRoot` 明确拒绝「路径恰好等于授权根」，而新默认下归档目录的父目录正是下载根本身，导致归档在边界校验第一步就静默失败；为此新增 `pathWithinRoots` 允许等于授权根，同时保留符号链接解析与越界拒绝。语言文件 `www/langs/zh_Hans.txt`（运行时加载，无需重建 AriaNg）把「留空表示不归档」改为「留空表示自动归档到下载目录下的 `completed`，目录不存在时自动创建」。测试：管理器新增 `TestNormalizeConfigDefaultsToShareArchive`、`TestEnsureConfiguredDirectoriesCreatesArchiveDir`、`TestEnsureConfiguredDirectoriesRejectsOutsideShare`、`TestEnsureConfiguredDirectoriesNoopWhenDisabled`，回调新增 `TestArchiveRecreatesMissingArchiveDirectory`；`gofmt -l` 与 `go vet` 干净，Windows 与 Linux 双平台测试通过，Linux 容器（`debian:12-slim`）实测 manager 28 通过 0 跳过、hook 14 通过。另在容器内用真实默认配置跑 `aria2-manager --ensure-ports` 做端到端确认：输出配置含 `download_root=/vol1/@appshare/aria2/downloads`、`archive_root=.../downloads/completed`、`archive_enabled=true`、`special_mode=move`，并打印「已自动创建归档目录」且磁盘上出现 `drwxrwx---` 的 `completed`。`fpk/cmd/install_callback` 与 `fpk/cmd/main` 通过 `bash -n` 语法检查。版本提升至 `0.1.89`，`GOOS=linux GOARCH=amd64 CGO_ENABLED=0` 交叉编译并校验 ELF 魔数后打包，`fnpack` 构建成功，`aria2.fpk` 为 46044517 字节，SHA-256 `AB5F14E4E1F5FA4F4B806CFBAA0FD679B8F68E1DD48E6C97A18D03BFE1E70939`。待用户在测试机手动卸载（选择清空应用配置）并安装后执行设备验收：确认 `config.json` 为新默认值、`completed` 自动创建且属主为 `aria2`、真实 HTTP 下载落入 `completed`、界面简体中文与「下载完成后归档=是」。

2026-09-04 归档目录跟随下载目录（0.1.90）：`0.1.89` 从未安装到测试机，其默认下载目录、自动创建 `completed`、默认开启归档三项需求与本轮修复合并为 `0.1.90` 发布。本轮补上一处会在用户改下载目录后立即复发的缺陷：`archive_root` 一旦被安装期写成旧下载目录下的 `completed`，用户在前台换目录后归档仍会把文件搬回旧目录，正是此前反复出现的“改了下载路径文件还跑到别处”。新增 `syncArchiveRootWithDownload`，只在保存配置时生效：下载目录变化且归档目录仍是旧默认派生值（或为空）时重新派生为“新下载目录/completed”，用户显式自定义过的归档目录原样保留，读取与安装路径不变。验证：`gofmt -l` 干净，`go vet` 通过，新增 `TestSyncArchiveRootWithDownload`（默认跟随、空值派生、自定义保留、未改目录无副作用四例）后 Windows 全绿；交叉编译 Linux 测试包并在测试机 `debian:12-slim` 容器内实测 manager 29 通过 0 跳过、hook 14 通过。另用新二进制复跑端到端默认值脚本：`aria2-manager --ensure-ports` 输出 `download_root=/vol1/@appshare/aria2/downloads`、`archive_root=.../downloads/completed`、`archive_enabled=true`、`special_mode=move`，并记录「已自动创建归档目录」，磁盘出现 `drwxrwx---` 的 `completed`。ADR-010 增补第 7 条决策并把版本指向 `0.1.90`。设备侧现状：测试机仍运行 `0.1.88`，其 `config.json` 保留旧 `download_root=/vol1/1000/download`，需用户手动卸载并选择「清空应用配置」后安装新包才会采用新默认值。`fnpack` 构建产物 `aria2.fpk` 为 46045887 字节，SHA-256 `6C81E1B6BD029E7666682200EDCCC4228A45D1E3A3A87DC659A865C32B5E8518`，包内 `manifest` 版本 `0.1.90`、两个原生二进制均为 ELF，中文说明文案已确认入包。待设备安装后执行验收：确认 `config.json` 三项新默认值、`completed` 属主为 `aria2`、真实下载落入 `completed`、在界面改一次下载目录后归档目录同步跟随、简体中文界面与「下载完成后归档=是」，并留存截图。

2026-09-04 `0.1.90` 设备实测验收：用户手动卸载并清空配置后安装 `0.1.90`，应用中心显示 `0.1.90 / running`，
`aria2-manager` 与 `aria2c` 均以 `aria2` 用户运行，RPC 仅绑定回环。本轮按前一条记录的验收清单在设备上逐项复验，
全部通过，未改动任何实现代码，版本保持 `0.1.90`，FPK 未重建，SHA-256 仍为
`6C81E1B6BD029E7666682200EDCCC4228A45D1E3A3A87DC659A865C32B5E8518`。

- 归档目录跟随下载目录：`tests/fnos-verify-archive-follow.sh` 改为从设备现行配置动态读取 `download_root`、`archive_root`、`authorized_roots`，
  两个方向各执行一次并在结束后还原原值与授权根。`@appshare` 到 `/vol1/1000/download`、以及 `/vol1/1000/download` 回到 `@appshare` 两次 PUT 均返回 204，
  `archive_root` 都自动跟随为“新下载目录/completed”并当场补建目录，还原后配置与测试前一致。
- 真实下载落盘：还原配置后下载 1.2 MB 文件，文件由下载目录移入 `/vol1/1000/download/completed`，属主 `aria2:AppUsers`，
  `aria2.log` 记录 `archive done: /vol1/1000/download/reverify.bin -> /vol1/1000/download/completed/reverify.bin`，下载目录无残留。
  用户自己那条 141 MB 磁力任务同样在完成时被自动搬进 `completed`，ext4 创建时间与完成时间一致，说明走的是 rename 快速路径而非复制。
- 运行期默认值：改用单次 `aria2.getGlobalOption` 全量返回后过滤，确认 `max-overall-upload-limit=2097152`（2M）、
  `bt-request-peer-speed-limit=10485760`（10M）、`listen-port=6888`、`rpc-listen-port=6801`（6800 被设备上第三方应用占用后自动切换）、
  `dht-listen-port=6888`、`dht-message-timeout=10`、`seed-time=0`，BT 服务器已填入 20 条 ngosang best 列表。
- 删除未完成任务连带清理：新增 `tests/fnos-verify-cleanup.sh`，用 `busybox httpd` 加 128K/s 限速造出真正处于 `active` 的任务
  （60 MB 预分配部分文件与 `.aria2` 控制文件同时存在），`forceRemove` 后两者均从磁盘消失，脚本 0 失败。
  首次不限速时 200 MB 在回环链路上瞬间完成，判定为无效用例，因此保留限速设计。
- 界面：`tests/fnos-verify-ui.mjs` 0 失败，简体中文文案（正在下载、正在等待、已完成、系统设置、已连接）无英文残留，
  左侧菜单可点击（`#!/stopped`），「下载完成后归档」为「是」，归档目录显示 `/vol1/1000/download/completed`，
  并在界面执行“关闭→刷新→开启”往返后确认持久化；截图留存于 `tests/ui-evidence/01-downloading.png` 至 `05-archive-on.png`。
  界面开关往返之后再下载 0.8 MB 仍正确归档。

设备现状（用户自行设定，保留不回滚）：`download_root=/vol1/1000/download`、`archive_root=/vol1/1000/download/completed`、
`archive_enabled=true`、`special_mode=move`、`rpc_port=6801`、`listen_port=6888`、`authorized_roots=["/vol1/1000/download"]`。
测试机日志实际位于 `/vol1/@appdata/aria2/manager.log` 与 `aria2.log`，不存在 `logs/` 子目录，归档钩子输出在 `aria2.log`。
`/vol1/1000` 下的目录经 `ls` 与 `stat` 一律显示 `d--------- admin Users`，但对本应用创建的目录直接 `stat` 可见真实的
`0770 aria2:AppUsers`，属设备既有表现，不再追查。

遗留问题：`authorized_roots` 置空时持久化为 JSON `null`（GET 已归一化为 `[]`，仅外观问题）；AriaNg 内修改的运行期选项会在重启后被
`aria2.conf` 覆盖；`fpk/app/www/scripts` 存在一份未被引用的副本；设备上第三方 `fnos-api` 以 `0.0.0.0:9932` 加弱 token 监听，与本应用无关；
`fnpack` 对同一目录重复打包时外层哈希不稳定；`tests/fnos-verify-0188.sh` 第 2 节的选项回显解析有误（仅显示层面）；
新默认值只在“卸载并清空应用配置”后重装时生效，因为 `install_callback` 仅在 `config.json` 缺失时写入配置。

下一步：`0.1.90` 已满足用户当前全部需求，无阻塞项。可选维护项为上述遗留问题，以及把本轮的设备验收脚本并入发布前检查清单。

2026-09-04 遗留问题集中处理（0.1.91）：一次性清掉 `0.1.90` 验收后挂出的可动手项，实现、测试与打包全部完成，`0.1.91` 已出包等待用户在飞牛应用中心手动安装。
决策记录见 `docs/adr/ADR-011-config-versioned-migration.md` 与 `docs/adr/ADR-012-runtime-option-persistence.md`。

- 运行期选项持久化（原“前台改完、重启就回退”）：管理器新增 `runtimeOverrides`，在 RPC 网关
  `proxyRPC` 注入 `token:` **之前**取出 `aria2.changeGlobalOption` 的选项对象，只有上游
  HTTP 200 且 JSON-RPC 无 `error` 时才落盘到 `TRIM_PKGVAR/aria2-overrides.conf`
  （`0600`、按键名字典序、原子重写）。`fpk/cmd/main` 新增 `merge_runtime_options`，在
  `write_aria2_config` 生成 `aria2.conf` 时合并这些选项。合并刻意分两遍：先净化覆盖文件，再用
  “净化后的键集合”删除基础配置同名行；单遍写法会让被篡改的覆盖文件借 `rpc-secret=` 删掉真实密钥，
  这一点在实现过程中真的踩到过。独占项清单在 Go 与 shell 两层各自维护同一份：`dir`、端口、
  Tracker、会话、日志、`conf-path`、`input-file`、`dht-file-path*` 以及 `rpc-`、`on-` 前缀一律拒绝。
  新增 `GET`/`DELETE /api/v1/runtime-options`；`TRIM_PKGVAR` 为空时整套机制自动禁用。
  所有失败路径返回 0 并清理临时文件，覆盖文件损坏不会阻止启动。
- 配置版本化迁移（ADR-011）：`config` 结构体新增 `config_version`，管理器新增 `--migrate-config`。
  迁移只处理机器写出的取值（`10s` 秒数、`TRIM_PKGVAR/downloads` 历史下载目录、其下的
  `completed` 归档目录），用户显式选择的目录与 `special_mode=off` 一律不动，且幂等
  （重写后字节一致时返回空变更）。`install_callback` 与 `upgrade_callback` 在配置已存在时调用它并以
  `|| true` 兜底，调用点放在二进制 `chmod 755` 之后；生命周期脚本改用 `if [ -x ... ]; then` 形式，
  避免 `[ -x ... ] && var=...` 在 `set -e` 下静默中断脚本。常规读取入口一律不改写文件。
  设备上用真实 `config.json` 副本演练：输出仅“补齐缺失配置项 config_version”，其余取值逐字不变。
- 四个配置入口统一：`--config-env` 与 `--config-stdin` 分支原先各自补默认值，现全部走
  `decodeConfig`，环境变量、标准输入、文件描述符、文件四条路径得到同一份配置。
- 卸载清理：`uninstall_callback` 的“清空应用配置”清单补入 `aria2.session.tmp`、`dht6.dat`、
  `aria2-overrides.conf`、`config.json.tmp`、`aria2.conf.tmp`，以及合并过程留下的 `aria2.conf.overrides` 与 `aria2.conf.merged`。
- 死重清理：删除 `fpk/app/www` 与 `build-output/ariang` 下未被 `index.html` 引用的
  `scripts/`（66 文件 551,868 B）与 `styles/`（11 文件 47,773 B），两树现各 34 个文件且逐字节一致；
  顺带修掉一处会让未来构建静默退回上游文案的隐患——`build-output/ariang/langs/zh_Hans.txt`
  比 `fpk/app/www` 落后 149 字节（缺少归档相关中文说明补丁），已回灌并重算
  `build-output/ariang.SHA256SUMS`（111 → 34 条）。规则写入 `ui-adapter/README.md`。
  另删除 `manager/aria2.fpk`（45,958,560 B）、`manager/fpk/`（13,666,601 B）、
  `build-output/aria2-manager-new`（7,000,226 B）三处无任何引用的历史产物，共约 66 MB。
- 验收脚本修复：`tests/fnos-verify-0188.sh` 第 2 节原先把选项名当作
  `aria2.getGlobalOption` 的第一个参数传入，会被当成错误的 rpc-secret，导致整段核对恒为
  `<none>`；改为单次全量拉取后过滤，并对取不到值的选项判失败。新增可选的
  `RUNTIME_TEST=1` 用例，验证改动落盘、API 回显、独占项被拒绝以及收尾清理。
- 测试：新增 `tests/shell-merge-runtime-options.sh`，用 `awk` 从 `fpk/cmd/main` 提取真实的
  `merge_runtime_options` 再执行，避免测试与实现各写一份逻辑，4 个用例覆盖普通覆盖、
  独占/注入/损坏拒绝、空文件不改动、含 `=` 与 `,` 的取值原样保留。新增
  `manager/cmd/aria2-manager/runtime_options_test.go`（拒绝清单、取值字面量、令牌位置、
  接受判定、落盘与重载、排序稳定性、数量上限、禁用态、损坏文件忽略、网关端到端持久化、
  四入口一致性、`authorized_roots` 空数组往返、迁移只动机器值、`migrateConfigFile` 幂等）。
  设备侧 Debian 容器内 `manager` 44 项通过、0 跳过，`hook` 14 项通过；Windows 宿主
  `go build`/`go vet`/`gofmt -l` 干净，`go test -count=1 ./...` 两包通过（POSIX 专属用例跳过）。
  9 个 `fpk/cmd` 脚本在设备 `bash -n` 全通过，行尾保持 LF、CR=0。
- 发布：`fpk/manifest` 升至 `0.1.91`，两个 Go 产物以 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 -trimpath`
  重新编译（不加 `-s -w`）。`aria2-manager` 10,345,056 B，SHA-256
  `fd39b760bd8d90e7e3259958bd1caf7faf9eef7a423c3feebdc2311dd45fbde0`；`aria2-hook` 3,486,340 B，
  SHA-256 `5a4d3ddabd7f803b1f03d6d87b33609f3ae714ed15d3cf3d9227b96efdd37a3f`；固定的 `aria2c`
  保持 92,036,512 B / `eefb98afb22dd3a4b7bf70705f064704257542c85faef424d6273cf0ccb5729e` 未重建。
  `aria2.fpk` 为 45,964,465 B，SHA-256
  `4C2CECFAE1DB1A918C3CE3A0BD3823C32CE06C12942E608E3FB9FBC9B06033CE`。设备解包核对：包内三个
  二进制的哈希与仓库构建逐一致，`www` 恰为 34 个文件且不再含 `scripts/`、`styles/`，
  `langs/zh_Hans.txt` 为打好中文补丁的那一份。

2026-09-04 取值分叉缺陷修复（0.1.92）：`0.1.91` 已由用户安装并完成设备实测，迁移与持久化
全部通过，但验收当场抓出两个真实缺陷，二者都会让「界面看到的取值」与「重启后生效的取值」分叉，
本轮一并修复并出包 `0.1.92`。

- 缺陷一：`aria2.changeGlobalOption` 只把 `params[0]` 当选项对象。AriaNg 未配置密钥时发
  `params:[选项对象]`，旧逻辑却在全部参数里挑第一个非空对象，于是 `[{}, {选项}]` 会被 Aria2
  应用 `{}` 并返回 `OK`，网关却把从未生效的取值落了盘。设备实测内存值停在 9、重启后凭空变成 11。
  现改为只读 `params[0]`。
- 缺陷二（更严重）：网关专属项此前只在落盘侧拦截，转发侧仍然放行。设备实测
  `changeGlobalOption {"dir":"/etc", ...}` 返回 `OK` 且 `getGlobalOption` 回读到 `dir=/etc`，
  也就是任意管理员 AriaNg 会话一次保存就能把 Aria2 的写入目录移出 `authorized_roots`，
  绕过目录授权与归档边界。新增 `filterGatewayOwnedOptions`，在注入密钥之前重写
  `request.Params[0]`，专属项既不转发也不落盘，并记一条「已拒绝网关专属选项」日志；
  转发与落盘共用同一份拒绝清单，两侧不可能分叉。整批都是专属项时转发 `{}`，读回真实值，
  行为诚实。作用域限于 `changeGlobalOption`；`addUri`/`addTorrent` 的 `dir` 仍走既有的
  `authorized_roots` 校验。决策见 `docs/adr/ADR-013-gateway-owned-option-forwarding-filter.md`。
- 顺带确认：AriaNg 在用户配置了 RPC 密钥时会把 `token:` 放进 `params[0]`，这种形状 Aria2 直接报
  `The parameter at 0 has wrong type.`，是显式失败而非静默生效，按原样放行，不改。
- 测试：`runtime_options_test.go` 新增 `TestProxyRPCStripsGatewayOwnedOptions`（断言上游请求体里
  不出现 `/etc`、`9999`、`/bin/sh`、`evil.invalid`、`"dir"`、`rpc-listen-port`、
  `on-download-complete`、`bt-tracker`，而 `split`、`max-concurrent-downloads` 正常转发、
  密钥仍注入、存储层只收到合法两项）与 `TestFilterGatewayOwnedOptionsKeepsLegitimateKeys`。
  Windows 宿主 `go build`/`go vet`/`gofmt -l` 干净，`go test -count=1 ./...` 两包通过；
  设备 Debian 12 容器内 `manager` 46 项通过、0 跳过、0 失败，`hook` 14 项通过（较上轮 +2）。
- 验收脚本 `tests/fnos-verify-runtime-options.sh` 修掉两处会误报的写法：Aria2 回读把 `/` 转义成
  `\/`，明文比较 `dir` 在修好的版本上也会失败，改为反转义后再比较；`bt-tracker` 与 `on-*`
  的注入检查改为在原始 JSON 里 grep 特征串，不受转义影响；写死的目录与端口提为 `EXPECT_DIR`、
  `EXPECT_RPC_PORT` 可覆盖变量。脚本机制已用探针在设备实测：`val dir` 现返回
  `/vol1/1000/download`，`retry-wait` 确认能被 Aria2 运行期接受，探针写入的覆盖项已清空。
- 发布：`fpk/manifest` 升至 `0.1.92`，`aria2-manager` 10,349,445 B / SHA-256
  `143b69a610f26a201435128f9f012df78cd57dd0409ba76a17ad504fab30c64f`；`aria2-hook` 与固定的
  `aria2c` 逐字节未变。`aria2.fpk` 45,965,354 B / 外层
  `B3844B340CBD03CC71A15760588F1BE46D577A80E06D37F4D2431CEF0AD6D410`，溯源以解包后的逐文件清单
  `build-output/fpk-0.1.92.inner.SHA256SUMS`（63 行：前 49 行沿用 `0.1.91` 路径顺序，后 14 行为
  外层归档条目）为准，与 `0.1.91` 相比仅 `server/aria2-manager` 一行不同。

设备现状（用户自行设定，保留不回滚）：第 11 检查点安装的是 `0.1.91 / running`，`0.1.92` 待安装；
`rpc_port=6801`（6800 被第三方 `fnnas.aria2` 占用后自动切换）、`listen_port=6888`、
`dht_listen_port=6888`、`dht_message_timeout=10`、`download_root=/vol1/1000/download`、
`archive_root=/vol1/1000/download/completed`、`special_mode=move`、
`authorized_roots=["/vol1/1000/download"]`、`config_version=1`、Tracker 自动更新开启。
两个常驻进程均以 `aria2` 用户运行，RPC 仅监听回环。`aria2-overrides.conf` 当前不存在。

遗留问题：`0.1.92` 的两项修复尚未经设备实测，必须等用户安装后验证；界面暂无「清空运行期选项」
按钮，只有 `DELETE /api/v1/runtime-options`（有意为之，避免误触）；`fnpack` 对同一目录重复打包
时外层哈希不稳定，发布一律以解包后的逐文件清单为准；设备上第三方 `fnos-api` 以 `0.0.0.0:9932`
加弱 token 监听、第三方 `fnnas.aria2` 占用 `6800`，均与本应用无关；`/vol1/1000` 经 `ls` 显示为
`d---------` 但实际是 `0770 aria2:AppUsers`，不要再追查；设备上 `/tmp/aria2verify`、
`/tmp/inner092*`、`/hosttmp/*.test` 等临时验证物待清理。

下一步：请用户在飞牛应用中心直接安装 `E:\aria2-pro\aria2.fpk`（`0.1.92`，走升级路径，无需先卸载）。
装后由本仓库执行设备验收：跑 `tests/fnos-verify-runtime-options.sh`（期望 `0 处失败`）与
`tests/fnos-verify-0188.sh`（8 项真实选项回读），确认管理器日志在探测步骤出现
「已拒绝网关专属选项」，复核 `dir` 等专属项改不动也留不住、普通选项改动重启后存活，并截图存档。

2026-09-05 种子元数据台账（0.1.93）：把「删除或清空任务时一并清理未完成任务的相关文件」这条长期未闭环的规则
补到了 Aria2 自动保存的上传种子上。取证结论：`rpc-save-upload-metadata` 在 Aria2 1.36.0 默认为 true
（`OptionHandlerFactory.cc:789`），Aria2 会把 RPC 上传的种子按内容 SHA-1 重命名为
`<sha1>.torrent`（`meta4` 同理）写进任务生效目录（`RpcMethodImpl.cc:295-303`、`:347-353`）；
`on-download-*` 回调只有 gid 与本地路径两个入参，反推不出这个文件名，P3TERX 上游脚本同样存在这个盲点，
所以清理只能发生在知道文件名的那一层——RPC 网关。

- 实现：网关在 `aria2.addTorrent`/`aria2.addMetalink` 成功后按 gid 登记目标文件名，删除与清空时按台账精确删除，
  重启后用 `aria2.getSessionInfo` 对账剔除失效条目；台账文件 `$TRIM_PKGVAR/aria2-torrents.tsv`，0600、原子重写、
  字典序，容量上限 1024 条。所有失败路径一律偏向「留下文件」而不是「删错文件」。
  `applyLedgerMutation` 自带守卫：台账缺失、变更为空、作用域缺失或响应未被接受时直接返回，不碰文件也不删文件。
  卸载「清空应用配置」列表新增台账与临时文件。决策与已知限制见 `docs/adr/ADR-014-rpc-uploaded-metadata-ledger.md`。
- 测试：新增 `manager/cmd/aria2-manager/torrent_ledger_test.go` 14 例，覆盖行解析与投毒输入、0600 与重载、
  损坏行容忍、容量上限、`metadataFileName` 的 Aria2 命名复刻（标准/URL-safe base64、`*` 前缀、空白、非补齐）、
  `ledgerMutationFor` 逐方法判定（含客户端自带 token 与 `/../../etc`）、登记到删除端到端、越权目录跳过、
  清空与在活任务对账、`ledgerKnownGIDs` 合并、以及完整 `proxyRPC` 网关往返（含无关方法不得改写台账）。
  Windows 宿主：`gofmt -l` 与 `go vet` 干净，`go test -count=1 ./...` 两包通过，管理器顶层 43 通过 / 15 跳过 / 0 失败
  （跳过项为 POSIX 路径语义专属，含新增 1 项）。设备 Debian 12 容器内以 `go test -c` 交叉产物复跑：
  管理器顶层 58 通过 / 0 跳过 / 0 失败，`hook` 14 通过 / 0 失败。
- 发布：`fpk/manifest` 升至 `0.1.93`，变化面 4 项——`server/aria2-manager`（10,396,304 B / SHA-256
  `504245d85e9fa31779d6e3bb424bae4e6fa7f42ce6d311236b981fa68adc1693`）、`cmd/main`（新增 `--torrent-ledger`）、
  `cmd/uninstall_callback`、`manifest`；`aria2c` 与 `aria2-hook` 及 34 个 `www` 静态资源逐字节未变。
  `aria2.fpk` 45,990,129 B / 外层 `91010CBC68B538CC18CFE4705A2F332C42A2F446F185361E9F3015EF4AA9B463`，
  解包逐文件清单 `build-output/fpk-0.1.93.inner.SHA256SUMS` 本轮起改为按归档内实际顺序生成（小写十六进制、63 行），
  与 `0.1.92` 清单路径顺序一致，可逐行 diff；实测差异恰为上列 4 项。上一版归档为
  `build-work/aria2-0.1.92.fpk`（外层 `B3844B340CBD03CC71A15760588F1BE46D577A80E06D37F4D2431CEF0AD6D410`）。
- `0.1.92` 设备实测补齐（此前只出包未验收）：`tests/fnos-verify-runtime-options.sh` 结果 `0 处失败`。
  `params[0]` 形状判定正确（`[{}, {选项}]` 不再被误落盘，内存值停在 9）；专属项在转发侧也被拦——
  `dir` 实测仍为 `/vol1/1000/download`、`rpc-listen-port` 仍为 6801、`bt-tracker` 与 `on-*` 未被注入；
  专属项与普通项混发时 `retry-wait=5` 照常生效且只落普通项；重启后 9/16 存活、`aria2.conf` 合并无重复键、
  `rpc-secret` 保留；`DELETE /api/v1/runtime-options` 后回到 5/64。测试结束已把覆盖文件清空，设备状态未改。
- 验收脚本加固：`tests/fnos-verify-torrent-cleanup.sh` 增加前置守卫——运行中的管理器命令行缺少
  `--torrent-ledger` 即中止，避免在修复前版本上执行本用例反而制造无法追踪的残留种子；该守卫已在 `0.1.92`
  设备上实测拦截（退出码 1，未产生任何任务与文件）。同时核对下载目录 `/vol1/1000/download`：
  上一轮泄漏的 `3068c5d3471aceda` 种子文件已不存在，当前无 `.torrent`/`.meta4`/`.aria2` 残留，
  `completed/` 下仅用户自有的 141 MB 文件，保持不动。

设备现状（第 15 检查点，用户自行设定，保留不回滚）：安装并运行的仍是 `0.1.92`——设备
`/var/apps/aria2/manifest` 为 `0.1.92`、`server/aria2-manager` 哈希为 `143b69a6…`、进程命令行无
`--torrent-ledger`、`aria2-torrents.tsv` 不存在、部署的 `cmd/main` 亦无该参数。用户本轮表示「已经安装」，
但设备上最后一次部署发生在 `00:09`，而 `0.1.93` 的包在 `01:01` 才产出，设备上也不存在 `0.1.93` 的包副本，
因此判定 `0.1.93` 尚未安装，不能把种子清理项记为已验收。其余配置与第 14 检查点一致：
`rpc_port=6801`、`listen_port=6888`、`dht_listen_port=6888`、`archive_enabled=true`、
`download_root=/vol1/1000/download`、`authorized_roots=["/vol1/1000/download"]`、两个进程均以 `aria2`
用户运行、RPC 仅监听回环。

遗留问题：`0.1.93` 尚未安装，故 RPC 上传种子的连带清理仍无设备证据（本地与容器测试已覆盖逻辑本身）；
台账只在网关看到 `addTorrent`/`addMetalink` 时才会登记，客户端自带 `dir` 指向授权根之外时不落记录，
因此那一类文件不会被删除，这是有意偏向保守的已知限制；`manager.log` 每次启动被删除重建，
跨重启断言日志无效。

下一步：请用户在飞牛应用中心安装 `E:\aria2-pro\aria2.fpk`，务必核对文件为 45,990,129 B、
修改时间 2026-09-05 01:01（版本号 `0.1.93`），避免装成 `build-work` 里的 `0.1.92` 归档。装后本仓库执行：
`tests/fnos-verify-torrent-cleanup.sh`（期望 `0 处失败`，守卫应放行）、复核 `aria2-torrents.tsv`
存在且为 0600 `aria2` 所有，再复跑 `fnos-verify-runtime-options.sh`、`fnos-verify-0188.sh`、
`fnos-verify-cleanup.sh`、`fnos-verify-archive-follow.sh`、`fnos-verify-ui.mjs` 并留存截图。

## 第 18 检查点（2026-09-05）：`0.1.94` 启动自愈与持久启动日志

用户报告 `0.1.93` 在部分机器上安装后无法启动。根因已在设备侧取证确认，结论与修复如下。

- 根因：`config.json` 的 `download_root`、`archive_root`、`authorized_roots` 是机器相关取值。
  换机、卷号不同、共享目录被改名或删除、`TRIM_DATA_SHARE_PATHS` 为空、配置跨 NAS 拷贝，
  都会让 `prepareConfig` 失败；`fpk/cmd/main` 是 `set -eu`，两次管理器探针
  （`--ensure-ports`、`--print-download-root`）一旦非零退出或标准输出为空即当场中止，
  而原因只写在标准错误，应用中心不收集它，用户只剩一句「无法启动」。界面进程同时没起来，
  用户没有入口改回配置，形成死锁。设备复现：`authorized_roots=["/tmp/h1/locked"]` 与
  `download_root=/vol1/1000/no-such-share` 均退出码 1、标准输出为空。
- 实现：新增 `manager/cmd/aria2-manager/startup_repair.go`，`main` 侧新增 `readConfigBytes`
  以在解析失败时复用原始字节。自愈只作用于这两个生命周期探针（`startupProbe`），
  前台保存 `configStore.update` 与网关侧 `decodeConfig` 保持严格。逐条修复而非整体否决：
  失效授权根单独移出、失效或越界的下载目录置空后重新派生、归档跟随下载目录、
  非法端口复位、非法 `special_mode` 回 `move`、非 https 的 Tracker 源回落随包默认、
  配置为空或不可解析时退回出厂默认。端口、DHT、网关、Tracker 开关与 `config_version`
  等用户设置一律不被顺带改写。最后兜底为 `$TRIM_PKGVAR/downloads`（`0770`，属主交还该目录
  属主）。修复结果由既有的 `cmd/main` 回写通道持久化，不产生额外落盘路径。
  决策、复现表与已知代价见 `docs/adr/ADR-015-startup-self-repair.md`。
- 可观测性：新增持久启动日志 `$TRIM_PKGVAR/startup.log`（`0660`、归属应用用户、只留最近
  200 行、不随启动清空，卸载「清空应用配置」时删除）。此前 `aria2.log` 与 `manager.log`
  每次启动被删除重建，跨重启线索全部丢失。生命周期脚本在每个 `exit 1` 前先记录原因，
  两个探针改用 `2>>"$startup_log"` 归集管理器的中文自愈说明；日志写入全程 `|| true`，
  日志不可写绝不反过来导致启动失败。
- 容器实测抓到两个真实缺陷，均已修复并补测试：
  1. 出厂 BT 端口恰好等于用户自定义 RPC 端口时（例如 `rpc=6888`），把 BT 复位到 `6888`
     等于没修，`validateConfig` 仍判端口冲突而拒绝启动。改为保住 RPC 不动，
     在 BT 侧按 `6888 → 6889 → 6800 → 6801` 挑第一个不冲突取值，空闲与否交给
     `ensureAvailablePorts`（`ADR-006`）。
  2. 私有兜底目录首次创建时，`normalizeConfig` 先看到空下载目录，连带把归档判成关闭，
     即使用户开着「下载完成后归档」也会静默失效。改为在 `normalizeConfig` 之前建好兜底目录。
     `sharedRoots()` 同时纳入已存在的私有目录，并新增 `dataShareRoots()` 供自愈判定
     「这台机器到底还有没有共享目录」，使降级必然留痕。
- 测试：新增 `startup_repair_test.go` 9 例。真实 Linux 复跑（`debian:12-slim` + 交叉编译测试产物）
  管理器顶层 67 通过 / 0 跳过 / 0 失败，`hook` 14 通过 / 0 失败；Windows 宿主 `go test ./...`
  两包通过，`gofmt -l` 与 `go vet ./...` 干净。新增 `tests/startup-repair-sim.sh`：在容器内用
  真机复现的坏配置驱动 `--ensure-ports`，6 组用例逐一断言「退出码 0 + 标准输出是合法配置 +
  逐条中文说明 + 用户端口未被改写 + 兜底后归档仍开启」，结果 `全部用例通过`。
  9 个 `fpk/cmd/*` 脚本全部通过 `bash -n`。
- 新工具：`tests/fnos-diagnose-startup.sh`（只读、一次性，`bash -n` 通过并已在设备实跑）。
  自动探测卷号，收齐机器上下文、版本、专属用户、ELF/CPU 兼容性自检、`startup.log`、
  配置取值与目录可写性、`TRIM_DATA_SHARE_PATHS`、端口占用、进程与网关 Socket 并给出结论。
  本机自检结果：`正常 13 项，注意 1 项，异常 0 项`（唯一「注意」是尚未安装 `0.1.94`，
  因此还没有 `startup.log`）。该脚本初版有三处自身缺陷并已修好：JSON 是缩进格式导致取值读空、
  manifest 的 `key = value` 两侧带空格、以及把本应用自己的 `aria2c` 误报为端口冲突、
  把 `ss` 的对端地址列 `0.0.0.0:*` 误判成 RPC 对外开放；现已改用 `ss -lntupH sport = :<port>`
  并按 pid 区分自身与外部占用。
- 发布：`fpk/manifest` 升至 `0.1.94`。变化面恰 4 项——`server/aria2-manager`
  （10,417,162 B / SHA-256 `7007c6bb450aa4b735955666e0d5ed89c25d6002fc2e8fde6c08481e30bf52f7`）、
  `cmd/main`（启动日志与逐条原因）、`cmd/uninstall_callback`（清理 `startup.log`）、
  `manifest`；`aria2c`、`aria2-hook` 与 34 个 `www` 静态资源逐字节未变。
  `aria2.fpk` 46,003,943 B / 外层 `834DDB72A3F12D9D96F8B545A06615409D1C91A9AC60FFBB2D1D17B481E13B41`，
  解包逐文件清单 `build-output/fpk-0.1.94.inner.SHA256SUMS`（63 行、归档内实际顺序、小写十六进制），
  与 `0.1.93` 清单逐行 diff 的差异恰为上列 4 项。上一版归档为 `build-work/aria2-0.1.93.fpk`
  （45,990,129 B / 外层 `91010CBC68B538CC18CFE4705A2F332C42A2F446F185361E9F3015EF4AA9B463`）。

设备现状（第 18 检查点）：仍在运行 `0.1.93`，配置为用户自行重装后的默认取值
`rpc_port=6800`、`listen_port=6888`、`dht_listen_port=6888`、`special_mode=move`、
`download_root=/vol1/@appshare/aria2/downloads`、`archive_root=…/completed`、
`authorized_roots=[]`、`config_version=1`，两进程以 `aria2` 运行、RPC 仅监听回环、
Tracker 自动刷新正常（20 个）。这些取值由用户设定，保留不回滚。

遗留问题：`0.1.94` 尚未经设备验证，自愈与 `startup.log` 的实机效果待用户安装后确认；
「有些机器无法启动」目前只有 1 台可用测试机，其余机器的具体病因需靠
`tests/fnos-diagnose-startup.sh` 的输出换取；`0.1.93` 的 RPC 上传种子连带清理仍无设备证据；
下载后归档不生效 / 完成后长期停留分享 / 多任务下载卡顿这几项行为问题属于 Phase 7 待办，
本轮未触及。

下一步：请用户在飞牛应用中心安装 `E:\aria2-pro\aria2.fpk`（`0.1.94`，
46,003,943 B，走升级路径即可，不必先卸载）。装后本仓库执行：确认 `startup.log` 生成且
含「启动完成」；跑一次「下载目录改成不存在路径 → 重启」验证自愈真机生效；
随后补 `tests/fnos-verify-torrent-cleanup.sh`、`fnos-verify-runtime-options.sh`、
`fnos-verify-0188.sh`、`fnos-verify-cleanup.sh`、`fnos-verify-archive-follow.sh`、
`fnos-verify-ui.mjs` 并留存截图。换机失败时先取 `tests/fnos-diagnose-startup.sh` 全文输出。

### 第 18 检查点补充：第二类启动根因与真实产物启动链路验证

同一天继续排查「有些机器无法启动」，发现配置类根因之外还有第二类原因，且两者在前台完全同形：
`server/aria2c` 是动态链接的，除 glibc 之外还依赖 `libz`、`libexpat`、`libsqlite3`、`libssl3`、
`libcrypto3`、`libssh2`、`libstdc++`。任一缺失时 `aria2c` 只往 `aria2.log` 留一行动态链接器报错就退出，
应用中心不收集该文件，前台仍然是一句没有信息的「无法启动」。容器实测里 `debian:12-slim` 缺这五个库，
`cmd/main` 于是稳定地卡在就绪门禁上，正好复现了这类机器的表现。
处理：`cmd/main` 在拉起进程前先执行一次 `aria2c --version`，失败即把真实原因写进 `startup.log` 再中止，
成功也记一行；取值不接管道，避免 `$?` 变成 `head` 的退出码而掩盖真实失败。
决策补为 `ADR-015` 第 11 条。彻底消除这一类需要静态链接 Core 或随包附带这些库，
涉及许可证与产物溯源，作为后续项单独评估，本轮不混合发布。

新增 `tests/startup-e2e-container.sh`：把解包后的真实发布产物放进容器，按设备的目录布局与 `TRIM_`
变量复刻，直接执行 `cmd/main start`，8 组用例逐一断言退出码、`startup.log` 内容、持久配置回写、
进程存活、服务用户属主、网关 Socket、日志轮转上限与「清空应用配置」的删除范围。
结果 `真实产物启动链路验证全部通过`（用例 1 健康配置、用例 2 换机双失效、用例 3 配置缺失带原因中止、
用例 4 半截 JSON、用例 5 端口自冲突、用例 6 整机无共享目录、用例 7 轮转、用例 8 卸载清理）。
其中用例 2 的 `startup.log` 完整打出了三条中文修复说明并以「启动完成」收尾，
是本轮修复的主要设备级证据；用例 6 同时证明兜底启动不会静默关掉自动归档。
构建可复现性一并确认：同一源码树二次 `go build -trimpath` 与入包产物逐字节一致。

排查过程中记录两条工具性事实，避免后续重复踩坑：`fnpack` 会向包内 `manifest` 注入自引用的
`checksum` 字段，所以包内 `manifest` 与源 `fpk/manifest` 必然哈希不同，且任何内容变化都会带动它变化，
逐文件清单里这一行的漂移属预期；归档内 `cmd/*` 的模式是 `666`，可执行位由应用中心在安装时补，
容器里复刻目录时必须自行 `chmod 755`，否则会把「回调没执行」误判成产品缺陷。
另外本轮修掉了 `cmd/uninstall_callback` 被重复插入的一行 `startup.log`（PowerShell 的 .NET 异常默认
不终止脚本，导致一次"失败"的补丁其实写进去了），发布清单已按修正后的产物重新生成。

最终发布物：`aria2.fpk` 版本 `0.1.94`，46,003,943 B，外层
`834DDB72A3F12D9D96F8B545A06615409D1C91A9AC60FFBB2D1D17B481E13B41`；
`server/aria2-manager` 10,417,162 B / `7007c6bb450aa4b735955666e0d5ed89c25d6002fc2e8fde6c08481e30bf52f7`。
解包逐文件清单 `build-output/fpk-0.1.94.inner.SHA256SUMS`（63 行、归档实际顺序、小写十六进制），
相对 `0.1.93` 差异恰为 `server/aria2-manager`、`cmd/main`、`cmd/uninstall_callback`、`manifest` 四项，
`aria2c`、`aria2-hook` 与 34 个 `www` 静态资源逐字节未变。上一版归档在
`build-work/aria2-0.1.93.fpk`。设备侧待办不变：等用户安装 `0.1.94` 后做真机验收。

## 第 19 检查点（2026-09-05）：`0.1.95` 静态 Core 消除第二类启动根因

承接第 18 检查点补充记录的第二类根因（Core 动态链接、宿主缺库即静默起不来）。
当时把它列为「涉及许可证与产物溯源，后续单独评估」，本轮直接做掉：Core 改为完全静态链接并进入发布物。

- 产物：`alpine:3.22` 容器内以 `ARIA2_STATIC=yes` 构建的 musl 静态 Core，10,739,352 B /
  SHA-256 `a482fb24722b87a0a719acee330e329d6dfa21075808c66202e451782812015b`。源码与补丁与
  第 18 检查点完全一致（`release-1.36.0` 加同一组 4 个补丁），唯一差异是链接方式。
  构建配方 `build-inputs/container-build-static.sh`（SHA-256
  `6534bf6c6512323298497b012f5e71a231b013bd12ca02b1bf5d8a5f174596a1`）。
- 关键陷阱：上游唯一的静态开关是环境变量 `ARIA2_STATIC=yes`；只传 `--enable-static-link`
  或 `LDFLAGS=-static` 会静默产出仍然动态链接的二进制。构建脚本因此设三道硬门禁
  （configure 摘要 `Static build: yes`、`src/Makefile` 含 `-all-static`、产物无 `NEEDED`）。
  Alpine 的 `*-dev` 包不含 `.a`，必须额外装 `zlib-static`、`expat-static`、`sqlite-static`、
  `libssh2-static`、`openssl-libs-static`。
- 决定性证据：在缺 5 个库的 `debian:12-slim` 容器里做 A/B，静态产物输出
  `aria2 version 1.36.0`，动态产物报
  `error while loading shared libraries: libexpat.so.1` —— 正是过去被 `aria2.log` 吞掉的那一行。
  同一容器再用解包后的真实发布产物跑 `tests/startup-e2e-container.sh`，8 组用例全部通过，
  `startup.log` 打出「Aria2 Core 自检通过」。
- 真机功能：`tests/fnos-verify-static-core.sh` 对入包的那一份字节（发货态复验，哈希与包内
  清单一致）在 glibc 2.36 设备上全绿：34 个现役选项无 `Unknown option`、RPC 握手、
  HTTP 与 HTTPS 真实下载、DHT 与 BT 的 IPv4/IPv6 监听、磁力提交与移除、无异常日志。
  能力面与动态 Core 逐项一致（`--version` 特性列表字节一致、完整 `--help` 逐行一致）。
  该脚本原有的 `readelf -d` 静态门禁在设备上因命令缺失而空转，已一并修成以 `file` 为主判据。
- 真实 BT 数据面：新增 `tests/fnos-verify-bt-transfer.sh` 与夹具 `tests/btfixture/`
  （独立 Go module，测试专用、不入发布包），在一台机器上离线跑端到端 BitTorrent：
  夹具造 `.torrent` 并起最小 HTTP Tracker 交换对等体，种子端与下载端都用这份静态 Core，
  传 8 MiB 随机数据后逐字节比对 SHA-256 一致，且日志互证
  `uploaded/downloaded=8.0MiB/8.0MiB` 与 `0B/8.0MiB`，即数据确实经过对等体传输。
  这一步补的正是此前只有「能建监听端口」而没有数据面的缺口。
  过程中固化两条上游行为：认领已存在的完整文件需 `--check-integrity=true`；
  `seed-ratio` 与 `seed-time` 同时为 0 等于不做种。
- 设备事实修正：fnOS 上没有 `readelf`、`objdump`、`nm`、`strings`，只有 `file`、`ldd`、`eu-readelf`。
  旧版 `tests/fnos-diagnose-startup.sh` 以 `readelf -d` 判链接方式，命令缺失时输出为空，
  把 92 MB 的动态 Core 误报成「静态链接、不依赖宿主共享库」——这个假阳性已在真机复现。
  现改为以 `file` 为主判据、`ldd` 兜底，无法判定时显式报错；验证脚本的静态门禁同步加固。
- 许可证：musl(MIT)、OpenSSL 3.5(Apache-2.0)、zlib、expat(MIT)、SQLite(PD)、
  libssh2(BSD-2)、libstdc++(GPLv3+ 含 Runtime Library Exception) 静态链接均可分发，
  且 Core 本身即 GPL-3.0，未引入比动态链接更重的义务。分析写入 `ADR-016` 与
  `docs/license-and-provenance-plan.md`。
- 发布：`fpk/manifest` 升至 `0.1.95`，`aria2.fpk` 13,853,128 B（原 46,003,943 B），外层
  SHA-256 `F6082363BB3021688FBA6C3BBEEAC64D1CC98D32F3760CBEB49D0DA27BD68760`。
  包内清单 `build-output/fpk-0.1.95.inner.SHA256SUMS`（63 行）相对 `0.1.94` 变化面为 3 项：
  `server/aria2c`（换静态）、`cmd/main`（下述病因归集）与 `manifest`
  （含 fnpack 注入的自引用 `checksum`，必然漂移），其余 60 项
  （`server/aria2-manager`、`aria2-hook`、其余 8 个 `cmd/*`、34 个 `www` 静态资源等）
  逐字节未变。管理器与 hook 本轮无源码改动，`gofmt -l`、`go vet ./...` 干净，
  `go test -count=1 ./...` 两包通过；9 个 `fpk/cmd/*` 与新脚本全部通过真机 `bash -n`。
- 文档：新增 `docs/adr/ADR-016-static-core-linking.md`；`ADR-015` 第 11 条改注为
  「自 `0.1.95` 起从缓解手段降级为安全网」；`docs/phase3-build-record.md` 重写为静态构建记录
  并保留退役产物的历史小节；`docs/license-and-provenance-plan.md` 新增静态链接义务一节；
  `README.md` 当前状态改为「两类根因各消一类」。
- 遗留：`0.1.95` 尚未经设备验收；`0.1.94` 的自愈与 `startup.log` 同样待装机确认，本轮
  容器证据不能替代真机；下载后归档不生效 / 完成后长期停留分享 / 多任务下载卡顿仍属 Phase 7 待办。
- 新增 `tests/startup-failure-attribution-container.sh`：用「`--version` 能过预检、真正拉起立刻以
  127 退出」的假 Core 逼出就绪门禁失败分支，断言 `aria2.log` 的病因文字被归集进持久
  `startup.log`，并断言启动仍非零退出、回滚不留残 pid。结果 `启动失败病因归集验证通过`。
  决策记为 `ADR-015` 第 13 条。
- 下一步：请用户在飞牛应用中心安装 `E:\aria2-pro\aria2.fpk`（`0.1.95`，13,853,128 B，
  走升级路径即可）。装后执行：确认 `startup.log` 末尾为「启动完成」且 Core 自检一行显示静态；
  跑一次「下载目录改成不存在路径 → 重启」验证自愈真机生效；复跑
  `fnos-verify-torrent-cleanup.sh`、`fnos-verify-runtime-options.sh`、`fnos-verify-0188.sh`、
  `fnos-verify-cleanup.sh`、`fnos-verify-archive-follow.sh`、`fnos-verify-ui.mjs` 并留存浏览器截图。
  换机仍失败时先取 `tests/fnos-diagnose-startup.sh` 全文输出。

### 第 19 检查点补充：`0.1.95` 发货产物全链路行为验证

设备验收需要等用户安装，因此先把「起来之后功能对不对」在容器里用**解包后的真实发布件**跑通。
新增 `tests/shipped-chain-container.sh`，与 `tests/startup-e2e-container.sh` 分工明确：
后者考察起不起来（退出码、自愈、日志、属主），本脚本考察功能链路
（生命周期脚本 -> 管理器网关 -> 静态 Core -> 完成回调 -> 归档搬移 -> 删除清理）。

容器内没有 `curl` 与 `python`，所以驱动、HTTP 源与轮询都由 `tests/btfixture` 承担，
为此给夹具补了 `fileserver`（含会挂住的 `/stall.bin`，用来制造真实的部分文件与 `.aria2`）、
`gateway`（走应用 Unix Socket，按管理员身份头模拟浏览器请求）和 `waitfile` 三个子命令。
夹具是测试专用工具，独立 Go module，不进入发布包。

结果 `发货产物全链路验证全部通过`：

- 服务启动后 `startup.log` 记录启动完成、网关 Socket 就绪、Core 预检通过。
- 经网关 `aria2.addUri` 提交本地 HTTP 任务，`tellStatus` 回读 `completedLength=4194304`、`errorCode=0`。
- 完成回调把文件搬进 `completed/`，归档件与源文件 **SHA-256 逐字节一致**，原位置清空，`move` 语义正确。
- 制造真实的部分下载（`stall.bin` 与 `stall.bin.aria2` 同时存在）后 `aria2.remove`，
  两者一并清除，下载目录再无 `stall.bin` 相关条目。
- 清理过程未误伤已归档文件。

这一步的意义：`0.1.95` 换 Core 只动链接方式，上述结果证明它与管理器、回调、归档与清理
逻辑的组合行为没有回归；同时把「删除未完成任务不清理文件」和「归档不生效」两条历史抱怨
在发货产物上做了正面证伪。真机上仍需按 `tests/fnos-verify-archive-follow.sh` 与
`tests/fnos-verify-cleanup.sh` 复核一次，因为容器的文件系统语义（同卷 rename 快速路径、
跨卷回退）与真机不完全一致。

## 第 20 检查点（2026-09-07）：`0.1.96` 运行期体检，前端批量调用恢复与启动锁收口

用户问「还有没有什么可以优化的」。`0.1.95` 已在测试机安装并连续运行 1 天 14 小时，
`startup.log` 只有自检通过与启动完成两行，两类「无法启动」根因（配置类与二进制依赖类）
均已消除，因此本轮不再找启动问题，改为带着假设做运行期取证。结论分「已修」「不改」两类。

### 设备取证

- 进程与资源：`aria2c`（pid 111703）与 `aria2-manager`（pid 111708）均以 `aria2` 运行，
  RSS 8 MB / 19.6 MB，CPU 0%，宿主 7.9 GB 内存。空载没有可挤的资源。
- 日志：`aria2.log` 258 B、`manager.log` 291 B（运行 1.5 天），`startup.log` 有 200 行上限；
  `manager.log` 显示 09-05 19:00 初次与 09-06 19:00 周期各刷新 20 个 Tracker，自动更新正常。
- 网关：`aria2.getGlobalStat` 经 Socket 返回 200 正常数据；`system.multicall` 与
  `system.listMethods` 均返回 `400 invalid RPC request`。
- 权限：默认下载目录 `drwx------ aria2`、归档目录 `drwxrwx--- aria2`，
  但 `/vol1/@appshare` 与 fnOS 官方 `fnaudio` 目录同为 `d--------- root:root`，卷挂载带 `trimacl`；
  用非 root 的 `admin` 实测能直接读出应用用户写的 0700 文件。**可见性由系统层负责，不改权限模型。**
- 残留：`/vol1/@appdata/aria2/.start.lock` 是 `drwxr-xr-x root:root`，mtime 停在启动时刻，
  证明启动路径从未释放它；`aria2.pid`/`manager.pid` 同样是 root 属主。

### 根因与修复一：网关把 AriaNg 的批量调用判为非法（前端功能缺失）

AriaNg 有三处依赖 `system.multicall`/`system.listMethods`：任务详情页把 `tellStatus` 与
`getPeers` 合成一次批量查询、重试前把 `tellStatus` 与 `getOption` 合成一次、调试页枚举方法。
而 `proxyRPC` 只放行 `aria2.` 前缀，这三处功能在 `0.1.95` 及以前一直是坏的，
非 silent 的失败还会弹错误提示并反复重试——与用户历史上「多任务下载界面很卡」的方向一致。

不能简单放行，两条 Core 契约都在真机取过证据：顶层 `params[0]` 放 token 会报
`The parameter at 0 has wrong type`（该位置要求调用数组），子调用各自不带 token 则整批
`Unauthorized`；而嵌套的 `aria2.changeGlobalOption {"dir":"/etc"}` 会被 Core 执行并返回 `OK`，
运行期目录真的被改掉，嵌套 `remove` 同理会绕开 `0.1.94` 起的种子台账。
因此实现为受控转发：只读白名单逐项校验（默认拒绝）、上限 128 项、剥除客户端自带 token 后
把真实密钥注入每个嵌套调用的 `params[0]`、顶层不带 token；`system.listMethods` 不注入且
`params` 缺失时补空数组。决策、契约证据与四条被否决的备选方案见 `ADR-017`，
`ADR-013` 增补「后续」一节说明绕过路径如何收口。

### 根因与修复二：启动锁只在 `stop` 释放（生命周期卫生）

`start` 用 `mkdir $TRIM_PKGVAR/.start.lock` 互斥，但只有 `stop` 分支删它，成功路径与
`write_aria2_config`、端口自愈里的 `exit 1` 都不删。现在抢到锁即 `trap 'rmdir' EXIT`，
`stop` 不再代删（它没抢过锁），强杀残留仍走原有回收分支；两条原本静默 `exit 0` 的分支
补上中文原因；`uninstall_callback` 用 `rmdir` 兜底（`rm -f` 删不掉目录）；`aria2.pid` 交回专属用户。
见 `ADR-015` 第 14 条。

### 测试

- `go test -count=1 ./...` 两包通过；新增 `manager/cmd/aria2-manager/multicall_test.go` 9 项：
  嵌套注入位置、剥除客户端 token、12 个非只读方法逐个拒绝、4 种畸形列表、超上限、平铺归一，
  以及三条经 `proxyRPC` 的端到端断言。`gofmt -l` 与 `go vet ./...` 干净。
- `debian:12-slim` 容器用解包后的 `0.1.96` 真实发布件复跑三条链路，全部通过：
  `startup-e2e-container.sh` 8 组用例新增「启动成功后已释放启动锁」；
  `startup-failure-attribution-container.sh` 新增「启动失败同样释放了启动锁」；
  `shipped-chain-container.sh` 新增用例 E，在真实静态 Core 前确认批量查询取到数据、
  详情页形状已过鉴权、嵌套 `changeGlobalOption` 与 `remove` 均返回 400、`dir` 未被改写、
  `listMethods` 放行。原有 A-D（真实下载、归档逐字节一致、部分文件与 `.aria2` 清理、不误伤）不回归。
- `fpk/cmd/main`、`install_callback`、`uninstall_callback`、`upgrade_callback` 与新脚本全部通过真机 `bash -n`。

### 发布

`fpk/manifest` 升至 `0.1.96`，`aria2.fpk` 13,861,296 B，外层 SHA-256
`F9F22A52F0E73F880FEAE9F29074BB22232F892C57D73A69C4E0C4D52B0353C9`。
包内清单 `build-output/fpk-0.1.96.inner.SHA256SUMS`（63 行）相对 `0.1.95` 变化面 4 项：
`server/aria2-manager`、`cmd/main`、`cmd/uninstall_callback`、`manifest`；其余 59 项逐字节未变，
含静态 Core（`a482fb24…015b`）与 34 个 `www` 资产。`aria2-hook` 二次构建与入包产物哈希一致，
再次确认构建可复现。

### 遗留与下一步

- 等设备验收：`tests/fnos-verify-multicall.sh`（六组断言）与浏览器确认任务详情页、
  BT peer 列表、重试按钮恢复；`0.1.95` 起待做的真机归档与清理复核（
  `fnos-verify-archive-follow.sh`、`fnos-verify-cleanup.sh`、`fnos-verify-torrent-cleanup.sh`）
  现在才有条件一并执行，因为需要用户先升级到 `0.1.96`。
- 记录在案但不值得动的：日志无上限（实测日增量几百字节）、
  `@appshare` 目录权限模型（由 `trimacl` 负责）。两份 pid 文件的属主本轮一并交回了专属用户。

### 本次事故与它改变的事实

`0.1.96` 收尾复验时，一条 `plink` 命令漏写 `docker run` 前缀，把
`tests/startup-e2e-container.sh` 直接执行在测试机宿主机上。该脚本首步为
`rm -rf /var/apps/aria2 /vol1 /run/aria2`（本意是在容器内复刻目录布局），
因此清空了测试机数据卷 `/vol1`：`/vol1/1000`、`/vol1/docker`、`/vol1/@appshare` 的共享内容
与各已安装应用的数据全部丢失，Docker 目前不可用。卷是 ext4 + LVM，无快照、无回收站，
恢复只能靠 VM 级快照或外部备份。fnOS 系统分区与应用中心本身正常（`5666` 返回 200）。

- 防范规则已写入 `AGENTS.md`「验证安全红线」：带 `rm -rf` 的测试脚本只允许
  `docker run ... bash <脚本>` 这一种执行形式；一次性脚本不得出现 `rm -rf` 绝对路径。
- 受影响的验证项：为 pid 属主新增的两条断言（`aria2.pid`/`manager.pid` 应为 `aria2`）
  **尚未在容器内跑过**；启动锁释放、multicall 用例 E 等其余结论均在事故之前已取得，仍然有效。
- 本机 Windows 侧无 Docker，无法替代验证；测试机恢复（或换一台）后需补跑
  `startup-e2e-container.sh` 一次，再谈 `0.1.96` 的设备验收。
- 项目侧不受影响：源码、静态 Core、AriaNg 产物、`0.1.96` 的 FPK 与全部构建记录都在本仓库内。
