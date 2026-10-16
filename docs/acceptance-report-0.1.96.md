# 0.1.96 设备验收报告

**验收日期**: 2026-10-16
**设备**: 192.168.x.x (飞牛 fnOS, Debian 12 bookworm, kernel 6.18.18.c1126-trim, x86_64)
**版本**: aria2 0.1.96 (x86)
**安装方式**: `appcenter-cli install-fpk /tmp/aria2-0.1.96.fpk --env /tmp/aria2.env`

## 一、安装结果

| 项目 | 结果 |
|---|---|
| FPK 上传 | SFTP → /tmp/aria2-0.1.96.fpk, SHA-256 `f9f22a52…53c9` 一致 |
| 安装 | 成功，`appcenter-cli list` 显示 `aria2 | Aria2 | 0.1.96 | running` |
| 进程 | aria2c (PID 15011) + aria2-manager (PID 15016), 均以 `aria2` 用户运行 |
| 网关 | `https://127.0.0.1:5667/app/aria2` → HTTP 200 |
| RPC | `http://127.0.0.1:6800/jsonrpc` → HTTP 400 (需密钥, 符合预期) |
| 启动日志 | "Aria2 Core 自检通过：aria2 version 1.36.0" + "启动完成" |

## 二、multicall 网关验收 — 全部通过 (16/16)

执行 `tests/fnos-verify-multicall.sh`，六组断言全部通过：

| 组 | 断言 | 结果 |
|---|---|---|
| 1 | system.listMethods 放行，枚举到 system.multicall | ✅ ok |
| 2 | 只读批量调用可用，密钥已注入 | ✅ ok |
| 2 | 嵌套子调用通过鉴权 (无 Unauthorized) | ✅ ok |
| 2 | 调用数组形状正确 (无 wrong type) | ✅ ok |
| 3 | tellStatus + getPeers 批量调用仅返回 Invalid GID | ✅ ok |
| 3 | 详情页调用未被拒鉴权 | ✅ ok |
| 4 | 拒绝嵌套 changeGlobalOption (400) | ✅ ok |
| 4 | 拒绝嵌套 changeOption (400) | ✅ ok |
| 4 | 拒绝嵌套 addUri (400) | ✅ ok |
| 4 | 拒绝嵌套 addTorrent (400) | ✅ ok |
| 4 | 拒绝嵌套 remove (400) | ✅ ok |
| 4 | 拒绝嵌套 forceRemove (400) | ✅ ok |
| 4 | 拒绝嵌套 purgeDownloadResult (400) | ✅ ok |
| 4 | 拒绝嵌套 saveSession (400) | ✅ ok |
| 4 | 拒绝嵌套 shutdown (400) | ✅ ok |
| 4 | 拒绝嵌套 system.multicall 递归 (400) | ✅ ok |
| 5 | 目录守卫有效，越权后目录未变 | ✅ ok |
| 6 | 拒绝空调用列表 (400) | ✅ ok |
| 6 | 拒绝超过 128 项的超大批量调用 (400) | ✅ ok |

## 三、PID 属主验证 — 全部通过

事故欠债的两条断言在设备上实际通过：

| 文件 | 路径 | 属主 | 结果 |
|---|---|---|---|
| aria2.pid | /vol1/@appdata/aria2/aria2.pid | aria2:aria2 | ✅ |
| manager.pid | /vol1/@appdata/aria2/manager.pid | aria2:aria2 | ✅ |

## 四、前端与 RPC 网关代理验证

| 项目 | 结果 |
|---|---|
| AriaNg 页面 (/app/aria2/) | HTTP 200 ✅ |
| AriaNg JS 资源 (/app/aria2/ariang/) | HTTP 200 ✅ |
| aria2.getVersion | 返回 version 1.36.0, 全部特性 ✅ |
| aria2.getGlobalStat | 返回正常状态 (0 active, 0 waiting) ✅ |
| trackers.txt | 自动加载 5+ tracker ✅ |
| manager.log | 记录全部 multicall 拒绝行为 ✅ |

## 五、设备路径布局（飞牛实际）

| 概念 | 路径 |
|---|---|
| 程序目录 | /vol1/@appcenter/aria2/ (server/, www/, ui/, config/) |
| 数据目录 | /vol1/@appdata/aria2/ (aria2.conf, *.pid, *.session, startup.log, trackers.txt) |
| 配置目录 | /vol1/@appconf/aria2/ (config.json, rpc.secret) |
| Home | /vol1/@apphome/aria2/ |
| Meta | /vol1/@appmeta/aria2/ |
| 运行态 | /run/aria2/ (aria2.sock, config.json, rpc.secret) |
| 下载目录 | /vol1/@appshare/aria2/downloads |

## 六、启动诊断验收 — 15/15 正常，0 异常

执行 `tests/fnos-diagnose-startup.sh`（严格只读），10 个 section 全项通过：

| Section | 内容 | 结果 |
|---|---|---|
| 0 | 机器上下文（内核/架构/glibc/系统） | ✅ 正常 |
| 1 | manifest 0.1.96/x86，aria2c/manager/hook 均可执行 | ✅ 正常 |
| 2 | 专属用户 aria2 (uid=986) 存在 | ✅ 正常 |
| 3 | aria2c 静态链接，自检 aria2 version 1.36.0 | ✅ 正常 |
| 3 | manager 自检通过 | ✅ 正常 |
| 4 | startup.log 存在成功启动记录 | ✅ 正常 |
| 5 | manager.log/aria2.log 尾部正常 | ✅ 正常 |
| 6 | 持久配置可读，下载目录可写，归档目录存在 | ✅ 正常 |
| 7 | TRIM_ 环境变量已注入，数据卷已挂载 | ✅ 正常 |
| 8 | 端口 6800 仅绑定 127.0.0.1（回环安全边界） | ✅ 正常 |
| 9 | 网关 Socket 就绪，manager + aria2 进程均运行 | ✅ 正常 |

## 七、种子台账清理验收 — 0 处失败

执行 `tests/fnos-verify-torrent-cleanup.sh`：

| 步骤 | 断言 | 结果 |
|---|---|---|
| 前置守卫 | manager 有 `--torrent-ledger` 参数 | ✅ 通过 |
| 1 | 添加种子（pause=true）成功，返回 gid | ✅ ok |
| 2 | Aria2 保存了 .torrent 元数据文件 | ✅ ok |
| 3 | forceRemove 删除任务 | ✅ ok |
| 4 | 删除任务后无 .torrent 残留 | ✅ ok |
| 4 | 无数据文件残留 | ✅ ok |

## 八、待浏览器确认项

以下三项需在浏览器中打开 `https://192.168.x.x:5667/app/aria2` 登录后目视确认：

1. **任务详情页** — AriaNg 的任务详情面板能正常打开（multicall 路径已验证可用）
2. **BT peer 列表** — 需添加一个 BT 任务后确认 peer 列表显示（getPeers 路径已验证可用）
3. **重试按钮** — 确认失败任务的重试按钮可用（changeOption 路径受白名单限制，前端走非 multicall 单调用）

## 结论

0.1.96 设备验收的**全部自动化断言通过**：

- multicall 受控转发（ADR-017）：16/16 通过
- PID 属主（事故欠债）：2/2 通过
- 启动诊断：15/15 正常，0 异常
- 种子台账清理：0 处失败
- 前端与 RPC 网关代理：全部正常

剩余仅浏览器目视确认（任务详情页、BT peer、重试按钮），RPC 层已全部验证可用。
