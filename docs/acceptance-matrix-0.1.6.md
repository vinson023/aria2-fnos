# Aria2 Pro 0.1.11 验收矩阵

| 类别 | 验收项 | 状态 | 证据/备注 |
| --- | --- | --- | --- |
| 构建 | Core 1.36.0 x86_64、固定源码/补丁、哈希、许可证 | 已完成 | `docs/phase3-build-record.md` |
| 构建 | AriaNg 1.3.14 静态资源、无运行时 Node/npm | 已完成 | `build-output/ariang.SHA256SUMS`、FPK 检查 |
| 生命周期 | 安装、启动、停止、重启、专属用户 | 已完成 | 测试机 App Center 与进程检查 |
| 配置 | RPC/BT 端口可修改、范围与冲突校验 | 已完成 | 6801/6889 回归与 Go 单元测试 |
| 网关 | 管理员身份、旧 Header 拒绝、RPC secret 服务端注入 | 已完成 | Unix Socket/HTTP 回归 |
| 行为 | Tracker 刷新、持久化、完成/错误/停止清理 | 已完成 | 设备回调与重启证据 |
| 行为 | session、DHT、TCP/UDP/IPv6 监听恢复 | 已完成 | 设备磁力任务回归 |
| UI | 登录后的 `/app/aria2` 页面加载 | 已完成 | `0.1.11` 桌面 iframe `src=/app/aria2/`；带前缀 Unix Socket 路径返回 HTTP 200 和 AriaNg HTML |
| UI | 浏览器网络与存储不含远程 RPC/secret | 已完成 | 静态 bundle 与 Socket 响应检查；RPC 仅由服务端注入 secret |
| 升级 | 正式升级接口、版本迁移与数据保留 | 已完成 | 飞牛应用中心安装流程已安装 0.1.11 running，配置和应用状态可查询 |
| 目录授权 | SDK 目录选择、服务端二次校验、有效下载根 | 已完成 | 真实 iframe 成功选择 `vol1/1000/test` 并写入 `download_root`/`authorized_roots`；取消后配置不变；共享目录候选仍经服务端根目录校验；Scope 与 `micro_app=true` 已生效 |
| 协议 | HTTP、FTP、BT、磁力、Metalink 最小下载样本 | 已完成 | HTTP、FTP、BT、磁力与 Metalink 均有最小样本证据 |
| 发布 | FPK 结构、离线资产、哈希、许可证和变更记录 | 已完成 | 当前工作区 `aria2.fpk` SHA-256 `2DCECA6046A99919B52DAD2E4D62DECC8497D4E6AF77026C330D6E3B27A46B08`；Core `EEFB98AFB22DD3A4B7BF70705F064704257542C85FAEF424D6273CF0CCB5729E`；管理器 `EEB13688116782AF400328E94CDD95F2D2DF2576F1ACFDCF724DBB359ACB8864`；hook `D78F1761DD772E72D874637B91622E8C569179F7BED2154BF53D7265B8C32204` |

## 观察项

自动化浏览器无法展示该内网 iframe 的像素内容，原因是其客户端拦截而非测试机响应；设备
Socket、桌面 iframe 目标与管理员 RPC 均已验证。
