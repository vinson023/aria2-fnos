# 飞牛版 Aria2

原生飞牛 fnOS FPK：集成 Aria2 Core、Go 管理器和 AriaNg。当前 x86_64 候选包为
`0.2.3`，运行时不依赖 Docker、Node.js、Python 或数据库，也不再依赖宿主的任何共享库。
<img width="1457" height="834" alt="image" src="https://github.com/user-attachments/assets/68ca58a0-fc10-49b2-8f98-5411c29b79de" />


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

最新候选包为 `0.2.3`（x86_64），相对 `0.2.0` 的三处界面/文案修复：归档目录字段从 AriaNg
基本设置页移除（`0.2.2`）、归档目录创建失败时降级而非阻塞保存（`0.2.1`）、连接提示与安装
向导文案统一为「Aria2」（`0.2.3`）。完整版本历史、产物指纹与逐文件溯源清单见
[候选发布物](docs/release-candidate.md)。
