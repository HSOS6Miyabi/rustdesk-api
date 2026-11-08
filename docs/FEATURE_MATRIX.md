# RustDesk 功能清单

本项目面向个人和小型团队，尽量提供开源、自托管的 RustDesk 商业版常用功能。此表按照当前开发分支的代码维护，不能以“计划中”代替实际完成情况。

| 功能 | 当前情况 | 对应模块 | 后续工作 |
| --- | --- | --- | --- |
| 用户账户、密码登录 | 已实现 | `service/user.go` | 持续兼容客户端 |
| OIDC/LDAP | 已实现 | `service/oauth.go`、`service/ldap.go` | 测试更多身份服务 |
| 用户群组、公共锚点组 | 已实现 | `service/group.go`、`service/group_members.go` | 与访问控制分别管理 |
| 联系人单向共享与共享设备 | 已实现 | `http/controller/admin/contact.go` | 增加批量操作 |
| 个人及共享地址簿 | 已实现基本功能 | `service/addressBook.go` | 跨组规则、批量导入导出 |
| 连接日志、文件传输及告警 | 已实现基本功能 | `service/audit.go` | 会话备注及远程结束 |
| WebClient v2、WebSocket、WebRTC | 已实现基本功能 | `resources/web2`、`resources/web/js` | 实际设备回归测试 |
| 设备策略心跳同步 | 已实现设备直接分配 | `service/device_policy.go` | 用户与设备组优先级 |
| 官方客户端 `--assign` | 开发中 | `http/controller/api/device_commands.go` | 执行集成测试及兼容性检查 |
| 官方客户端 `--deploy` | 开发中 | `http/controller/api/device_commands.go` | 配套服务端端到端测试 |
| 管理设备部署记录 | 开发中 | `http/controller/admin/device_deployment.go` | 操作管理、状态调整 |
| 严格设备注册（可选） | 开发中 | API 与 `rustdesk-server` | 客户端实测 |
| SMTP、邀请注册、密码恢复 | 待实现 | 账户服务 | 加入控制台 |
| TOTP 与恢复码 | 待实现 | 账户服务 | 原生客户端二次验证 |
| 独立 API 管理令牌 | 待实现 | 账户服务 | 权限与到期管理 |
| 管理员角色、控制角色 | 待实现 | API 与服务端 | 分级权限管理 |
| 服务器执行的连接访问规则 | 待实现 | API 与 `hbbs` | 可选严格模式 |
| 地理位置中继选择 | 待实现 | `hbbs` | 节点管理与故障替换 |
| 自定义客户端构建器 | 待实现 | 独立构建工作流 | 各系统安装包 |

相关代码与测试：API 草稿 PR [#6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)，Server 草稿 PR [#4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4)。

## 发布说明

发布采用里程碑确认制。日常开发及测试提交不创建 Release，也不推送发行版 Docker 镜像。只有在 `v*` 版本标签上手动启动工作流，并勾选 `milestone_release` 后才发布。
