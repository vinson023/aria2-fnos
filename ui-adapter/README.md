# AriaNg fnOS 安全适配

本目录记录对固定 AriaNg `1.3.14`（revision
`d6a765377e1eecfbcc387dcb824124df114decfb`）的最小安全补丁。补丁仅在构建工作副本中
应用，最终 FPK 只包含构建后的静态文件，不包含 Node.js、npm、源码或依赖目录。

适配后的 AriaNg 只使用相对网关路径 `/app/aria2/jsonrpc`，固定 HTTP `POST`，禁用
WebSocket。启动时会清除浏览器已保存的 RPC 主机、端口、Header、secret 与附加服务器列表；
设置页不显示 RPC 配置、多服务器或 secret 输入控件。服务端仍会对每一个请求验证网关管理
员上下文，并在转发到回环 Aria2 时注入 secret。

重新构建步骤：在受审查、已安装依赖的 `build-work/AriaNg` 副本应用
`ariang-fnos-adaptation.md` 记录的改动，执行 `npm run build`，检查输出只含静态资源，随后复制
`dist/` 到 `build-output/ariang` 和 `fpk/app/www`。两个目标目录同步前必须先删除旧的
`js/aria-ng-*.min.js`，确保只保留当前 `index.html` 引用的单个 bundle；同步完成后再重新生成
`build-output/ariang.SHA256SUMS`。不得在安装或运行时执行该过程。

## 打包范围约束

`fpk/app/www` 与 `build-output/ariang` 必须逐字节一致，两份目录树都由
`build-output/ariang.SHA256SUMS` 覆盖（当前 34 个文件）。AriaNg 的仓库源码目录
`scripts/` 与 `styles/` **永不允许进入这两个目录**：`index.html` 只引用
`css/`、`js/`、`fonts/`、`vendor/` 和 `fnos-directory-adapter.js`，松散源码既不会被加载，
还会把未压缩的上游源码混进发布包。同步时如果 `dist/` 之外出现了这两个目录，应直接删除
并回到构建步骤检查，而不是把它们带进 FPK。

`langs/zh_Hans.txt` 带有本项目的中文文案补丁（归档说明、下载完成后归档等）。它属于
打包树而不是构建产物，任何一次 `dist/` 同步之后都必须用 `fpk/app/www/langs/zh_Hans.txt`
回灌 `build-output/ariang/langs/zh_Hans.txt` 并重算哈希清单，否则下一次构建会静默把界面
退回上游文案。