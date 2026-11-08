# 开发交接说明

本项目面向小型团队，优先补齐官方客户端常用功能，并保留简单的默认部署方式。

## 当前工作

- API 开发分支：`development/m1-device-management`；草稿 PR [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)。
- Server 开发分支：`development/m1-device-management`；草稿 PR [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4)。
- 两个默认分支已经要求明确确认里程碑才可发布。禁止通过普通提交创建 Release。
- 第一个阶段包含设备命令行分配、可选设备部署登记和对应的服务器检查。
- 准确进展和测试结果记录于 [IMPLEMENTATION_LEDGER.md](IMPLEMENTATION_LEDGER.md)。

## 接手时从哪里开始

1. 分别克隆 API 和 Server 仓库，切换到两个仓库的 `development/m1-device-management` 分支，读取当前最新提交。
2. 查看两个草稿 PR 的提交和 GitHub Actions 结果。
3. 阅读 [FEATURE_MATRIX.md](FEATURE_MATRIX.md)、[ARCHITECTURE.md](ARCHITECTURE.md)、[TESTING.md](TESTING.md)。
4. 先运行现有 Go、Rust 和网页测试，再继续实现尚未完成的功能。
5. 每完成一个可独立验证的功能，就更新两个仓库的实施台账及相关兼容版本。

## 当前开发结构

- `http/controller/api/device_commands.go`：官方客户端设备分配、部署和内部登记校验接口。
- `model/device_deployment.go`：已部署设备的信息。
- `http/controller/admin/device_deployment.go`：部署记录查询。
- `cmd/apimain.go`：数据库迁移。
- `resources/admin/app.js`：设备备注、部署记录页面。
- Server：`src/device_deployment.rs` 和 `src/rendezvous_server.rs`。

## 必须保留的现有功能

个人群组、联系人单向共享、公共锚点组、地址簿、审计、WebClient v2、WebRTC、已有策略及多数据库支持。

## 发布限制

不要在开发过程中创建 `v*` 标签并勾选 `milestone_release` 执行发布工作流。只有里程碑功能完成、两个仓库测试成功且对应版本明确后，才进行正式发布。
