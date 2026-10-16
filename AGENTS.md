# 项目协作规范

本仓库用于构建原生飞牛 fnOS FPK 应用「Aria2 Pro」。产品由架构相关的 Aria2 Pro Core 二进制、飞牛适配的管理/网关进程和 AriaNg Web 界面组成，运行时不依赖 Docker。

## 工作规则

- 实现前阅读 `PROJECT_CONTEXT.md` 和 `docs/adr/` 中相关决策。
- 第一版运行环境尽量自包含，不得仅为便利引入 Node.js、Python、Docker 或数据库运行时。
- 使用 `TRIM_APPDEST`、`TRIM_PKGETC`、`TRIM_PKGVAR` 等变量，禁止硬编码 `/var/apps/...` 或卷路径。
- 常驻进程使用飞牛专属应用用户运行，禁止以 root 运行服务。
- Aria2 RPC 只绑定回环地址；公开入口走飞牛统一网关并校验用户上下文。
- 所有路径、RPC 数据和配置输入均视为不可信，文件操作前必须校验和规范化。
- 保留任务恢复、清理回调、Tracker 更新和权限策略，不照搬公开 RPC 等不安全默认配置。
- 安装和启动时不得下载可执行代码、配置或依赖；发布包必须包含审查过且版本固定的内容。
- 重要架构选择必须记录在 `docs/adr/`。

## 打包与验证

- 使用 `tools/fnpack.exe` 创建和校验 FPK。
- 包含原生二进制的应用必须按架构独立打包，禁止声明 `platform=all`。
- 本地构建成功后才可在授权测试机验证，优先执行只读检查和安装 dry-run。
- 测试机应用的卸载与安装由用户手动在飞牛应用中心执行；开发侧只负责产出并校验 FPK、
  在安装完成后做设备侧检查、真实下载验收和截图，不自行调用卸载或安装流程。
- 禁止在日志中输出 RPC 密钥、认证 token 或用户凭据。

## 验证安全红线

- `tests/*-container.sh` 与任何带 `rm -rf` 的测试脚本**只能在容器内执行**。宿主机上直接执行会
  毁掉真实数据卷，已经造成过一次事故。禁止把这类脚本单独 `bash` 跑，也不允许先跑脚本再补
  `docker run`；唯一形式是同一条命令里的 `docker run ... bash /hosttmp/<脚本>`。
- 通过 `plink` 组装远程命令时，若命令里出现 `docker run`，必须确认待执行脚本本身也在这个
  `docker run` 之内；同一次调用里混用「宿主机命令」与「容器命令」时，容器前缀逐条写全，
  不允许依赖上一条命令的写法。
- 一次性脚本里不要出现 `rm -rf` 绝对路径。需要清理时改用 `mktemp -d` 出来的目录，
  或改用只在容器内成立的相对布局变量。

## 参考资料

- 离线资料：`reference-snapshot/`
- 官方页面映射：`reference-snapshot/fnnas-portal/manifest.json`
- 上游源码：`reference-snapshot/upstream/`
- 飞牛 Web SDK：`reference-snapshot/npm/trimjs-web-app/package-0.4.2/`
