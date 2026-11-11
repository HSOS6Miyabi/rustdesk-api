# 本分支相对于上游的功能改动

本仓库基于 [lejianwen/rustdesk-api](https://github.com/lejianwen/rustdesk-api) 持续维护，保留原项目账户、地址簿、Web 客户端及 Go API 的主要结构。下表按实际使用的功能整理增加的模块，便于接手开发人员理解用途、启用条件与受影响的位置。

当前正式发布为 [v3.0.0-api.108](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.108)。设备管理、用户搜索、管理导航和 WebClient 的相关说明见 [拉取请求 #13](https://github.com/HSOS6Miyabi/rustdesk-api/pull/13)。

| 功能 | 管理页面或使用入口 | 主要代码 | 使用条件 |
| --- | --- | --- | --- |
| 现代管理控制台与移动端布局 | `/_admin/` | `resources/admin/` | 导航分类可折叠，管理表格可排序，支持电脑与手机 |
| 个人设备与管理员设备管理 | 我的设备、设备管理 | `http/controller/admin/my/peer.go`、`resources/admin/app.js` | 个人设备操作检查所有者，管理员设备列表检查授权范围 |
| 设备与用户详情 | 设备列表、审计记录、搜索结果、共享设备 | `resources/admin/app.js`、`resources/web2/app.js` | 根据当前账户权限显示详情和管理入口 |
| WebClient 双全屏与设备搜索 | `/webclient2/` | `resources/web2/` | 支持网页全屏和显示器全屏，搜索本人和共享的设备 |
| 个人群组与公共成员 | 我的群组、用户分组 | `service/group_access.go`、`service/group_owned.go` | 用户自行创建群组或由管理员分配 |
| 联系人与共享设备 | 联系人、共享设备 | `service/shared_devices.go`、`model/contact.go` | 主动添加联系人或设置共享组 |
| 地址簿权限与导入导出 | 地址簿、地址簿回收站 | `service/addressBook.go`、`service/address_book_csv.go`、`service/address_book_trash.go` | 按用户或群组授予地址簿访问权限 |
| 官方客户端设备命令 | API 管理令牌、设备部署记录 | `http/controller/api/device_commands.go`、`service/api_token.go` | `--assign` 或 `--deploy`；严格登记需配套 hbbs 配置 |
| 设备、账户与设备组策略 | 设备安全策略、策略分配 | `service/device_policy.go`、`service/device_policy_target.go` | 创建策略并分配后生效 |
| 活动远程连接与结束请求 | 活动远程连接、结束连接记录 | `service/session_management.go` | 客户端通过心跳报告连接 |
| 控制角色与连接访问规则 | 远程控制角色、连接访问规则 | `service/control_role.go`、`service/connection_access.go` | 服务端必须启用相应内部接口；未配置规则时沿用原有连接处理 |
| 管理员委派权限 | 管理员角色、角色分配 | `service/admin_role.go`、`http/middleware/admin_privilege.go` | 管理员创建角色并指定作用范围 |
| SMTP、账户恢复、邮箱邀请 | SMTP 邮件服务、成员邀请 | `service/mail.go`、`service/password_recovery.go`、`service/user_invitation.go` | SMTP 可用；保存密码需长期保留加密密钥 |
| TOTP 与邮箱验证码 | 个人设置 | `service/totp.go`、`service/email_login.go` | 用户自行启用；邮件验证码还需要 SMTP |
| 通知与发送失败重试 | 邮件事件通知、消息推送 | `service/notification.go`、`service/push_delivery.go`、`service/event_outbox.go` | 管理员或用户启用订阅后处理事件 |
| 多中继和地理位置 | 中继节点管理 | `service/relay_node.go`、`service/geoip.go` | 需要多个中继时启用；国家识别还需本地 GeoLite2 数据 |
| 安装包生成与下载 | 客户端安装包构建 | `service/client_build.go`、`service/client_build_worker.go` | 选择官方客户端平台并建立任务 |
| 审计记录与保留期限 | 日志记录、日志保留设置 | `service/audit_management.go`、`model/audit_management.go` | 日志保留天数可分别配置 |
| 设置中心与启动配置覆盖 | 设置中心 | `config/settings.go`、`service/settings.go`、`http/controller/admin/settings.go` | 仅系统管理员；覆盖值优先于环境变量，修改后重新启动 API |

## 主要数据库变化

相对于上游新增了群组成员、联系人、策略分配、会话管理、管理员角色、个人推送和通知记录等业务表。现有数据库在启动时由 `cmd/apimain.go` 按迁移版本更新。当前正式版本 `v3.0.0-api.108` 使用数据库迁移版本 **290**。

设置中心增加 `system_setting_overrides`、`system_setting_revisions` 和 `system_setting_histories`。通知事件表还保存处理状态、失败次数、下一次重试时间和共享处理权的标识。升级前需备份数据库，回退旧版程序之前先核对其对新增字段的兼容性。

## 与 RustDesk Server 的配合

严格设备登记、服务器执行的连接访问规则、多中继动态选择与控制角色等功能需要 API 和本项目配套 Server 共同配置。各功能的内部访问地址、密钥和默认状态见 [部署说明](DEPLOYMENT.md)。默认关闭的功能无需增加周期性的远程检查。

## 接手开发

优先阅读 [功能清单](FEATURE_MATRIX.md)、[架构说明](ARCHITECTURE.md)、[API 使用说明](API.md)、[设置中心使用说明](SETTINGS_CENTER.md) 和 [上游合并流程](UPSTREAM_SYNC.md)。所有新功能应明确写出页面入口、启用条件、API、数据库变化、普通安装与 Docker 操作以及可执行的测试方法。