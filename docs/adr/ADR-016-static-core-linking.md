# ADR-016：Core 改为静态链接以消除换机启动失败

- 状态：已采纳
- 日期：2026-09-05
- 关联：ADR-001（原生 FPK、不含 Docker）、ADR-003（静态 Go 管理器）、ADR-008（许可证与溯源）、ADR-015 第 11 条（Core 启动前预检）、`docs/phase3-build-record.md`

## 背景

用户报告 `0.1.93` 在部分机器上安装后「无法启动」。`ADR-015` 已解决第一类根因（配置与机器取值耦合导致的启动死锁）。
第二类根因是二进制依赖：入包的 `server/aria2c` 是动态链接的，除 glibc 外还需要
`libz`、`libexpat`、`libsqlite3`、`libssl3`、`libcrypto3`、`libssh2`、`libstdc++`、`libgcc_s`。
任一缺失时 `aria2c` 只往 `aria2.log` 留一行动态链接器报错就退出，而应用中心不收集该文件，
前台表现与配置类故障完全同形：一句没有信息的「无法启动」。

容器实测已复现：`debian:12-slim` 缺这五个库，`cmd/main` 稳定卡在就绪门禁上。
不同机器的系统库集合并不一致，因此这一类故障无法靠预检彻底避免，只能靠消除依赖。

## 决策

1. **Core 改为完全静态链接**（musl libc 加全部第三方依赖静态库），产物不含动态段，
   不再依赖宿主任何共享库。`ADR-015` 第 11 条的 `aria2c --version` 预检保留，
   但定位从「缓解手段」降为「安全网」，用于兜底 CPU 指令集与 ELF 架构不匹配等剩余可能。
2. **构建器固定为 `alpine:3.22`**（实测 3.22.5，镜像 ID
   `sha256:b66e0ce64844f5c6435b0c4bfd965558199ab0f53270846861c979cb1ac29365`，
   仓库摘要 `alpine@sha256:14358309a308569c32bdc37e2e0e9694be33a9d99e68afb0f5ff33cc1f695dce`，
   gcc 与 g++ 14.2.0）。Aria2 源码与补丁不变，仍是 `release-1.36.0`
   （`782B5DAD48CB333E0CBEF8795F868F7963455EFECFD2A818FEB8DAB1CFA973F5`）
   加同一组 4 个锁定补丁，因此两个 Core 之间唯一的差异就是链接方式。
   Docker 仅作临时构建环境，运行时的 FPK 不依赖 Docker（`ADR-001` 不变）。
3. **静态链接的唯一正确开关是环境变量 `ARIA2_STATIC=yes`**。上游 `configure.ac` 仅在它被设置时
   才改用 `PKG_CONFIG --static` 并追加 `-all-static`。单独传 `--enable-static-link` 或
   `LDFLAGS=-static` 会静默产出仍然动态链接的二进制（实测 `file` 报
   `dynamically linked, interpreter /lib/ld-musl-x86_64.so.1` 且带 9 条 `NEEDED`），
   这种产物在 glibc 设备上根本无法运行，却能在 Alpine 容器里自检通过。
   因此 `build-inputs/container-build-static.sh` 设三道硬门禁：configure 日志必须出现
   `Static build: yes`；`src/Makefile` 必须含 `-all-static`；产物必须通过
   `file` 与 `readelf -d` 的无 `NEEDED` 检查。
4. **Alpine 的 `*-dev` 包不含静态库**，必须额外显式安装
   `zlib-static`、`expat-static`、`sqlite-static`、`libssh2-static`、`openssl-libs-static`，
   否则 `--static` 的 pkg-config 查找会失败并退回动态链接。
5. **静态二进制与全部取证文件（`file`、`readelf`、`ldd`、`sha256`、`version`）随构建记录归档**，
   安装包与运行时一律不下载任何可执行代码。

## 备选方案与不采用原因

- 随包附带这些共享库并设置 `LD_LIBRARY_PATH`：需要改 `cmd/main` 的进程环境，
  且会把宿主 glibc 版本假设写死进包里；一旦某台机器 glibc 更低，仍然起不来，
  排查时又多一层可被覆盖的搜索路径。静态链接一步到位，不留这类隐患。
- 改用更旧基线的发行版动态构建：只降低版本号不匹配的概率，
  不消除库集合缺失的可能，且与现有构建基线分叉。
- 只保留预检、失败时报更清楚的错：诊断变好了，但用户仍然装不上，不解决问题。

## 许可证分析（对应 `ADR-008`）

静态链接会放大 GPL 与许可证义务，逐项核对如下（版本为 Alpine 3.22 的打包版本）：

| 组件 | 版本 | 许可证 | 静态链接结论 |
| --- | --- | --- | --- |
| musl libc | 1.2.5 | MIT | 允许静态链接，保留 MIT 声明即可 |
| OpenSSL | 3.5.0h | Apache-2.0 | 许可证第 4 条明确允许与任意程序链接后分发，无需额外授权 |
| zlib | 1.3.2 | zlib 许可证 | 允许静态链接 |
| expat | 2.8.4 | MIT | 允许静态链接 |
| SQLite | 3.49.2 | Public Domain | 无义务 |
| libssh2 | 1.11.1 | BSD-2 风格 | 允许静态链接，保留声明 |
| libstdc++ | 14.2.0 | GPLv3+ 含 Runtime Library Exception | 例外条款允许被任意程序链接而不施加 GPL；且本 Core 本身即 GPL-3.0，结论不受影响 |
| Aria2 | 1.36.0 | GPL-3.0 | 必须提供对应完整源码与 GPL 文本，见下 |

对照：改静态前的 Core 链接的是 OpenSSL 3.0.0t 系的动态库；本次构建器统一后
OpenSSL 升到 3.5.0h，属同一 Apache-2.0 体系，不改变结论。

GPL-3.0 对应源码义务：随包说明提供的仍是固定 SHA-256 的
`build-inputs/aria2-release-1.36.0.tar.gz` 加 `build-inputs/patch/` 的同版补丁，
以及 `build-inputs/container-build-static.sh` 这一可复现构建脚本，
满足「以固定 revision 的源码加明确构建方式」提供对应源码的要求。

## 后果

- 正面：`server/aria2c` 从 92,036,512 B 降到 10,739,352 B，FPK 从 46,003,943 B 降到 13,853,128 B；
  「缺库型无法启动」这一整类根因被消除；容器端到端测试里那种
  需要往容器塞库的 workaround 不再需要。
- 代价与边界：musl 与 glibc 行为差异需靠真机功能验收覆盖，尤其 DNS、IPv6、
  线程栈大小与证书路径；静态二进制不随宿主证书更新，但设备实测 HTTPS 下载与
  Tracker 自动刷新均正常，说明其证书查找走的是宿主的系统信任库，未被打进二进制。
- 剩余风险：CPU 指令集或 ELF 架构不匹配仍会导致失败，由 `ADR-015` 第 11 条预检
  与 `tests/fnos-diagnose-startup.sh` 的架构自检负责给出明确线索。

## 验收

### 缺库环境的 A/B 对照（决定性证据）

在 `debian:12-slim` 容器内（缺 `libexpat.so.1`、`libsqlite3.so.0`、`libssl.so.3`、
`libcrypto.so.3`、`libssh2.so.1` 共 5 项，`ldd` 报 5 条 `not found`）同时放入
`0.1.93` 的动态 Core 与 `0.1.95` 的静态 Core：

```text
=== 静态 0.1.95 ===
aria2 version 1.36.0

=== 动态 0.1.93 ===
error while loading shared libraries: libexpat.so.1: cannot open shared object file: No such file or directory
```

这一行正是过去那些机器上真实发生、却被 `aria2.log` 吞掉、前台只剩「无法启动」的报错。
同一容器里再用解包后的真实发布产物跑 `tests/startup-e2e-container.sh`，
8 组用例全部通过，其中 `startup.log` 打出「Aria2 Core 自检通过：aria2 version 1.36.0」。

### 静态性与功能门禁

- 静态性：`file` 报 `statically linked`；`readelf -d` 报 `There is no dynamic section in this file`；
  `ldd` 报 `not a dynamic executable`；`env -i ./aria2c --version` 在无 `LD_LIBRARY_PATH` 下成功。
  注意 fnOS 设备**没有** `readelf`、`objdump`、`nm`、`strings`，因此设备侧脚本以 `file` 为主判据、
  `ldd` 为兜底：早期版本的 `tests/fnos-diagnose-startup.sh` 用 `readelf -d` 判静态，
  命令不存在时输出为空，把 92 MB 的动态 Core 误报成「静态链接、不依赖宿主共享库」，
  已在设备上实测复现并修正。`tests/fnos-verify-static-core.sh` 的静态门禁同样加固，
  `readelf` 只在存在时作为加强项。
- 能力对齐：与现役动态 Core 逐项比对 `--version` 的 Enabled Features 与 Hash Algorithms，
  结果字节一致；完整 `--help` 逐行 diff 亦一致。注意 `--help` 不列出本项目的补丁选项，
  `--help=all` 是子串过滤（实测只命中 118 行），两者都不能用来证明补丁选项存在。
- 真机功能：`tests/fnos-verify-static-core.sh` 从现役 `/vol1/@appdata/aria2/aria2.conf`
  重建配置（携带 34 个选项，仅替换机器相关与生命周期相关行，去掉 RPC 密钥与回调），
  在临时端口上启动，断言：无 `Unknown option`、RPC `getVersion`、HTTP 真实下载、
  HTTPS 真实下载（证明静态 OpenSSL 证书链与异步 DNS 可用）、DHT 的 IPv4/IPv6 监听、
  BT 的 TCP IPv4/IPv6 监听、磁力任务提交与移除。全部通过。
  该脚本对**入包的那一份字节**（从 FPK 解包后上传的发货态产物）复跑，同样全绿。

### 真实 BT 数据面（不只是"能建监听"）

`tests/fnos-verify-bt-transfer.sh` 与夹具 `tests/btfixture/`（独立 Go module，测试专用、不入库发布包）
在一台机器上离线跑通端到端 BitTorrent：夹具为待传文件生成 `.torrent`（bencode 加 32 个分片 SHA-1），
并起一个最小 HTTP Tracker 交换对等体；种子端与下载端**都使用同一份待测 Core**，
各自独立目录与 BT 端口，传输 8 MiB 随机数据后逐字节比对 SHA-256。

```text
OK: BT 传输完成且分片校验一致：SHA-256 b571afb01866ef3c…
seed.log : Your share ratio was 1.0, uploaded/downloaded=8.0MiB/8.0MiB
leech.log: Your share ratio was 0.0, uploaded/downloaded=0B/8.0MiB
```

两端字节数互证，说明这 8 MiB 确实经过对等体协议传输，不是本地复制或种子端自检。
构建夹具时踩到两个上游行为，已在脚本注释里固化：
认领已存在的完整文件必须 `--check-integrity=true`（否则 aria2 为防止截断而直接取消）；
`seed-ratio` 与 `seed-time` 同时为 0 等于不做种，种子端会立刻打 `Seeding is over` 退出。
- 首启 `dht.dat` 缺失会让 aria2 打两条 ERROR：`Exception caught while loading DHT routing table from …`
  以及详情行 `Failed to load DHT routing table …`。已做 A/B 取证：同一份临时配置下
  现役动态 Core 与新静态 Core 都恰好出现 2 条，属上游首启行为、与静态链接无关，
  因此该门禁按 `DHT routing table` 关键字排除，其余异常一律判失败。
