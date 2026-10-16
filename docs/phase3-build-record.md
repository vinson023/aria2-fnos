# Phase 3 构建记录

## Core（静态链接，`0.1.95` 起为发布产物）

- 目标：fnOS x86_64，完全静态链接，不依赖宿主任何共享库（决策见 `docs/adr/ADR-016-static-core-linking.md`）。
- Aria2 源码：`release-1.36.0`，归档 `build-inputs/aria2-release-1.36.0.tar.gz`，
  SHA-256 `782B5DAD48CB333E0CBEF8795F868F7963455EFECFD2A818FEB8DAB1CFA973F5`。
- Core 补丁：`docs/phase3-build-input-lock.md` 锁定的同一组 4 个补丁，在 `--fuzz=3` 下干净应用；
  设备侧补丁文件需先 `tr -d "\r"` 归一为 LF，否则 `patch` 会因行尾差异失败。
- 构建器：测试机 Docker 的 `alpine:3.22` 临时容器（3.22.5，镜像 ID
  `sha256:b66e0ce64844f5c6435b0c4bfd965558199ab0f53270846861c979cb1ac29365`，
  仓库摘要 `alpine@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce`，
  gcc 与 g++ 14.2.0）。Docker 未纳入 FPK 运行时。
- 构建脚本：`build-inputs/container-build-static.sh`（实测约 9 分钟）。
- 关键开关：环境变量 `ARIA2_STATIC=yes`。这是上游唯一的静态链接开关，它同时让 `pkg-config`
  走 `--static`（带出 `Libs.private`）并给最终链接加 `-all-static`；只传 `--enable-static-link`
  或 `LDFLAGS=-static` 会静默产出仍然是动态链接的二进制。
- 必需静态归档：`zlib-static`、`expat-static`、`sqlite-static`、`libssh2-static`、
  `openssl-libs-static`；Alpine 的 `*-dev` 包不含 `.a`，只装 `-dev` 会退回动态链接。
- 配置参数：`--enable-static --disable-shared --with-openssl --without-gnutls --disable-nls
  --disable-dependency-tracking --without-libuv --without-libgmp`，
  `CFLAGS/CXXFLAGS="-O2 -pipe"`，安装后 `strip`。
- 静态库版本：zlib 1.3.2、expat 2.8.4、SQLite 3.49.2、OpenSSL 3.5.0h、libssh2 1.11.1、musl 1.2.5。
- 产物：`build-output/aria2c`，入包路径 `server/aria2c`。
- 产物 SHA-256：`A482FB24722B87A0A719ACEE330E329D6DFA21075808C66202E451782812015B`。
- 产物大小：10,739,352 B（对照：改静态前 92,036,512 B）。
- 取证文件：`build-output/aria2c.file`、`aria2c.readelf`、`aria2c.ldd`、`aria2c.sha256`、`aria2c.version`。
- 链接形态：`file` 报 `ELF 64-bit LSB executable, x86-64, version 1 (SYSV), statically linked, stripped`；
  `readelf -d` 报 `There is no dynamic section in this file`；`ldd` 报 `not a dynamic executable`；
  `env -i ./aria2c --version` 在没有 `LD_LIBRARY_PATH` 的环境下成功。
- 真机验收：在 glibc 2.36（Debian 12 基线）的 fnOS 测试机上，用现役 `aria2.conf` 的 34 个选项启动，
  `tests/fnos-verify-static-core.sh` 全绿：HTTP 真实下载、HTTPS 真实下载（拉取 Tracker 列表）、
  DHT 与 BT 的 IPv4/IPv6 监听、磁力任务提交与移除，且无 `Unknown option`、无异常日志。
  该脚本对从 FPK 解包出来的发货态产物复跑，结果一致。
- 能力对齐：与退役的动态 Core 比对 `--version` 的 Enabled Features 与 Hash Algorithms 字节一致，
  完整 `--help` 逐行 diff 一致（各 285 行、30 个选项行）。注意 `--help` 不含本项目的补丁选项，
  `--help=all` 是子串过滤（实测只命中 118 行），二者都不能用于证明补丁选项存在。

## Core 真实 BT 数据面验收

`tests/fnos-verify-bt-transfer.sh` 配夹具 `tests/btfixture/`（独立 Go module，仅测试用、不入发布包）
在一台机器上离线跑通端到端 BitTorrent：夹具生成 `.torrent`（bencode 加 32 个分片 SHA-1）并起一个
最小 HTTP Tracker 交换对等体；种子端与下载端使用同一份待测 Core、各自独立目录与 BT 端口，
传 8 MiB 随机数据后逐字节比对 SHA-256。结果：

```text
OK: BT 传输完成且分片校验一致：SHA-256 b571afb01866ef3c…
seed.log : Your share ratio was 1.0, uploaded/downloaded=8.0MiB/8.0MiB
leech.log: Your share ratio was 0.0, uploaded/downloaded=0B/8.0MiB
```

两端字节互证，说明数据确实经过对等体协议传输，而不是本地复制或种子端自检。
两个上游行为值得记录：认领已存在的完整文件必须 `--check-integrity=true`，
否则 aria2 为防止截断而直接取消；`seed-ratio` 与 `seed-time` 同时为 0 等于不做种，
种子端会立刻打 `Seeding is over` 退出。

## 设备工具可用性事实（影响所有设备侧脚本）

fnOS 设备上没有 `readelf`、`objdump`、`nm`、`strings`，只有 `file`、`ldd` 和 `eu-readelf`。
任何以 `readelf` 为唯一判据的检查在设备上都会静默空转：早期版本的
`tests/fnos-diagnose-startup.sh` 就因此把 92 MB 的动态 Core 报成「静态链接、不依赖宿主共享库」。
现已改为以 `file` 为主判据、`ldd` 为兜底，并在无法判定时显式报错而不是给结论；
`tests/fnos-verify-static-core.sh` 的静态性门禁同样加固，`readelf` 仅在其存在时作为加强项。

## AriaNg

- 源码：`mayswind/AriaNg` revision `d6a765377e1eecfbcc387dcb824124df114decfb`，版本 `1.3.14`。
- 固定源码归档：`build-inputs/ariang-1.3.14-source.tar.gz`。
- 源码 SHA-256：`83D8651490FDF225D635461FF3090D7CC1CDDB321EB02D807D2220037FB7F389`。
- 构建器：本机 Node.js `24.19.0` / npm `11.18.0`，仅用于生成静态资源。
- 依赖安装：在 `build-work/AriaNg` 副本中以 `npm install --ignore-scripts --legacy-peer-deps` 执行；旧 lockfile 不与现代 npm 的 `ci` 规则兼容，构建工作副本不进入 FPK。
- 构建命令：`npm run build`，成功生成 32 个静态文件。
- 产物目录：`build-output/ariang`。
- 文件哈希清单：`build-output/ariang.SHA256SUMS`。2026-09-01 清理历史 `aria-ng-*.min.js`
  后只保留当前入口引用的 `js/aria-ng-2b80650589.min.js`，清单重新生成并在同步目录后校验；
  当前清单 SHA-256 为 `62B22D5B07024A4A7E13E932BD08F172179F1323F1795E62C3E4CC509129E2DD`。

## 许可证

- Core GPL-3.0 文本：`build-output/LICENSES/Aria2-Pro-Core-GPL-3.0.txt`。
- AriaNg MIT 文本：`build-output/LICENSES/AriaNg-MIT.txt`。
- Trim Web App SDK MIT 文本：`build-output/LICENSES/Trim-Web-App-MIT.txt`。
- 两份文本与离线快照中的许可证文件哈希一致。
- 静态链接引入的 musl、OpenSSL、zlib、expat、SQLite、libssh2、libstdc++ 义务逐项分析见
  `ADR-016` 的「许可证分析」一节；对应源码获取方式仍按 `ADR-008` 执行。

## 后续门槛

- Core 自包含性已在 glibc 真机上验收，`0.1.95` 的 FPK 不再需要宿主共享库或 `LD_LIBRARY_PATH`。
- 仍需覆盖：其它 CPU 架构（如 aarch64）需按架构独立构建静态 Core，禁止声明 `platform=all`；
  以及构建器镜像升级后的重跑与逐字节复核。
