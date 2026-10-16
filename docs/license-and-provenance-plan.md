# 许可证与产物溯源计划

本文件是第一版发布前的合规设计记录，不构成法律意见。实际分发前应依据最终纳入的
文件、构建方式和发布渠道进行复核；特别是 GPL 组件的修改、链接方式和对应源码范围。

## 已核对的离线基线

| 资产或资料 | 固定位置 | 当前用途 | 许可证 | 第一版处理 |
| --- | --- | --- | --- | --- |
| Aria2 Pro Core | `upstream/Aria2-Pro-Core`，revision `4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062` | 目标架构的 `aria2c` Core 来源 | GPL-3.0 | 随 FPK 二进制分发时附 GPL 文本、对应源码获取方式和精确构建记录 |
| AriaNg | `upstream/AriaNg`，revision `d6a765377e1eecfbcc387dcb824124df114decfb` | Web 静态界面来源 | MIT | 分发静态资源时保留 MIT 许可证和版权声明 |
| P3TERX Aria2 Pro Docker | `upstream/Aria2-Pro-Docker`，revision `4ec58da2901947f57e2e53f30077723627c8eaeb` | 行为与默认参数参考 | MIT | 未直接分发时仅记为行为参考；若复制任何文件，纳入许可证清单 |
| P3TERX aria2.conf | `upstream/aria2.conf`，revision `02b9d95ea155e66f7e3c4340cd22717f8bc7401c` | 配置、Tracker 与回调行为参考 | MIT | 未直接分发时仅记为行为参考；若改编/复制脚本，保留 MIT 声明 |
| `@trimjs/web-app` | `npm/trimjs-web-app/package-0.4.2` | 飞牛页面宿主交互 SDK | 当前快照未发现 `license` 字段或 `LICENSE` 文件 | 实际打包前从权威发布来源核对许可证和传递依赖，再列入发布清单 |
| 自研 Go 管理器与 FPK 脚本 | Phase 2 后产生 | 服务、网关和生命周期 | 待项目发布策略确认 | 在首次分发前固定项目许可证与版权声明 |

“当前用途”为设计结论，不表示这些文件已经被复制进应用包。`reference-snapshot/`
仅作分析证据，不能整体打包为运行时依赖。

## 每个架构产物必须记录的字段

1. 应用版本、`appname`、FPK 文件名和目标 CPU 架构。
2. Aria2 Core 的来源 URL、固定 revision、源码包 SHA-256、二进制 SHA-256。
3. Core 构建脚本/命令、构建器镜像或工具链版本、构建日期和可复现构建说明。
4. 管理器、AriaNg、配置模板、回调文件和第三方 SDK 的来源、revision/版本、SHA-256、许可证。
5. 随包包含的 `LICENSES/` 内容，以及用户可读的第三方声明页。
6. GPL Core 对应源码的同版归档位置或稳定下载地址，及其完整性哈希。

## 发布边界

- 不以“上游 latest”“默认分支”或可变 CDN URL 作为某一 FPK 的唯一溯源。
- 不在安装或启动时下载 Core、AriaNg、脚本、许可证或构建依赖来补全发行包。
- P3TERX 的 Docker 启动逻辑、远程脚本和 Rclone 安装流程不会因参考其源码而自动进入产物。
- 发现新第三方文件、重新生成 AriaNg 资源或变更 Core 构建参数时，必须刷新该架构的产物清单并重新审查许可证。

## 静态链接的额外义务（`0.1.95` 起）

Core 自 `0.1.95` 起改为完全静态链接（musl libc 与全部第三方依赖的静态库），静态链接会把
「链接方式」相关的许可证义务从可分项变成不可分项，因此单独记录。逐组件结论、版本号与
构建器镜像见 `docs/adr/ADR-016-static-core-linking.md`，要点：

- musl (MIT)、OpenSSL 3.5 (Apache-2.0)、zlib、expat (MIT)、SQLite (Public Domain)、
  libssh2 (BSD-2 风格) 均允许静态链接后分发。
- libstdc++ 为 GPLv3 加 Runtime Library Exception，例外条款允许被任意程序链接而不施加 GPL；
  且本 Core 自身即 GPL-3.0，不存在额外的传染问题。
- 因此静态链接**没有**引入比动态链接更重的新义务，也没有改变 Aria2 的 GPL 源码提供义务范围。
- 每次更换构建器镜像或依赖版本，必须重跑这张表并把新版本记入 `docs/phase3-build-record.md`。

对应源码提供方式保持固定：`build-inputs/aria2-release-1.36.0.tar.gz`（SHA-256
`782B5DAD48CB333E0CBEF8795F868F7963455EFECFD2A818FEB8DAB1CFA973F5`）
加 `build-inputs/patch/` 的同版补丁，加 `build-inputs/container-build-static.sh`
（SHA-256 `6534bf6c6512323298497b012f5e71a231b013bd12ca02b1bf5d8a5f174596a1`）作为可复现构建说明。

## 阶段职责

| 阶段 | 必须完成的溯源工作 |
| --- | --- |
| Phase 1 | 固定来源、许可证结论和发布证据模型；不下载或打包资产 |
| Phase 3 | 建立架构产物清单、哈希、构建记录、许可证目录和 Core 对应源码归档 |
| Phase 5 | 核查 AriaNg 与 SDK 实际打包内容、许可证和源码/构建信息 |
| Phase 7 | 将 FPK、物料清单、许可证、对应源码获取方式和发布说明作为同版验收物 |

## 参考依据

- `reference-snapshot/upstream/Aria2-Pro-Core/LICENSE`
- `reference-snapshot/upstream/AriaNg/LICENSE`
- `reference-snapshot/upstream/Aria2-Pro-Docker/LICENSE`
- `reference-snapshot/upstream/aria2.conf/LICENSE`
- `docs/reference-baseline.md`
- `docs/adr/ADR-005-pinned-offline-assets.md`
- `docs/adr/ADR-008-license-provenance-and-source-offer.md`
