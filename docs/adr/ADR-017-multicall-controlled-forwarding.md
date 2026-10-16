# ADR-017：`system.multicall` 的受控转发

- 状态：已采纳
- 日期：2026-09-07
- 关联：ADR-002（飞牛统一网关）、ADR-013（专属选项转发侧拦截）、ADR-014（种子元数据台账）
- 落地版本：`0.1.96`

## 背景

`0.1.96` 的优化排查中，直接在测试机上对运行中的 `0.1.95` 打网关 `/jsonrpc`：

```
system.multicall    -> 400 invalid RPC request
system.listMethods  -> 400 invalid RPC request
aria2.getGlobalStat -> 200（对照组，说明链路本身正常）
```

原因是 `proxyRPC` 只放行 `aria2.` 前缀的方法。而 AriaNg 前端确实在用这两个方法：
`aria2TaskService.getTaskStatusAndBtPeers` 用 multicall 合并 `tellStatus` 与 `getPeers`，
`aria2TaskService.retryTask` 用 multicall 合并 `tellStatus` 与 `getOption`，
`controllers/debug.js` 用 `system.listMethods` 枚举可用方法。`/jsonrpc` 的地址与空密钥由
`fnos-directory-adapter.js` 固化，前端没有绕开网关的路径，所以这三处功能在 `0.1.95`
及之前一直是坏的：任务详情页打不开、看不到 peer 列表、重试按钮无效，
并且非 silent 的失败会弹错误提示并反复重试。这与用户反馈的「多任务下载时界面很卡」方向一致。

## 为什么不能简单放行

Core 的 JSON-RPC 在 multicall 上有两处与直觉相反的契约，均在真机取过证据：

1. **顶层不放 token，鉴权按嵌套调用逐个进行**。`params[0]` 放 token 时 Core 直接报
   `The parameter at 0 has wrong type`（该位置要求的是调用数组）；子调用各自不带 token 时，
   整批返回 `Unauthorized`：

   ```
   params:[[{getGlobalStat},{tellActive}]]         -> [{Unauthorized},{Unauthorized}]
   params:[[{getGlobalStat,params:["token:…"]},…]] -> 正常数据
   ```

   本应用的 RPC 密钥只存在于网关内部，`getCurrentRpcSecret()` 被适配层固定为空，
   因此子调用永远不可能自带有效密钥，必须由网关逐个注入。

2. **嵌套调用会真的执行，并且绕过 ADR-013 的转发侧守卫**。真机实测：

   ```
   system.multicall params:[[{"methodName":"aria2.changeGlobalOption","params":[{"dir":"/etc"}]}]]
   -> [["OK"]]，随后 aria2.getGlobalOption 读回 dir 已是 /etc
   ```

   同理，嵌套的 `addUri`、`remove` 也会被执行，而 `0.1.94` 起的种子台账只识别顶层调用，
   从这条路径过去就会漏掉删除任务时的残留文件与种子清理。

## 决策

1. 放行 `system.multicall` 与 `system.listMethods`，其余仍只允许 `aria2.` 前缀
   （`forwardableRPCMethod`）。
2. multicall 做结构校验：解析调用数组（归一化 AriaNg 的 `params[0]` 数组形式、开头带 token 的
   变体、以及把调用平铺在 `params` 里的写法），逐项要求 `methodName` 命中**只读白名单**，
   单次上限 128 项。白名单按「默认拒绝」实现。
3. 密钥注入位置：先剥掉客户端自带的 `token:`，再把真实密钥注入每个嵌套调用的 `params[0]`，
   顶层不带 token。
4. `system.listMethods` 不注入密钥（Core 对它不做鉴权）；`params` 缺失时补成空数组，
   避免序列化成 `null` 后被 Core 拒收。
5. 台账与运行期选项持久化不改语义：嵌套写方法已被拒绝，multicall 的台账变更恒为空。

## 备选方案

- **整体放行 `system.*`**：否决。等于把 ADR-013 的目录守卫与 ADR-014 的台账同时作废。
- **写方法黑名单代替只读白名单**：否决。Core 有 38 个方法，黑名单随版本升级会漏项。
- **改前端让它别用 multicall**：否决。要动 AriaNg 产物并长期维护补丁，换来的仍是功能缺失。
- **让前端自带密钥**：否决。密钥一旦离开网关就等于公开，直接违反回环 RPC 与网关鉴权边界。

## 后果

- 正向：任务详情页、重试、调试页恢复；目录越权与台账绕过两条路径被显式封死，并有断言守着。
- 代价：网关多了一层需要与 Core 报文契约对齐的解析。若将来 Core 或 AriaNg 改了 multicall 形状，
  容器用例 E 与设备脚本会先失败，不会静默退化。
- 有意保留的限制：任何依赖嵌套写操作的前端能力会拿到 400 而不是被执行。确有需求时，
  应作为独立的顶层调用重新走一遍守卫，而不是放宽白名单。

## 验证

- `manager/cmd/aria2-manager/multicall_test.go`：注入位置、剥除客户端自带 token、
  12 个非只读嵌套方法逐个拒绝、4 种畸形列表、超上限、平铺归一，
  以及经 `proxyRPC` 的三条端到端断言（批量放行、嵌套越权 400 且 `/etc` 不出网关、
  `listMethods` 不带密钥）。
- `tests/shipped-chain-container.sh` 用例 E：在真实静态 Core 前跑批量查询、详情页形状、
  嵌套 `changeGlobalOption` 与 `remove` 拒绝、目录未受影响、`listMethods` 放行。
- `tests/fnos-verify-multicall.sh`：装机后的设备验收，同样六组断言，
  越权用例额外回读 `dir` 确认未被改写。
