# 阶段一审查清单

## 审查范围

阶段一负责项目初始化、需求分析、技术方案、架构、目录设计、开发计划和架构决策记录。该阶段不下载或编译业务运行文件，也不执行设备安装；阶段一完成后允许创建 Phase 2 空 FPK 骨架。

## 当前证据

| 检查项 | 结果 | 证据 |
| --- | --- | --- |
| 中文项目入口说明 | 已完成 | `README.md` |
| 协作规则 | 已完成 | `AGENTS.md` |
| 产品目标和约束 | 已完成 | `PROJECT_CONTEXT.md` |
| 功能、风险和非目标 | 已完成 | `docs/requirements.md` |
| 技术栈及选型理由 | 已完成 | `docs/technology-stack.md` |
| 飞牛能力与 SDK 职责边界 | 已完成 | `docs/fnos-integration-matrix.md` |
| 上游行为保真与安全差异 | 已完成 | `docs/upstream-behavior-matrix.md` |
| 组件、数据、网络边界 | 已完成 | `docs/architecture.md` |
| 源码与 FPK 目录职责 | 已完成 | `docs/directory-structure.md` |
| 需求到验收证据追踪 | 已完成 | `docs/requirements-traceability.md` |
| 分阶段开发计划 | 已完成 | `docs/development-plan.md` |
| 重要架构决策 | 已完成 | `docs/adr/ADR-001` 至 `ADR-008` |
| 许可证与产物溯源计划 | 已完成 | `docs/license-and-provenance-plan.md` |
| Web SDK 接口与安全契约 | 已完成 | `docs/web-sdk-integration-contract.md` |
| 持续推进机制 | 已完成 | `PROJECT_CONTEXT.md` 的“持续推进机制” |
| 发布前产品决策 | 已完成（2026-08-27） | `docs/phase1-decisions.md` 中 D-001 至 D-005 |

## 阶段一门槛判定

当前判定为 **已完成，Phase 2 已开始**。五项产品决策已于 2026-08-27 确认，且空 FPK 骨架已完成本地构建验证。

阶段一完成前，以下行为明确禁止；当前已进入 Phase 2 后，仍继续禁止下载或打包未审查的运行资产：

- （历史门槛）创建 `manager/`、`ui-adapter/` 等尚未到阶段的源码目录；
- （历史门槛）在未确认格式前生成生产用途的生命周期实现；
- （持续约束）下载、编译或复制未经审查的 Aria2 Core、AriaNg 到应用目录；
- （持续约束）在测试机安装或升级未确认范围的 FPK。

## 重新审查触发条件

出现以下任一情况时，应重新审查本阶段文档：

1. 用户确认或修改 D-001 至 D-005；
2. 用户新增端口、权限、下载目录、PT/Rclone 或多实例要求；
3. 飞牛文档、fnpack 工具或上游项目发生影响架构的版本变化；
4. 进入 Phase 2 前发现现有 ADR 与确认后的产品边界不一致。

## 进入下一阶段的操作

五项决策确认后已更新 `PROJECT_CONTEXT.md`、本表和决策表，并完成空 FPK 本地构建；
后续按 `docs/development-plan.md` 继续 Phase 2。阶段一文档在后续审计中仍作为边界和
追踪基线使用，不因进入实现阶段而删除或改写历史决策。
