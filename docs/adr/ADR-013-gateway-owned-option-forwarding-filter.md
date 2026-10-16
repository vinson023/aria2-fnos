# ADR-013：网关专属选项在转发侧同样拦截

- 状态：已采纳
- 日期：2026-09-04
- 关联：ADR-012（运行期选项持久化）、`AGENTS.md`「Aria2 RPC 只绑定回环地址」「所有路径、RPC 数据和配置输入均视为不可信」

## 背景

ADR-012 引入运行期选项持久化时，拒绝清单只作用在**落盘侧**：`aria2.changeGlobalOption`
的选项对象会原样转发给 Aria2，只有写入 `aria2-overrides.conf` 时才过滤掉 `dir`、端口、
`rpc-*`、`on-*`、会话与 Tracker 等网关专属项。

`0.1.91` 的设备验收暴露了后果。验收脚本用
`params:[{"dir":"/etc","rpc-listen-port":"9999","bt-tracker":"..."}]` 探测，Aria2 返回 `OK`，
随后 `aria2.getGlobalOption` 读回 `dir` 已经是 `/etc`。也就是说：

1. 拒绝清单只挡住了「重启后仍然有效」，没有挡住「当前进程立刻生效」。
2. 任何能打开 AriaNg 的管理员会话，都能把 Aria2 的运行期下载目录改到
   `authorized_roots` 之外，绕过目录授权、归档边界和「专属用户 aria2 只能写授权目录」这一整层设计。
3. 界面显示与实际状态是否一致取决于用户是否恰好重启过，属于本项目已经明令禁止的
   「凭空生效 / 凭空失效」这一类缺陷的反向形态。

## 决策

网关专属选项在**转发侧**同样拦截，而不是只在落盘侧拦截：

1. `proxyRPC` 处理 `aria2.changeGlobalOption` 时，先用 `changeGlobalOptionParams` 取出
   `params[0]` 选项对象（Aria2 只读第一个参数，见 ADR-012 的取值语义修正）。
2. 再用 `filterGatewayOwnedOptions` 按同一份 `runtimeOverrideDeniedKey` 清单拆分保留项与被拒项，
   并用保留项重写 `params[0]` 后才注入令牌转发。
3. 被全部拒绝时转发空对象 `{}`，Aria2 正常返回 `OK`；客户端回读 `getGlobalOption` 得到的仍是
   真实取值，不会出现「界面显示已改、实际未改」的假象。
4. 落盘侧继续使用同一份清单，两侧共用一个判定函数，避免清单分叉。

## 影响

- 正向：目录、端口、密钥、会话、回调脚本这些受管项在运行期和重启后表现一致，前台无法绕过；
  `on-*` 回调劫持（把 `on-download-complete` 指向任意可执行文件）在网关层就被剥掉。
- 代价：AriaNg「Aria2 设置」里的下载目录、RPC 端口等字段保存后不会生效，回读即恢复原值。
  这些项的唯一入口是本应用自己的设置页，属于有意为之。
- 范围：只处理 `aria2.changeGlobalOption`。任务级 `aria2.addUri` / `addTorrent` 的 `dir`
  仍按既有的 `authorized_roots` 校验路径处理，本 ADR 不改变其语义。
- 测试：`manager/cmd/aria2-manager/runtime_options_test.go` 新增
  `TestProxyRPCStripsGatewayOwnedOptions`（专属项不得出现在上游请求中、普通项必须仍生效并落盘）
  与 `TestFilterGatewayOwnedOptionsKeepsLegitimateKeys`（混合与全拒两种形态）。
  设备侧 `tests/fnos-verify-runtime-options.sh` 新增第 2 节的运行期回读断言与第 2a 节混发用例。

## 后续

本 ADR 的守卫只覆盖**顶层** `aria2.changeGlobalOption`。`0.1.96` 排查发现
`system.multicall` 的嵌套调用同样会被 Core 执行，因此是一条绕过本 ADR 的现成路径：
真机实测嵌套的 `changeGlobalOption {"dir":"/etc"}` 返回 `OK` 并确实改掉了运行期目录。
处理方式与配套断言见 `ADR-017-multicall-controlled-forwarding.md`。
