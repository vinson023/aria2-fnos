# AriaNg fnOS 适配记录

该记录对应固定 AriaNg 1.3.14 构建副本中已应用的改动：

1. 在 `src/scripts/services/ariaNgSettingService.js` 新增 `lockGatewayRpcSetting`，每次读取设置时清空 `rpcHost`、`rpcPort`、`rpcRequestHeaders`、`secret` 和 `extendRpcServers`，并固定别名、接口和 HTTP POST。
2. 将 `getCurrentRpcUrl` 改为返回 `/app/aria2/jsonrpc`；将请求 Header 和 secret 返回为空；将 WebSocket 判定固定为 false。
3. 在 `src/views/settings-ariang.html` 为 RPC 标签、添加服务器按钮和 RPC 配置面板加入 `ng-if="false"`。
4. 在 `src/scripts/services/aria2TaskService.js` 增加 `getTaskCleanupPaths`：删除或清空已停止任务前，先通过相对网关 `api/v1/cleanup` 清理未完成任务（`error`/`removed`）的主文件、`.aria2`/`.torrent` 控制文件及 BT 的 `infoHash.torrent`；`complete` 任务仅移除记录、不触碰下载文件。`removeTasks` 与 `clearStoppedTasks` 均执行清理，控制器 `clearStoppedTasks` 传入当前任务列表。管理器清理接口只接受授权根内的常规文件或目录，目录会递归检查符号链接后删除，越界或含链接时拒绝。
5. 在“下载完成后归档”之外新增 `fnos-archive-dir` 配置项，值为飞牛授权归档目录；
   `setSpecialMode(true)` 且当前没有归档目录时自动调用目录选择器
   （`window.fnosArchiveDirectory.pick`），用户取消则不保存开启状态，避免“开关显示开启但
   实际不生效”。`saveFNosConfig` 在管理器返回非 2xx 时回显中文纯文本原因，不再静默失败。
6. `src/index.html` 固定引入 `fnos-directory-adapter.js?v=2026904`，中文语言资源使用
   `langs/zh_Hans.txt?v=fnos-zh-20260904` 缓存破坏参数；改动语言文件必须同时提升该参数，
   否则浏览器会继续沿用旧的英文缓存。

这是构建变更记录，不是可独立应用的通用补丁。重新构建时必须在已锁定 revision 的工作副本中逐项复核以上改动，完成构建后检查产物只引用一个 AriaNg bundle 且其中包含 `/app/aria2/jsonrpc`。
