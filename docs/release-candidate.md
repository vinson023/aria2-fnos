# Aria2 候选发布物

## 产物

- 文件：`aria2.fpk`（版本 `0.2.4`，10,646,580 B；GitHub Release 中名为 `aria2-0.2.4-x86_64.fpk`）
- 架构：x86_64（manifest 使用 `platform = x86`）
- FPK 外层 SHA-256：`10FD5D4513C209FB8C396004AFD80D4F4E14EE90CF9E6B2E7D36FBE80F956FD1`
  （仅作当次产物指纹；因 `fnpack` 元数据不稳定，溯源以下方解包清单为准）
- 应用标识：`appname = aria2`，专属账号与组 `aria2`，开发者与发布者 `vinson023`。
  这是相对 `0.1.x` 的**破坏性改名**，应用中心按新应用处理，旧 `aria2pro` 应用不会被覆盖升级
  （见 `docs/adr/ADR-018-identity-rename-and-archive-automation.md`）。
- `0.2.4` 新增：安装向导可设置 RPC/BT/DHT 三个端口、RPC 密钥、下载完成后自动归档
  （`SPECIAL_MODE`）与下载文件夹路径；新增 `wizard/config`，安装完成后可在应用中心
  「应用设置」里修改同一组参数，保存后按需自动重启生效。落盘、校验与目录创建统一收口到
  `aria2-manager --apply-wizard`（见 `docs/adr/ADR-019-wizard-configurable-parameters.md`）。
- `0.2.3` 修复：AriaNg 连接提示与安装向导文案仍显示「Aria2 Pro」——飞牛定制版设置加载
  函数把 RPC 别名写死为 `"Aria2 Pro"`，本版改为 `"Aria2"`，并同步两处安装向导端口校验提示。
- `0.2.2` 修复：AriaNg 原生「基本设置」页面仍显示「归档目录」和「下载完成后归档」字段，
  直接在 AriaNg 压缩包的 `basicOptions` 数组中删除 `special-mode` 和 `fnos-archive-dir`，
  使这两个字段不再出现在基本设置页面上。
- `0.2.1` 修复：归档目录创建失败（下载目录对 aria2 用户不可写）时降级为关闭归档，
  不再阻塞下载目录保存；前端适配层移除"特殊模式" select 与"设置特殊模式"菜单项。
- Core：Aria2 1.36.0 加同一组 4 个补丁，**完全静态链接**（musl），10,739,352 B，
  SHA-256 `A482FB24722B87A0A719ACEE330E329D6DFA21075808C66202E451782812015B`
  （自 `0.1.95` 起逐字节未变）。
- 管理器：静态 Go x86_64，7,819,424 B（`0.2.4` 重新编译，含向导配置写入、目录自动授权与
  `--repair-permissions` 运行期自愈入口），
  SHA-256 `B1587A1FE0BEEB2E6BBC550BC2692FBF26E30AE545BC9FE69BDC9F2FFAA5AFCD`。
- 回调：静态 Go x86_64，2,793,632 B（`0.2.4` 重新编译，路径白名单纳入配置里的下载/归档根），
  SHA-256 `359DAA4A030CD503E3440854459DE17ABE48DB5865B7FE6811C242F2A775D503`。
  Go 二进制自 `0.2.4` 起带 `-ldflags="-s -w"` 构建，体积较此前版本明显下降，行为不变。
- AriaNg：1.3.14 fnOS 适配 bundle（网关路径已改为 `/app/aria2/jsonrpc`）；构建时使用 Node，运行时仅静态文件。
  `0.2.2` 起 `basicOptions` 数组不再包含 `special-mode` 和 `fnos-archive-dir`；
  `0.2.4` 起移除了 `dir`（网关本就拒绝运行期修改下载目录），下载目录只在应用设置里维护；
  `0.2.3` 起 RPC 别名默认值由 `Aria2 Pro` 改为 `Aria2`。
- SDK：固定 `@trimjs/web-app@0.4.2`，仅纳入静态 `dist/index.js`；FPK Scope 为 `trim.file.userAccess`、`trim.file.sharedAccess`，manifest `micro_app=true`。

本版溯源清单：`build-output/fpk-0.2.4.inner.SHA256SUMS`（49 个文件）。
外层 `wizard/` 目录自 `0.2.4` 起多出 `wizard/config`（应用设置表单），它不在 `app.tgz` 内，
因此内层文件数仍为 49。

## 已验证

> 以下为 `0.1.x` 系列在设备上的既有验证结论。`0.2.0` 因应用标识改名，需重新在设备
> 完成安装与功能验收，目前尚未执行。

测试机为 fnOS Debian 12 x86_64。已验证管理员网关身份、旧身份头拒绝、回环 RPC、
6801/6889 TCP/UDP 监听、Tracker 刷新与重启持久化、完成/错误/停止回调清理与归档、session/DHT
文件保留、卸载数据保留以及 FPK 内无 Node/npm/node_modules。飞牛桌面入口实际以 iframe
打开，`src` 为 `/app/aria2/`；经应用 Unix Socket 访问根路径返回 AriaNg HTML，管理员
请求 `aria2.getVersion` 返回 Core 1.36.0，旧 `X-Trim-User` 请求返回 401。

## 观察项

Codex 浏览器对测试机 `/app/aria2/` 注入了 `ERR_BLOCKED_BY_CLIENT` 拦截页，因而无法从
该自动化客户端取得 AriaNg 的像素截图或网络面板；这不影响设备实际 iframe 路径、Socket
响应和 RPC 代理的验证结果。

`fnpack` 对同一源目录的重复打包会写入不同归档元数据，因此 FPK 的外层 SHA-256 不保证
逐字节稳定；本文件记录的是当前工作区发布文件的哈希。Core、管理器、hook 与固定静态资源
的版本和 SHA-256 溯源见 Phase 3 构建记录。

2026-09-04 对 `0.1.91` 做过一次实证：源目录 `fpk/` 完全不变，连续两次 `fnpack build` 得到
45,964,237 B（外层 `36D360E83D931A5F9FB556C7F93F341BD62234B09D888770592CE7C3C261B9EA`）与
45,964,480 B（外层 `41AD472151858FAC2C4F75EEBEB44D1617C2E1455FE85483D96FDF09C11A7005`），
连内层 `app.tgz` 的哈希都随之变化；把两个归档分别解包后按文件计算 SHA-256，49 行清单逐行一致。
因此发布溯源一律以解包后的逐文件清单为准，而不是 FPK 外层哈希：
`build-output/fpk-0.1.91.inner.SHA256SUMS` 记录最终发布归档（45,964,465 B / 外层
`4C2CECFAE1DB1A918C3CE3A0BD3823C32CE06C12942E608E3FB9FBC9B06033CE`）解包后的 49 个文件，
覆盖 `server/aria2c`、两个 Go 产物、三个 hook 包装脚本、`cmd/*`、`config/*`、`wizard/*`
与 34 个 `www` 静态资源。

2026-09-04 发布 `0.1.92`：修复 `0.1.91` 设备验收抓出的两处取值分叉缺陷
（`aria2.changeGlobalOption` 只认 `params[0]`、网关专属项在转发侧未拦截），
决策见 `docs/adr/ADR-013-gateway-owned-option-forwarding-filter.md`。
解包逐文件清单换为 `build-output/fpk-0.1.92.inner.SHA256SUMS`，共 63 行：
前 49 行沿用 `0.1.91` 的路径顺序，另 14 行为外层归档条目（`manifest`、`ICON*.PNG`、
9 个 `cmd/*`、2 个 `wizard/*`），因此行数是扩充而非变更；与 `0.1.91` 相比内容差异
只有 `server/aria2-manager` 一行，`www/` 仍为 34 个文件不变。上面 49 行的表述
保留为 `0.1.91` 时期的历史记录。

2026-09-05 发布 `0.1.93`：补上「删除任务连带清理 RPC 上传的种子元数据」这条长期未实现的规则。`rpc-save-upload-metadata` 在 Aria2 1.36.0 默认开启，Aria2 会把上传的种子按内容 SHA-1 改名存进任务目录，而 `on-download-*` 回调只拿到 gid 与本地路径，无法反推这个文件名，因此改由网关侧记台账：`aria2.addTorrent`/`addMetalink` 成功后按 gid 登记目标文件名，删除、清空（`purgeDownloadResult`）时按台账精确删除，重启后用 `aria2.getSessionInfo` 对账剔除失效条目。台账落在 `$TRIM_PKGVAR/aria2-torrents.tsv`（0600、原子重写），卸载选择「清空应用配置」时一并清除；所有失败路径一律偏向「留下文件」而不是「删错文件」。决策见 `docs/adr/ADR-014-rpc-uploaded-metadata-ledger.md`。变化面：`server/aria2-manager`、`cmd/main`（新增 `--torrent-ledger`）、`cmd/uninstall_callback`（清理台账）、`manifest`（版本号）共 4 项，`aria2c`、`aria2-hook` 与 `www/` 逐字节未变。解包逐文件清单 `build-output/fpk-0.1.93.inner.SHA256SUMS` 改为按归档内实际顺序生成，与 `0.1.92` 清单的路径顺序完全一致，方便逐行 diff。同时给 `tests/fnos-verify-torrent-cleanup.sh` 加了前置守卫：运行中的管理器若没有 `--torrent-ledger` 就直接中止，避免在修复前的版本上执行测试反而留下无法追踪的种子文件（该守卫已在 `0.1.92` 设备上实测拦截成功）。`0.1.92` 的两项修复也已在本设备通过 `tests/fnos-verify-runtime-options.sh` 实测，`0 处失败`：`params[0]` 形状判定正确、`dir` 停在 `/vol1/1000/download` 改不动也留不住、`rpc-listen-port` 保持 6801、`retry-wait` 等普通项混发时仍生效、重启后 `max-concurrent-downloads=9`/`split=16` 存活、`DELETE` 后回到 5/64。`0.1.93` 的验收需等用户安装后执行。

2026-09-05 发布 `0.1.94`：解决「`0.1.93` 在部分机器上安装后无法启动」。根因经设备取证确认为
配置中的机器相关取值（`download_root`、`archive_root`、`authorized_roots`）在换机、卷号不同、
共享目录改名或删除、配置跨 NAS 拷贝时无法解析，而 `fpk/cmd/main` 是 `set -eu`：两次管理器探针
（`--ensure-ports`、`--print-download-root`）只消费标准输出，失败原因留在标准错误且不被应用中心收集，
于是前台只剩一句「无法启动」，界面进程同时没起来，用户没有任何入口改回配置，形成死锁。
设备复现样本：`authorized_roots=["/tmp/h1/locked"]` 与 `download_root=/vol1/1000/no-such-share`
均退出码 1、标准输出为空。

修复口径是**保存严格、启动自愈**：前台 `configStore.update` 与网关 `decodeConfig` 保持不变，
只有两个生命周期探针走 `repairStartupConfig`，逐条修复而不整体否决，保住端口、DHT、网关、
Tracker 开关与 `config_version` 等用户设置，最后兜底到 `$TRIM_PKGVAR/downloads`（`0770`、属主交还
目录属主）。新增持久启动日志 `$TRIM_PKGVAR/startup.log`（`0660`、只留最近 200 行、不随启动清空、
卸载「清空应用配置」时删除），每条自愈改动与每个 `exit 1` 原因都落在那里。决策见 `ADR-015`。

容器实测（`debian:12-slim` + 交叉编译产物）抓出并修好两个真实缺陷：出厂 BT 端口恰好等于用户自定义
RPC 端口时，把 BT 复位到 `6888` 等于没修，`validateConfig` 仍判冲突而拒绝启动，现改为保住 RPC 不动、
在 BT 侧按 `6888 → 6889 → 6800 → 6801` 选不冲突取值；私有兜底目录首次创建时 `normalizeConfig` 会先把
归档判成关闭，导致「下载完成后归档」在兜底启动后静默失效，现改为在派生归档之前建好兜底目录。
验证：管理器真实 Linux 复跑 67 通过 / 0 跳过 / 0 失败（新增 `startup_repair_test.go` 9 例），
`hook` 14 通过；`tests/startup-repair-sim.sh` 用真机坏配置驱动 `--ensure-ports`，6 组用例断言
「退出码 0 + 标准输出是合法配置 + 逐条中文说明 + 用户端口未被改写 + 兜底后归档仍开启」，结果 `全部用例通过`；
9 个 `fpk/cmd/*` 全部通过 `bash -n`。新增只读排查脚本 `tests/fnos-diagnose-startup.sh`，已在设备实跑，
自检 `正常 13 项 / 注意 1 项 / 异常 0 项`。

变化面恰 4 项：`server/aria2-manager`（10,417,162 B / `7007c6bb…52f7`）、`cmd/main`、
`cmd/uninstall_callback`、`manifest`；`aria2c`、`aria2-hook` 与 34 个 `www` 静态资源逐字节未变。
解包逐文件清单 `build-output/fpk-0.1.94.inner.SHA256SUMS`（63 行、归档内实际顺序、小写十六进制），
路径顺序与 `0.1.93` 清单一致、可逐行 diff，实测差异恰为上列 4 项。上一版归档为
`build-work/aria2-0.1.93.fpk`（45,990,129 B / 外层 `91010CBC68B538CC18CFE4705A2F332C42A2F446F185361E9F3015EF4AA9B463`）。
`0.1.94` 与 `0.1.93` 的设备验收（含 RPC 上传种子的连带清理）均需等用户安装后执行。

同一天补记：「无法启动」其实有两类同形根因。除配置取值不可解析外，`server/aria2c` 是动态链接的，
除 glibc 外还依赖 `libz`、`libexpat`、`libsqlite3`、`libssl3`、`libcrypto3`、`libssh2`、`libstdc++`，
任一缺失时 Core 只往 `aria2.log` 留一行动态链接器报错就退出，而应用中心不收集该文件，前台表现与
配置类故障完全一样。因此 `0.1.94` 的 `cmd/main` 在拉起进程前先执行一次 `aria2c --version`，
把真实原因写进 `startup.log` 再中止（`ADR-015` 第 11 条）。彻底消除这一类需要静态链接 Core
或随包附带这些库，涉及许可证与产物溯源，留待后续单独评估。

发布前的最后一道验证是 `tests/startup-e2e-container.sh`：把上面这份发布物解包后放进 `debian:12-slim`，
按设备目录布局与 `TRIM_` 变量复刻，直接执行 `cmd/main start`，8 组用例（健康配置、换机双失效、
配置缺失、半截 JSON、端口自冲突、整机无共享目录、日志轮转、卸载清理）全部断言退出码、
`startup.log` 内容、持久配置回写、进程存活与服务用户属主、网关 Socket 与删除范围，
结果 `真实产物启动链路验证全部通过`。同一源码树二次 `go build -trimpath` 与入包产物逐字节一致。
另记两条工具性事实：`fnpack` 会向包内 `manifest` 注入自引用的 `checksum` 字段，
所以包内 `manifest` 与源文件必然哈希不同、且随任何内容变化而漂移，逐文件清单里这一行属预期；
归档内 `cmd/*` 的模式是 `666`，可执行位由应用中心安装时补上。

2026-09-05 发布 `0.1.95`：把上面「留待后续单独评估」的第二类根因直接消除——Core 改为完全静态链接。
构建器为 `alpine:3.22` 临时容器（3.22.5，镜像 ID `sha256:b66e0ce64844f5c6435b0c4bfd965558199ab0f53270846861c979cb1ac29365`，
gcc 与 g++ 14.2.0），源码、补丁与 Aria2 版本与动态 Core 完全一致，唯一差异是链接方式，
配方为 `build-inputs/container-build-static.sh`。上游唯一的静态开关是环境变量 `ARIA2_STATIC=yes`，
只传 `--enable-static-link` 或 `LDFLAGS=-static` 会静默产出仍然动态的二进制，
因此脚本设三道硬门禁（configure 摘要 `Static build: yes`、`src/Makefile` 含 `-all-static`、产物无 `NEEDED`）。
Alpine 的 `*-dev` 包不含 `.a`，必须额外装五个 `*-static` 包。

决定性证据：在缺 `libexpat`、`libsqlite3`、`libssl3`、`libcrypto3`、`libssh2` 的 `debian:12-slim` 容器里做 A/B，
静态 Core 输出 `aria2 version 1.36.0`，动态 Core 报
`error while loading shared libraries: libexpat.so.1` —— 正是过去被 `aria2.log` 吞掉、前台只剩「无法启动」的那一行。
同一容器内用解包后的真实发布产物复跑 `tests/startup-e2e-container.sh`，8 组用例全部通过。
真机侧 `tests/fnos-verify-static-core.sh` 针对**入包的那一份字节**在 glibc 2.36 设备上全绿：
34 个现役选项无 `Unknown option`、RPC 握手、HTTP 与 HTTPS 真实下载、DHT 与 BT 的 IPv4/IPv6 监听、
磁力提交与移除、无异常日志；能力面与动态 Core 逐项一致（`--version` 特性列表字节一致、完整 `--help` 逐行一致）。
许可证分析（musl/MIT、OpenSSL 3.5/Apache-2.0、zlib、expat/MIT、SQLite/PD、libssh2/BSD-2、
libstdc++/GPLv3+ 含 Runtime Library Exception）见 `ADR-016` 与 `docs/license-and-provenance-plan.md`，
静态链接没有引入比动态链接更重的义务。

一处设备事实修正值得所有后续脚本遵守：fnOS **没有** `readelf`、`objdump`、`nm`、`strings`，
只有 `file`、`ldd`、`eu-readelf`。以 `readelf` 为唯一判据的检查会在设备上静默空转——
旧版 `tests/fnos-diagnose-startup.sh` 就因此把 92 MB 动态 Core 报成「静态链接、不依赖宿主共享库」，
该假阳性已真机复现并修正为以 `file` 为主判据、`ldd` 兜底、无法判定时显式报错。

发布前又补了一处可观测性修复，因此变化面为 3 项：`server/aria2c`（换静态）、
`cmd/main`（就绪门禁失败时归集日志尾部）与 `manifest`（版本号加自引用 `checksum`）。
解包逐文件清单 `build-output/fpk-0.1.95.inner.SHA256SUMS`（63 行、归档实际顺序、小写十六进制），
与 `0.1.94` 清单逐行 diff 实测仅上述 3 行不同，其余 60 项（管理器、hook、其余 8 个 `cmd/*`、
2 个 `wizard/*`、34 个 `www` 静态资源、3 份许可证文本等）逐字节未变。

### 启动失败病因归集

`aria2.log` 与 `manager.log` 每次启动被删除重建，应用中心又不收集它们，所以
「进程拉起来了但立刻挂掉」这类故障跨一次重启后线索全灭，前台照样只剩「无法启动」。
`cmd/main` 现在在就绪门禁失败分支里把两个日志的尾部若干行归集进持久的 `startup.log`。
配套新增 `tests/startup-failure-attribution-container.sh`：用一个「`--version` 能过预检、
真正拉起立刻以 127 退出」的假 Core 把这条路径逼出来，断言病因文字
（示例里是 `error while loading shared libraries: libsomething.so.9`）确实进入 `startup.log`，
并断言启动仍按预期非零退出、回滚不留残 pid。结果 `启动失败病因归集验证通过`。

### 发货产物全链路行为

`tests/shipped-chain-container.sh` 用解包后的 `0.1.95` 真实发布件在容器里走完
生命周期脚本 -> 网关 -> 静态 Core -> 完成回调 -> 归档 -> 清理 一整条链路，
结果 `发货产物全链路验证全部通过`：真实 HTTP 下载完成后文件被搬进 `completed/`
且与源文件 SHA-256 逐字节一致、原位置清空；制造出的部分文件与 `.aria2`
在 `aria2.remove` 后一并清除；已归档文件未被清理误伤。
容器内没有 `curl` 与 `python`，因此 `tests/btfixture` 追加了 `fileserver`（含会挂住的
`/stall.bin`）、`gateway`（走应用 Unix Socket 并带管理员身份头）与 `waitfile` 子命令。
同一轮把 `tests/startup-e2e-container.sh` 的 8 组用例在重建后的产物上复跑，仍然全部通过。

## `0.1.96`：运行期体检与受控的批量调用转发（2026-09-07）

`0.1.95` 已于测试机安装并稳定运行 1 天 14 小时，`startup.log` 干净，静态 Core 未再出现任何
启动类故障，因此 `0.1.94`/`0.1.95` 的「无法启动」验收结论可以关闭。本轮改为排查「起来之后
还有什么可优化」，取证结论分三类。

### 前端功能缺失：批量调用被网关判为非法

对运行中的 `0.1.95` 直接打网关 `/jsonrpc`，`system.multicall` 与 `system.listMethods` 均返回
`400 invalid RPC request`，而对照组 `aria2.getGlobalStat` 返回 `200`。AriaNg 用 multicall 合并
任务详情页的 `tellStatus` + `getPeers` 与重试前的 `tellStatus` + `getOption`，
`/debug` 页用 `system.listMethods` 枚举方法，所以这三处功能此前一直是坏的，失败还会触发反复重试。
修法是受控转发而不是简单放行：Core 对嵌套调用逐个鉴权（顶层放 token 反而报
`The parameter at 0 has wrong type`），且真机实测嵌套的 `changeGlobalOption {"dir":"/etc"}`
被 Core 接受并确实改掉了运行期目录，嵌套 `remove` 同样会执行并绕开种子台账。
因此按只读白名单逐项校验、剥除客户端自带 token 后把真实密钥注入每个嵌套调用。
决策、契约证据与备选方案见 `ADR-017`，`ADR-013` 增补「后续」一节说明这条绕过路径如何收口。

### 生命周期卫生：启动锁与 pid 属主

设备上 `.start.lock` 在正常启动后仍以 `root:root` 残留在应用数据目录：只有 `stop` 分支删它，
成功路径与 `write_aria2_config`、端口自愈里的 `exit 1` 都不删。现在抢到锁即 `trap ... EXIT`，
`stop` 不再代删，「清空应用配置」用 `rmdir` 兜底；另外两条原本静默 `exit 0` 的分支补上中文原因，
避免应用中心以为启动成功而前台只得到一个空界面。`aria2.pid` 交回专属用户属主。见 `ADR-015` 第 14 条。

### 体检中排除的三条线索（不改）

默认下载目录 `drwx------ aria2` 看着像权限缺失，实测非 root 的 `admin` 可直接读取其中 0700 的文件，
且 `/vol1/@appshare` 下 fnOS 官方 `fnaudio` 同样是 `d---------`，卷挂载带 `trimacl`——可见性由系统层负责。
`aria2.log`/`manager.log` 运行 1.5 天各 258/291 字节（`summary-interval=0`、`show-console-readout=false`），
`startup.log` 有 200 行上限，无膨胀风险。空载 `aria2c` RSS 8 MB、管理器 19.6 MB、CPU 0%，
不存在需要优化的资源占用。

### 验证与发布

`go test ./...` 两包通过（`multicall_test.go` 新增 9 项，含三条经 `proxyRPC` 的端到端断言），
`gofmt -l` 与 `go vet ./...` 干净。真实 Linux 容器复跑三条链路全绿：
`startup-e2e-container.sh` 8 组用例（新增「启动成功后已释放启动锁」）、
`startup-failure-attribution-container.sh`（新增「启动失败同样释放了启动锁」）、
`shipped-chain-container.sh` 用例 A-E（新增真实 Core 前的批量查询可用、
嵌套 `changeGlobalOption`/`remove` 被拒、`dir` 未被改写、`listMethods` 放行）。
9 个 `fpk/cmd/*` 与新脚本全部通过真机 `bash -n`。
发布：`fpk/manifest` 升至 `0.1.96`，`aria2.fpk` 13,861,296 B，外层 SHA-256
`F9F22A52F0E73F880FEAE9F29074BB22232F892C57D73A69C4E0C4D52B0353C9`，
包内清单 `build-output/fpk-0.1.96.inner.SHA256SUMS`（63 行）相对 `0.1.95` 变化面 4 项：
`server/aria2-manager`、`cmd/main`、`cmd/uninstall_callback`、`manifest`；
其余 59 项逐字节未变（含静态 Core 与全部 `www` 资产）。`aria2-hook` 二次构建与入包产物
逐字节一致，再次确认构建可复现。
设备验收已完成（2026-10-16，测试机 192.168.x.x）：
`tests/fnos-verify-multicall.sh` 六组断言全部通过（16/16），
`tests/fnos-diagnose-startup.sh` 15 项正常 0 异常，
`tests/fnos-verify-torrent-cleanup.sh` 0 处失败（种子台账清理规则验证通过），
PID 属主（`aria2.pid`/`manager.pid` 均属 `aria2:aria2`）在真机直接验证通过，
浏览器前端（AriaNg 任务详情页、BT peer、重试按钮）用户确认可正常打开。
完整报告见 `docs/acceptance-report-0.1.96.md`。

### 测试机数据卷事故（2026-09-07，记录以免被遗忘）

本轮在把 `cmd/main` 的 pid 属主改动纳入复验时，一条 `plink` 命令漏写 `docker run` 前缀，
导致 `tests/startup-e2e-container.sh` 被**直接执行在宿主机上**。该脚本第一步是
`rm -rf /var/apps/aria2 /vol1 /run/aria2`，本意是在容器里复刻目录布局，结果清空了
测试机的数据卷 `/vol1`（ext4 + LVM，无快照、无回收站）：`/vol1/1000`、`/vol1/docker`、
`/vol1/@appshare` 下的共享内容以及已安装应用的数据全部丢失，Docker 暂不可用。
fnOS 系统分区与应用中心本身未受影响（`5666` 仍返回 200）。

防范规则已写进 `AGENTS.md` 的「验证安全红线」：带 `rm -rf` 的测试脚本只能在
`docker run ... bash <脚本>` 这一条命令内执行，不允许先跑脚本再补容器前缀；
一次性脚本不得出现 `rm -rf` 绝对路径。

受本次事故影响的验证项：`startup-e2e-container.sh` 为 pid 属主新增的两条断言
（`aria2.pid`/`manager.pid` 属主应为 `aria2`）**尚未在容器内跑过**，其余三条链路
（含启动锁释放与 multicall 用例 E）在事故之前已全部通过。测试机恢复后需补跑。

## `0.2.4`：六个运行参数可在安装向导与应用设置中修改（2026-11-24）

此前只有 RPC 端口与 BT 监听端口能在安装时指定，`wizard/config`、`cmd/config_init`、
`cmd/config_callback` 均为空壳，DHT 端口、RPC 密钥、归档模式与下载目录只能手工改
`config.json`。`0.2.4` 把六项参数统一收口（决策见 ADR-019）：

| 参数 | 向导字段 | 安装默认值 | 应用设置中的空值语义 |
|---|---|---|---|
| RPC 监听端口 | `wizard_rpc_port` | `6800` | 留空 = 保持当前值 |
| BT 监听端口 | `wizard_listen_port` | `6888` | 留空 = 保持当前值 |
| DHT 监听端口 | `wizard_dht_listen_port` | 同 BT 端口 | 留空 = 保持当前值 |
| RPC 密钥 | `wizard_rpc_secret` | 留空自动生成 48 位 | 留空 = 不修改 |
| 自动归档（`SPECIAL_MODE`） | `wizard_special_mode` | `move`（开启） | 单选「保持当前设置 / 开启 / 关闭」 |
| 下载文件夹 | `wizard_download_root` | 飞牛数据共享目录 | 留空 = 保持当前值 |

实现要点：

- 校验与落盘集中在 `aria2-manager --apply-wizard`，复用常规配置路径的端口互斥、
  下载目录授权范围、归档目录派生与属主继承逻辑；`install_init`/`config_init` 的
  shell 校验是同一口径的提前反馈（安装阶段二进制尚未落地，无法调用管理器）。
- RPC 密钥下限由 32 位放宽到 16 位，限定 `[A-Za-z0-9._-]`。`aria2.conf` 由 shell 逐行
  写出，含空白、引号、反斜杠或 `#` 的取值会破坏文件，因此在入口拒绝，而不是让它
  “看起来保存成功、重启后连不上”。
- 下载目录允许尚不存在：在已授权根内逐级创建，新建目录属主跟随上级目录，避免 root
  建出应用用户无法穿越的中间目录。
- `config_callback` 在服务运行时执行一次受控重启（约 8 秒）使新端口/密钥/目录生效；
  应用未运行时跳过重启，配置在下次启动时读取。

**验证状态：已在 fnOS 设备（192.168.5.141，x86_64，fnOS 6.18.18.c1186-trim）完成端到端验收。**

| 验证项 | 方法与结果 |
|---|---|
| 安装向导写入 | `appcenter-cli install-fpk -e <env>`，六个字段全部生效：`config.json` 得到 `rpc_port=6805`、`listen_port=6885`、`dht_listen_port=6886`、`special_mode=move`、`download_root=/vol1/@appshare/aria2/downloads/movies`；密钥文件为提交值；下载目录与 `completed` 自动创建且属主为 `aria2` |
| 运行期生效 | 启动后 `aria2.conf` 为 `rpc-listen-port=6805`、`listen-port=6885`、`dht-listen-port=6886`、`dir=<新目录>`、`rpc-secret=<新密钥>`；`ss` 显示 RPC 仅监听 `127.0.0.1:6805` |
| 应用设置表单 | `GET /app-center/v1/config/wizard?appName=aria2` 返回六个字段，且**飞牛会用当前值回填 `initValue`**（端口、密钥、下载目录均为现值，密钥以密码框回显） |
| 应用设置保存 | `POST` 修改 DHT 端口 6886→6887、归档 `move`→`off`：`config.json` 与 `aria2.conf` 同步更新，服务自动重启（PID 变化），`startup.log` 记 `应用设置已更新并重启生效` |
| 原样保存不重启 | 以完全相同取值再次提交：PID 未变化、无重启日志——`--apply-wizard` 退出码 `2` 生效 |
| 密钥轮换 | 提交新密钥后 `rpc.secret` 与 `aria2.conf` 同时更新，重启后经网关 `aria2.getVersion` 仍返回 Core 1.36.0，AriaNg 显示「已连接」 |
| 非法输入 | RPC 端口与 BT 端口相同、下载目录越界（`/etc/...`）、密钥过短三种提交均被拒绝（返回 `code 10000`），配置未被改写 |
| 下载目录改为数据卷路径 | 提交 `/vol1/1000/downloads`（不在随包共享目录内）成功：`config.json`、`aria2.conf` 与运行中 `dir` 同步更新，归档目录 `completed` 自动创建且属主为 `aria2`；停止再启动后配置不被启动自愈重置 |
| 数据卷内新建子目录 | 提交 `/vol1/1000/downloads/movies`：目录被创建并归属 `aria2`，应用可写入 |
| 系统目录拒绝 | 提交 `/etc/aria2-test` 仍被拒绝（`code 10000`），配置未改写 |
| 删除任务 | 经网关添加真实下载 → 完成后按 AriaNg 顺序调 `/app/aria2/api/v1/cleanup`（204）+ `aria2.removeDownloadResult`：文件已删除，`tellStopped` 中不再出现该 gid |
| 清空已结束任务 | 同样流程改用 `aria2.purgeDownloadResult`：文件删除、已停止列表归零 |
| 越界路径不阻塞删除 | 同一请求里混入下载目录之外的文件路径：该路径被跳过并记日志，接口仍返回 204，任务删除不被中断 |
| 界面保存不再报异常 | 在应用中心「应用设置 → 运行设置 → 编辑」把 RPC 端口改为 6806 后点「保存并重启」：先提示「保存成功」，无「服务异常」；`config.json` 与 `aria2.conf` 同步为 6806，RPC 监听 6806 |
| 单次重启 | `startup.log` 只在应用中心的 `/v1/restart` 后出现一次启动记录，回调不再重复重启 |
| 启动窗口 | 把 `cmd/main` 的固定 `sleep 3` 换成进程与网关 Socket 的就绪轮询：`appcenter-cli start` 从 6.2s 降到 4.2s，重启期间“未运行”的窗口相应缩短 |
| 保存期间的接口 | 保存→重启的 15 秒内每 0.5s 查询一次应用详情：状态依次为 stopping → stopped → starting → running，全部 `code 0`，`/v1/restart` 返回 `code 0` |
| 下载目录选择器 | 安装向导不支持目录选择（仅有 7 种字段类型）。应用内 Aria2 右上角文件夹图标 →「选择并授权目录」使用飞牛 SDK 选择器：设备上 `TRIM_DATA_ACCESSIBLE_PATHS=/vol1/1000/downloads`（对应「可访问的文件」），`config.json` 的 `download_root` 同步为同一路径（对应「运行设置」） |
| RPC 密钥修改 | `password` 类型时保存请求携带的仍是旧值（输入框已是新值），密钥实际改不掉；改为 `text` 后请求携带新值，`rpc.secret` 与 `aria2.conf` 同步更新，`startup.log` 出现「RPC 密钥已更新」 |
| 含符号的密钥 | `Vinson@2024`（11 位、含 `@`）保存成功：请求携带该值，`rpc.secret` 与 `aria2.conf` 同步，重启后 `aria2.getVersion` 正常返回 1.36.0 |
| 开机后目录授权 | 模拟飞牛开机重置权限（`chgrp Administrators` + 755）后，aria2 立刻失去写权限（实测 `Permission denied`）；看护脚本在 60 秒周期内重新授权，`startup.log` 记录「运行期权限自愈：... 已把目录 ... 划给专属应用组」，写入恢复 |
| 多任务 fd 上限 | 5.20 上 14 个 BT 任务时 `aria2c` 软限制仅 1024、实际已用 1011，出现 4 个 `errorCode=15`（`No file descriptors available`）与一批 tracker `Failed to resolve the hostname ... System error`。启动脚本加 `ulimit -n 65536` 后：`/proc/<pid>/limits` 为 65536/65536，`startup.log` 记录「aria2c 文件描述符软限制已设为 65536」，14 个任务全部恢复下载，fd 用量约 870 |
| 下载完成后归档 | 下载根改为 `/vol1/1000/downloads` 后，修复前回调报 `cleanup skipped: download path cannot be resolved`（白名单只含随包共享目录与空的 authorized_roots），文件停在下载根；修复后日志为 `archive done: /vol1/1000/downloads/COPYING -> /vol1/1000/downloads/completed/COPYING`，文件确实移入 `completed` |
| 归档后用户可读 | 归档文件权限显示为 `rwxrwx--- aria2 AppUsers`（trimacl 以 ACL 掩码呈现组位），以 `admin`(uid 1000) 实测可读取、可在 `completed` 内创建文件 |
| 安装时填已有共享目录 | 用户填 `/vol1/1000/Download`（管理员属主、权限 755、未经应用内选择器授权）：修复前写探测失败导致安装中止；修复后目录被划给专属组 `aria2`（`drwxrwxr-x admin aria2`），aria2 可写、admin 仍可写，配置写入该路径，`completed` 一并创建 |
| 不合规密钥的前端拦截 | 填入 `abc`（3 位）后点「保存并重启」：**没有任何 `config/wizard` 请求**，字段变红并在下方显示「RPC 密钥需为 8-128 位可见字符，不能包含空格或中文」 |
| 下载目录唯一入口 | AriaNg「基本设置」页移除 `dir` 字段（网关本就拒绝运行期修改 `dir`），下载目录只在应用设置里修改 |
| 单元测试 | `go vet` 与 `go test` 在 ubuntu-24.04 runner 通过，覆盖端口范围与互斥、密钥字符集、目录越界、数据卷判定、空值保持、安装期创建、配置回读、密钥未变化判定 |

一处框架限制：配置流程失败时，App Center 返回的 `msg` 是 `15002:exit code: 1`，没有带出
脚本写入 `TRIM_TEMP_LOGFILE` 的中文原因（安装流程会带出）。脚本侧已同时写 stderr 与
`TRIM_TEMP_LOGFILE`，该文案是否展示由飞牛决定，应用侧无法进一步控制。

## ARM64 (aarch64) 版本（实验性，未设备验收）

`0.2.3` 起新增 ARM64 构建支持。Go 管理器与回调用本机 Go 交叉编译
（`GOOS=linux GOARCH=arm64 CGO_ENABLED=0`），产物已验证为 ELF aarch64。
`0.2.4` 起 Go 二进制统一改为 `-ldflags="-s -w"` 构建，因此体积与哈希相对 `0.2.3` 变化：

| 组件 | 大小 | SHA-256 |
|---|---|---|
| `aria2-manager` (arm64) | 7,209,120 B | `6263B6C34E3C06590FFE4352061B7FCD8D9DF8F2C79112F92B70B3BD2952C886` |
| `aria2-hook` (arm64) | 2,687,136 B | `68CE81CD93B3BD52EE493C5B0C7F5F9588E87EB190E557F2CE3F66426D562411` |
| `aria2c` (arm64) | 10,070,168 B | `57097DA852528679971FA2BD43EBA7DD0742F72EE7538B6FD05053CB6D048627` |

ARM64 `aria2c` 静态核心由 GitHub Actions 在 `alpine:3.22` arm64 容器中构建
（`.github/workflows/build-static-core.yml`），使用与 x86_64 完全相同的源码、补丁和
`container-build-static.sh` 配方。CI 构建耗时约 24 分钟（QEMU 模拟 arm64）。

FPK 目录树由 `tools/build-arm64-fpk.py` 自动组装：复制 x86 FPK 模板，替换 Go 二进制，
改 `manifest` 的 `platform = arm`（fnpack 1.2.3 用 `arm` 表示所有 ARM 变体包括 aarch64）。
`0.2.4` 产物 `aria2-0.2.4-arm64.fpk`（10,431,960 B，外层 SHA-256
`00E34117D1890E06D2713A575FE52E7C528757804D4B44CD47B1775E3074DE95`）。

ARM64 构建验证：
- `aria2c`: ELF 64-bit ARM aarch64, statically linked, no NEEDED, `aria2 version 1.36.0`，
  `built by aarch64-unknown-linux-musl`，特性列表与 x86_64 版一致。
- `aria2-manager`: ELF 64-bit aarch64, CGO_ENABLED=0 纯静态。
- `aria2-hook`: ELF 64-bit aarch64, CGO_ENABLED=0 纯静态。

**状态：实验性。** 未经任何 ARM 设备实机验证，仅供测试。待有 ARM fnOS NAS 后
需完成与 x86 版相同的验收流程。

## GitHub Release（2026-10-22）

`0.2.3` 已发布到 <https://github.com/vinson023/aria2-fnos/releases/tag/v0.2.3>，
tag `v0.2.3`。发布资产：

| 资产 | 大小 | SHA-256 |
|---|---|---|
| `aria2-0.2.3-x86_64.fpk` | 11,160,627 B | `531FFE273CE16D25FAEB116A70ED728655DED66F3369F8AC50FB04A7F31DBCE4` |
| `aria2-0.2.3-arm64.fpk` | 13,449,441 B | `D4940DC5E9536FB2ADEC6C5A12CAAF714E4A6AAD60DFAD025C9CBDF7D7B0DCBA` |
| `aria2-core-source-offer-0.2.3.tar.gz` | 8,364 B | `D89FB98C1AB8492A805B55C15003FA1936440A1DBDED1F9960470F985059715C` |

第三个资产是 GPL-3.0 对应源码提供材料（`ADR-008` 要求）：含上游 `aria2` `release-1.36.0`
固定地址与校验和、4 个补丁、静态构建脚本与复现步骤。归档内文件统一 LF 行尾，
`mtime` 固定，可重复生成同一哈希。

发布前重新构建了 x86 包（原 `aria2.fpk` 在 ARM64 构建时被 fnpack 覆盖并改名）。
重建后解包 49 个文件与 `build-output/fpk-0.2.3.inner.SHA256SUMS` **逐字节一致**，
仅外层归档元数据变化；两个架构包之间只有 3 个原生二进制不同，其余 46 个文件
（含 `0.2.3` 修正过的 AriaNg bundle，SHA-256 `7835FEF4…66CB1`）完全相同。
