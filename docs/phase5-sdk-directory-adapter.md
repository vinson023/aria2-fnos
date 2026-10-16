# Phase 5：Web SDK 目录授权适配

本阶段纳入固定的 `@trimjs/web-app` `0.4.2` 浏览器构建文件，仅用于飞牛桌面
iframe 中的目录选择和文件管理器跳转。SDK 运行时不提供认证 Header、RPC secret
或管理器权限。

## 固定输入

- 来源：`reference-snapshot/npm/trimjs-web-app/package-0.4.2/dist/index.js`
- SHA-256：`681CF705A0C846362ADB567AD2EA155D86CB8E9E92C88E465D682F221AB366B7`
- 许可证：MIT，随包提供 `fpk/app/LICENSES/Trim-Web-App-MIT.txt`。
- FPK Scope：`trim.file.userAccess`、`trim.file.sharedAccess`；`manifest` 声明 `micro_app=true`。

## 适配行为

`fpk/app/www/fnos-directory-adapter.js` 使用 `pickUserFile({directory:true})` 或
`pickSharedFile()` 获得候选目录，再提交到 `/api/v1/config`。浏览器只提交候选值，
不保存授权结论；管理器重新解析目录、检查 `/volN` 数据卷、共享目录和应用可访问性，
并原子持久化 `authorized_roots`。启动时由管理器输出有效 `download_root`，再写入
Aria2 配置。打开文件管理器前也重新读取管理器返回的已验证目录。

SDK 缺失、用户取消、宿主拒绝或管理器拒绝时，不改变配置；界面只显示通用错误。
