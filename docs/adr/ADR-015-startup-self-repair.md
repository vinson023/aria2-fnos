# ADR-015：启动路径配置自愈与持久启动日志

## 状态

已接受，2026-09-05 起生效，对应发布版本 `0.1.94`。

## 背景

用户报告 `0.1.93` 在部分机器上安装后"无法启动"，另一些机器完全正常。设备取证结论：

`config.json` 里的 `download_root`、`archive_root`、`authorized_roots` 都是**机器相关**取值。
`resolveDirectory` 用 `EvalSymlinks` + `Stat.IsDir` 判定，目录不存在即报错；
`normalizeAuthorizedRoots` 要求路径落在 `^/vol[0-9]+` 数据卷内。于是换机、卷号不同
（`/vol1` 与 `/vol2`）、共享目录被改名或删除、`TRIM_DATA_SHARE_PATHS` 为空、
配置从另一台 NAS 拷贝过来，都会让 `prepareConfig` 失败。

`fpk/cmd/main` 是 `set -eu`，而它要执行两次管理器探针：

- `--ensure-ports`（约 229 行）把标准输出当配置读回；
- `--print-download-root`（约 141 行）把标准输出当目录读回。

任一探针非零退出，或标准输出为空，脚本当场中止。失败原因只写在**标准错误**，
而应用中心不收集生命周期脚本的标准错误，用户看到的只有一句"无法启动"。
更糟的是界面进程同时没起来，用户没有任何入口改回配置 —— 形成死锁，只能卸载重装。

设备实测复现（`0.1.93`）：

| 输入 | 退出码 | 标准输出 | 唯一线索（标准错误） |
| --- | --- | --- | --- |
| `authorized_roots=["/tmp/h1/locked"]` | 1 | 空 | `authorized root is not on an fnOS data volume` |
| `download_root=/vol1/1000/no-such-share` | 1 | 空 | `directory cannot be resolved` |

## 决策

1. **区分保存路径与启动路径**。前台保存（`configStore.update`）保持严格：改错了就明确
   报错，绝不静默搬迁用户数据。启动路径改为尽最大努力恢复，先让应用可用，再由用户在
   界面上重新选定目录。
2. **自愈只发生在两个生命周期探针上**。`main` 用 `startupProbe := ensurePorts || printDownloadRoot`
   判定；`decodeConfig` 保持严格不变，因此网关侧的常规读取行为不受影响。
3. **逐条修复，不做整体否决**。单个越界授权目录不得连带否决其余可用目录：

   | 失效取值 | 处理 |
   | --- | --- |
   | 无法解析的 `authorized_roots` 条目 | 逐条移出列表，其余保留 |
   | 不可解析 / 越界 / 不可写的 `download_root` | 置空，交 `normalizeConfig` 重新派生 |
   | 失效或不在下载目录内的归档目录 | 重新派生；父目录不可写时关闭归档 |
   | 非法端口 | 复位到出厂值（见第 4 条） |
   | 非法 `special_mode` | 复位为 `move` |
   | 非 https 的 `tracker_source` | 置空回落随包默认源 |
   | 配置文件为空 / 无法解析 | 退回出厂默认 |

   端口、DHT、网关、Tracker 开关与 `config_version` 等用户设置一律不被顺带改写。
4. **端口复位必须真正解除冲突**。出厂 BT 端口可能恰好等于用户自定义的 RPC 端口
   （例如 `rpc=6888`），此时把 BT 复位到 `6888` 等于什么都没修，`validateConfig`
   仍会以 `invalid or conflicting ports` 拒绝启动。因此保住用户可见的 RPC 端口不动，
   在 BT 侧按 `6888 → 6889 → 6800 → 6801` 顺序挑第一个不冲突的取值；
   真正的空闲与否交给随后的 `ensureAvailablePorts` 避让（`ADR-006`）。
5. **最后兜底是应用私有下载目录**。当本机没有任何可用数据共享目录时，
   创建 `$TRIM_PKGVAR/downloads`（`0770`，并把属主交还给该目录的属主，因为生命周期
   探针通常以 root 运行），保证 `aria2c` 一定有可写的 `dir`。
   该目录必须在 `normalizeConfig` **之前**建好：`normalizeConfig` 见到空下载目录会连带把
   归档判成关闭，否则"下载完成后归档"会在兜底启动后静默失效。容器实测已抓到这个回归。
6. **每条改动都必须在启动日志留下中文说明**，包括"下载目录沿用应用私有目录"这类
   非降级的降级。标准自愈路径不得静默改变任何取值。
7. **管理器把自愈说明写到标准错误**，因为标准输出是被生命周期脚本消费的返回值通道；
   `fpk/cmd/main` 用 `2>>"$startup_log"` 归集。网关路径禁止向标准输出写任何东西。
8. **新增持久启动日志 `$TRIM_PKGVAR/startup.log`**。此前每个阶段的原因只进
   `aria2.log` / `manager.log`，而这两个文件在每次启动时被删除重建，跨重启的线索全部
   丢失。`startup.log` 不随启动清空，只保留最近 200 行，`0660` 归属应用用户，
   并在卸载选择"清空应用配置"时随其余配置一并删除（`cmd/uninstall_callback`）。
   生命周期脚本在每个 `exit 1` 之前先 `log_startup` 记下原因。
9. **修复结果自然持久化**。`fpk/cmd/main` 本就把 `--ensure-ports` 的输出写回
   `/run` 与 `TRIM_PKGETC/config.json`，因此自愈结果无需额外落盘通道，也不会与持久配置分叉。
10. 提供 `tests/fnos-diagnose-startup.sh`（只读、一次性）：自动探测卷号，收齐机器上下文、
    版本、专属用户、ELF/CPU 兼容性自检、`startup.log`、配置取值与目录可达性、
    `TRIM_DATA_SHARE_PATHS`、端口占用（区分本应用自身与真正的外部冲突）、进程与网关
    Socket，并给出结论。用于换取机失败机器上的一次性完整证据。
11. **启动前预检 Core 可执行性**。`0.1.93` 及之前的 `server/aria2c` 是动态链接的，除 glibc 外还依赖
    `libz`、`libexpat`、`libsqlite3`、`libssl3`、`libcrypto3`、`libssh2`、`libstdc++`。任一依赖缺失时
    `aria2c` 只往 `aria2.log` 丢一行动态链接器报错就退出，而应用中心不收集它，前台表现与配置类
    故障完全一样：一句没有信息的「无法启动」。因此 `cmd/main` 在拉起进程前先执行一次
    `aria2c --version`，失败就把真实原因写进 `startup.log` 再中止，成功也记一行。取值写成
    `core_check=$(...)` 且**不接管道**：管道会让 `$?` 变成 `head` 的退出码，反而掩盖真实失败。
    **自 `0.1.95` 起这一条从缓解手段降级为安全网**：Core 已改为完全静态链接，宿主共享库缺失这一类
    根因被消除（见 `ADR-016`）；预检仍保留，用于兜底 CPU 指令集与 ELF 架构不匹配等剩余可能。
12. `fnpack` 打包时会向 `manifest` 注入自引用的 `checksum` 字段，因此**包内 `manifest` 与源文件
    `fpk/manifest` 必然哈希不同，且任何内容变化都会带动它变化**。逐文件清单里 `manifest` 一行的
    漂移不是异常，比对差异时应当预期。
13. **就绪门禁失败时归集日志尾部**（`0.1.95` 起）。`aria2.log` 与 `manager.log` 每次启动被删除重建，
    应用中心又不收集它们，因此「进程拉起来了但立刻挂掉」这一类故障在跨一次重启之后线索全灭，
    前台仍然只剩「无法启动」。`cmd/main` 在就绪门禁失败分支里把两个日志的尾部若干行写进持久的
    `startup.log` 再回滚。验证方式见 `tests/startup-failure-attribution-container.sh`：
    构造一个「`--version` 能过预检、真正拉起立刻以 127 退出」的假 Core 把这条路径逼出来，
    断言病因文字确实进入 `startup.log`。

14. **启动锁只覆盖启动过程本身**（`0.1.96` 起）。`start` 用 `mkdir $TRIM_PKGVAR/.start.lock`
    做互斥，此前只有 `stop` 分支删它，成功路径与 `write_aria2_config`、端口自愈里的
    `exit 1` 都不删，结果一次正常启动之后就在应用数据目录里永久留下一个 root 属主的锁目录，
    而并发保护也退化成了「第一次之后永远走残留回收分支」。现在抢到锁后立即
    `trap 'rmdir "$start_lock"' EXIT`，无论成功、失败还是回滚都释放；`stop` 不再代 `start`
    删锁（它没抢过锁，删别人的锁只会破坏互斥）。启动被强杀留下的残留仍由原有的
    「无存活进程且无 Socket 即回收」分支处理。同时补上两条静默出口的中文原因：
    原先 `rmdir` 或 `mkdir` 失败直接 `exit 0`，应用中心以为启动成功，前台只看到一个打不开的空界面。
    `cmd/uninstall_callback` 在「清空应用配置」时用 `rmdir` 兜底删除该目录（`rm -f` 删不掉目录）。

## 后果

- 换机、改共享目录名、删卷这类问题从"应用起不来且无解"变成"应用照常启动 +
  `startup.log` 写明改了什么"，用户能在界面上把目录选回去。
- 严格校验仍然存在于所有非启动路径，坏配置不会被静默接受，也不会悄悄搬迁用户数据。
- 自愈的代价是"启动时可能改写配置里的机器相关字段"，因此第 6 条的逐条留痕是强制要求，
  不是可选项；新增自愈分支时必须同时补日志与测试。
- 若连应用私有目录都不可用（数据卷未挂载），仍然失败，但 `startup.log` 会带上真实病因，
  不再是一句无信息的"无法启动"。
- 生命周期脚本对 `startup.log` 的写入全程 `|| true`，日志不可写绝不能反过来导致启动失败。

## 验证

- `manager/cmd/aria2-manager/startup_repair_test.go`：9 例（含越权授权根、失效下载目录、
  健康配置零变更、半截 JSON、端口自冲突、兜底目录权限与幂等、兜底后归档仍开启）。
- 真实 Linux 复跑（`debian:12-slim` 容器 + 交叉编译测试产物）：管理器 67 通过 / 0 跳过 / 0 失败，
  `aria2-hook` 14 通过 / 0 失败。Windows 宿主跳过 POSIX 专属用例外全部通过。
- `tests/startup-repair-sim.sh`：在同一容器内用真机复现的坏配置驱动 `--ensure-ports`，
  6 组用例全部要求"退出码 0 + 标准输出是合法配置 + 逐条中文说明"，结果 `全部用例通过`。
  该脚本同时抓到并锁死了第 4、5 条描述的两个真实缺陷。
- `tests/startup-e2e-container.sh`：把**解包后的真实发布产物**放进 `debian:12-slim`，按设备的目录
  布局与 `TRIM_` 变量复刻，直接执行 `cmd/main start`。8 组用例逐一断言退出码、`startup.log` 内容、
  持久配置回写结果、进程存活、服务用户属主（走 pid 文件加 `/proc/<pid>/status`，不依赖容器里
  没有的 `pgrep`/`ps`）、网关 Socket、日志轮转上限与「清空应用配置」的删除范围，
  结果 `真实产物启动链路验证全部通过`。其中还确认了 `--ensure-ports` 的修复结果在 `stop` 回写后仍然保留。
- 构建可复现：同一源码树二次 `go build -trimpath` 产出与入包产物逐字节一致的 `7007c6bb…52f7`。
- 9 个 `fpk/cmd/*` 全部通过 `bash -n`。注意归档内 `cmd/*` 的模式是 `666`，可执行位由应用中心在
  安装时补上；容器里复刻目录结构时必须自己 `chmod 755`，否则会把「回调没执行」误判成产品缺陷。
