# ADR-012：AriaNg 运行期全局选项的持久化

## 状态

已接受，2026-09-04 起生效，对应发布版本 `0.1.91`。

## 背景

AriaNg 的"Aria2 设置"页通过 `aria2.changeGlobalOption` 修改的参数只存在于 Aria2 进程
内存中。本应用在每次启动时都会从 `config.json` 重新生成 `aria2.conf`，于是这些改动在
下一次重启后被随包默认值整体覆盖，表现为"改完当时生效、重启就回退"。这是长期挂在
遗留清单上的第一项功能性缺陷。

可选的落点有三个，各有明确问题：

1. 写回 `config.json`。`fpk/cmd/main` 在启动时用 `--ensure-ports` 把 Go `config` 结构体
   `json.MarshalIndent` 后同时写入 `/run` 与 `TRIM_PKGETC`，任何不在结构体里的键都会被
   静默抹掉（见 `ADR-011`）。要为几十个 Aria2 选项逐一建模，等于把界面变成配置文件
   的翻版，得不偿失。
2. 用 `aria2 --input-file` 或运行期重载。Aria2 只在启动时读取配置文件，没有重载入口，
   这条路不成立。
3. 单独维护一份覆盖文件，在生成 `aria2.conf` 时合并。改动面最小，且与现有"每次启动
   重写配置"的模型天然契合。

## 决策

1. 采用第 3 种方案，新增 `TRIM_PKGVAR/aria2-overrides.conf`，内容是排过序的
   `key=value` 行，权限 `0600`，与 `aria2.conf` 同格式，因此生命周期脚本无需额外的转义
   规则。它位于持久化目录，卸载时"清空应用配置"必须一并删除。
2. 捕获点在管理器的 RPC 网关。`proxyRPC` 在**注入 `token:` 之前**取出
   `aria2.changeGlobalOption` 的选项对象（AriaNg 不保证令牌位置，逐个参数尝试而不是假定
   下标），并且只有在上游返回 HTTP 200 **且** JSON-RPC 响应不含 `error` 时才落盘。被
   Aria2 拒绝过的取值如果也被记住，重启后会把同一个非法值再塞进 `aria2.conf`。
3. 独占项一律拒绝，Go 与 shell 两层维护同一份清单。精确键名包括 `dir`、
   `listen-port`、`bt-listen-port`、`dht-listen-port`、`dht-message-timeout`、
   `enable-dht`、`enable-dht6`、`bt-tracker`、`conf-path`、`input-file`、`log`、
   `save-session*`、`dht-file-path*`、`bt-metadata-save-directory`、
   `save-upload-metadata-file-dir`、`enable-rpc`，另有 `rpc-` 与 `on-` 前缀。这些键分别
   由 `config.json`、端口自动切换、授权目录校验和任务回调独占；放行等于让前台绕过安全
   边界，历史上正是这类绕过造成过应用无法启动。任务级选项（`aria2.addUri` 的 `out`
   等）不持久化。
4. `fpk/cmd/main` 的 `merge_runtime_options` **必须两遍处理**。第一遍把覆盖文件净化成
   临时文件（键名字符集校验、独占项过滤、同名去重），第二遍才用"净化后的键集合"去基础
   配置里删除同名行并追加。若写成单遍——对覆盖文件里出现的每个键都先删基础配置行——
   被篡改过的覆盖文件只要包含 `rpc-secret=`，就能让基础配置里的真实密钥被删掉而又不被
   追加回来，直接导致 RPC 全线失败。这是实现过程中踩到的一次真实回归。
5. 合并失败一律返回成功并清理临时文件。覆盖文件损坏绝不能变成"应用起不来"；此时退回
   只含随包默认值的 `aria2.conf`。读取侧同理：解析器对坏行整体报错而不静默跳过，
   `load()` 遇到解析失败就忽略整个文件，避免坏行在下一次写入时被带回配置。
6. 数量与取值上限：最多 64 个键、键名 64 字节、取值 512 字节；键名必须匹配
   `^[a-z0-9][a-z0-9-]*$`，取值拒绝 `\x00`、`\r`、`\n` 和空串。数值与布尔按 JSON 字面量
   原样落盘，字符串取其内容，`null`、对象、数组一律拒绝，防止任意文本进入
   `aria2.conf`。
7. 提供 `GET /api/v1/runtime-options` 与 `DELETE /api/v1/runtime-options`（管理员）用于
   查看和清空。`--runtime-options` 未指定时由 `TRIM_PKGVAR` 推导；`TRIM_PKGVAR` 为空
   时整套机制自动禁用，写路径在 `path == ""` 处短路，单元测试与本机调试不受影响。

## 后果

- 用户在 AriaNg 里改的调优参数（并发数、分片、限速等）重启后仍然有效，而端口、目录、
  Tracker、回调这类受管键依旧只能走应用自己的设置页与 `config.json`。
- 多出一个需要随配置清理的状态文件；`uninstall_callback` 的"清空应用配置"清单已包含它，
  漏掉会让下一次安装继承上一次的运行期取值。
- 界面暂时只有只读回显与整体清空 API，没有"清空运行期选项"按钮；已记录为后续增强项。
- 覆盖文件是纯文本且可被有 root 权限的维护者直接编辑，因此净化逻辑独立于写入逻辑，
  两层互不信任。