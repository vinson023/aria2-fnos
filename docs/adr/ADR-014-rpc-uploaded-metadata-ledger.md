# ADR-014：RPC 上传元数据由网关侧台账回收

- 状态：已采纳
- 日期：2026-09-05
- 关联：ADR-009（归档与做种默认值）、ADR-012（运行期选项持久化）、ADR-013（网关专属选项转发过滤）、`AGENTS.md`「所有路径、RPC 数据和配置输入均视为不可信」

## 背景

用户反馈并已在设备复现：删除未完成的任务会带走数据文件和 `.aria2` 控制文件，但下载目录里
仍然留着上传用的种子文件。

根因在上游默认值。Aria2 1.36.0 的 `rpc-save-upload-metadata` 出厂即为 `true`
（`src/OptionHandlerFactory.cc` 中该选项默认 `A2_V_TRUE`），设备侧 `getGlobalOption` 也读回
`"rpc-save-upload-metadata":"true"`。开启后 `addTorrent` / `addMetalink` 会把 RPC 收到的元数据
原文落盘，文件名规则见 `src/RpcMethodImpl.cc`：

```
filename = applyDir(dir, hexSha1(base64 解码后的元数据原文) + ".torrent" | ".meta4")
```

三个事实决定了它难以在常规位置处理：

1. 文件名只由元数据内容决定，与任务 gid 无关，也与下载出来的文件名无关。
2. `on-download-complete` / `on-download-error` 回调只能拿到 gid、文件数量和已下载文件的本地
   路径，拿不到上传时的元数据原文，因此钩子进程**无法**反推这个文件名。上游示例配置同样是
   只删 `<下载文件名>.torrent`，存在相同盲区。
3. `dir` 属于每任务可覆盖项，落盘目录不固定。

## 决策

由管理器承担回收责任，新增 gid 到元数据文件路径的**种子元数据台账**
（`$TRIM_PKGVAR/aria2-torrents.tsv`，实现见 `manager/cmd/aria2-manager/torrent_ledger.go`）：

1. 网关是所有 RPC 的必经点，`addTorrent` 的元数据原文一定经过它，因此在这里记账：
   `ledgerMutationFor` 在**注入 RPC 令牌之前**用客户端原始参数识别本次调用，避免密钥前插导致
   `params` 整体后移而取错参数。
2. 上游确认接受（HTTP 200 且 JSON-RPC 无 `error`）之后才落账，gid 取自响应 `result`。
3. `aria2.remove` / `forceRemove` / `removeDownloadResult` / `forceRemoveDownloadResult`
   成功后按 gid 取回路径并删除元数据文件；删除失败则回填记账，宁可残留也不丢归属。
4. `aria2.purgeDownloadResult`（前台「清空列表」）之后向 Aria2 询问 `tellActive` +
   `tellWaiting` + `tellStopped` 的完整存活集合，台账里剩下的才是真孤儿，连带删除其元数据文件。
5. `on-download-*` 钩子**故意不改**：它能拿到的信息不足以定位这些文件，把删除逻辑放进钩子只会
   制造「看起来实现了」的假象。归档与数据文件清理仍归钩子与 `/api/v1/cleanup` 负责。

## 安全与失败语义

- 记账前必须通过 `ledgerPathFor`：目标目录取任务显式 `dir`，否则退回全局下载目录；且必须落在
  `authorized_roots` 与 `TRIM_DATA_SHARE_PATHS` 之内（复用 `taskContainedRoot` 的符号链接解析）。
  越界直接放弃记账。
- 删除只接受 `^[0-9a-f]{40}\.(torrent|meta4)$` 形状的文件名，且必须是授权根内的普通文件，
  符号链接与非普通文件一律拒绝。台账本身按行解析，非法行只丢该行。
- 一切失败方向都是**残留**而不是误删：存活集合查询失败即放弃整轮清理；
  `prune(nil)` 语义为「未知，不清理」，与非 nil 空集合「确实没有任务」严格区分。
- 台账上限 1024 条，超限拒绝新增并记录日志，不静默丢账。文件 0600、临时文件加改名原子写入，
  行按字典序排序保证同一状态落盘字节一致。
- 已知局限：客户端在 `addTorrent` 里显式指定授权根之外的 `dir` 时，Aria2 自身仍会在该目录落盘，
  此时网关不记账（也就不会删）。这是「宁残留不误删」的刻意取舍，目录边界本身由 ADR-013 的
  `changeGlobalOption` 过滤与归档策略守住。

## 后果

- 卸载选择「清空应用配置」时，`aria2-torrents.tsv` 与 `.tsv.tmp` 一并删除；保留配置升级时台账
  跨重启继续生效，因此残留种子不会在升级后变成永久孤儿。
- 台账是派生数据而非用户数据：损坏或丢失的最坏结果等于本 ADR 之前的状态（残留一个 `.torrent`），
  不会造成任务不可用。
- 设备验收以 `tests/fnos-verify-torrent-cleanup.sh` 为准，期望「0 处失败」。