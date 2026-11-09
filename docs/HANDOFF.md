# 开发交接说明

> **统一交接入口：[RustDesk OSS API 与 Server 完整实施台账](IMPLEMENTATION_LEDGER.md)。** 包含当前两仓库提交及 CI、完整功能状态、T001 至 T022 待办任务、测试命令、部署设置和发布条件。请先阅读台账第 1 至第 10 节，再按需阅读下方模块说明。

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

优先按照 [实施台账](IMPLEMENTATION_LEDGER.md) 中 T001 至 T012 的实际客户端、数据库和部署检查推进；之后根据 T013 至 T022 的优先级继续扩展自定义客户端构建器、更多客户端策略和其他功能。每个功能均需更新测试及交接记录。

两个仓库仍处于草稿 PR 阶段。版本确认之前不发布 Release。

## 历史连接备注

`model/audit.go` 的 `AuditConn` 保存 `note` 和 `note_updated_by`。`service/audit_note.go` 按连接记录编号修改备注。管理员 HTTP 接口位于 `http/controller/admin/audit_note.go`，管理页面位于 `resources/admin/app.js` 的“全部连接记录”。

执行 `go test ./service -run TestConnectionAuditNotes -count=1` 和 `go test ./http/controller/admin -run TestConnectionAuditNoteHTTPAndCSV -count=1` 检查备注与 CSV 导出，另使用 Chromium 管理界面测试确认编辑功能。

## 双重认证功能

TOTP 实现位于 `service/totp.go`，数据模型位于 `model/totp.go`。API 个人设置接口：`http/controller/admin/totp.go`。双阶段登录分别位于 `http/controller/api/login.go`、`http/controller/admin/login.go`，管理网页位于 `resources/admin/app.js`。

启动环境必须配置 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 才能绑定 TOTP；用户无需同时启用 SMTP。此密钥需要在数据库备份和服务重启时持续保持一致。更改密钥前必须处理已保存的 TOTP 密钥和 SMTP 密码。

在开发分支运行：
```bash
go test ./service -run 'TestTOTP|TestDeleteUser' -count=1
go test ./http/controller/api -run TestNativeRustDeskLoginRequiresTOTP -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

TOTP 数据和恢复码会随账户删除一并清理。具体提交和 CI 运行链接参见 [IMPLEMENTATION_LEDGER.md](IMPLEMENTATION_LEDGER.md)。

## 邮箱验证码登录

- `model/email_verification.go`：邮箱验证设置及一次性验证码请求。
- `service/email_login.go`：生成、发送、核对邮箱验证码。
- `service/second_factor.go`：根据登录请求选择 TOTP 或邮箱验证码。
- `http/controller/admin/email_verification.go`：个人设置接口。
- `http/controller/api/login.go`、`http/controller/admin/login.go`：官方客户端和 Web 登录。
- `resources/admin/app.js`：个人邮箱登录验证设置和验证码输入页面。

SMTP 服务必须事先配置；用户启用邮箱验证前，需要确认自己账户中的邮箱可接收邮件。当前邮箱发生变化时，之前确认的邮箱不会自动授权新邮箱。旧用户默认继续使用原有密码或 OIDC/LDAP 登录方式。

测试：

```bash
go test ./service -run 'TestEmailLogin|TestEmailCode' -count=1
go test ./http/controller/api -run TestNativeRustDeskEmailCodeLogin -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

具体提交见 [实施台账](IMPLEMENTATION_LEDGER.md)。

## 多中继节点配置

API：

- `model/relay_node.go`、`service/relay_node.go`：节点数据、地址和 CIDR 验证、优先级。
- `http/controller/admin/relay_node.go`：管理员增删修改及查询节点。
- `http/controller/api/internal_relay.go`：供 Server 读取的内部配置接口。
- `resources/admin/app.js`：中继节点管理页面。

Server：

- `src/relay_policy.rs`：获取 API 配置、解析 CIDR 和选择中继节点。
- `src/rendezvous_server.rs`：定时同步节点，执行在线检查及故障时替代选择。
- `scripts/test_managed_relays.py`：运行真实 `hbbs`、临时 API 和中继进行功能测试。

两端使用相同的 `RUSTDESK_API_INTERNAL_KEY`；Server 通过 `RUSTDESK_API_RELAY_NODES_URL` 查找内部接口。未配置同步地址时保留原有中继选择方法。

接手时应检查 API 和 Server 两个草稿 PR 的最新 CI，并运行上述真实服务测试。GeoLite2 Country CSV 国家识别已实现，详细内容见下文“GeoLite2 国家识别”。

## GeoLite2 国家识别

API 的 `RelayNode.Countries` 以逗号分隔保存两位国家代码，`EnabledRelayNodes` 向 hbbs 传输 `countries: ["JP","US"]`。该字段由数据库迁移版本 277 建立。

Server 的 `src/geoip_country.rs` 读取 GeoLite2 Country MMDB 或 CSV，`src/relay_policy.rs` 结合国家代码与精确 CIDR 规则选择中继。`src/rendezvous_server.rs` 在启动时载入数据，并使用本地控制命令 `reload-geo` 重新载入数据。

需要测试配置同步、地理位置选择和断线替代选择时，运行 `python3 scripts/test_managed_relays.py target/debug/hbbs`。

本功能的对应提交和 CI 详见 [实施台账](IMPLEMENTATION_LEDGER.md)，在完成里程碑前两个 PR 继续保持草稿状态。

## 远程控制角色

- `model/control_role.go`：14 项官方远程会话权限及 2 位编码。
- `service/control_role.go`：角色分配、优先级计算和数据维护。
- `http/controller/admin/control_role.go`：管理员创建、修改、删除角色及分配。
- `http/controller/api/internal_control.go`：供 hbbs 查询目标设备权限。
- `resources/admin/app.js`：控制角色编辑和分配页面。

API 数据库迁移版本为 278，包含 `control_roles`、`control_role_assignments` 两张表。Server 使用 `RUSTDESK_API_CONTROL_ROLE_URL` 连接 API，使用共同的 `RUSTDESK_API_INTERNAL_KEY` 验证内部请求。

角色优先级为单台设备、设备所有者、设备组；未分配角色时，客户端继续沿用原有权限。运行 `go test ./service -run TestControl -count=1` 和 `go test ./http/controller/api -run TestInternalControl -count=1` 检查后端，再运行管理页面测试。

两个仓库的最新提交和 CI 见 [实施台账](IMPLEMENTATION_LEDGER.md)。

## 管理员分级权限

核心文件：

- `model/admin_role.go`：管理员角色、用户与角色的多对多分配。
- `service/admin_role.go`：权限清单、角色范围检查和角色叠加。
- `service/admin_role_routes.go`：后台 HTTP 方法与管理权限的对应规则。
- `http/middleware/admin_privilege.go`：判断系统管理员或已委派成员的访问资格。
- `http/controller/admin/admin_role.go`：角色与分配的管理接口。
- `http/controller/admin/user.go`、`peer.go`：对限定用户/群组角色筛选列表和详情。
- `service/admin_mfa_reset.go`：管理员恢复账户双重认证。
- `resources/admin/app.js`：角色权限选择、分配和受委派成员可访问的网页。

管理员角色在迁移版本 279 中增加了 `admin_roles`、`admin_role_assignments`；当前开发分支数据库迁移版本为 280。系统管理员默认仍拥有全部权限。一个用户可被分配多个角色。

限定用户或用户组范围的角色支持查看、修改和删除所辖账户与设备，并可以在两个均属授权范围的账户之间转移设备。相关权限为 `users.view`、`users.edit`、`users.delete`、`devices.view`、`devices.edit_info`、`devices.delete` 和 `devices.transfer`。账户修改不得改变所属用户组或管理员身份，设备修改不得更换设备标识；批量删除和设备转移会检查目标归属。新增其他管理端点时，应先在 `AdminPermissionForRoute` 中登记权限，再逐项检查操作对象的归属。

继续开发时运行：

```bash
go test ./service -run 'TestAdministrator|TestDelegated|TestAdminResetSecondFactors' -count=1
go test ./http/controller/admin -run 'TestDelegatedAdmin|TestAdminRoles' -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

完成后根据 [实施台账](IMPLEMENTATION_LEDGER.md) 更新最新提交与 CI。

## 可选的服务器连接访问检查

API：

- `model/access_rule.go`、`service/connection_access.go`：规则结构、引用验证、用户和群组访问判断。
- `http/controller/admin/access_rule.go`：管理员增删修改规则。
- `http/controller/api/internal_connection_access.go`：检查设备 ID 和账户令牌的内部接口。
- `resources/admin/app.js`：连接访问规则管理页面。
- 数据库迁移版本 280，表名 `access_rules`。

Server：

- `src/connection_access.rs`：按需调用内部 API，返回连接许可。
- `src/rendezvous_server.rs`：普通连接及中继请求的授权检查。
- `scripts/test_connection_access.py`：启动真实 hbbs 测试连接许可及 API 不可用的情况。

启用需要在 hbbs 中设置 `RUSTDESK_API_CONNECTION_ACCESS_URL` 和 `RUSTDESK_API_INTERNAL_KEY`，其中内部密钥必须与 API 相同。

模式默认关闭。开启前先在管理页面添加需要的允许规则，并确认测试账户能够成功登录。严格模式下只有有效账户登录令牌才能参与访问判断；目标设备须在 API 中登记。设备所有者可以访问自己设备，其他账户按规则判断，明确禁止优先于允许。

提交和测试记录参见 [实施台账](IMPLEMENTATION_LEDGER.md)。实际桌面客户端测试应包括连接转发、中继连接及没有登录的客户端。

## 本阶段新增的自动化与批量管理

独立 API 令牌的权限在 `service/api_token.go` 中登记，验证由 `http/middleware/automation_auth.go` 执行，独立管理接口由 `http/router/automation.go` 注册。令牌所绑定的账户需保持系统管理员身份，权限不足的请求由中间件拒绝。详细权限及请求路径见 [API.md](API.md)。

新增批量管理分别由 `service/address_book_batch.go`、`http/controller/admin/contact_batch.go`、`http/controller/admin/address_book_rule_batch.go`、`http/controller/admin/device_deployment_batch.go` 处理；管理页面统一在 `resources/admin/app.js`。新增操作在数据库事务中处理，并限制每次处理数量。开发时同时运行后端 Go 测试与浏览器交互测试。

