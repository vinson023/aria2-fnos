# 飞牛 Web SDK 集成契约

本文件依据离线固定的 `@trimjs/web-app@0.4.2` 类型声明和 README，约束 Phase 5 的
UI 适配层如何使用飞牛宿主能力。它不是前端实现，也不授予浏览器额外的文件或服务端
权限。

## 适用范围与前提

- SDK 版本固定为 `0.4.2`，来源和哈希见 `docs/reference-baseline.md`。
- 仅在飞牛桌面 iframe 或受支持宿主中调用；宿主缺失、调用取消和返回错误均为正常可
  恢复状态，UI 不能假设调用必定成功。
- D-003 已于 2026-08-27 确认；目录选择和保存已由 `fnos-directory-adapter.js` 实现，仍必须遵循本契约的
  授权、规范化和服务端二次校验要求。
- SDK 返回的路径、授权标识、主题、语言和任何宿主消息都必须被视为前端输入。管理器
  仍是下载目录、归档目录和文件操作的最终授权与规范化检查点。

## 首版允许的 SDK 调用

| 场景 | 首选接口 | UI 行为 | 服务端必须再次验证 |
| --- | --- | --- | --- |
| 选择并授权管理员目录 | `pickUserFile`，选择目录；或 `authorizeUserFile` | 只提交成功回调中的候选路径 | 路径规范化、授权根、符号链接逃逸、应用用户可访问性 |
| 选择飞牛共享目录 | `pickSharedFile` 或 `authorizeSharedFile` | 只提交成功回调中的候选路径；由宿主控制共享目录选择语义 | 共享资源声明、授权根、实际读写权限 |
| 打开下载/归档位置 | `openFileManager` | 仅在管理器已确认该路径属于已授权目录后启用 | 不把 UI 参数直接传给 SDK 或本地文件操作 |
| 打开飞牛应用设置 | `openAppSetting` | 用于跳转到飞牛管理入口，不在 AriaNg 中显示 RPC secret | 无服务端文件或端口权限变化 |
| 主题、语言适配 | `getPlatformConfig`、`$on('os/theme')`、`$on('os/language')` | 只调整 AriaNg 外层显示和本地化 | 不影响 Aria2、网关或身份授权 |

## 禁止与限制

1. 不使用 `getHostSnapshot`、`query`、`refreshToken` 或宿主身份信息替代统一网关的
   服务端管理员校验。
2. 不向 AriaNg 暴露 SDK token、`X-Trim-*` Header、RPC secret 或任意宿主会话数据。
3. 不使用 `openURL` 把用户输入的 URL 当作可信跳转目标；第一版不需要该能力。
4. 不把 `pickFile` 的普通路径选择视为授权成功；需要目录授权时使用带授权语义的
   `pickUserFile`、`pickSharedFile`、`authorizeUserFile` 或 `authorizeSharedFile`。
5. SDK 目录授权不绕过 `config/resource`、`TRIM_DATA_SHARE_PATHS`、应用用户 UNIX 权限
   或管理器的路径白名单。

## 回调与错误处理

1. UI 将候选路径和用户动作提交给管理器，不在浏览器持久化“已授权”结论。
2. 管理器验证失败时，UI 显示通用错误；不得回显 secret、完整内部路径、认证 Header 或
   系统权限细节。
3. 用户取消选择、SDK 返回 `undefined` 或返回非成功响应时，不变更当前配置。
4. 配置持久化成功前，不更新 UI 的已生效目录或归档状态；端口/目录变更均由管理器
   返回最终状态。

## Phase 5 验收证据

- 用户目录和共享目录的成功、取消、拒绝、过期授权及路径逃逸用例均有记录。
- 浏览器请求、存储、SDK 调试输出与 UI 资源扫描不含 RPC secret、网关身份 Header 或
  宿主 token。
- SDK 不可用时页面仍能安全显示，涉及目录修改的操作失败但不会采用任意默认路径。
- 打开文件管理器只能定位到管理器已确认的下载或归档目录。

## 许可证待办

当前离线 npm 快照的 `package.json` 未声明 `license` 字段，包根目录也没有 `LICENSE`
文件。Phase 3/5 在实际将 SDK 或其构建产物纳入 FPK 前，必须从其权威发布来源确认
许可证、传递依赖和应包含的声明，并补入同版产物清单；未完成前不能把 SDK 作为可
发布资产声明为已审查。

## 参考依据

- `reference-snapshot/npm/trimjs-web-app/package-0.4.2/README.md`
- `reference-snapshot/npm/trimjs-web-app/package-0.4.2/dist/index.d.ts`
- `reference-snapshot/npm/trimjs-web-app/package-0.4.2/dist/types.d.ts`
- `docs/fnos-integration-matrix.md`
- `docs/security-model.md`
