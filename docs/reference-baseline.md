# 参考资料基线

本文件固定阶段一使用的规范资料和上游行为资料。它们的用途不同：飞牛官方文档用于判断 FPK、统一网关、权限和 API 的正确用法；上游项目用于核对 Aria2 Pro 的功能行为；npm SDK 用于飞牛宿主交互设计。

## 快照时间

- 抓取日期：2026-08-27。
- 飞牛官方页面：38 个页面，原始 HTML 保存在 `reference-snapshot/fnnas-portal/raw/`。
- P3TERX Aria2 Pro 介绍页：1 个页面，保存在 `reference-snapshot/p3terx/`。
- URL 映射：`reference-snapshot/fnnas-portal/manifest.json`，共 39 条记录（38 条官方页面和 1 条 P3TERX 页面）。

## 官方飞牛资料

| 本地资料 | 用途 |
| --- | --- |
| `docs/core-concepts/framework` | FPK 安装目录、生命周期脚本和运行状态约定 |
| `docs/core-concepts/manifest` | `appname`、架构、网关入口、端口和依赖字段 |
| `docs/core-concepts/privilege` | 专属应用用户、附加用户组和 root 风险 |
| `docs/core-concepts/resource` | 数据共享目录和资源声明 |
| `docs/core-concepts/gateway-registration` | 统一网关、Unix Socket、用户 Header 和 WebSocket |
| `docs/core-concepts/app-entry` | 桌面 iframe、入口配置和用户可见性 |
| `docs/core-concepts/wizard` | 安装/配置向导和环境变量 |
| `docs/core-concepts/environment-variables` | `TRIM_*` 路径、用户、端口和系统变量 |
| `docs/core-concepts/runtime` | 可选运行时依赖及其声明方式 |
| `docs/examples/native` | 原生应用、Node 服务和网关 Socket 的完整示例 |
| `docs/cli/fnpack` | fnpack 创建、构建和基础校验 |
| `docs/quick-started/test-application` | 测试设备上的应用验证流程 |
| `api/calling`、`api/authorization/*`、`api/page/*` | 后续需要系统 API、授权和页面集成时的协议依据 |

## 上游行为资料

| 项目 | 本地路径 | 固定修订 |
| --- | --- | --- |
| Aria2 Pro Core | `reference-snapshot/upstream/Aria2-Pro-Core` | `4542bf85a0ce5a9ebbc9a86cfc8018b8d6608062` |
| Aria2 Pro Docker | `reference-snapshot/upstream/Aria2-Pro-Docker` | `4ec58da2901947f57e2e53f30077723627c8eaeb` |
| Aria2 配置与回调脚本 | `reference-snapshot/upstream/aria2.conf` | `02b9d95ea155e66f7e3c4340cd22717f8bc7401c` |
| AriaNg | `reference-snapshot/upstream/AriaNg` | `d6a765377e1eecfbcc387dcb824124df114decfb` |

Docker 项目和配置仓库只用于核对行为，不意味着 FPK 要复制 Docker 的启动方式。尤其是运行时远程下载配置、公开 RPC 和自动安装 Rclone，都必须经过本项目安全边界重新设计。

## 飞牛 Web SDK

- 包名：`@trimjs/web-app`。
- 固定版本：`0.4.2`。
- 本地包：`reference-snapshot/npm/trimjs-web-app/trimjs-web-app-0.4.2.tgz`。
- SHA-512：`6a956468e0883e30bff2138172bd8589a6211864309dff30c594eb1839c0f28eb8b7e9e57cba720bc4efd73c93bf1f596c09ecf119e4489d2465c1d12134ce90`。

SDK 只作为 UI 宿主适配工具，不能替代 FPK 生命周期、Aria2 RPC 代理或服务端鉴权。

## 使用规则

1. 规范性问题优先查飞牛官方离线页面。
2. Aria2 Pro 行为优先查固定修订的上游源码和配置脚本。
3. 当前设计与上游 Docker 行为冲突时，以本项目安全基线和飞牛应用约束为准，并新增或更新 ADR 说明原因。
4. 外部资料发生变化时，不直接覆盖旧快照；应重新抓取到带日期的新目录并更新本文件。
