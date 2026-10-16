# 0.1.96 遗留验收执行清单

本文件记录**尚未执行完的验收项**、执行方式与前置条件。与 `docs/acceptance-matrix-0.1.6.md`
（需求到验收的映射）不同，这里只关心"还有哪些断言没跑、怎么跑"。

当前发布物：`aria2.fpk`，版本 `0.1.96`，13,861,296 B，外层 SHA-256
`F9F22A52F0E73F880FEAE9F29074BB22232F892C57D73A69C4E0C4D52B0353C9`。

## 一、本地已完成的核验（无需设备与容器）

| 项 | 方法 | 结果 |
| --- | --- | --- |
| 产物完整性 | 解包 `aria2.fpk`，对 `build-output/fpk-0.1.96.inner.list` 逐文件比对 SHA-256 | **63 项全部一致，0 缺失 0 不符** |
| 外层指纹 | `sha256sum aria2.fpk` | 与发布记录一致 |
| 包内版本 | 解包 `manifest` | `version = 0.1.96`、`platform = x86`、`appname = aria2` |
| 生命周期脚本语法 | `bash -n fpk/cmd/*`（9 个） | 全部通过 |
| 测试脚本语法 | `bash -n tests/*.sh` | 全部通过 |

注意：包内 `manifest` 与源文件 `fpk/manifest` 必然哈希不同（`fnpack` 注入自引用
`checksum` 字段），逐文件清单里这一行属预期，见 `ADR-015` 第 12 条。

## 二、容器验证（欠跑，来自 2026-09-07 数据卷事故）

事故经过见 `docs/release-candidate.md` 的「测试机数据卷事故」一节。受影响的是
`0.1.96` 为 pid 属主新增的两条断言，其余链路在事故前已通过。

| 脚本 | 覆盖 | 默认入参 |
| --- | --- | --- |
| `tests/startup-e2e-container.sh` | 8 组：健康启动、换机双失效自愈、配置缺失、半截 JSON、端口自冲突、无共享目录兜底、日志轮转、卸载清理；**含 `aria2.pid`/`manager.pid` 属主与启动锁释放断言** | `PKG=/hosttmp/pkg094` |
| `tests/startup-failure-attribution-container.sh` | 就绪门禁失败时把 `aria2.log`/`manager.log` 尾部归集进 `startup.log` | `PKG=/hosttmp/pkg0195` |
| `tests/shipped-chain-container.sh` | 用例 A–E：真实 HTTP 下载→归档→清理、multicall 放行与嵌套越权拒绝 | `PKG=/hosttmp/pkg0195`、`FIX=/hosttmp/btfixture-linux` |
| `tests/startup-repair-sim.sh` | 6 组真机坏配置驱动 `--ensure-ports` | `MGR=/hosttmp/aria2-manager` |

需要准备（放到容器可见的宿主目录，例如 `/hosttmp`）：

1. 解包后的 `0.1.96` 包体（外层 tar 解开，再把 `app.tgz` 展开到 `app/`），目录名如
   `pkg0196`；容器脚本按 `$PKG/app/.`、`$PKG/cmd/.`、`$PKG/manifest` 取用。
2. 交叉编译的 Linux 测试产物：`mgr.test`、`hook.test`（供 `runct.sh` 用），以及
   `btfixture-linux`（仓库内已有：`build-work/btfixture-linux`，9,681,240 B）。
3. 管理器 Linux 产物 `aria2-manager`（供 `startup-repair-sim.sh` 用）。

### 执行方式（安全红线，必须逐字遵守）

`AGENTS.md`「验证安全红线」要求：这些脚本**只能在容器内执行**，且必须是
**同一条命令里的 `docker run ... bash <脚本>`**，不允许先跑脚本再补容器前缀。

```bash
docker run --rm -v /hosttmp:/hosttmp debian:12-slim bash /hosttmp/startup-e2e-container.sh
```

`startup-e2e-container.sh` 第一步就是 `rm -rf /var/apps/aria2 /vol1 /run/aria2`，
在宿主机上直接执行会清空真实数据卷 —— 该脚本已用 `PKG` 环境变量参数化，但**不能**在宿主
直接 `bash` 它。`startup-repair-sim.sh` 与 `shipped-chain-container.sh` 同理。

## 三、设备验证（待装机后执行）

测试机 `192.168.x.x`（fnOS Debian 12 x86_64，glibc 2.36）。设备侧以 root 执行。

| # | 脚本 / 动作 | 期望 |
| --- | --- | --- |
| 1 | `tests/fnos-verify-multicall.sh` | 六组断言全过：`listMethods` 放行、只读批量可用且子调用带密钥、详情页形状、10 个嵌套写方法逐个 400、`dir` 未被改写、空列表与 >128 项被拒 |
| 2 | 浏览器 | 任务详情页可打开、BT peer 列表可见、重试按钮有效（`ADR-017` 修复的三处功能） |
| 3 | `tests/fnos-verify-torrent-cleanup.sh` | 删除任务连带清理 RPC 上传的 `.torrent`；脚本有前置守卫，管理器无 `--torrent-ledger` 时直接中止 |
| 4 | `tests/fnos-diagnose-startup.sh` | 只读排查，输出 `正常/注意/异常` 计数；用于换机失效场景取证 |
| 5 | `tests/fnos-verify-runtime-options.sh` | 运行期选项持久化回归（0.1.92 项复跑） |
| 6 | `tests/fnos-verify-static-core.sh` | 静态 Core 在 glibc 真机上的功能面复跑 |

## 四、收尾动作

每完成一项，在 `docs/release-candidate.md` 追加一行实测结论（含日期、版本、
退出码或断言计数），并把 `README.md`「当前状态」段对应的"待验收"改为结论。
`0.1.93`/`0.1.94`/`0.1.96` 三版的验收全部关闭后，才可在 `README.md` 中标记该轮需求闭环。
