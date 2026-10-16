# 上游行为保真矩阵

本文件把固定修订的 P3TERX Aria2 Pro Docker、`aria2.conf` 和 AriaNg 行为映射到
飞牛原生 FPK 的第一版设计。它只定义迁移边界和未来验收证据，不包含运行脚本、
配置文件或二进制产物。

## 判定规则

- **保留**：第一版必须提供等价能力，但实现方式可改变。
- **适配**：保留用户价值，同时替换 Docker、root、公开 RPC 或任意路径等不安全前提。
- **排除**：第一版不得提供该上游默认能力，后续需要新的产品确认和 ADR。
- **待确认**：仅用于尚未完成产品确认的事项；D-001 至 D-005 已于 2026-08-27 确认，
  相关能力仍须按本矩阵的安全边界分阶段实现。

## Aria2 Core 与任务状态

| 上游行为 | 第一版判定 | 飞牛原生设计 | 未来验收证据 |
| --- | --- | --- | --- |
| `input-file`、`save-session`、`save-session-interval` 保存未完成任务 | 保留 | 会话文件放在 `$TRIM_PKGVAR`，启动时恢复，停止和定期写入均使用应用用户权限 | 中断 HTTP、BT 和磁力任务后重启，未完成任务恢复且状态文件仍存在 |
| IPv4/IPv6 DHT 路由表 | 保留 | `dht.dat` 与 `dht6.dat` 放在 `$TRIM_PKGVAR`；IPv6 是否启用由受控配置决定 | 启用对应协议后重启，DHT 文件保留且服务可继续使用 |
| HTTP(S)、FTP、BT、磁力、Metalink | 保留 | 使用固定修订、目标架构对应的 Aria2 Pro Core；不因协议需要加入 Docker 或解释型运行时 | 每种协议有最小可重复下载用例 |
| 高并发、重试、磁盘缓存、umask、BT 加密等 Aria2 参数 | 适配 | 仅暴露经过白名单校验的配置字段；保守默认值由后续配置模板固定 | 非法或未知字段被拒绝；已允许字段生成有效配置 |
| 单一共享 Aria2 守护进程 | 保留 | 第一版只运行一个专属应用用户的实例，配置、会话和端口属于该实例 | 管理员能操作；非管理员与第二实例请求均被拒绝 |

## 网络、RPC 与 AriaNg

| 上游行为 | 第一版判定 | 飞牛原生设计 | 未来验收证据 |
| --- | --- | --- | --- |
| 上游 `rpc-listen-all=true` 与公开映射 RPC 端口 | 排除 | Aria2 仅监听 `127.0.0.1`；浏览器经 `/app/aria2` 和 Unix Socket 管理器访问 | NAS 局域网地址不能直连 RPC；经网关的管理员请求可用 |
| 默认 RPC 端口 `6800` | 适配 | 默认值为 `6800`，持久化且可独立修改；变更不公开 RPC | 新端口回环监听、旧端口关闭、冲突配置回滚 |
| BT/DHT 使用单一监听端口 | 适配 | 默认 `6888`，作为 TCP BT 与 UDP DHT 共用值；与 RPC 端口独立配置 | 修改后 TCP/UDP 监听一致，非法或占用端口被拒绝 |
| 上游固定或环境变量传入 RPC secret | 适配 | 安装时生成随机 secret，保存于 `$TRIM_PKGETC` 的受限文件，只由管理器向本机 Aria2 注入 | 静态资源、浏览器存储、请求和日志均无 secret |
| AriaNg 多 RPC 主机、任意 RPC 地址和浏览器保存 secret | 排除 | 仅提供一个由应用管理器固定的本地代理目标；隐藏远程主机、端口、Header、secret 和多服务器配置入口 | 篡改浏览器设置或请求不能把代理指向任意 URL；无 secret 落入浏览器存储 |
| AriaNg 相对访问路径 | 保留 | AriaNg 面向网关相对路径工作，不写死 `127.0.0.1:6800` 或实际 secret | 端口变更后 UI 无需修改地址仍能操作 |

## Tracker、清理与归档

| 上游行为 | 第一版判定 | 飞牛原生设计 | 未来验收证据 |
| --- | --- | --- | --- |
| Tracker 列表刷新 | 适配 | 管理器获取数据而不执行远程脚本；固定 3 个 HTTPS 源、禁止重定向、15 秒超时和 256 KiB 上限；仅持久化过滤去重后的 `http`/`https`/`udp` URI，失败不影响 Aria2 | 单元测试覆盖恶意、超大、失效响应；测试机刷新 144 条并在重启后恢复 |
| `clean.sh` 清理 `.aria2`、`.torrent` 和空目录 | 适配 | 回调只处理已校验的任务路径；完成、失败、删除三种状态分别定义允许删除的对象 | 穿越、符号链接逃逸和错误任务不会删除授权根外文件 |
| 失败/删除任务删除下载文件 | 适配 | 默认保守：必须区分未完成文件、完成文件和用户显式删除意图；高风险删除受结构化策略约束 | 完成文件不会因失败/删除回调被误删；预期控制文件按策略清理 |
| `move.sh` 将完成文件移动到目标目录 | 适配 | “下载完成后归档”以 `special_mode` 为唯一事实源，未选归档目录时由管理器归一化为关闭；源和目标都必须是飞牛授权目录，跨设备时复制后删除，同名目标追加 ` (1)` | 已验证（0.1.88 测试机）：HTTP 与 12 个并发任务完成即移动到归档目录，`aria2.log` 出现 `archive done: 源 -> 目标`，源目录清空；未选目录时开关自动归一化为关闭 |
| 上游 `SEED_TIME` 环境变量控制做种时长 | 适配 | 默认 `seed-time=0`、`seed-ratio=1.0`、`bt-detach-seed-only=true`，任务完成即停止做种并释放并发槽位；需要分享时由用户在前台 BT 设置调大 `seed-time` | 已验证（0.1.88 测试机）：真实 BT 任务完成后 Aria2 记录 `Your share ratio was 0.0` 并立即结束做种，任务离开活动列表且不再上传 |
| `upload.sh`、Rclone 与 `rclone.env` | 排除 | 第一版不打包、不安装、不自动更新 Rclone，也不执行上传脚本 | 产物与启动流程不含 Rclone 下载、安装或调用 |

## 与上游不同的明确默认值

1. 不使用上游 `/root/.aria2`、`/root/completed` 或容器卷路径；分别使用
   `$TRIM_APPDEST`、`$TRIM_PKGETC`、`$TRIM_PKGVAR` 和飞牛授权数据目录。
2. 不以 root 运行 Aria2、管理器或回调。
3. 不把 Tracker 更新脚本、Rclone、可变配置或可执行文件作为安装/启动时下载内容。
4. 不将 AriaNg 的多服务器、远程 RPC、任意 Header 或 secret 管理能力作为应用功能。
5. 不因端口可配置而放宽 RPC 的回环绑定和统一网关鉴权。

## 参考依据

- `reference-snapshot/upstream/Aria2-Pro-Docker/README.md`
- `reference-snapshot/upstream/aria2.conf/aria2.conf`
- `reference-snapshot/upstream/aria2.conf/script.conf`
- `reference-snapshot/upstream/aria2.conf/README.md`
- `reference-snapshot/upstream/aria2.conf/tracker.md`
- `reference-snapshot/upstream/AriaNg/README.md`
- `reference-snapshot/upstream/AriaNg/src/scripts/services/ariaNgSettingService.js`
