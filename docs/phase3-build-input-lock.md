# Phase 3 构建输入锁定清单

## 结论

当前 `Aria2-Pro-Core` 上游构建脚本不能直接用于本项目的发布构建。其默认流程会在构建时
拉取 `aria2/aria2` 的 `master` 分支，并下载未固定的第三方依赖；这与 ADR-005 的固定、
可审查离线资产要求冲突。

因此，本项目只复用其补丁集合和构建意图；正式构建必须先准备下列已审查的离线输入，并将
每个文件的来源 URL、版本/revision、SHA-256、许可证和获取日期记录在本清单中。

## 已锁定

| 输入 | 标识 | SHA-256/证据 |
|---|---|---|
| Aria2 Pro Core 脚本与补丁 | tag `1.36.0_2021.08.22`，revision `4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062` | 本地离线快照；GPL-3.0 |
| Core 补丁集合 | `patch/*.patch`，随上述 revision | 构建时必须逐个记录补丁哈希 |
| AriaNg 源码 | revision `d6a765377e1eecfbcc387dcb824124df114decfb`，版本 `1.3.14` | 本地离线快照；MIT |

### Core 补丁哈希

| 补丁 | SHA-256 |
|---|---|
| `0001-options-unlock-connection-per-server-limit.patch` | `85C365921A88FC311D1B6D770FAF356CB0C3004AA0DD7AEFAFDAD5AD3935409F` |
| `0002-download-retry-on-slow-speed-and-reset.patch` | `5ADD11177EDD265059F7EB20061777D956D35A82C2F1034515DC5E6B019827B8` |
| `0003-option-add-option-to-retry-on-http-4xx.patch` | `04FECE3E451F2EA6923CD342BE79C35F215ED591D60C62615CFB5424EA71CF96` |
| `0004-option-add-option-to-disable-want-digest-header.patch` | `9A822528F72DCAF98AD88B4B759A24A19F977EF70A9A7C49EEA5194E697FCFB1` |

构建前必须验证这四个补丁与表中哈希一致，并先在锁定的 Aria2 主源码上执行
`git apply --check`。校验失败时不得继续编译或生成发布产物。

## 必须补充后才能构建

| 输入 | 当前上游脚本行为 | 项目要求 |
|---|---|---|
| Aria2 主源码 | 克隆 `master` | 固定 tag/commit 的源码 tarball 或 Git bundle，离线校验 SHA-256 |
| zlib | `1.2.11` URL 下载 | 离线文件、SHA-256、许可证 |
| Expat | `2.4.1` URL 下载 | 离线文件、SHA-256、许可证 |
| c-ares | `1.17.2` URL 下载 | 离线文件、SHA-256、许可证 |
| OpenSSL | `1.1.1k` URL 下载 | 离线文件、SHA-256、许可证与安全复核 |
| SQLite | `3360000` URL 下载 | 离线文件、SHA-256、许可证 |
| libssh2 | `1.9.0` URL 下载 | 离线文件、SHA-256、许可证 |

## 构建执行规则

1. 构建脚本不得联网；禁用 `git pull`、`git clone`、`curl` 和包管理器在线安装路径。
2. 所有源码包在解压、编译前通过 SHA-256 校验。
3. 记录 Linux 发行版、编译器、链接器、`configure` 参数和所有补丁哈希。
4. 产物需完成 `file`、`readelf -d`、`ldd` 和 `aria2c --version` 验证后，才可进入 FPK。
