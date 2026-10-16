# 目录结构与职责

本目录结构是实现约定。Phase 2 已创建 `fpk/` 空包骨架；其余业务源码目录按开发阶段逐步创建。

```text
aria2-pro/
├── AGENTS.md                         # 协作规范
├── PROJECT_CONTEXT.md                # 当前目标、约束和确认项
├── README.md                         # 项目入口说明
├── docs/
│   ├── requirements.md               # 功能、风险和非目标
│   ├── technology-stack.md           # 技术栈及选型依据
│   ├── architecture.md               # 组件、数据和网络边界
│   ├── directory-structure.md        # 目录职责
│   ├── development-plan.md           # 分阶段开发计划
│   └── adr/                          # 重要架构决策
├── fpk/                              # FPK 源目录（Phase 2 已创建空骨架）
│   ├── app/
│   │   ├── bin/                      # aria2c、manager 等运行二进制（后续阶段）
│   │   ├── config/                   # 固定模板和默认配置
│   │   └── ui/                       # AriaNg、适配层、入口配置
│   ├── cmd/                          # install/start/stop/upgrade 脚本
│   ├── config/                       # privilege、resource
│   ├── wizard/                       # install/config 向导
│   ├── manifest
│   ├── ICON.PNG
│   └── ICON_256.PNG
├── manager/                          # Go 管理器源码（Phase 5 创建）
├── ui-adapter/                       # 飞牛 Web SDK 适配层源码（Phase 5 创建）
├── scripts/                          # 构建、校验和发布辅助脚本
├── tests/                            # 单元、集成和设备验收测试
├── tools/                            # 本地 fnpack、trim-cli 等开发工具
└── reference-snapshot/               # 官方文档和上游源码离线快照
```

## FPK 内部运行目录

安装后的飞牛目录由系统管理，应用代码只能使用环境变量定位：

- `$TRIM_APPDEST`：只读的 target 运行文件和 Unix Socket。
- `$TRIM_PKGETC`：`aria2.conf`、`script.conf`、RPC 密钥和用户配置。
- `$TRIM_PKGVAR`：session、DHT 数据、PID、日志和运行时临时状态。
- `$TRIM_DATA_SHARE_PATHS`：资源声明创建的数据共享目录。

## 文件命名原则

- 应用标识、网关前缀和入口 ID 在发布后保持稳定。
- 配置字段使用稳定的 `snake_case` 名称，例如 `rpc_port`、`listen_port`。
- 版本化迁移脚本只处理已知旧版本，失败时不得覆盖原配置。
- 秘密、会话和日志文件不得放入 `app/` 静态资源目录。
