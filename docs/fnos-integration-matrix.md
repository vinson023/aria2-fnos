# 飞牛集成能力矩阵

本文件明确 Aria2 Pro 与飞牛 fnOS 各能力的职责边界，避免在实现阶段重复造轮子或错误使用开放 API。

## 能力分配

| 能力 | 首选机制 | 负责组件 | 是否首版必需 | 说明 |
| --- | --- | --- | --- | --- |
| 应用识别与版本 | `manifest` | FPK 外壳 | 是 | 固定 `appname`、版本、架构、网关入口和运行策略 |
| 应用专属用户 | `config/privilege` | FPK 外壳/fnOS | 是 | 生命周期钩子以 root 初始化受限目录；`cmd/main` 使用 `start-stop-daemon --chuid aria2` 启动常驻服务，服务不以 root 运行 |
| 下载共享目录 | `config/resource` 的 `data-share` | FPK 外壳/fnOS | D-003 已确认 | 由 fnOS 创建并授权，路径通过 `TRIM_DATA_SHARE_PATHS` 获取 |
| 安装/配置输入 | `wizard/install`、`wizard/config` | FPK 外壳/fnOS | 是 | 端口、模式等字段需在生命周期脚本中再次校验 |
| 桌面打开入口 | `app/ui/config` | FPK 外壳/fnOS | 是 | 使用 `type=iframe`，统一网关前缀为 `/app/aria2` |
| 登录态校验 | 统一网关 | fnOS | 是 | 网关先校验会话，再通过 Unix Socket 转发 |
| 当前用户身份 | `X-Trim-*` Header | 管理器 | 是 | 只信任网关转发的身份上下文，不信任浏览器自带 Header |
| Aria2 RPC | 本机 HTTP JSON-RPC | 管理器 + aria2c | 是 | 管理器注入 secret；AriaNg 固定访问相对 `/app/aria2/jsonrpc`；aria2 仅监听回环地址 |
| 文件选择/授权 | `@trimjs/web-app` | UI 适配层 | D-003 已确认，主体已实现 | 通过固定 SDK 选择并提交目录候选值；管理器重新校验，不直接信任浏览器结果；取消/拒绝回调待设备补证 |
| 打开文件管理器 | `TrimApp.openFileManager` | UI 适配层 | 否 | 仅用于辅助用户定位下载和归档目录 |
| 应用设置跳转 | `TrimApp.openAppSetting` | UI 适配层 | 否 | 配置入口由飞牛 App Center 提供 |
| 主题/语言 | `getPlatformConfig`、`$on` | UI 适配层 | 否 | 适配 AriaNg 外层，不改变下载核心 |
| 系统文件操作 API | 飞牛开放 API | 管理器/适配层 | 暂不调用 | 首版尽量使用应用资源、授权和本地文件操作；只有明确需求才接入 |
| 用户/权限管理 API | 飞牛开放 API | 管理器 | 暂不调用 | 第一版不自建用户体系，不修改飞牛用户权限 |
| 任务下载 API | Aria2 RPC | 管理器/AriaNg | 是 | 不使用飞牛 Download Center API 代替 Aria2 |

## 网关入口约定

```text
/app/aria2                    页面和静态资源
/app/aria2/jsonrpc            Aria2 HTTP JSON-RPC 代理
/app/aria2/healthz            管理器健康检查
/app/aria2/api/v1/config      管理员配置接口
/app/aria2/api/v1/trackers/refresh  管理员 Tracker 刷新接口
/app/aria2/ws                 预留的 WebSocket 路径
```

实际实现时，所有路径必须以稳定网关前缀为根；AriaNg 使用相对路径，不能把
`127.0.0.1:6800`、可变 RPC 端口或 secret 写入前端配置。

## 配置字段边界

| 字段 | 来源 | 保存位置 | 修改影响 |
| --- | --- | --- | --- |
| `rpc_port` | 安装/配置向导 | `$TRIM_PKGETC` | 重启管理器与 aria2，管理器仍代理同一网关路径 |
| `listen_port` | 安装/配置向导 | `$TRIM_PKGETC` | 重启 aria2，变更 TCP/UDP BT/DHT 监听 |
| `rpc_secret` | 安装脚本生成 | `$TRIM_PKGETC` | 不提供普通 UI 编辑，必要时走显式重置流程 |
| `download_dir` | 共享目录或授权流程 | `$TRIM_PKGETC` | 修改前校验授权，可能需要迁移或重启 |
| `archive_mode` | 配置向导 | `$TRIM_PKGETC` | 只控制完成回调，不改变 Aria2 核心协议 |

## 不应混用的能力

- 不用 CGI 代替统一网关常驻服务。
- 不用 Docker Compose 资源声明包装原生 aria2c。
- 不用飞牛下载中心任务接口替代 Aria2 会话和 RPC。
- 不用 SDK 的目录选择结果绕过管理器的授权根校验。
- 不把飞牛 API token 或网关 Header 直接暴露给 AriaNg。
