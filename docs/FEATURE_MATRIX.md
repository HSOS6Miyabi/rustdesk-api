# RustDesk 功能清单

接手开发与完整待办任务请查看 [实施台账](IMPLEMENTATION_LEDGER.md)。当前开发分支尚未批准发布，以下“已实现”均指开发代码及相应自动化测试，不包含待执行的真实客户端检查。

本项目面向个人和小型团队，尽量提供开源、自托管的 RustDesk 商业版常用功能。此表按照当前开发分支的代码维护，不能以“计划中”代替实际完成情况。

| 功能 | 当前情况 | 对应模块 | 后续工作 |
| --- | --- | --- | --- |
| 用户账户、密码登录 | 已实现 | `service/user.go` | 持续兼容客户端 |
| OIDC/LDAP | 已实现 | `service/oauth.go`、`service/ldap.go` | 测试更多身份服务 |
| 用户群组、公共锚点组 | 已实现 | `service/group.go`、`service/group_access.go` | 与访问控制分别管理 |
| 联系人单向共享与共享设备 | 已实现 | `http/controller/admin/contact.go` | 增加批量操作 |
| 个人及共享地址簿 | 已实现个人、群组和所有用户共享，含附加群组权限 | `service/addressBook.go`、`http/controller/api/ab.go` | 继续增强客户端兼容 |
| 地址簿 CSV 导入导出 | 开发分支已实现，支持普通账户及管理员 | `service/address_book_csv.go`、`http/controller/admin/address_book_csv.go` | 更多第三方格式 |
| 地址簿回收站 | 开发分支已实现删除、恢复和彻底删除 | `service/address_book_trash.go`、`http/controller/admin/address_book_trash.go` | 批量恢复 |
| 连接日志、文件传输及告警 | 已实现主要日志、远程结束会话与历史连接备注 | `service/audit.go`、`service/session_management.go`、`service/audit_note.go` | 更多查询条件 |
| WebClient v2、WebSocket、WebRTC | 已实现基本功能 | `resources/web2`、`resources/web/js` | 实际设备回归测试 |
| 设备策略心跳同步 | 开发分支已实现三级分配及优先级 | `service/device_policy.go`、`service/device_policy_target.go` | 扩充更多客户端设置项 |
| 官方客户端 `--assign` | 已实现 API 接口及集成测试 | `http/controller/api/device_commands.go` | 使用真实官方客户端完成分配操作测试（T001） |
| 官方客户端 `--deploy` | 已实现 API 接口和真实 hbbs 测试 | `http/controller/api/device_commands.go` | 使用真实官方客户端测试安装、部署与撤销（T001） |
| 管理设备部署记录 | 开发分支已实现撤销和恢复许可 | `http/controller/admin/device_deployment.go` | 批量管理 |
| 严格设备注册（可选） | 开发分支已实现并通过实际服务进程测试 | API 与 `rustdesk-server` | 真实客户端安装测试 |
| SMTP 邮件发送和管理 | 开发分支已实现 SMTP 设置、凭据加密和测试邮件 | `service/mail.go`、`http/controller/admin/mail.go` | 更多邮件发送通知 |
| 邮箱验证码登录 | 开发分支已实现可选启用、邮件绑定和原生客户端验证流程 | `service/email_login.go`、`http/controller/admin/email_verification.go` | 更多邮件模板及实际客户端测试 |
| 邮件邀请注册 | 开发分支已实现，含有效期和邀请撤销 | `service/user_invitation.go` | 批量邀请 |
| 邮箱密码恢复 | 开发分支已实现，含一次性恢复链接和旧会话撤销 | `service/password_recovery.go` | 更多恢复方式 |
| TOTP 与恢复码 | 开发分支已实现，完成 Go、原生客户端协议及浏览器自动化测试 | `service/totp.go`、`http/controller/admin/totp.go` | 实际客户端多平台测试 |
| 独立 API 管理令牌 | 开发分支已实现设备命令权限与到期管理 | `service/api_token.go`、`http/middleware/device_command_auth.go` | 扩充可用接口与权限类别 |
| 远程控制角色 | 开发分支已实现 14 项官方客户端会话权限、设备/用户/设备组分配、hbbs 协议传递 | `service/control_role.go`、Server `src/control_roles.rs` | 真实桌面连接及各操作权限测试 |
| 管理员分级角色 | 开发分支已实现多角色授权、细分操作权限及限定用户/群组的查看范围 | `service/admin_role.go`、`http/middleware/admin_privilege.go` | 增加更多限定范围的修改操作 |
| 服务器执行的连接访问规则 | 开发分支已实现可选严格模式、用户与群组规则、普通连接与中继授权 | `service/connection_access.go`、Server `src/connection_access.rs` | 跨平台客户端实际连接测试 |
| 多中继节点与 GeoLite2 国家识别 | 开发分支已实现 IP 网络、国家代码选择、自动同步和节点故障替换 | `service/relay_node.go`、Server `src/relay_policy.rs`、`src/geoip_country.rs` | 真实跨地区连接测试、可选 MMDB 文件支持 |
| 自定义客户端构建器 | 待实现 | 独立构建工作流 | 各系统安装包 |

相关代码与测试：API 草稿 PR [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)，Server 草稿 PR [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4)。

## 发布说明

发布采用里程碑确认制。日常开发及测试提交不创建 Release，也不推送发行版 Docker 镜像。只有在 `v*` 版本标签上手动启动工作流，并勾选 `milestone_release` 后才发布。

## 近期增加的管理界面

开发分支的管理员控制台包含以下入口：

- 设备部署记录：查看、撤销和恢复设备登记许可。
- 用户策略分配和设备组策略分配：批量设置客户端安全策略。
- 活动远程连接和结束连接记录：向客户端发送结束指令并查看执行情况。
- API 管理令牌：创建可独立撤销、具有指定操作权限的令牌。
- 个人地址簿 CSV：导入、导出，避免重复录入并保留连接凭据。
- 地址簿回收站：恢复误删设备条目，或将其彻底删除。
- SMTP 邮件服务：保存邮件设置并发送测试邮件。
- 成员邮件邀请与账户密码恢复：通过一次性邮件链接完成操作。

以上功能均在 GitHub 草稿 PR 中开发。当前公开 Release 仍应按照其对应版本的功能说明使用。

- 普通用户个人设置：可启用 TOTP、查看剩余恢复码数量、重新生成恢复码或解除双重认证。

- 中继节点管理：可设置中继地址、优先级、CIDR 网络范围、ISO 国家代码及启停。支持 GeoLite2 Country CSV、本地查询、运行时更新和节点断开后的替代选择。

- 远程控制角色：为设备、设备所有者或设备组指定权限，支持键盘、剪贴板、文件传输、终端、远程重启、录制、USB 等官方协议字段。

- 管理员分级角色：可授予普通成员具体管理权限；支持多个角色、全局角色、指定账户和用户组的只读目录权限。系统管理员保留全部管理功能。

- 可选的连接访问规则：指定哪些用户或用户组可访问单台设备、指定成员拥有的设备或某个设备组；显式禁止规则优先。默认保持原有连接方式。
