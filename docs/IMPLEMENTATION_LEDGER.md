# RustDesk OSS API 与 Server 完整实施台账

## 接手总览（截至 2026 年 11 月 9 日）

本文件是两仓库的**统一实施台账和工作交接入口**。最新交接快照：[HANDOFF_SNAPSHOT_2026-11-09.md](HANDOFF_SNAPSHOT_2026-11-09.md)；逐项状态、验证步骤和待办条件：[TASK_VERIFICATION_RECORD.md](TASK_VERIFICATION_RECORD.md)。本节记录当前结果与后续事项，后面的“历史实施记录”保留每项功能的具体提交、测试、设计和数据库修改过程。接手人员应先读取本节，再查阅历史记录中的相关功能。

### 1. 项目定位与开发原则

项目面向个人和小型团队，优先实现 RustDesk 官方客户端可直接使用的管理功能，保持普通服务器安装、Docker 和现有自托管用户数据的兼容性。API 提供账户、管理界面、地址簿、策略和内部授权接口；RustDesk Server 的 `hbbs`、`hbbr` 负责设备登记、连接协商和中继通信。

维护原则：

1. **优先补齐常用功能。** 新功能尽量由 API 和现有 RustDesk 客户端直接使用，不要求团队购买商业授权。
2. **保持默认安装简单。** 预先登记设备、服务器强制登录、集中连接访问检查、TOTP、邮箱登录验证码、托管中继和控制角色均保留可选设置。
3. **不得破坏已有数据。** 数据库修改使用版本迁移；升级前备份，重要修改同时验证 SQLite、MySQL 和 PostgreSQL。
4. **每次功能修改都完成代码、测试和文档。** 测试未通过时继续修正。每项任务必须记录修改位置、测试命令和实际结果。
5. **没有完成并确认里程碑前严禁发布。** 日常提交和草稿 PR 仅进行测试，不创建 Release、不更新正式发行镜像、不合并到默认分支。

### 2. 接手时的准确仓库状态

| 项目 | API | RustDesk Server |
| --- | --- | --- |
| GitHub 仓库 | [HSOS6Miyabi/rustdesk-api](https://github.com/HSOS6Miyabi/rustdesk-api) | [HSOS6Miyabi/rustdesk-server](https://github.com/HSOS6Miyabi/rustdesk-server) |
| 开发分支 | `development/m1-device-management` | `development/m1-device-management` |
| 默认分支 | `master` | `forapi` |
| 最近通过自动测试的代码提交 | [`31dbf4b7db0dd79f3cb3fda4b6ac2a1f6efb73d1`](https://github.com/HSOS6Miyabi/rustdesk-api/commit/31dbf4b7db0dd79f3cb3fda4b6ac2a1f6efb73d1) | [`66a2093127b452f2783d80386d2fd12c6501cae5`](https://github.com/HSOS6Miyabi/rustdesk-server/commit/66a2093127b452f2783d80386d2fd12c6501cae5) |
| 开发拉取请求 | [草稿 PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) | [草稿 PR #4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) |
| 综合 CI 检查 | [42635225816：成功](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42635225816) | [42634291061：成功](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42634291061) |
| 正式发行版本 | `v3.0.0-api.96`（2026-11-08） | `v1.1.17-api.9`（2026-11-08） |
| 里程碑发布 | 未执行 | 未执行 |

以上是已通过完整自动化测试的功能代码提交。文档修改可以产生更新的分支提交；部署时应同时查看两仓库分支最新提交、对应的 CI 结果和本台账记录。**API 与 Server 开发分支必须配合部署，不能直接把未发布的 API 功能与旧版 Server 混用并假定兼容。**

当前 API 数据库迁移版本：**282**。当前状态：代码仍在草稿 PR 内，没有合并，没有发行新版本，也没有更新正式发行镜像。

CI 已实际通过的内容：

- API：Go 测试、Go 编译、JavaScript 语法及交互测试、WebClient 构建与 WebRTC 检查、Chromium 桌面/平板/手机页面操作、三种数据库模拟升级及官方 Windows、Linux、macOS 客户端安装文件完整性检查。
- Server：Rust 编译及单元测试；实际 `hbbs`/`hbbr` 的 TCP、UDP、WebSocket、WebRTC 信令、登记及访问权限检查；AMD64/ARM64 的普通进程部署、Docker 组合部署、重启和数据保持测试。
- CI **没有**替代对正式 SMTP、实际多平台桌面客户端、真实互联网中继部署和生产数据库的检查。其具体待办项目列于下文。

### 3. 功能实施状态

状态“已实现且自动测试通过”表示当前开发分支代码和相应 CI 具备该功能；功能的真实客户端兼容性仍按“待办事项”检查。

| 编号 | 功能 | 当前状态 | 主要代码或说明 |
| --- | --- | --- | --- |
| F01 | 原有账户密码、OIDC、LDAP、群组、公共锚点组、联系人单向共享 | 原有功能保留，自动测试通过 | `service/user.go`、`service/group_access.go`、`service/oauth.go`、`service/ldap.go` |
| F02 | 官方客户端 `--assign`、`--deploy` | 已实现接口和集成测试；真实客户端安装与部署待验证 | `http/controller/api/device_commands.go` |
| F03 | 可选严格设备预先登记及撤销许可 | 已实现且真实 `hbbs` 进程测试通过；真实客户端待验证 | API `model/device_deployment.go`；Server `src/device_deployment.rs` |
| F04 | `MUST_LOGIN` 下的连接和中继令牌校验 | 已实现，Server 协议测试通过 | Server `src/jwt.rs`、`src/rendezvous_server.rs` |
| F05 | 设备、所属用户、设备组三层策略 | 已实现，含访问模式、键盘鼠标、剪贴板、文件传输及连接确认方式，Go 与浏览器测试通过 | `service/device_policy.go`、`service/device_policy_target.go` |
| F06 | 活动远程连接与向客户端发送结束请求 | 已实现且心跳与浏览器测试通过；真实桌面会话待验证 | `service/session_management.go` |
| F07 | 独立 API 管理令牌 | 已实现创建、到期、撤销、细分操作权限及专用自动化管理接口，测试通过 | `service/api_token.go`、`http/middleware/automation_auth.go`、`http/router/automation.go` |
| F08 | 共享地址簿、附加群组、所有用户共享 | 已实现且接口测试通过 | `service/addressBook.go`、`http/controller/api/ab.go` |
| F09 | 地址簿 CSV 导入导出与回收站 | 已实现，含批量恢复、永久删除和数据库事务测试 | `service/address_book_csv.go`、`service/address_book_trash.go`、`service/address_book_batch.go` |
| F10 | SMTP 配置、测试邮件、邀请注册和密码恢复 | 已实现，含本地模拟 SMTP 测试；实际服务商待验证 | `service/mail.go`、`service/user_invitation.go`、`service/password_recovery.go` |
| F11 | TOTP、六个一次性恢复码、邮箱验证码登录 | 已实现且协议及浏览器测试通过；真实多平台客户端待验证 | `service/totp.go`、`service/email_login.go` |
| F12 | 14 项远程控制角色及三级分配 | 已实现，`hbbs` 协议消息测试通过；真实桌面远程操作待验证 | API `service/control_role.go`；Server `src/control_roles.rs` |
| F13 | 管理员分级角色及多角色授权 | 已实现限定账户和用户组范围内的查看、编辑、删除及设备所有者转移，测试通过 | `service/admin_role.go`、`http/middleware/admin_privilege.go`、`http/controller/admin/peer_transfer.go` |
| F14 | 多中继节点、按 CIDR 和国家代码选择 | 已实现，含实际 `hbbs` 进程测试 | Server `src/relay_policy.rs`；API `service/relay_node.go` |
| F15 | 本地 GeoLite2 Country MMDB/CSV 识别 IPv4/IPv6、`reload-geo` | 已实现双格式读取与示例数据库测试；正式数据库由部署者提供 | Server `src/geoip_country.rs` |
| F16 | 服务器集中连接访问规则 | 已实现且真实服务测试通过；默认关闭；实际客户端待验证 | API `service/connection_access.go`；Server `src/connection_access.rs` |
| F17 | 连接、文件、登录与操作审计、历史连接备注和导出 | 已实现，含设备清单 CSV、审计日期和设备筛选、历史连接备注，自动测试通过 | `service/audit.go`、`service/audit_note.go`、`http/controller/admin/peer_export.go` |
| F18 | WebClient v2、WebSocket 和 WebRTC | 现有功能保留，CI 编译和浏览器测试通过；真实设备待验证 | `resources/web2`、`resources/web/js` |
| F19 | 可视化自定义客户端构建器 | 已实现 Windows、Linux、macOS 的 x64/ARM64 官方安装文件和配置脚本打包，自动测试通过；真实系统安装待检 | `service/client_build.go`、`service/client_build_worker.go`、`service/client_packages.go`、`http/controller/admin/client_builder.go` |
| F20 | GeoLite2 MMDB 读取、更多策略和 API 令牌功能、批量管理增强 | MMDB、客户端策略、独立管理令牌和四类批量管理已实现并通过自动化测试 | 见 T014–T018 |

### 4. 完整待办事项及处理顺序

优先级定义：**P0 为正式里程碑发布前必须完成的检查；P1 为下一阶段应实现的功能；P2 为后续使用体验和兼容扩展。** 所有条目均由接手人明确分配负责人、记录新提交并补充实际测试结果。

| 任务编号 | 优先级 | 状态 | 具体工作 | 完成判断 |
| --- | --- | --- | --- | --- |
| T001 | P0 | 官方客户端 Linux 设备部署与分配已在隔离本地 API 上通过；真实完整服务器重连待检查 | 使用一套真实 API、`hbbs`、`hbbr` 和官方 RustDesk 客户端运行 `--assign` 与 `--deploy`；验证 UUID、公钥、重复部署、设备归属及撤销许可 | 真实设备能够安装、注册、重新连接；预期拒绝的设备得到正确反馈 |
| T002 | P0 | 待执行 | 实测 Windows、Linux、macOS 及可用的 Android/iOS 客户端登录；分别检查密码、TOTP、邮箱验证码、OIDC/LDAP、恢复码及退出登录 | 支持的客户端版本与测试结果逐项记入记录；异常可重现并修复 |
| T003 | P0 | 待执行 | 使用两台真实设备验证远程控制角色：键盘、剪贴板、文件传输、音频、终端、录制、隐私模式、USB 等适用权限 | 禁止和允许设置在目标客户端实际生效，且未分配角色时行为保持不变 |
| T004 | P0 | 同一 hbbs 中严格登记、JWT、连接规则与控制角色组合测试通过；实际客户端和完整 API 集成待检查 | 在同一环境组合验证设备严格登记、`MUST_LOGIN=Y`、集中连接授权、中继连接、WebSocket 和 WebRTC | 合法连接成功，无权限连接被拒绝；API 不可用时严格模式按现有设计处理 |
| T005 | P0 | 待执行 | 实际配置公网和跨地区的多中继服务器，使用真正 GeoLite2 Country CSV 检查地区选择、节点失效及 `reload-geo` | 真实客户端能够选择可达节点，节点失效后改用其他中继 |
| T006 | P0 | 三种数据库模拟升级、备份和恢复通过；正式数据恢复演练待执行 | 在 SQLite、MySQL、PostgreSQL 上从旧结构迁移至数据库版本 282，备份并恢复历史记录，检查设备策略、审计和客户端构建任务 | 三种数据库迁移与回退演练成功，旧数据保持可读取 |
| T007 | P0 | 待执行 | 使用真实 SMTP 服务验证 STARTTLS/TLS、邀请、密码恢复和邮箱验证码；检查链接公开地址 | 收件人能完成整个流程，密码和原有账户资料保持正常 |
| T008 | P0 | 普通进程与 Docker 自动化部署及重启检查通过；正式环境待检查 | 使用常规进程和 Docker 部署两套完整系统，核对端口、内部 HTTP 服务名称、WebClient、数据目录和 GeoLite2 挂载 | 两种部署都可以运行、重启并恢复数据 |
| T009 | P0 | Chromium 多设备自动测试及截图已完成；人工使用检查待执行 | 浏览器人工查看桌面、平板、移动端页面；操作新功能，并记录截图；检查中文文本及错误提示 | 管理页面可独立完成主要管理操作，无阻碍使用的问题 |
| T010 | P0 | 官方多平台 1.5.1 启动和 Android 模拟器已通过，Linux --deploy/--assign 测试通过；真实登录及多版本连接待检查 | 检查新旧客户端的 API 与 Protobuf 版本兼容情况；核对官方版本更新是否改变了设备命令和双重认证字段 | 已支持的客户端版本清单明确，自动化测试能覆盖主要字段 |
| T011 | P0 | 已完成自动化预演 | 对正式发布工作流执行一次仅构建和部署检查的预演，验证跨架构可执行文件和 Docker 镜像启动 | 构建包与镜像可以运行，且在正式确认前不推送 Release 或正式镜像 |
| T012 | P0 | 待执行 | 汇总所有 P0 结果，更新功能清单和两个仓库的兼容版本，提交正式里程碑审核 | 未完成项目均明确标注，里程碑经维护者确认后才可发布 |
| T013 | P1 | 六个平台的安装文件准备与配置脚本已实现并通过自动测试；真实系统安装待检查 | 自定义客户端构建器：管理客户端服务器地址、公钥、预设值、构建任务、历史版本和安装包 | Web 页面能够创建构建任务、显示任务状态，并提供测试过的安装包 |
| T014 | P1 | 已实现并通过心跳及界面自动测试；实际远程设备待检查 | 扩充设备策略项目，研究官方客户端支持的更多连接和安全设置 | 管理页面能配置新增项目，客户端心跳后能够应用并撤销 |
| T015 | P1 | 已实现，CI 通过 | 扩展独立 API 令牌权限至适用的只读、设备和自动化管理接口 | 权限可分别选择，超出授权范围的请求被拒绝 |
| T016 | P1 | 已实现，CI 通过 | 允许管理员在特定账户或用户组范围内执行经过检查的修改操作 | 编辑、删除和转移设备时均检查目标归属，不影响其他用户 |
| T017 | P1 | 已实现，CI 通过 | 联系人、共享地址簿、设备部署和回收站的批量导入、共享、恢复及整理 | 管理界面能批量处理，失败时明确说明未成功的项目 |
| T018 | P1 | 已实现，Rust CI 通过 | 可选 GeoLite2 MMDB 文件支持，并与现有 Country CSV 方式共存 | 两种数据库格式的国家判断结果一致，未配置时正常选择中继 |
| T019 | P1 | 邮件通知及重试已实现，模拟 SMTP 测试通过；正式 SMTP 待检查 | 活动会话结束、登录异常、部署状态、文件传输等消息的邮件通知 | 管理员可选择通知项目，消息发送结果可以查询 |
| T020 | P1 | 设备和审计筛选、CSV 导出及备注功能已实现，自动测试通过 | 为设备、共享地址簿及连接审计增加更实用的筛选、导出及备注管理 | 小型团队可快速查询、导出和维护历史记录 |
| T021 | P2 | 官方 1.5.1 功能候选已整理；持续版本检查保留 | 官方商业版新增功能与 RustDesk 客户端更新情况，定期更新功能清单 | 每个新增候选功能附使用场景、客户端版本和预计修改位置 |
| T022 | P2 | 中文管理界面及移动端新增页面测试通过；其他语言继续完善 | 文档与界面的多语言、移动端操作、管理表格可读性 | 中文文档完整，其他语言与终端界面符合目标用户需求 |

P0 检查属于正式发布前的必要工作，不妨碍下一位开发者继续在开发分支增加其他功能。**完成代码自动化测试与完成真实客户端验证分别记录，不得互相代替。**

### 5. 重要实现细节与当前限制

**官方客户端兼容。** `/api/devices/cli` 和 `/api/devices/deploy` 已接入，端到端自动测试使用 Gin/SQLite 和真实 `hbbs`，最终还应使用官方客户端命令进行完整操作。独立 API 令牌支持设备分配、部署及细分的自动化管理权限，通过 `Authorization: Bearer rdapi_...` 调用 `/api/automation/v1`。管理接口每次验证令牌权限和所属账户的系统管理员身份，令牌摘要与普通登录令牌分开保存。

**设备策略与控制角色。** 两者分别管理：设备策略经客户端心跳应用，优先级为单设备、用户、设备组；控制角色在连接协商时写入 `ControlPermissions`，按目标设备、所属用户、设备组选择。实际客户端的控制操作仍需验证。

**集中连接访问。** 只在 `RUSTDESK_API_CONNECTION_ACCESS_URL` 配置后启用。API 检查登录令牌、已登记的目标设备和访问规则；设备所有者始终可以连接自己的设备；其他用户需匹配允许规则，明确的禁止规则优先。API 访问失败时拒绝严格模式的新连接。直接使用 IP 地址连接设备的方式不经 `hbbs` 的连接协商处理。

**管理员角色。** 一个用户可拥有多个角色。具有 `is_admin=true` 的现有系统管理员保留全部管理权限。当前限定用户与用户组的角色支持所属账户和设备的查看、受限编辑、删除与设备所有者转移。修改账户时不得变更群组及管理员身份，转移设备时同时检查原所有者和目标所有者。管理角色分配本身只允许系统管理员操作。

**邮箱与认证。** TOTP 和邮箱验证码登录均为用户可选。SMTP 密码和 TOTP 密钥依赖持久化的 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，该密钥必须与数据库一起备份；变更密钥前需要重新处理已保存的加密资料。普通用户重置密码会撤销原有登录会话；管理员帮助成员重置第二因素时也会撤销已有会话。

**多中继。** `hbbs` 可根据活动中继、CIDR 网络规则和 GeoLite2 国家数据选择节点；管理员指定的 CIDR 规则优先。GeoLite2 Country MMDB 文件或 CSV 目录由部署者自行提供，程序未包含正式地理位置数据库。未配置托管中继时使用原有中继配置。

**数据库与兼容。** 当前 API 迁移版本 280；其中，设备登记、三级策略、会话管理、独立令牌、共享地址簿、SMTP/TOTP/邮箱验证码、GeoLite2 节点设置、控制角色、管理员角色和访问规则分别有对应的新表或字段。历史数据库必须先备份再迁移。详见 [数据库升级说明](UPGRADE.md)。

### 6. 双仓库代码索引

| 功能 | API 主要文件 | Server 主要文件 |
| --- | --- | --- |
| 设备部署 | `http/controller/api/device_commands.go`、`model/device_deployment.go` | `src/device_deployment.rs`、`src/rendezvous_server.rs` |
| 连接许可 | `model/access_rule.go`、`service/connection_access.go`、`http/controller/api/internal_connection_access.go` | `src/connection_access.rs`、`src/rendezvous_server.rs` |
| 控制角色 | `model/control_role.go`、`service/control_role.go`、`http/controller/api/internal_control.go` | `src/control_roles.rs`、`src/rendezvous_server.rs` |
| 多中继和国家识别 | `model/relay_node.go`、`service/relay_node.go`、`http/controller/api/internal_relay.go` | `src/relay_policy.rs`、`src/geoip_country.rs` |
| 管理员角色 | `model/admin_role.go`、`service/admin_role.go`、`service/admin_role_routes.go`、`http/middleware/admin_privilege.go` | 无须修改 Server |
| 设备策略 | `service/device_policy.go`、`service/device_policy_target.go` | 已有 hbbs 兼容模块 |
| 活动连接 | `service/session_management.go`、`http/controller/admin/device_session.go` | 与客户端心跳协议配合 |
| 地址簿 | `service/addressBook.go`、`service/address_book_csv.go`、`service/address_book_trash.go` | 通过现有 RustDesk 客户端 API |
| 邮件与账户 | `service/mail.go`、`service/totp.go`、`service/email_login.go`、`service/password_recovery.go`、`service/user_invitation.go` | `src/jwt.rs` 负责部分连接令牌检查 |
| Web 管理页面 | `resources/admin/app.js`、`resources/admin/browser.e2e.cjs` | 无 |
| 发布与验证 | `.github/workflows/verify-release.yml` | `.github/workflows/verify-server.yml`、`.github/workflows/release-server.yml` |

完整接口说明见 [API.md](API.md)，部署说明见 [DEPLOYMENT.md](DEPLOYMENT.md)，测试命令见 [TESTING.md](TESTING.md)，架构说明见 [ARCHITECTURE.md](ARCHITECTURE.md)。

### 7. 两仓库内部接口和环境配置

| 作用 | hbbs 关键环境变量 | API 内部地址 |
| --- | --- | --- |
| 设备预先登记 | `REQUIRE_DEVICE_DEPLOYMENT=Y`、`RUSTDESK_API_INTERNAL_URL` | `POST /internal/v1/device/authorize` |
| 多中继动态同步 | `RUSTDESK_API_RELAY_NODES_URL`、`RUSTDESK_API_RELAY_SYNC_SECONDS` | `GET /internal/v1/relay/nodes` |
| 控制角色 | `RUSTDESK_API_CONTROL_ROLE_URL`、`RUSTDESK_API_CONTROL_CACHE_SECONDS` | `GET /internal/v1/control/permissions` |
| 集中连接许可 | `RUSTDESK_API_CONNECTION_ACCESS_URL` | `POST /internal/v1/connection/authorize` |
| 令牌验证 | `MUST_LOGIN=Y`、`RUSTDESK_API_JWT_KEY` | 与 API 共同使用 JWT 密钥 |
| 内部通信 | `RUSTDESK_API_INTERNAL_KEY` | API 与 Server 必须一致 |
| GeoLite2 Country 数据 | `RUSTDESK_GEOIP_MMDB_FILE` 或 `RUSTDESK_GEOIP_CSV_DIR` | 从本地 MMDB/CSV 文件读取，无 HTTP 查询 |
| SMTP 与 TOTP 密钥 | API 端 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` | API 本地使用；须备份 |

这些变量均按实际部署决定是否配置。内部地址样例中的 `127.0.0.1` 仅适用于同一网络命名空间中的服务；Docker 跨容器访问请使用可解析的服务名称或实际内部地址。禁止在文档、提交或日志中记录真实密钥。

### 8. 测试和复现方法

**API 开发与完整测试**

```bash
git clone --branch development/m1-device-management https://github.com/HSOS6Miyabi/rustdesk-api.git
cd rustdesk-api
go test ./... -count=1
go build -o /tmp/rustdesk-api ./cmd/apimain.go
node --check resources/admin/app.js
node --test resources/admin/app.test.cjs
node --test resources/web2/app.test.cjs
cd resources/web/js
npm ci
npm run build
node --test tests/webrtc.test.cjs tests/websock.test.cjs
```

浏览器测试在 API 项目根目录运行；需预先安装 Playwright 和 Chromium：

```bash
npm --prefix resources/admin install --no-save --no-package-lock playwright@1.56.0
resources/admin/node_modules/.bin/playwright install --with-deps chromium
node --test resources/admin/browser.e2e.cjs resources/admin/browser.webrtc.e2e.cjs
```

涉及 Redis 的测试设置 `REDIS_TEST_ADDR`；GitHub Actions 会自动启动测试 Redis。具体要求见 [TESTING.md](TESTING.md)。

**Server 开发与真实服务进程检查**

```bash
git clone --recurse-submodules --branch development/m1-device-management https://github.com/HSOS6Miyabi/rustdesk-server.git
cd rustdesk-server
git submodule update --init --recursive
cargo test --lib --tests
cargo build --bins
python3 scripts/test_runtime.py target/debug/hbbs target/debug/hbbr
python3 scripts/test_device_deployment.py target/debug/hbbs
python3 scripts/test_managed_relays.py target/debug/hbbs
python3 scripts/test_control_roles.py target/debug/hbbs
python3 scripts/test_connection_access.py target/debug/hbbs
```

具体测试脚本使用本地临时 API 或模拟中继，尽量避免依赖外部服务。这里列出的检查与实际公网跨地区、真实客户端、正式 SMTP 和旧数据库升级测试仍需分别执行。

### 9. 里程碑和发布控制

目前首个里程碑**尚未正式确认完成**。当前两个草稿 PR 的成功 CI 是开发检查结果，不意味着已经批准发布。

发布之前依次完成：

1. 将 P0 任务的结果写入本台账；未完成时说明原因及影响。
2. 固定两仓库相互兼容的提交和客户端测试版本。
3. 更新 [FEATURE_MATRIX.md](FEATURE_MATRIX.md)、[UPGRADE.md](UPGRADE.md) 和部署文档。
4. 备份正式数据库并完成迁移、回退及普通进程与 Docker 部署检查。
5. 获取维护者对里程碑完成及版本编号的明确确认。
6. 仅在相应 `v*` 版本标签上手动运行 GitHub Actions，选择 `milestone_release`，再创建 Release 和正式镜像。

当前工作流的发布任务具有 `workflow_dispatch`、`milestone_release` 和 `v*` 标签条件。不要在日常开发中触发发布，也不要使用合并或自动提交绕过上述条件。

### 10. 接手人员开始工作的顺序

1. 查看 [API PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 和 [Server PR #4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4) 的最新提交、检查与审查意见。
2. 按第 2 节记录的提交检查仓库。如果分支已有新提交，以实际代码为准，检查本台账与新增提交的差异后再继续修改。
3. 运行第 8 节所列测试；若当前环境没有真实客户端或生产数据库，则明确记录尚未执行的检查。
4. 优先认领第 4 节 P0 项目，其后依使用需求推进 P1；可以把每个任务编号写入 PR 描述或 Issue。
5. 实现新功能时，在相应功能模块增加自动化测试，实际运行并核查结果；更新本节功能表、待办状态、提交与 CI 链接。
6. 保持两仓库草稿 PR，直到满足第 9 节的里程碑发布条件。

**本次交接不包含**真实 SMTP 凭据、数据库密码、JWT 密钥或 GeoLite2 授权资料。接手人员需要从部署环境的合法管理者处获取这些配置。

### 11. 开发记录维护方法

本文件的第 2 节只记录一个经过检查的双仓库版本组合。每次形成新的稳定开发版本时，用最新提交、CI URL 和数据库版本更新该节。任务完成时应记录任务编号、代码位置、提交、测试命令、实际结果、影响的部署设置和剩余工作。历史提交清单保留在下方，按功能和时间连续追加，便于后来的人阅读项目演进。

---

## 历史实施记录

以下各节保留开发过程中每项功能的详细提交记录。早期章节描述的是当时的完成程度和计划；**当前功能状态、待办事项和已检查版本以本文件第 2 至第 4 节为准**。

## 2026-11-09：第一个开发阶段开始

目标：兼容官方客户端设备分配与部署命令，建立可选设备登记检查，并保留现有小型团队默认注册方式。

仓库：
- API：`HSOS6Miyabi/rustdesk-api`，开发分支 `development/m1-device-management`
- Server：`HSOS6Miyabi/rustdesk-server`，开发分支 `development/m1-device-management`

### 发布流程

| 仓库 | 提交 | 更改 |
| --- | --- | --- |
| API | [b9da00b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/b9da00b71d3ff43b0c943a8817302b71e0548d3d) | 只有手动确认里程碑的标签工作流可以发布 |
| Server | [b4e1fb6](https://github.com/HSOS6Miyabi/rustdesk-server/commit/b4e1fb6ec93c74196cd91461b710176c07f0a8cc) | 仅允许手动确认后发布二进制程序和镜像 |

### 功能修改

| 仓库 | 提交 | 代码与行为 |
| --- | --- | --- |
| API | [edbeb5d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/edbeb5d374fe4c46084d9aaac742eab2a318a5ae) | 官方 `/api/devices/cli`、`/api/devices/deploy`；设备登记记录；内部登记检查；迁移和配置 |
| API | [0e66647](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0e666477447fddbe6bcc7ef14d1f942b4d9f259a) | 后台部署记录、设备备注、命令行分配及登记接口测试 |
| Server | [33a6108](https://github.com/HSOS6Miyabi/rustdesk-server/commit/33a6108638d7cbc7bb240fd03768355a4bb6b105) | `RegisterPk` 可选 API 登记检查、`RequestRelay` 令牌检查、缺少 JWT 密钥时拒绝启动 |

### 配置、数据库及兼容性

API 新增 `app.require-device-deployment` 和 `app.internal-key`；数据库自动迁移 `DeviceDeployment` 表、设备和地址簿的 `note` 字段。

Server 新增 `REQUIRE_DEVICE_DEPLOYMENT`、`RUSTDESK_API_INTERNAL_KEY`、`RUSTDESK_API_INTERNAL_URL`。

严格登记默认关闭，API 的 `--deploy` 返回 `NOT_ENABLED`。启用时必须同时配置 API 和 Server，并使用相同的内部访问密钥。现有设备与地址簿原始记录不删除。

### 测试和后续检查

测试说明见 [TESTING.md](TESTING.md)。本阶段开发分支通过草稿 PR 执行 CI，实际通过情况以 PR 的检查结果为准。连接真实桌面、文件传输及严格设备部署还需要在具备客户端的环境验证。

待实施功能持续更新于 [FEATURE_MATRIX.md](FEATURE_MATRIX.md)；交接说明见 [HANDOFF.md](HANDOFF.md)。

## 2026-11-09：设备管理完善、三级策略、活动连接与 API 令牌

此阶段继续使用两个仓库的 `development/m1-device-management` 分支，所有修改通过草稿 PR 检查。以下是本轮新增和扩展功能的具体提交。

| 仓库 | 提交 | 功能 |
| --- | --- | --- |
| API | [47a71c2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/47a71c288a6dfae9028d7216b978ebc4fc0ce675) | 完善部署管理路由和 SQLite 集成测试 |
| API | [97e3ef3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/97e3ef3235f4f5c89df24cb9e1637126cd2c30e9) | 可撤销和恢复设备部署许可，并检查对应设备身份 |
| API | [79c2d9d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/79c2d9d7fbc5617ede0e7dfd416a34c057a3c08d) | API 与 Server 支持共同的设备部署环境变量 |
| API | [bdf646a](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bdf646ae81f76ce62edaffe87faddf9e5e899a56) | 加入设备、用户、设备组三层策略和后台管理 |
| API | [2a64af3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a64af37b3ba92f0650f744ece62428acaf286da) | 验证策略优先级、禁用处理、撤销和数据清理 |
| API | [722c595](https://github.com/HSOS6Miyabi/rustdesk-api/commit/722c5958f50949b2faaa7fe237b96b6711f100d8) | 记录客户端活动会话并通过心跳发送结束指令 |
| API | [ff2facd](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ff2facdec81e87a0f20bec769169ccc9245f4c3d) | 活动连接与结束连接记录管理页面 |
| API | [bb7a1c9](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bb7a1c9ff54956e437c1c8d62afbd25cf6db405b) | 会话心跳、指令执行和管理页面自动化测试 |
| API | [0c570fb](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0c570fb77ca6daf1599659993ff9dba1ac396ba4) | 独立 API 管理令牌、权限校验和自动化设备接口 |
| API | [c6135a1](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c6135a151a22ff3bc55dab38b00cd26b6a67da9a) | 管理控制台的令牌创建、查看与撤销 |
| API | [81a656b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/81a656be715b8c9c4e31b53b134dfb1c9d45fd93) | API 令牌权限与官方设备命令集成测试 |
| API | [468bddf](https://github.com/HSOS6Miyabi/rustdesk-api/commit/468bddfbb90521d559cea0a31a8aadc79cf372d7) | 改善客户端认证组件的初始化检查 |
| API | [5310011](https://github.com/HSOS6Miyabi/rustdesk-api/commit/53100114da346d9d544a8686f9340920bebe763c) | 调整令牌创建后的即时显示 |
| API | [c4b1302](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c4b130274e80910878e3992859103eb77cac03b6) | 令牌创建页面独立处理保存操作 |
| Server | [8995074](https://github.com/HSOS6Miyabi/rustdesk-server/commit/89950741e6c156d6ed7cb24c348a97441f3c0ce0) | 增加服务端实施台账与接手文档 |
| Server | [cd67739](https://github.com/HSOS6Miyabi/rustdesk-server/commit/cd6773901134fd38ca566af73de906e014257637) | 真实 `hbbs` 设备登记通信测试 |
| Server | [ec48c9b](https://github.com/HSOS6Miyabi/rustdesk-server/commit/ec48c9bb5414d93d9b486cc158cbc42216fb8199) | 设备恢复连接时重新检查登记许可 |
| Server | [bfd1d8e](https://github.com/HSOS6Miyabi/rustdesk-server/commit/bfd1d8e2ca3f7d513e7686f42ead026826606b95) | 仅接受 API 明确启用后的严格登记许可 |
| Server | [902293d](https://github.com/HSOS6Miyabi/rustdesk-server/commit/902293d49f17d831922237718ff389b3a1178267) | 完善服务端环境变量、测试说明和接手文档 |

### 数据变更与兼容性

新增 `DevicePolicyUserAssignment`、`DevicePolicyGroupAssignment`、`DeviceActiveConnection`、`DeviceDisconnectRequest` 和 `APIToken`，并为 `DevicePolicy` 增加启停状态。旧策略保持有效。

活动连接使用客户端 `/api/heartbeat` 中的 `conns` 信息；结束指令使用 `disconnect`，对不支持报告活动连接的旧客户端不会显示虚假的活动会话。管理令牌采用 `rdapi_` 前缀，仅用于有相应权限的设备命令接口。

### 已确认的自动化测试

- [API 设备命令、部署管理与数据库测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42519664684)。
- [API 三级策略、回退规则及浏览器测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520109512)。
- [API 活动连接、心跳指令及浏览器测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42520568742)。
- [Server 严格设备登记、通信协议和 Rust 测试通过](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42519807127)。
- [Server 文档更新后的自动化测试通过](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42521370716)。

API 管理令牌的最终浏览器测试及完整 Go 测试，应以[API 草稿 PR](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 最新检查结果为准。

### 后续功能

商业版仍待实现的功能以 [FEATURE_MATRIX.md](FEATURE_MATRIX.md) 为准。继续开发时，更新本台账、对应提交编号、测试结果和 [HANDOFF.md](HANDOFF.md)。

## 2026-11-09：共享地址簿的附加群组与所有用户共享

这一组修改主要帮助小型团队快速共享常用设备列表。新增功能保留现有个人群组、共享组及公共锚点组的行为。

| 仓库 | 提交 | 完成内容 |
| --- | --- | --- |
| API | [ef495dc](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ef495dc3fea1a615dc152200e9dac64da4ee9ba7) | 支持附加群组和所有用户共享；客户端共享地址簿列表使用实际权限 |
| API | [5938de3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/5938de324923bbc2e5e0ddbff0a412f8e4c7f243) | 覆盖多群组共享、权限优先级、所有用户规则修改及管理页面测试 |
| API | [bafa58d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/bafa58da217efab387eb3db4b46e7f8f1f746f5c) | 验证原生客户端 `/api/ab/shared/profiles` 在多种权限组合下的响应 |

### 主要修改

- `service/addressBook.go`：读取个人、所有群组及公开共享规则，计算具体用户、群组和公开规则的优先级。
- `model/addressBook.go`：共享对象新增“所有用户”，使用 `type=3, to_id=0`。
- `http/controller/api/ab.go`：返回每个地址簿的最终权限，保留公共锚点组原有的可见范围。
- `http/controller/admin/addressBookCollectionRule.go`、`http/controller/admin/my/addressBookCollectionRule.go`：管理规则时支持所有用户。
- `resources/admin/app.js`：个人共享规则页面及所有用户选项。

无需迁移已有共享规则或改变原有设备 ID。新增的共享类型使用现有数据表字段。

### 自动化测试

- [共享功能实现后的检查通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522339510)。
- [多群组和公开规则测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522474127)。
- [原生共享地址簿接口集成测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42522571693)。

详见 [API 使用说明](API.md)、[测试说明](TESTING.md) 及 [功能清单](FEATURE_MATRIX.md)。后续还可补充更多第三方地址簿导入格式和批量共享功能。

## 2026-11-09：地址簿导入导出、恢复、SMTP 与账户邮件功能

本次继续开发以小型团队日常使用为重点，所有程序均保留在 `development/m1-device-management`，两个草稿 PR 均不发布发行版。

### 已提交的功能

| 仓库 | 提交 | 内容 |
| --- | --- | --- |
| API | [a002498](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a0024985c0bb07e3627a25f9a56e7455addf45b1) | 地址簿 CSV 导入导出，按用户限制访问，最多 5000 条、4 MB，导入事务处理 |
| API | [d30ad8c](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d30ad8c9cc91aea2f3172aa48f41bf7c1374a8b0) | CSV 导入导出、中文内容、重复记录与跨用户访问的测试 |
| API | [2a43f80](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a43f80d1db8c06282aa4a10f8b5b700b2edd804) | 管理控制台 CSV 导入导出按钮及浏览器操作 |
| API | [c07cc6f](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c07cc6f2df867f4aaddf84b146b2806bcbf4780d) | 地址簿分类编号导入测试 |
| API | [21f70c7](https://github.com/HSOS6Miyabi/rustdesk-api/commit/21f70c7977d4c9ae1fea9f2f7535d41053fcd282) | 地址簿条目回收站与删除整个分类时的记录处理 |
| API | [918089c](https://github.com/HSOS6Miyabi/rustdesk-api/commit/918089c83023bd1023deb114efcbb3df764e189e) | 恢复、彻底删除、查询回收站接口及账户删除关联处理 |
| API | [6e13012](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6e130123e3d06d524cc7accd78592ca20a7cc95c) | 回收站管理界面与浏览器测试 |
| API | [6885810](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6885810b0643aa5972c498f1aa9a2636e32e80b7) | 恢复设备配置、归属验证、重名检查及分类删除的测试 |
| API | [d7997c0](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d7997c012dcf5287e40a6b4e53bd8736279b1f3b) | SMTP 服务器配置、密码加密存储、发送邮件与测试接口 |
| API | [df0f509](https://github.com/HSOS6Miyabi/rustdesk-api/commit/df0f509a01ca9ce60282dab31ddf96dee4c817cf) | 模拟 SMTP 服务器发送测试、加密保存和配置修改 |
| API | [1a06af2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/1a06af26d5c28fa2b24700c939fcb90c8fca6ab0) | SMTP 管理页面及发送测试邮件操作 |
| API | [78358d8](https://github.com/HSOS6Miyabi/rustdesk-api/commit/78358d887f827537e980b5607af55743b2a53f92) | 密码恢复请求、15 分钟一次性链接、旧会话撤销 |
| API | [6eb9ee6](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6eb9ee61b7650afbf92d0f9e3455b755969914cb) | 邮件恢复完整服务测试与账户状态处理 |
| API | [200cfad](https://github.com/HSOS6Miyabi/rustdesk-api/commit/200cfad69ec2cdd1b6b781ce46990a39ffe48a8c) | 网页登录入口和密码恢复表单 |
| API | [a922f32](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a922f3235e537dd82cd2bdfc4e796c357277b55f) | 登录页面与恢复表单浏览器测试 |
| API | [2a42e2d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2a42e2d4bd1f66470e936433c54a181025b6679d) | 团队邮箱邀请、按指定群组注册和邀请撤销 |
| API | [c482a4b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c482a4b4d29cc366031ac712d325db558db98c0f) | 邀请邮件与注册、失效、撤销的自动测试 |
| API | [0a37914](https://github.com/HSOS6Miyabi/rustdesk-api/commit/0a37914e573fa8d2b5e7a3048e0b7da0a0858316) | 管理员邀请页面、登录页面处理邮件链接 |
| API | [453ce53](https://github.com/HSOS6Miyabi/rustdesk-api/commit/453ce536e146dd1f9ac9179638409a23d6262848) | 邀请发送、撤销及注册页面的浏览器自动化测试 |

### 数据库及部署

地址簿条目新增 `deleted_at`，普通删除后可从回收站恢复。删除整个地址簿分类时，其关联的活动条目和已删除条目都会被彻底清除。

新增 `smtp_settings`、`password_reset_tokens`、`user_invitations`。已有用户、密码与群组保持原样。SMTP 邮箱凭据使用 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 加密保存。更换该密钥前，应在管理控制台重新配置 SMTP 密码。

密码恢复仅修改本地账户密码，并撤销该账户已有登录会话；OIDC 与 LDAP 的身份配置保持不变。邮件邀请创建的新账户会归属于指定用户组。

### 最新测试状态

- [地址簿 CSV 完整测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42558570753)，通过。
- [回收站恢复及管理界面测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559071829)，通过。
- [SMTP 配置及邮件传送测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559555373)，通过。
- [密码恢复服务测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42559793707)，通过。
- [邀请注册、邮件找回及管理页面的最新综合测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42560569953)，通过。

当前开发仓库对应 Server 开发提交仍是 [902293d](https://github.com/HSOS6Miyabi/rustdesk-server/commit/902293d49f17d831922237718ff389b3a1178267)。未创建里程碑版本标签或新的 Release。

## 2026-11-09：历史连接备注

新增管理员在已有连接记录中填写与修改备注的功能，备注会随连接日志一起查询和导出。原有连接 ID、会话信息与认证记录保持不变。

| 仓库 | 提交 | 具体内容 |
| --- | --- | --- |
| API | [507210f](https://github.com/HSOS6Miyabi/rustdesk-api/commit/507210f3183b534db3858be3ccdc6a82939c69d5) | 为连接审计记录增加备注、更新人编号、HTTP 接口与网页编辑功能 |
| API | [9664256](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9664256840d3c89edc188abbf729f77cc7b59413) | 测试备注保存、清除、非法长度、CSV 导出和 Chromium 页面操作 |

数据库 `audit_conns` 增加 `note`、`note_updated_by`；使用现有数据库迁移方式，旧连接记录保持可用。

管理员通过 `POST /api/admin/audit_conn/note` 修改备注，最多 2000 个字符。管理操作自动记录在原有操作日志中。备注同样包含在 `GET /api/admin/audit/export?kind=audit_conn` 的 CSV 文件中。

自动化测试见 [API 草稿 PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6) 的最新提交及 [GitHub Actions](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42561282556)。这部分仅修改 API，无须更新 Server。

## 2026-11-09：TOTP 双重认证和一次性恢复码

本模块对小型团队默认关闭。用户可以通过个人设置开启身份验证器，登录时按官方客户端双重认证请求格式验证。支持 30 秒动态验证码、6 个一次性恢复码及验证码重复使用检查。

| 仓库 | 提交 | 内容 |
| --- | --- | --- |
| API | [6115846](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6115846009eed5c7e01a4d2b58ff01da4938193e) | TOTP 时间校验、绑定、恢复码、登录挑战及数据库迁移 |
| API | [6cfd42b](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6cfd42bae6040d16c6eeae9af82a8b560313ac1f) | 官方客户端和网页密码登录的双重认证步骤 |
| API | [1299065](https://github.com/HSOS6Miyabi/rustdesk-api/commit/12990659d510794924d44ce2b8f46140f3d30707) | 个人设置中的双重认证启停、恢复码管理 |
| API | [2943406](https://github.com/HSOS6Miyabi/rustdesk-api/commit/29434069a2fa6e6d6d2d08c03ff62f2c9b903c86) | RFC 6238 标准验证码及绑定、恢复码与挑战限制测试 |
| API | [760fbb2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/760fbb25add2190ce75b626e649857622c19c425) | 官方 RustDesk 客户端登录协议的请求与响应测试 |
| API | [eefe3f2](https://github.com/HSOS6Miyabi/rustdesk-api/commit/eefe3f2b97df1d5b6d7bfe386e448b3cc70df786) | 管理网页 TOTP 绑定、恢复码显示及登录浏览器测试 |
| API | [ec31834](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ec3183425294169f22ec64938cab0365a0f41217) | 删除账户时删除 TOTP 数据、登录挑战及密码恢复申请；撤销该账户发出的待接受邀请 |

### 数据、接口及兼容性

增加 `user_totps`、`totp_recovery_codes`、`totp_login_challenges`，数据库迁移版本更新到 274。TOTP 密钥加密保存；恢复码只在生成时显示，数据库只保存摘要。

官方客户端登录首先返回 `type=email_check`、`tfa_type=tfa_check` 和一次性 `secret`；用户提交动态验证码后再创建实际登录令牌。网页通过 `/api/admin/mfa` 管理绑定状态。

目前复用邮件模块使用的 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 加密 TOTP 密钥。该环境变量在不使用 SMTP 的安装中也需要设置，开启双重认证前应保存并备份。

### 验证

- [TOTP 标准算法与恢复码测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562369988)，通过。
- [RustDesk 官方客户端双重认证协议测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562498241)，通过。
- [Web 控制台绑定、恢复码及登录浏览器测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42562596974)，通过。

真实 Windows、macOS、Linux、Android 客户端登录需要在完成里程碑前结合实际设备进行检查。

## 2026-11-09：可选邮箱验证码登录

通过已配置的 SMTP 服务器发送 6 位数字验证码。用户登录后可在个人设置中启用邮箱登录验证；登录时由服务器发送验证码，验证成功才创建账户会话。双重认证继续保持可选，已启用 TOTP 时优先使用身份验证器。

| 仓库 | 提交 | 实施内容 |
| --- | --- | --- |
| API | [2919335](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2919335af73d5f6bd8e81463599fd2fff82df854) | 邮箱验证数据表、验证码生成及发送、登录验证服务、统一双重认证接口 |
| API | [5a254fd](https://github.com/HSOS6Miyabi/rustdesk-api/commit/5a254fdb0b65255a4f15b08ae2f9a008e0e3e6f9) | 绑定与关闭接口、官方客户端和网页登录流程 |
| API | [c4e025e](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c4e025eb324d24825ee81532e40078a8074ecd0b) | 本地 SMTP 验证、验证码有效期和尝试次数测试 |
| API | [32a3a88](https://github.com/HSOS6Miyabi/rustdesk-api/commit/32a3a88cb9ac1e41b414d7a600f8fd4cd8a07568) | 个人设置中的邮箱验证开关与两步登录 |
| API | [7babd12](https://github.com/HSOS6Miyabi/rustdesk-api/commit/7babd1221d94515ca6f25ab2916df3eeb7c7a426) | 验证码发送频率和重复申请测试 |
| API | [1f6e49c](https://github.com/HSOS6Miyabi/rustdesk-api/commit/1f6e49cee055082629845820ba039ce8ab20449e) | 官方 RustDesk 客户端邮箱验证码请求与登录响应测试 |
| API | [6a74cda](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6a74cda828b0696b1a879f614df06f363d83f0b7) | 网页账户绑定和邮箱验证码登录浏览器测试 |
| API | [e8d54f3](https://github.com/HSOS6Miyabi/rustdesk-api/commit/e8d54f3f157c97ab8086fad729cf949698aa6309) | 删除账户时清除邮箱验证设置与未完成的验证码请求 |

### 数据库及默认行为

增加 `email_login_settings`、`email_code_challenges`，数据库迁移版本更新至 275。用户默认不开启邮箱验证码登录，原有登录方式继续有效。

邮箱验证码通过已配置的 SMTP 发送，10 分钟内有效，最多尝试 5 次；单个账户每小时最多申请 6 次。绑定时先向当前账户邮箱发送验证码，成功验证后记录已验证的邮箱地址。账户邮箱变更后，需要重新进行邮箱验证。

官方客户端复用 `type=email_check`、`tfa_type=email_check` 和 `verificationCode` 字段。已有 TOTP 的用户优先使用身份验证器，其余启用邮箱验证的用户使用邮箱验证码。服务器不会在验证码通过前发放登录令牌。

### 测试

- [邮箱验证码服务及发送频率测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42563626241)，通过。
- [完整原生客户端 HTTP 接口测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42563745799)，按后续综合测试结果核对。
- [网页邮箱验证及登录测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42563796255)，按最新综合测试结果核对。

与此前相同，日常提交不创建 GitHub Release 或发行版 Docker 镜像。

## 2026-11-09：多中继节点管理及网络优先选择

这一功能适合在多个地区部署 `hbbr` 的小型团队。管理员在 Web 控制台配置节点，`hbbs` 可定时读取 API 并根据连接双方的 IP 网络优先选择中继。没有配置托管中继时，继续使用现有 `--relay-servers` 设置。

| 仓库 | 提交 | 实施内容 |
| --- | --- | --- |
| API | [33476e0](https://github.com/HSOS6Miyabi/rustdesk-api/commit/33476e0b87e92adb9695fc3806dc928a314412a3) | 中继节点表、创建修改删除、地址和 CIDR 校验、内部同步接口 |
| API | [850db22](https://github.com/HSOS6Miyabi/rustdesk-api/commit/850db223278dbf1fb4fd311a54e71f0c9823dd71) | 中继节点管理页面、优先级和 IP 网络编辑 |
| API | [83d2628](https://github.com/HSOS6Miyabi/rustdesk-api/commit/83d26283346bdb839f045a44c76655f24fafca5a) | SQLite 节点管理、禁用、内部联系服务身份验证及浏览器测试 |
| API | [c76f5cf](https://github.com/HSOS6Miyabi/rustdesk-api/commit/c76f5cffee49946e6de57c7fa80f1ab22171e435) | 列表刷新完成后显示保存提示 |
| API | [d892c04](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d892c0404806cb1649d51a7a9cf7b5e8e4f1629a) | 中继节点管理页面保存和关闭编辑窗口的自动化检查 |
| Server | [6183fb1](https://github.com/HSOS6Miyabi/rustdesk-server/commit/6183fb15f9969fc64cd2868d2ed48a6fb4306c15) | IP 网络选择策略，保留未配置时的轮流选择逻辑 |
| Server | [cf3b616](https://github.com/HSOS6Miyabi/rustdesk-server/commit/cf3b6162831416e4af2ed2086f3e117f5c9b5d54) | hbbs 定期获取 API 节点配置，更新活动中继列表 |
| Server | [1a82c2c](https://github.com/HSOS6Miyabi/rustdesk-server/commit/1a82c2c3668ae4c8057466f25cbd853246f68a41) | 真实 hbbs 进程、临时中继服务器与内部 API 的联调测试 |
| Server | [1dd7c0e](https://github.com/HSOS6Miyabi/rustdesk-server/commit/1dd7c0e85c10bf71f2bcfe86eebead8e09d4c495) | 中继离线后的替代选择、全部节点不可用时回到原有地址 |

### 运行设置

API 新增 `relay_nodes` 表，初始数据库迁移版本为 276。节点启用状态、优先级和 CIDR 范围由管理员在“中继节点管理”页面修改。国家代码字段在数据库迁移版本 277 中增加。

Server 读取以下环境变量：

```bash
export RUSTDESK_API_RELAY_NODES_URL="http://127.0.0.1:21114/internal/v1/relay/nodes"
export RUSTDESK_API_INTERNAL_KEY="与 API 使用相同的内部服务密钥"
export RUSTDESK_API_RELAY_SYNC_SECONDS=30
```

`RUSTDESK_API_RELAY_NODES_URL` 不配置时，`hbbs` 继续使用启动参数和原有中继服务器列表。服务器定期同步成功后，不需要重启即可使用新节点。使用公开主机部署时，应通过 HTTPS 或受保护的内部网络访问 API。

### 选择方式与测试

优先选择能够匹配连接双方 IP 网络范围的在线节点。双方均匹配时优先于只有一方匹配的节点；匹配程度一致时，优先级数字较小的节点优先。如果没有匹配规则，则在活动节点之间轮流选择。

已有中继节点健康检查继续检查 TCP 端口。节点不可达时自动选择其他活动节点；托管配置全部取消时恢复 `hbbs` 启动时的中继配置。

- [API 中继管理、内部同步及浏览器测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42565133053)，通过。
- [Server IP 网络规则与动态同步的自动化测试](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42564891450)，通过。
- [Server 中继断线替代选择测试](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42565263648)，执行结果见工作流。

管理员可以继续维护 IP 网络规则，也可以通过 GeoLite2 Country CSV 自动识别 IP 所属国家。具体实现与测试记录见下文。

## 2026-11-09：GeoLite2 Country CSV 地理位置选择

本轮增加国家级 IP 地理位置识别。管理员可给每个中继设置 `countries`（两位 ISO 国家代码），RustDesk Server 从服务器本地的 GeoLite2 Country CSV 文件读取 IPv4/IPv6 网络和国家关系。精确 CIDR 规则具有更高选择优先级。未启用国家匹配时，继续使用现有中继规则。

### 实施及测试记录

| 仓库 | 提交 | 实现内容 |
| --- | --- | --- |
| API | [e986914](https://github.com/HSOS6Miyabi/rustdesk-api/commit/e986914be9379d5d36d7c77eb5f4532898fc7a1a) | 节点国家代码字段、输入规范化、Web 管理页面和内部 API 的国家列表 |
| API | [ec007b5](https://github.com/HSOS6Miyabi/rustdesk-api/commit/ec007b5030664a01f2e0962d05dcaae9333080d4) | 提升数据库迁移版本至 277，兼容已存在节点的数据 |
| Server | [fc0b3ce](https://github.com/HSOS6Miyabi/rustdesk-server/commit/fc0b3ce3831981c47984f10120b1abe70aa7196f) | 加载 MaxMind GeoLite2 Country 的 Locations-en 和 IPv4/IPv6 Blocks CSV |
| Server | [f3b67c0](https://github.com/HSOS6Miyabi/rustdesk-server/commit/f3b67c0aac07007f7eaf13ae3a28b626968946e6) | 根据国家代码和精确 CIDR 规则选择节点 |
| Server | [ab61a6f](https://github.com/HSOS6Miyabi/rustdesk-server/commit/ab61a6f9f7ef574cdd48930ffaf2865dd14aa26d) | hbbs 启动时读取国家数据库，支持运行时 `reload-geo` 命令 |
| Server | [9429bda](https://github.com/HSOS6Miyabi/rustdesk-server/commit/9429bda97a5c07b8b2539fc413512f4109928bef) | 真实 hbbs 进程测试：国家选择、热更新、节点断线与原有中继恢复 |

### 配置说明

MaxMind 官方 GeoLite2 Country CSV 解压目录应包含 `GeoLite2-Country-Locations-en.csv`、`GeoLite2-Country-Blocks-IPv4.csv` 和 `GeoLite2-Country-Blocks-IPv6.csv`。

```bash
export RUSTDESK_GEOIP_CSV_DIR="/opt/rustdesk/geoip"
```

从 [MaxMind 开发者文档](https://dev.maxmind.com/geoip/docs/databases/city-and-country/)了解 Country CSV 格式和获取方法。用户自行获取、存储并更新数据库。程序不随安装包分发 GeoLite2 数据。

运行中的 hbbs 可通过本地控制命令 `reload-geo` 重新读取文件。未配置 `RUSTDESK_GEOIP_CSV_DIR` 时，国家代码规则不参与选择，仍然使用精确 CIDR、节点优先级及原有轮流选择方式。

### 选择规则

1. 已启用且可连接的节点进入候选列表。
2. 如果有节点的 CIDR 范围匹配连接双方的地址，优先使用精确网络匹配。
3. 没有精确网络匹配时，使用 GeoLite2 推断的两位国家代码匹配节点。
4. 匹配客户端数量相同，优先使用优先级数字较小的节点；都不匹配则轮流选择。
5. 节点不可连接时改选其他可用节点，动态配置被清空时恢复旧的中继地址。

### 测试结果

- [API 国家代码、IPv4/IPv6 CIDR、内部配置和浏览器测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42567027724)，通过。
- [Rust 单元测试及 GeoLite2 CSV 读取测试](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42567365799)，通过。
- [真实 hbbs 的 GeoLite2 国家级中继选择和数据重新载入测试](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42567610418)，通过。

对应的两个仓库最新功能提交分别为 API `ec007b5` 与 Server `9429bda`。继续开发时应先读取各自分支的最新提交，再执行相应测试。

## 2026-11-09：远程控制角色及官方客户端协议传递

管理员可以建立远程控制角色，管理官方 RustDesk 客户端支持的 14 项会话权限。可直接给单台设备、设备所有者和设备组分配角色，优先顺序为设备、设备所有者、设备组。未分配角色时保留现有设备设置。

| 仓库 | 提交 | 主要内容 |
| --- | --- | --- |
| API | [43975d8](https://github.com/HSOS6Miyabi/rustdesk-api/commit/43975d8003a4072456994ab788ed0e5003790604) | 控制角色及分配模型，14 项权限的 2 位编码 |
| API | [90bed78](https://github.com/HSOS6Miyabi/rustdesk-api/commit/90bed7869b1634dcd88aa973b564f705b20dd252) | 管理员角色接口、查询生效角色及内部会话权限接口 |
| API | [4537369](https://github.com/HSOS6Miyabi/rustdesk-api/commit/4537369045362c9b3133880f3c54cf3a0073baf7) | 网页角色编辑、权限选择和分配管理 |
| API | [13ff560](https://github.com/HSOS6Miyabi/rustdesk-api/commit/13ff560f7bbd18837d3d57795ae0f6e76312ac11) | 位编码、角色优先级、权限继承及分配测试 |
| API | [7cec413](https://github.com/HSOS6Miyabi/rustdesk-api/commit/7cec413688738623bed53e2ef29a2cc80f9a9d51) | 删除用户与设备组时清理相关角色分配 |
| API | [2bf36df](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2bf36dfed5d3139c111ee5b061e8945f505ca0cf) | 内部会话接口和浏览器权限编辑测试 |
| API | [9aed985](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9aed985e56b6896c2abd37aacbe6302913a7c103) | 完整 Go、JavaScript 与 Chromium 自动化测试 |
| Server | [85bdbab](https://github.com/HSOS6Miyabi/rustdesk-server/commit/85bdbab7e14e852115787085b095bb143afa6bbd) | 从内部 API 读取控制角色及缓存 |
| Server | [c97ba9d](https://github.com/HSOS6Miyabi/rustdesk-server/commit/c97ba9de6144eee27e45e8d37d5b3c7c0a98ca94) | 将控制权限写入 PunchHole、FetchLocalAddr、RequestRelay |
| Server | [ee9ff73](https://github.com/HSOS6Miyabi/rustdesk-server/commit/ee9ff73be768677f24ee883e33bca59cfe21c48d) | 实际 hbbs 进程中的客户端协议测试 |

### 数据库、接口与配置

API 新增 `control_roles`、`control_role_assignments`，迁移版本为 278。权限沿用 RustDesk 官方 `ControlPermissions` 编码：每项占 2 位，`0` 为沿用设备设置、`1` 为禁止、`2` 为允许，保留值 `3` 无效。

API 提供管理员控制角色及分配接口，并通过 `GET /internal/v1/control/permissions?device_id=...` 给 hbbs 返回当前目标设备生效的位编码。该内部接口需要与部署服务共享 `RUSTDESK_API_INTERNAL_KEY`。

Server 需设置 `RUSTDESK_API_CONTROL_ROLE_URL` 才会查询控制角色。设置 `RUSTDESK_API_CONTROL_CACHE_SECONDS` 可控制缓存时间，默认 5 秒，范围 0 至 60 秒。未配置时保留原有会话权限处理方式。

### 自动化测试

- [API Go、JavaScript、Chromium 测试通过](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42569840801)。
- [Server Rust 协议结构和编译测试通过](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42569499247)。
- [Server 实际运行进程协议检查](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42569942719)，以该工作流最终状态为准。

实际桌面连接的键盘、文件传输、剪贴板、终端等操作行为仍需要在各平台客户端上进一步验证。

## 2026-11-09：管理员角色、细分权限和成员管理委派

这一部分适合小型团队由多名成员共同维护设备。系统管理员可以创建管理角色，指定允许的管理操作，再将多个角色分配给同一位成员。全局角色可执行指定的管理操作；指定用户和用户组的角色目前支持查看范围受限的用户与设备目录。

### 具体提交

| 提交 | 主要工作 |
| --- | --- |
| [5c39269](https://github.com/HSOS6Miyabi/rustdesk-api/commit/5c39269fdf24026f74ddbe350ded19b71309884e) | 管理角色及多角色分配、权限清单、全局及限定范围的目录访问 |
| [9033661](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9033661efbd147243c48cd15fc6e4c544099ab4e) | 后台请求权限检查及登录页面使用的管理权限名称 |
| [7167e7f](https://github.com/HSOS6Miyabi/rustdesk-api/commit/7167e7fcab95fa4751fb613921fc36d41947149f) | 管理角色创建、修改、查询、删除及角色分配接口 |
| [e408c55](https://github.com/HSOS6Miyabi/rustdesk-api/commit/e408c55c2259163b59c06cfd0d7e6f4aed70068e) | 限定用户组或指定账户时，筛选用户和设备列表及详情 |
| [1dc3185](https://github.com/HSOS6Miyabi/rustdesk-api/commit/1dc31854c64759617a03285a5cf9085198d3feeb) | 清理被删除账户或用户组对应的角色分配 |
| [9eb73ac](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9eb73ac0ec84edb208d109537499112f0eb96323) | 网页管理界面创建角色、勾选权限、分配给用户 |
| [e7ff42d](https://github.com/HSOS6Miyabi/rustdesk-api/commit/e7ff42d5bc8b4b4844afad8c32759b8e82a9843e) | 测试角色叠加、用户组范围、停用、删除及接口权限映射 |
| [2f713df](https://github.com/HSOS6Miyabi/rustdesk-api/commit/2f713dff3a50cda30dcbf55af4f8f0dc1d7a1349) | Gin 实际管理接口的限定范围和越权访问测试 |
| [8ed6790](https://github.com/HSOS6Miyabi/rustdesk-api/commit/8ed67903d6884ef5434e6c763191002e105d0d55) | 统一限定范围访问时的拒绝响应 |
| [f320640](https://github.com/HSOS6Miyabi/rustdesk-api/commit/f320640567bb8ecc7a22e9fc773b17097861a26f) | 网页角色管理及受限成员导航测试 |
| [9a74eb1](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9a74eb1f014ed139106c8fe25f07204340482263) | 管理角色页面提交操作 |
| [d92959e](https://github.com/HSOS6Miyabi/rustdesk-api/commit/d92959e6847bd8b0111eee0f049e292091a95718) | 管理员帮助用户重置 TOTP 和邮箱验证 |
| [373b9bf](https://github.com/HSOS6Miyabi/rustdesk-api/commit/373b9bf370de7bbda1ba3dbd150483e85fcb1991) | 重置双重认证时撤销旧登录会话，清理失效角色授权 |
| [b4a9b02](https://github.com/HSOS6Miyabi/rustdesk-api/commit/b4a9b02d50ac4987d0316b1fd36821e4eef75b84) | 双重认证恢复、登录会话撤销与浏览器操作测试 |

### 管理范围

- 全局：使用该角色可执行的所有管理操作，并访问全体相应数据。
- 用户组：对该组的主要成员及附加成员提供用户目录、设备目录只读访问。
- 指定用户：仅可查看该用户资料和名下设备。

**限定范围角色支持用户与设备查看、编辑、删除以及设备所有者转移。** 相关修改操作由服务器检查目标账户的归属，设备转移要求原所有者及目标所有者均属于授权范围。系统设置及管理角色分配仍由系统管理员负责。

已经拥有 `is_admin=true` 的系统管理员继续拥有全部管理权限。普通成员未被分配管理角色时，现有个人功能不改变。管理角色接口本身仅允许系统管理员访问，普通受委派成员不能创建或提升自己的角色。

### 数据迁移与清理

新建 `admin_roles`、`admin_role_assignments`，数据库迁移版本为 279。一个角色可以分配给多个成员，一个成员可以同时拥有多个角色。禁用角色后立即停止该角色授权，删除角色时同时删除相关分配记录。

删除账户或用户组时，会删除相关分配和失效的范围角色。管理员重置成员的 TOTP、邮箱登录验证后，成员原有登录会话同时被撤销，可重新选择登录方式。

### 测试情况

- [角色叠加、权限范围与基本接口测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42573060628)：通过。
- [角色管理和双重认证重置综合测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42574010348)：通过。
- 最新开发提交的全部自动化测试参见 [API 草稿 PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)。

本模块只修改 API 和管理控制台，不需要改变当前 Server 二进制程序。

## 2026-11-09：服务器执行的可选连接访问规则

本阶段实现按用户、用户组和目标设备范围管理远程连接许可。通过管理员 Web 控制台配置规则，并在 hbbs 收到普通远程连接或中继请求时调用 OSS API 检查。此功能面向需要集中控制设备访问的小型团队，默认保持关闭。

### 两个仓库的具体提交

| 仓库 | 提交 | 功能 |
| --- | --- | --- |
| API | [146ab63](https://github.com/HSOS6Miyabi/rustdesk-api/commit/146ab634d546185ad6e1ee91343c2188ce1bddee) | 建立 `AccessRule`、连接访问判断、管理员管理接口及内部授权接口 |
| API | [95d62ac](https://github.com/HSOS6Miyabi/rustdesk-api/commit/95d62ac3497b87b01669b3d1afcc246291892be6) | 注册管理路由与 `POST /internal/v1/connection/authorize`，数据库迁移版本 280 |
| API | [087b43e](https://github.com/HSOS6Miyabi/rustdesk-api/commit/087b43ec9c4ed747bb6c03d1f132d01b0d3d5a15) | 用户、群组、设备授权以及内部接口的 Go 集成测试 |
| API | [3f0ce67](https://github.com/HSOS6Miyabi/rustdesk-api/commit/3f0ce67964bb602f6a006436c77eb44105e6e0e9) | 管理界面新增“连接访问规则”及条件编辑 |
| API | [f92b362](https://github.com/HSOS6Miyabi/rustdesk-api/commit/f92b3621be1702b8bb68ed6c91c7e694812d484c) | 管理页面的规则编辑自动化测试 |
| API | [6b62ea6](https://github.com/HSOS6Miyabi/rustdesk-api/commit/6b62ea6f9dfad169a52a0ac64d470837d1d2055b) | 规则编辑提交和浏览器自动化测试 |
| API | [9788dc5](https://github.com/HSOS6Miyabi/rustdesk-api/commit/9788dc51550ca9d592ad4851d842f44d2070eaaa) | 删除账户、用户组、设备或设备组时清理相关访问规则 |
| API | [97cb3e1](https://github.com/HSOS6Miyabi/rustdesk-api/commit/97cb3e1b43c88e49fbc0cc30e34443ddc2f37fcb) | 清理规则后的 Go 测试 |
| Server | [8aff47f](https://github.com/HSOS6Miyabi/rustdesk-server/commit/8aff47fbf25227078821ff2058cdf161b30368a1) | hbbs 向 API 发出授权请求；仅按配置启用严格检查 |
| Server | [3ceef25](https://github.com/HSOS6Miyabi/rustdesk-server/commit/3ceef2570694ca8d2e78467d0147b375a3647dab) | 普通连接 `PunchHoleRequest` 与中继 `RequestRelay` 的访问检查 |
| Server | [0fa1e56](https://github.com/HSOS6Miyabi/rustdesk-server/commit/0fa1e56e873f000bf13e88b9a31dfaab37c17a50) | 真实 hbbs 与模拟 API 的连接许可测试 |
| Server | [4ba02a9](https://github.com/HSOS6Miyabi/rustdesk-server/commit/4ba02a96a0b4eca0aab7bd77ed5c4bafe6d2d0b7) | 实际客户端协议拒绝信息及完整运行测试 |

### 数据和权限处理

新增 `access_rules` 表，数据库迁移版本为 280。每条规则包括来源类型、来源账户或用户组、目标类型、目标 ID、允许或禁止操作以及启用状态。

来源类型支持所有已登录用户、指定用户、指定用户组。目标类型支持所有已登记设备、指定设备 ID、指定设备所有者、指定设备组。

启用严格模式后，所有普通连接和中继请求都需要有效的账户登录令牌。设备所有者可以访问自己的已登记设备；其他连接必须匹配至少一条允许规则。若同时匹配允许和禁止规则，禁止规则优先。未匹配规则时拒绝访问。

群组来源同时考虑用户的主要群组和附加群组。该检查使用 API 中的已登记设备信息及登录会话，在数据库中修改或撤销账户令牌后，下一次连接请求将应用新结果。

### Server 设置

只有在 hbbs 进程中设置 `RUSTDESK_API_CONNECTION_ACCESS_URL` 时才开启严格模式：

```bash
export RUSTDESK_API_CONNECTION_ACCESS_URL="http://127.0.0.1:21114/internal/v1/connection/authorize"
export RUSTDESK_API_INTERNAL_KEY="与 API 相同的内部通信密钥"
```

Server 通过带内部密钥的 POST 请求，将设备 ID 和客户端登录令牌发送给 API。授权接口只向 hbbs 提供，返回 `allow` 和原因。API 无法访问、登录令牌无效或没有许可时，严格模式拒绝新的连接请求。

未设置 `RUSTDESK_API_CONNECTION_ACCESS_URL` 时，现有普通连接和中继工作方式保持不变。需要注意：这里管理的是通过 hbbs 协商的连接请求，直接通过 IP 地址建立的连接仍由客户端和目标设备控制。

### 自动化测试

- [API 授权规则服务、内部接口及数据库测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42577911849)：通过。
- [API 管理页面及 Chromium 浏览器测试](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42578812266)：通过。
- [Server 普通连接、中继连接及 API 无法访问的实际进程测试](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42578682536)：通过。

真实桌面客户端的各类网络环境、登录方式及直接 IP 连接仍应在里程碑发布前检查。


## 2026-11-09：自动化管理、限定范围修改、批量管理及双格式国家数据

本阶段在两个开发分支持续提交，并通过草稿 PR 的自动化检查。首个正式发布里程碑仍须完成 P0 真实环境检查与维护者确认。

### 实现和代码位置

| 任务 | 主要提交 | 实现结果 |
| --- | --- | --- |
| T011 | [Server 32967aa](https://github.com/HSOS6Miyabi/rustdesk-server/commit/32967aa4bf) | 增加双架构 API/Server 组合镜像预演，运行实际容器检查 API、hbbs、hbbr 与 WebSocket |
| T015 | [API 自动化令牌](https://github.com/HSOS6Miyabi/rustdesk-api/blob/development/m1-device-management/http/router/automation.go) | 为独立令牌增加查看和管理权限；从 `/api/automation/v1` 调用设备、部署、中继、会话、策略、角色与审计接口 |
| T016 | [API 设备所有者转移](https://github.com/HSOS6Miyabi/rustdesk-api/blob/development/m1-device-management/http/controller/admin/peer_transfer.go) | 允许按指定账户或用户组授权修改、删除设备和用户；设备转移同步更新登记记录并撤销旧设备会话 |
| T017 | [API 回收站批量服务](https://github.com/HSOS6Miyabi/rustdesk-api/blob/development/m1-device-management/service/address_book_batch.go) | 联系人批量添加和撤销、地址簿批量共享、部署许可批量修改、回收站批量恢复与永久删除 |
| T018 | [Server MMDB/CSV 国家识别](https://github.com/HSOS6Miyabi/rustdesk-server/blob/development/m1-device-management/src/geoip_country.rs) | 支持 `RUSTDESK_GEOIP_MMDB_FILE`，与原有 CSV 读取共存；可执行 `reload-geo` 重新读取数据 |

### 已完成的检查

- [API 令牌权限与管理接口：42584017460](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42584017460)，通过。
- [管理员范围检查、设备转移与 Chromium 交互：42585567417](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42585567417)，通过。
- [四类批量管理 Go 测试与页面交互：42587648927](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42587648927)，通过。
- [Server 双架构组合容器预演：42585097640](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42585097640)，通过。
- [Server MMDB/CSV 国家读取、示例数据及 Rust 测试：42588361437](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42588361437)，通过。

### 继续维护时的检查顺序

- 先读取两个开发分支最新提交和 PR CI；文档更新后可能产生新的 CI 运行。
- 在相应分支运行 Go、Rust、浏览器及双架构容器测试。MMDB 验证示例位于 Server 的 `tests/fixtures`，正式地理数据库仍由部署人员提供。
- T001–T010、T012 需要实际客户端、SMTP、数据库或发布审核的项目仍按第 4 节执行。T013、T014、T019–T022 后续事项保持在待办清单中。
- 两个拉取请求继续保持草稿，所有日常开发提交不得发布 Release 或正式 Docker 镜像。


## 2026-11-09：客户端构建、邮件通知、设备策略及部署检查

本阶段已实现 P1 类别的 T013–T020 中列出的主要管理功能，并完成对应自动化测试。正式发布前仍需完成真实客户端、实际邮箱、公网中继与数据库备份恢复检查。以下记录当前开发代码的功能和验证范围。

### Windows、Linux、macOS 客户端安装文件

客户端构建页面支持保存服务器地址、公钥、API 和中继地址，选择 Windows、Linux、macOS 的 x64 或 ARM64，创建安装任务、查看历史记录、重新提交失败任务及下载已准备好的文件。构建服务保留 RustDesk 1.5.1 官方 EXE、DEB 或 DMG 原始字节。Windows 使用官方客户端识别的文件名保存服务器配置；Linux/macOS 在 ZIP 内另附配置脚本和使用说明。系统在下载官方文件及管理员下载文件时核对 SHA-256。

关键文件：`model/client_build.go`、`service/client_build.go`、`service/client_build_worker.go`、`service/client_packages.go`、`http/controller/admin/client_builder.go`、`resources/admin/app.js`。

- [官方安装程序核对及 Go 单元测试 #42620321148](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620321148)：通过。
- [Chromium 桌面、平板与手机交互及截图检查 #42620320903](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620320903)：通过。
- [六个平台官方安装文件检查 #42623413828](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42623413828)：通过；包含 Linux 和 macOS ZIP 及配置脚本的 Go 测试。\n- Windows、Linux、macOS 的真实安装、权限及连接检查，继续在发布前任务中执行。

### 邮件通知与管理数据

邮件事件通知已覆盖登录失败、设备登记许可变化、远程会话结束请求与确认、文件传输记录。管理员可选择事件和收件人、查看发送记录并重新提交失败邮件。发送处理与原有业务请求分开，模拟 SMTP、重复事件处理及失败重试已通过测试。数据库版本 281 新增邮件通知表。

设备管理新增筛选及 CSV 导出，审计记录支持时间、设备、来源账户和连接事件筛选。联系人、地址簿共享、设备许可和回收站批量管理已完成，批量修改使用数据库事务检查。

- [API 综合检查 #42620320903](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620320903)：通过。
- 正式 SMTP 服务商的实际收件、邀请链接和账户恢复流程仍按 T007 检查。

### 客户端策略和数据库版本

设备安全策略增加访问模式、键盘鼠标、剪贴板、文件传输和连接确认方式。API 记录受服务器管理的选项，支持更新和清除。数据库版本 282 新增客户端构建配置和任务记录。

在临时 SQLite、MySQL 8.4、PostgreSQL 16 上模拟旧结构升级，检查策略、审计和构建任务保存，并验证重复迁移：

- [数据库兼容检查 #42620320922](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620320922)：通过。
- 正式部署前备份数据库、检验恢复流程、确认所用数据库版本和旧数据完整性，仍属于 T006。

### 普通进程与 Docker 的双架构检查

Server 通过普通进程启动 API、`hbbs`、`hbbr`，检查 TCP、UDP、WebSocket、WebClient、服务重启、账户及数据库保持。Docker 测试覆盖 AMD64、ARM64，验证容器重建及 API 数据与服务器密钥保存。

- [Server 实际服务测试 #42618004098](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42618004098)：通过。
- [双架构组合服务与数据保存检查 #42618003889](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42618003889)：通过。

### 客户端版本与界面

根据 RustDesk 1.4.5、1.5.0、1.5.1 的官方源代码检查 `--assign`、`--deploy` 请求字段及返回状态。1.4.5 可使用所检查的 `--assign` 参数；1.5.0 与 1.5.1 同时支持这两项命令。核对程序位于 `scripts/verify_upstream_client_commands.py`。官方 1.5.1 新增功能和兼容性候选记录于 [UPSTREAM_FEATURE_WATCH.md](UPSTREAM_FEATURE_WATCH.md)。

Chromium 检查已保存客户端构建器及邮件通知的桌面、平板、手机截图。多平台真实设备登录、远程操作及其他语言支持仍按 T002、T003、T009、T010 和 T022 检查。

两个仓库继续使用草稿 PR，未创建新版本标签、Release 或正式 Docker 镜像。


## 2026-11-09：安装配置核对与数据库恢复演练

六平台客户端安装文件制作已完善核对流程。Linux、macOS 的设置脚本在执行每条 RustDesk `--option` 命令后，再次读取对应设置。如果客户端未安装、权限不足或设置未成功保存，脚本会停止并显示错误。源代码兼容性检查确认 RustDesk 1.5.1 的 `--option` 需要客户端已经安装并具备管理权限，且支持读取及写入。安装文件下载使用完成 SHA-256 核对的同一文件对象，避免重新打开可能已被替换的路径。

- [API 综合检查 #42627684694](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42627684694)：Go 编译、浏览器管理页面及其他测试通过。
- [官方安装文件和客户端命令检查 #42627684647](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42627684647)：通过。
- [数据库迁移与恢复检查 #42627684759](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42627684759)：在临时 SQLite、MySQL 8.4 和 PostgreSQL 16 环境通过。数据库恢复演练使用单独的目标数据库，保留原始数据；具体操作见 [数据库备份恢复说明](DATABASE_BACKUP_RESTORE.md)。

数据库检查包含旧结构升级、SQLite 停止连接后的文件恢复、MySQL 的 `mysqldump` 导出与恢复，以及 PostgreSQL 的 `pg_dump`、`pg_restore`。恢复后检查历史策略、审计记录与客户端构建任务。

正式发布前还需要使用实际部署环境和真实业务数据的备份恢复，以及实际客户端与公共网络的端到端测试。草稿 PR 和正式版本保持原有状态。
