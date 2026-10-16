# Phase 3 原生产物构建基线

## 目的

固定 Aria2 Pro Core、AriaNg 和构建输入，形成可审查、可复现的 x86 首发构建基线。

## 已固定的上游输入

| 组件 | 版本/标识 | 来源与证据 |
|---|---|---|
| Aria2 Pro Core | `1.36.0 build 2021.08.22` | `P3TERX/Aria2-Pro-Core`；本地快照 revision `4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062` |
| AriaNg | `1.3.14` | `mayswind/AriaNg` `package.json`；本地快照 revision `d6a765377e1eecfbcc387dcb824124df114decfb` |
| Aria2 配置/脚本参考 | 本地快照 | `reference-snapshot/upstream/aria2.conf`，仅用于审查，不直接作为运行时依赖 |

P3TERX Core 的公开发行页显示 `1.36.0 build 2021.08.22` 为当前最新正式 release；不能把“上游 aria2 新版本”自动等同于 Core 新版本。AriaNg 采用快照中已审查的 `1.3.14`，升级必须重新审计并更新哈希。

## 首发平台

- 目标：`x86_64` / fnOS x86。
- 测试机：Debian 12、glibc 2.36、fnOS 内核 `6.18.18.c952-trim`。
- Core 上游声明 amd64 最低 Linux 内核为 3.2；本地构建后仍需使用 `readelf`、`ldd` 和执行自检确认实际产物不引入动态运行时依赖。
- 不声明 `platform=all`；未来 ARM 必须独立构建、独立哈希和独立验收。

## 首个构建产物

已在测试机 Docker Debian 12 x86_64 环境完成 Core 构建：源码 `release-1.36.0` 与 4 个
补丁均通过校验/应用，`aria2c --version` 报告 `1.36.0`。产物 SHA-256 为
`EEFB98AFB22DD3A4B7BF70705F064704257542C85FAEF424D6273CF0CCB5729E`。

当前产物为动态链接 ELF，依赖 fnOS 提供的 zlib、expat、sqlite3、OpenSSL 3、libssh2、
libstdc++、libgcc_s、libm 和 glibc；`file`/`ldd` 检查已完成，测试机修正执行权限后
`aria2c --version` 成功。纳入 FPK 前必须完成目标 fnOS ABI 矩阵验证。

## 构建与验收门槛

1. 固定源码 revision、补丁集合和构建工具版本。
2. 生成 `aria2c` 后记录 SHA-256、文件类型、架构和依赖检查结果。
3. 生成 AriaNg 静态资源并记录源码 revision、许可证和资源哈希。
4. 提供 GPL-3.0/MIT 许可证、第三方清单和对应源码获取方式。
5. 仅在本地构建与校验通过后，才允许使用 PuTTY 连接测试机进行 dry-run。

## 当前状态

- 版本和源码 revision：已固定。
- Core/AriaNg 可执行运行资产：尚未构建或纳入 FPK。
- Core 与 AriaNg 静态运行资产：已构建，尚未纳入 FPK。
- SHA-256、许可证清单和构建记录：已生成，见 `docs/phase3-build-record.md`。

补充环境结论：测试机存在 Docker 客户端但 daemon 未运行；项目仅允许临时构建隔离，不允许作为最终运行时依赖。宿主编译器缺失且 sudo 安装未获授权，Phase 3 仍等待可用 Linux 工具链。

## 本地构建前置检查（2026-08-27）

- Windows 主机可用 `wsl.exe`，但当前未安装可用的 WSL Linux 发行版；`wsl --status`/`wsl -l -q` 仅返回安装提示。
- 当前快照只包含 Core 的构建脚本、补丁和许可证，不包含 Aria2 主源码，因此不能在没有额外源码输入的情况下生成 `aria2c`。
- 本机未发现 `gcc`、`make`、`go`、`readelf` 或 `ldd`；不能把 Windows 工具链结果当作 fnOS 兼容性证据。
- 结论：Phase 3 版本基线已完成，但产物构建被 Linux 构建环境和固定 Aria2 源码输入阻塞。下一步应准备审查过的源码 tarball/commit 与隔离 Linux 构建环境，再执行构建和哈希记录。

2026-08-27 已补充并校验 Aria2 `release-1.36.0` 源码 tarball：SHA-256 为
`782B5DAD48CB333E0CBEF8795F868F7963455EFECFD2A818FEB8DAB1CFA973F5`，并通过 PuTTY 上传
至测试机 `/tmp` 后再次校验。测试机缺少完整编译工具链；尝试使用 `sudo` 安装依赖时，
设备 sudo 密码与 SSH 密码不一致，安装未执行成功。

## 离线输入审计（2026-08-27）

本轮未下载或执行远程内容。已对现有离线快照完成输入清点：

- `Aria2-Pro-Core` revision：`4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062`。
- `AriaNg` revision：`d6a765377e1eecfbcc387dcb824124df114decfb`。
- `Aria2-Pro-Docker` revision：`4ec58da2901947f57e2e53f30077723627c8eaeb`。
- Core `LICENSE` SHA-256：`230184F60BAE2FEAF244F10A8BAC053C8FF33A183BCC365B4D8B876D2B7F4809`。
- AriaNg `LICENSE` SHA-256：`0ACB83664DEEA9703E04C8C8F38BC0650A149B2716E3DEFDD30948F4CD5C69B6`。
- Docker 参考 `LICENSE` SHA-256：`7CF9A66F98C3EB33E2ECE45E5DA00F5776BDBC56CB8F8ED5753CF5842963F902`。
- AriaNg `package.json` SHA-256：`D96EB930F77A13A356962B760638D58175DA77CB746B2BE97C54B14E6F38282B`。

Core 快照中未发现 Aria2 主源码 tarball 或解压后的源码树；该事实与当前构建阻塞一致。

上游 Core 脚本还会在构建时拉取 `aria2/aria2` 的 `master` 分支和多项网络依赖，不能原样
用于发布构建。已新增 `docs/phase3-build-input-lock.md`，明确离线输入、哈希和禁网构建
门槛。
