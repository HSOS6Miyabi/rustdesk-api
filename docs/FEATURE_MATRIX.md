# RustDesk 功能清单

以下清单记录当前分支中已经实现的功能，实际发布范围应与 GitHub Release 对应提交核对。详细模块位置见 [分支功能说明](FORK_CHANGELOG.md)，历史实施记录见 [实施台账](IMPLEMENTATION_LEDGER.md)。

本项目面向个人和小型团队，提供开源、自托管的 RustDesk 常用管理功能。本表以当前 `master` 分支和自动化测试结果为依据；实际设备、生产数据库及邮件服务商的检查另行列明。正式发行版 `v3.0.0-api.109` 的数据库迁移版本为 **290**。客户端构建器支持 Windows、Linux 和 macOS 的 x64/ARM64 官方 1.5.1 安装文件，均通过完整性检查。

| 功能 | 当前情况 | 对应模块 | 后续工作 |
| --- | --- | --- | --- |
| 用户账户、密码登录 | 已实现 | `service/user.go` | 持续兼容客户端 |
| OIDC/LDAP | 已实现 | `service/oauth.go`、`service/ldap.go` | 测试更多身份服务 |
| 用户群组、公共锚点组 | 已实现 | `service/group.go`、`service/group_access.go` | 与访问控制分别管理 |
| 联系人单向共享与共享设备 | 已实现，支持批量添加和撤销，自动测试通过 | `http/controller/admin/contact.go`、`contact_batch.go` | 真实团队账户测试 |
| 个人及共享地址簿 | 已实现个人、群组和所有用户共享，含附加群组权限 | `service/addressBook.go`、`http/controller/api/ab.go` | 继续增强客户端兼容 |
| 地址簿 CSV 导入导出 | 已实现，支持普通账户及管理员 | `service/address_book_csv.go`、`http/controller/admin/address_book_csv.go` | 更多第三方格式 |
| 地址簿回收站 | 已实现恢复、永久删除和批量处理，自动测试通过 | `service/address_book_trash.go`、`service/address_book_batch.go` | 正式数据备份与恢复 |
| 连接日志、文件传输及告警 | 已实现连接备注、日期与设备筛选及 CSV 导出，自动测试通过 | `service/audit.go`、`service/session_management.go`、`service/audit_note.go`、`http/controller/admin/peer_export.go` | 正式数据查询与导出 |
| WebClient v2、WebSocket、WebRTC | 支持网页全屏、显示器全屏和设备搜索 | `resources/web2`、`resources/web/js` | 在实际设备上检查远程连接 |
| 设备与用户详情 | 点击设备名称、设备 ID、用户名可查看相关信息；设备状态显示为在线、离线或暂时无法确定 | `resources/admin/app.js`、`resources/web2/app.js` | 根据访问权限显示管理操作 |
| 管理导航和数据表格 | 一级分类可折叠，导航滚动位置保留；表格支持当前页按列升序或降序排序 | `resources/admin/app.js`、`resources/admin/style.css` | 已加入电脑与手机浏览器测试 |
| 用户与设备选择器 | 在设置及关联记录编辑时搜索用户、设备和相关分组 | `resources/admin/app.js`、`service/directory_search.go` | 查询范围受账户权限限制 |
| 个人地址簿与共享规则编辑 | 保留未提交字段，根据登录账户确认记录归属 | `http/controller/admin/my/addressBook.go`、`addressBookCollection.go`、`addressBookCollectionRule.go` | 仅记录所有者可修改 |
| 设备标签透明颜色 | 支持保存所有有效 ARGB 整数颜色 | `http/request/admin/tag.go`、`resources/admin/app.js` | 颜色与透明度均可修改 |
| 前端接口检查 | 根据浏览器请求验证路由、响应格式和授权操作 | `scripts/test_frontend_api_contract.py` | 独立 SQLite 测试服务，无需访问生产数据库 |
| 服务器管理操作 | 根据操作说明填写参数，浏览器和 API 分别检查后执行 | `resources/admin/app.js`、`http/controller/admin/server_command_validation.go` | 需要配套 RustDesk Server 的管理端口 |
| 设备策略心跳同步 | 已实现三级分配及访问模式、键盘、剪贴板、文件传输和连接确认方式 | `service/device_policy.go`、`service/device_policy_target.go` | 真实客户端设置生效检查 |
| 官方客户端 `--assign` | 已实现 API 接口及集成测试 | `http/controller/api/device_commands.go` | 使用真实官方客户端完成分配操作测试（T001） |
| 官方客户端 `--deploy` | 已实现 API 接口和真实 hbbs 测试 | `http/controller/api/device_commands.go` | 使用真实官方客户端测试安装、部署与撤销（T001） |
| 管理设备部署记录 | 已实现批准、撤销及批量状态修改，自动化测试通过 | `http/controller/admin/device_deployment.go`、`device_deployment_batch.go` | 真实客户端完整操作 |
| 严格设备注册（可选） | 已实现并通过实际服务进程测试 | API 与 `rustdesk-server` | 真实客户端安装测试 |
| SMTP 邮件发送和管理 | 已实现 SMTP 凭据加密、测试邮件、事件通知及失败重试 | `service/mail.go`、`service/notification.go`、`http/controller/admin/notification.go` | 真实 SMTP 服务商及收件测试 |
| 邮箱验证码登录 | 已实现可选启用、邮件绑定和原生客户端验证流程 | `service/email_login.go`、`http/controller/admin/email_verification.go` | 更多邮件模板及实际客户端测试 |
| 邮件邀请注册 | 已实现，含有效期和邀请撤销 | `service/user_invitation.go` | 批量邀请 |
| 邮箱密码恢复 | 已实现，含一次性恢复链接和旧会话撤销 | `service/password_recovery.go` | 更多恢复方式 |
| TOTP 与恢复码 | 已实现，完成 Go、原生客户端协议及浏览器自动化测试 | `service/totp.go`、`http/controller/admin/totp.go` | 实际客户端多平台测试 |
| 独立 API 管理令牌 | 已实现细分管理权限、有效期、撤销和自动化接口 | `service/api_token.go`、`http/middleware/automation_auth.go`、`http/router/automation.go` | 实际自动化脚本测试 |
| 远程控制角色 | 已实现 14 项官方客户端会话权限、设备/用户/设备组分配、hbbs 协议传递 | `service/control_role.go`、Server `src/control_roles.rs` | 真实桌面连接及各操作权限测试 |
| 管理员分级角色 | 已实现多角色授权和限定用户/群组范围的用户、设备管理及所有者转移 | `service/admin_role.go`、`http/controller/admin/peer_transfer.go` | 真实团队管理测试 |
| 服务器执行的连接访问规则 | 已实现可选严格模式、用户与群组规则、普通连接与中继授权 | `service/connection_access.go`、Server `src/connection_access.rs` | 跨平台客户端实际连接测试 |
| 多中继节点与 GeoLite2 国家识别 | 已实现 IP 网络、国家代码选择、自动同步、节点故障替换与 MMDB/CSV 读取 | `service/relay_node.go`、Server `src/relay_policy.rs`、`src/geoip_country.rs` | 真实跨地区连接测试 |
| 独立 API 令牌与自动化管理 | 已实现细分操作权限、有效期、撤销及管理员身份复核，Go 和浏览器测试通过 | `http/router/automation.go`、`http/middleware/automation_auth.go` | 对接实际自动化脚本 |
| 批量联系人、共享、部署和回收站管理 | 已实现，支持事务检查与管理界面操作，Go 和浏览器测试通过 | `service/address_book_batch.go`、`http/controller/admin/contact_batch.go` | 在真实团队数据上检查操作 |
| 自定义客户端构建器 | 已实现 Windows、Linux、macOS 的 x64/ARM64 任务、校验下载和配置脚本打包，自动测试通过 | `service/client_build.go`、`service/client_packages.go`、`http/controller/admin/client_builder.go` | 对应系统真实安装、登录与连接检查 |

既有功能的合并记录包括 API [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 与 Server [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4)。设置中心与可靠性改进已通过 API [#11](https://github.com/HSOS6Miyabi/rustdesk-api/pull/11) 合并至 `master`，并纳入正式发行版 `v3.0.0-api.107`。

## 发布说明

发布采用里程碑确认制。日常开发及测试提交不创建 Release，也不推送发行版 Docker 镜像。只有在 `v*` 版本标签上手动启动工作流，并勾选 `milestone_release` 后才发布。

## 近期增加的管理界面

当前管理控制台包含以下入口：

- 设备部署记录：查看、撤销和恢复设备登记许可。
- 用户策略分配和设备组策略分配：批量设置客户端安全策略。
- 活动远程连接和结束连接记录：向客户端发送结束指令并查看执行情况。
- API 管理令牌：创建可独立撤销、具有指定操作权限的令牌。
- 个人地址簿 CSV：导入、导出，避免重复录入并保留连接凭据。
- 地址簿回收站：恢复误删设备条目，或将其彻底删除。
- SMTP 邮件服务：保存邮件设置并发送测试邮件。
- 成员邮件邀请与账户密码恢复：通过一次性邮件链接完成操作。

具体发布状态见 [兼容版本](COMPATIBILITY.md)，启用方法见 [部署说明](DEPLOYMENT.md)。

- 普通用户个人设置：可启用 TOTP、查看剩余恢复码数量、重新生成恢复码或解除双重认证。

- 中继节点管理：可设置中继地址、优先级、CIDR 网络范围、ISO 国家代码及启停。支持 GeoLite2 Country CSV、本地查询、运行时更新和节点断开后的替代选择。

- 远程控制角色：为设备、设备所有者或设备组指定权限，支持键盘、剪贴板、文件传输、终端、远程重启、录制、USB 等官方协议字段。

- 管理员分级角色：可授予普通成员具体管理权限，支持多个角色，以及指定账户或用户组范围内的查看、修改、删除和设备转移。系统管理员保留全部管理功能。

- 可选的连接访问规则：指定哪些用户或用户组可访问单台设备、指定成员拥有的设备或某个设备组；显式禁止规则优先。默认保持原有连接方式。

## 设置中心与运行可靠性

正式版 `v3.0.0-api.107` 已提供设置中心、网页配置覆盖、数据库迁移失败检查、通知失败隔离与共享处理保护，并改善管理员角色查询、批量推送和安装包任务处理。系统管理员可以查看配置当前运行值和来源，修改后按提示重新启动 API。使用方法见 [设置中心](SETTINGS_CENTER.md)、[性能测试](PERFORMANCE_TESTING.md) 和 [版本发行记录](releases/v3.0.0-api.107.md)。
