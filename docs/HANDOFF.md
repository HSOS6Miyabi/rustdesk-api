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

## 当前已经提交的主要模块

| 位置 | 作用 |
| --- | --- |
| `model/device_policy_target.go`、`service/device_policy_target.go` | 设备、用户和设备组策略优先级 |
| `http/controller/admin/device_policy.go` | 策略分配、启停和生效策略查询 |
| `model/device_session.go`、`service/session_management.go` | 活动连接、结束请求、客户端心跳 |
| `http/controller/admin/device_session.go` | 管理员查看活动连接并请求结束 |
| `model/api_token.go`、`service/api_token.go` | 管理令牌、有效期和权限 |
| `http/middleware/device_command_auth.go` | 官方设备命令的登录令牌及管理令牌验证 |
| `http/controller/admin/api_token.go` | 创建、列出和撤销管理令牌 |
| `resources/admin/app.js` | 本轮新增的管理页面 |

## 已通过的分项测试

- [设备命令和部署](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42519664684)
- [三级策略分配](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520109512)
- [活动连接管理](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520568742)
- [Server 设备注册](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42519807127)

每次继续开发前，请查看草稿 PR #6 和 #4 的最新提交与检查结果，不要假设之前的运行结果适用于后来修改的代码。

## 共享地址簿扩展

本阶段还增加了附加群组和所有用户的地址簿共享。个人共享规则的界面位于 `resources/admin/app.js`，核心权限计算位于 `service/addressBook.go`。

接手时重点测试：

- 用户主要群组与附加群组同时生效。
- 指定用户规则优先于群组规则，群组规则优先于所有用户规则。
- 退出群组后重新获取共享地址簿，权限立即重新计算。
- RustDesk 客户端调用 `/api/ab/shared/profiles` 时返回最终有效权限。
- 公共锚点组的原有可见范围保持有效。

相关提交和测试结果记录于 [IMPLEMENTATION_LEDGER.md](IMPLEMENTATION_LEDGER.md)。

## 本轮新增的地址簿与邮件功能

- `service/address_book_csv.go`、`http/controller/admin/address_book_csv.go`：UTF-8 CSV 导入导出、账户访问范围与事务保存。
- `service/address_book_trash.go`、`http/controller/admin/address_book_trash.go`：误删条目恢复与永久删除。
- `model/addressBook.go`：新增软删除时间；历史记录保持可用。
- `service/mail.go`、`http/controller/admin/mail.go`：SMTP 连接方式、密码加密与测试邮件。
- `service/password_recovery.go`：邮箱密码恢复、单次使用链接与会话撤销。
- `service/user_invitation.go`、`http/controller/admin/invitation.go`：团队邮件邀请、按指定群组创建账户。
- `resources/admin/app.js`：CSV、回收站、SMTP、邀请及密码恢复的页面。

### 环境配置

在使用带密码的 SMTP 服务前设置 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，至少 16 个字符，升级和重启后继续使用同一个值。SMTP 服务地址、端口、连接方式、发件地址和公开控制台 URL 由管理员在网页中设置。密码恢复和邀请链接依赖公开控制台 URL。

新功能测试记录：[API 综合测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42560569953)。

### 接手后的优先工作

读取 [功能清单](FEATURE_MATRIX.md) 中尚未完成的功能，首先检查 TOTP、邮箱登录验证、控制角色、多中继服务器与自定义客户端功能，并继续保持功能实现、前端操作、自动测试和文档记录同步更新。

两个仓库仍处于草稿 PR 阶段。版本确认之前不发布 Release。
