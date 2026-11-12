# RustDesk API 与 Server 全部工作事项及测试记录

更新日期：2026 年 11 月 12 日。维护对象为 [rustdesk-api](https://github.com/HSOS6Miyabi/rustdesk-api) 的 `master` 分支及 [rustdesk-server](https://github.com/HSOS6Miyabi/rustdesk-server) 的 `forapi` 分支。详细实施记录见[统一实施台账](IMPLEMENTATION_LEDGER.md)。本文逐项记录 T001–T022 的检查范围、已完成工作、实际环境条件与结果。

当前 API 数据库迁移版本为 **290**。已合并 [API PR #14](https://github.com/HSOS6Miyabi/rustdesk-api/pull/14) 并发布 [API v3.0.0-api.109](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.109)；Server 继续使用 [v1.1.17-api.21](https://github.com/HSOS6Miyabi/rustdesk-server/releases/tag/v1.1.17-api.21)。两个仓库的正式版本应分别核对。

## 状态说明与测试记录

“自动化测试已通过”表示对应代码已通过列明的 GitHub Actions 工作流；“实际环境待检查”表示还需要客户端、基础设施或外部服务；“持续维护”表示需在之后的官方版本更新时重新核查。单独通过的单元、模拟协议、实际进程和公网设备检查分别记录，不能互相代替。

| 检查项目 | 已通过的自动化运行 |
| --- | --- |
| API Go、WebClient、Chromium 桌面、平板和手机交互及发行文件构建 | [42978795619](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42978795619) |
| 官方安装文件、客户端命令和 Linux/macOS 配置包 | [42666929902](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42666929902) |
| SQLite、MySQL 8.4、PostgreSQL 16 数据库升级与备份恢复 | [42975509940](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42975509940) |
| Server Rust 单元、协议和实际服务检查 | [42669744309](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42669744309) |
| API 与 Server 的 AMD64、ARM64 普通进程及 Docker 组合部署 | [42669744598](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42669744598) |
| 官方 1.5.1 客户端可执行文件跨系统启动 | [42666929318](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42666929318)，三种桌面系统及 Android 模拟器成功 |

### T001：官方设备分配、设备部署与撤销许可

**当前状态：官方 Linux 客户端部署与分配、API 与 hbbs 的设备登记、许可撤销与恢复、撤销后的新连接拒绝及设备所有者变更后的会话撤销测试通过；双设备远程重连待检查。**

代码位于 API 的 `http/controller/api/device_commands.go` 和 Server 的 `src/device_deployment.rs`。API 的 `http/controller/api/device_commands_integration_test.go` 使用 SQLite 检查设备所有者、设备组、策略、地址簿及 UUID。Server 的 `scripts/test_device_deployment.py` 启动真实 hbbs 验证未登记、已登记、ID 冲突和撤销结果。另有官方 1.4.5、1.5.0、1.5.1 源代码与请求字段检查。新增 API `scripts/test_official_linux_real_api.py` 和 Server `scripts/test_real_api_registration.py` 在实际运行的服务中检查设备所属用户、身份信息及撤销许可；相关检查分别见 API #42646358750 和 Server #42669744598。

**实际检查方法：** 在隔离测试环境启用 `REQUIRE_DEVICE_DEPLOYMENT`，分别用已安装的官方 1.5.1 客户端运行 `--deploy --token <测试令牌>`、`--assign --token <测试令牌> --device_name Office-PC`。记录设备 ID、UUID、公钥、API 返回、数据库所有者和客户端重新连接结果。测试重复部署、不同 UUID、公钥冲突及撤销后重新注册。令牌仅用于测试，完成后立即撤销。

**完成条件：** 在真实设备和配套 API、hbbs、hbbr 环境中完成上述操作；合法请求成功，其他情形明确拒绝，设备归属和重新连接结果符合预期。

### T002：不同系统客户端登录

**当前状态：实际 API 的密码登录、错误密码、退出登录、JWT 唯一性与会话撤销联合测试通过；TOTP、邮箱验证码及恢复码的接口测试通过；多系统正式客户端与外部身份服务待检查。**

账户、OIDC、LDAP、TOTP、邮箱验证码和恢复码代码位于 `service/user.go`、`service/totp.go`、`service/email_login.go`、`service/oauth.go`、`service/ldap.go`。JWT 随机唯一标识位于 `lib/jwt/jwt.go`。Server 的 `scripts/test_real_api_strict_combination.py` 通过实际 API 登录与登出接口检查会话有效性、JWT 签名和重新登录后的令牌撤销。API 综合检查 #42670400984 和 Server 双架构检查 #42669744598 已通过。

**实际检查方法：** 使用独立测试账户，分别在 Windows、Linux、macOS、Android 和可用的 iOS 设备记录客户端版本与登录结果。依次检查密码登录、错误密码、TOTP 成功与拒绝、邮箱验证码过期和重复使用、恢复码一次性使用、OIDC/LDAP 跳转、退出登录及会话失效。外部身份提供者必须先由维护者配置测试账号。不得在记录中保存密码、恢复码及原始令牌。

**完成条件：** 每个计划支持的系统和登录方式都有真实操作结果，失败流程不会错误建立会话。

### T003：远程控制角色与十四项客户端权限

**当前状态：控制角色 API、Protobuf 编码和实际 hbbs 信令检查通过；与实际 API 的权限读取和传递已通过 AMD64/ARM64 测试；真实远程会话待执行。**

API 文件：`service/control_role.go`。Server 文件：`src/control_roles.rs`。实际服务测试：`scripts/test_control_roles.py`。在未分配角色时保留客户端原有权限。

**实际检查方法：** 准备两台设备，分别设置每项权限为允许、禁止、沿用设备设置。在直连、中继及实际支持的桌面系统上检查键盘、鼠标、剪贴板、文件传输、音频、终端、录制、隐私模式、USB 等十四项对应能力。每项记录角色配置、连接模式和实际客户端行为；系统不支持的能力应明确标注。

**完成条件：** 目标客户端的权限与服务端配置一致，修改权限后新连接生效，不影响未分配角色的设备。

### T004：严格模式组合运行

**当前状态：真实 API、hbbs、JWT 登录与登出、连接访问规则、设备撤销后拒绝新连接、控制角色、中继请求和 WebRTC SDP/ICE 信令已在 AMD64/ARM64 通过；实际客户端连接与公网中继待检查。**

Server 的 `scripts/test_real_api_strict_combination.py` 在实际 API 与 hbbs 中检查有效 JWT、非法签名、失效会话、允许及禁止规则、控制权限与 API 中断，`scripts/test_real_api_registration.py` 检查设备身份及许可。两种架构的完整组合构建、服务通信、容器重新创建与数据保存检查已在 CI #42669744598 通过。

**实际检查方法：** 在一套隔离部署中同时开启 `REQUIRE_DEVICE_DEPLOYMENT=Y`、`MUST_LOGIN=Y` 和集中连接访问规则，验证合法设备经直连或中继连接成功，未部署设备、失效账户、无许可用户以及 API 服务暂时不可用时的处理；分别检查 WebSocket、WebRTC 客户端。

**完成条件：** 各严格设置组合启用后授权准确，拒绝和正常连接都能在客户端实际复现。

### T005：公网多中继与 GeoLite2 数据

**当前状态：多中继选择、失败后替代选择、Country CSV 与 MMDB 解析及 reload-geo 已通过 Rust 单元和真实 hbbs 测试；跨地区公网与正式 GeoLite2 数据待执行。**

Server 文件：`src/relay_policy.rs`、`src/geoip_country.rs`。测试使用 MaxMind 公开样例文件 `tests/fixtures/GeoLite2-Country-Test.mmdb`，通过 CIDR、国家代码、节点状态及重载逻辑检查。

**实际检查方法：** 准备至少两个位于不同地区的可公开访问的中继节点，由部署者提供已获授权的 GeoLite2 Country MMDB 或 CSV。在每个地区的真实客户端检查连接选择与延迟，暂停一个中继节点后检查重新连接；更新 GeoLite2 文件并执行 `reload-geo`，复查国家匹配。

**完成条件：** 不同地区的客户端能连接到可达中继，节点不可用后完成替代选择，国家匹配及文件重载正确。

### T006：SQLite、MySQL 和 PostgreSQL 升级与恢复

**当前状态：自动化检查已覆盖数据库迁移版本 290，并在 SQLite、MySQL 和 PostgreSQL 上检查升级及恢复；生产数据恢复演练仍需执行。**

文件：`service/database_multi_driver_test.go`、`service/database_restore_test.go`、`scripts/test_database_restore.sh`。工作流：`.github/workflows/verify-database-upgrades.yml`。自动测试在 SQLite、MySQL 8.4 和 PostgreSQL 16 检查旧策略、审计、客户端任务及重复迁移；SQLite 采用停止连接后的文件复制，另两种数据库使用原生逻辑备份与恢复。

**实际检查方法：** 按 [数据库备份恢复说明](DATABASE_BACKUP_RESTORE.md) 从实际环境生成备份，在独立测试环境恢复并升级到版本 290，核对用户、群组、设备、审计、地址簿、设备策略、通知和安装任务。数据库之外还须保存 Server 身份密钥及客户端安装文件。

**完成条件：** 实际数据恢复后可登录、可查询历史记录、服务器身份不变，恢复及重新启动通过。

### T007：SMTP、邀请与邮箱认证

**当前状态：模拟 SMTP 发送、账户邀请、密码恢复、邮箱验证码及 TLS/STARTTLS 加密连接测试通过；真实外部邮件服务商待执行。**

主要代码：`service/mail.go`、`service/mail_tls_test.go`、`service/user_invitation.go`、`service/password_recovery.go`、`service/email_login.go`。加密测试包括隐式 TLS、STARTTLS、SMTP 用户认证及错误时拒绝。邮箱服务凭据使用加密字段存储。

**实际检查方法：** 在测试域名和真实邮箱配置 STARTTLS 或 TLS，发送测试邮件，再创建邀请链接、完成新用户注册、密码恢复、邮箱验证码以及通知邮件接收。分别检查过期链接、无效验证码、TLS 证书不正确、错误 SMTP 凭据及未授权操作。请在正式环境安全地配置 `RUSTDESK_API_MAIL_ENCRYPTION_KEY`，不要将密码提交到仓库。

**完成条件：** 真实邮箱成功收到并完成所有预期业务流程，失败操作不会改变账户密码和安全设置。

### T008：普通服务器与 Docker 部署

**当前状态：AMD64、ARM64 上普通进程与 Docker 自动化部署、实际服务启动、重启及数据保存通过；正式主机部署检查待执行。**

Server 脚本：`scripts/test_native_stack.py`、`scripts/test_container_image.py`、`scripts/test_persistent_container.py`。组合工作流：`.github/workflows/milestone-preflight.yml`。测试检查 API、hbbs、hbbr、WebClient、TCP、UDP、WebSocket，重启后管理员账户、数据库和 hbbs 公钥不变。

**实际检查方法：** 用普通系统服务和 Docker 各部署一套测试环境，设置内部 API URL、密钥、数据卷、GeoLite2 挂载及自动重启。重启系统、升级测试镜像后检查历史数据与文件下载；同时检查反向代理和外部网络端口。

**完成条件：** 两种部署均可在目标主机运行及重启，并保留账户、数据、服务密钥和客户端安装文件。

### T009：管理控制台界面

**当前状态：Chromium 已完成桌面、平板、手机页面及设置中心的交互检查并保存截图；实际部署环境的管理员操作仍需检查。**

文件：`resources/admin/app.js`、`resources/admin/app.test.cjs`、`resources/admin/browser.e2e.cjs`。CI 保存 `rustdesk-admin-browser-screenshots`，含安装包构建与邮件通知页面的桌面、平板和手机截图。已测试关键表单、任务操作及页面水平滚动范围。

**实际检查方法：** 用管理员和只读账户分别在真实桌面与移动浏览器运行账户、设备、地址簿、策略、部署、审计、通知和安装包操作，检查错误信息、按钮可见性、表格及移动端可操作性。记录浏览器版本、屏幕宽度和页面截图。

**完成条件：** 主要管理功能可在预期浏览器独立完成，权限不足的用户看不到无权执行的操作，界面文字清晰。

### T010：官方客户端与协议版本兼容

**当前状态：官方 1.4.5、1.5.0、1.5.1 源代码字段已检查；官方 1.5.1 在 Windows、Linux、macOS 的运行和 Android 模拟器启动已通过，Linux 客户端已对实际 API 完成部署与分配；真实账号登录和旧版本远程连接仍需检查。**

文件：`scripts/verify_upstream_client_commands.py` 和 `.github/workflows/verify-client-runtime.yml`。源码检查覆盖 `--assign`、`--deploy` 的请求字段及预期返回码，以及 `--option` 的管理员权限要求。

**实际检查方法：** 在真实系统中分别使用官方客户端版本登录当前 API、注册设备、部署及变更策略；比较不同版本的 Protobuf 消息、错误提示和双重认证结果。不能把只验证请求字段视为完整兼容。

**完成条件：** 已计划支持的版本均有明确的实际运行结果和失败流程检查。

### T011：里程碑发布前构建预演

**当前状态：API v3.0.0-api.109 已完成正式发行，Linux AMD64、Linux ARM64、Windows AMD64 安装文件及双架构 Docker 镜像均已发布。**

Server 的 `.github/workflows/milestone-preflight.yml` 在 AMD64、ARM64 上构建配套 API 和 Server，运行普通进程及 Docker 服务检查，不创建 Release，也不推送正式镜像。

**完成条件：** 已在 AMD64 和 ARM64 上通过构建、启动、WebClient 和数据保存检查；正式发行文件与容器镜像已经生成。

### T012：正式里程碑核查与批准

**当前状态：API PR #14 已合并并发布 v3.0.0-api.109，Server 维持 v1.1.17-api.21；发行文件、Docker 镜像和自动化检查均已核对。实际设备和外部服务检查继续保留。**

本次发行已经核查两个版本的源代码提交、GitHub Actions、数据库版本与发行文件。后续具备实际设备、外部邮箱和业务数据库的测试条件时，继续完成 T001–T010 中相应的实际环境检查。

**完成条件：** 本阶段的维护者批准、合并与正式版本发布已经完成；需要真实运行环境的测试继续按任务记录执行。

### T013：客户端安装文件制作

**当前状态：Windows、Linux、macOS 的 x64 与 ARM64 六种官方 1.5.1 文件已完成配置、下载、校验与任务管理；真实目标系统安装待执行。**

文件：`service/client_build.go`、`service/client_build_worker.go`、`service/client_packages.go`、`http/controller/admin/client_builder.go`。Windows 使用官方可识别的服务器配置文件名；Linux/macOS ZIP 保留官方 DEB/DMG 并包含设置脚本，脚本会读取核对配置。下载时以 SHA-256 检查后直接发送已核对的文件内容。

使用说明：[客户端安装文件制作](CLIENT_BUILDER.md)。还需在真实 Windows、Debian/Ubuntu、macOS 设备检查签名、安装、权限、服务器配置、登录及连接。

**完成条件：** 六类文件均可下载并通过完整性核对；目标设备的实际安装与连接符合预期。

### T014：设备安全策略扩充

**当前状态：Go、管理界面、设备心跳、严格登记许可和设备所有者一致性检查通过；真实客户端策略效果待检查。**

API 的 `service/device_policy.go`、`model/device_policy.go` 增加访问模式、键盘鼠标、剪贴板、文件传输和连接确认方式，并记录受服务器管理的配置项，支持取消策略时清除。`http/controller/api/index.go` 会核对严格登记设备的部署许可，未登记或已撤销许可的设备无法继续通过心跳获取策略与会话管理指令。

**完成条件：** 管理页面可修改选项，真实客户端心跳后正确更新；撤销时恢复预期行为。

### T015：独立 API 令牌

**当前状态：已实现并通过自动化测试。**

文件：`service/api_token.go`、`http/middleware/automation_auth.go`、`http/router/automation.go`。测试验证有效期、撤销、管理员身份复核、只读与修改权限分离及未知令牌拒绝。

**完成条件：** 持有对应权限的令牌可以调用指定接口，其他调用被拒绝；撤销及账户停用立即生效。现有自动化检查已满足。

### T016：指定账户及用户组范围内的管理操作

**当前状态：已实现并通过自动化测试；设备所有者转移后同步撤销目标设备的旧会话。**

文件：`service/admin_role.go`、`http/middleware/admin_privilege.go`、`http/controller/admin/peer_transfer.go`。测试检查更新、删除与设备所有者转移，要求原所有者和目标所有者均在授权范围；设备转移同步修改部署记录。

**完成条件：** 合法范围内管理操作可执行，跨账户及管理员身份变更等受限制操作被拒绝。现有自动化检查已满足。

### T017：批量管理

**当前状态：已实现并通过 Go 和浏览器自动测试。**

包含联系人添加与撤销、地址簿共享、设备部署许可批量修改、回收站批量恢复与永久删除。相关文件：`http/controller/admin/contact_batch.go`、`address_book_rule_batch.go`、`device_deployment_batch.go`、`service/address_book_batch.go`。

**完成条件：** 多项输入成功时记录完整，出现无效或跨账户目标时不产生部分修改；界面给出明确结果。现有自动化检查已满足。

### T018：GeoLite2 MMDB 和 CSV

**当前状态：已实现并通过 Rust 单元及实际 hbbs 测试。**

Server 的 `src/geoip_country.rs` 支持两种格式，优先查询 MMDB，未取得国家代码时再查询 CSV；可通过 `reload-geo` 重新读取，加载失败时保留可用数据。MaxMind 的公开示例文件保存在 `tests/fixtures`。

**完成条件：** IPv4/IPv6 查询、格式切换与配置刷新均正确。正式 GeoLite2 文件的使用与跨地区连接仍由 T005 检查。

### T019：邮件事件通知

**当前状态：已实现，模拟 SMTP、事件过滤、失败重试和记录保存测试通过；正式服务商收件待检查。**

文件：`service/notification.go`、`model/notification.go`、`http/controller/admin/notification.go`。覆盖登录失败、部署变更、会话结束请求及完成、文件传输记录。发送由单独的队列任务处理，失败状态可查询及重新提交。

**完成条件：** 管理员可选事件和收件人，相关邮件按设置发送且错误可查询。真实收件由 T007 继续检查。

### T020：设备、地址簿及连接审计管理

**当前状态：设备查询与 CSV、审计日期和设备筛选、连接备注及共享地址簿批量管理已通过自动化测试；实际团队使用检查待执行。**

文件：`http/controller/admin/peer_export.go`、`audit_management.go`、`service/audit_note.go` 及地址簿相关控制器。CSV 检查会限制当前管理员可读取的用户范围并处理电子表格危险输入。

**完成条件：** 真实管理账号能够筛选、导出并维护相关信息，结果与数据库中可访问的记录一致。

### T021：官方客户端及功能更新研究

**当前状态：已整理 RustDesk 官方 1.5.1 功能候选，后续需在版本更新时持续检查。**

记录：[官方版本与功能研究](UPSTREAM_FEATURE_WATCH.md)。候选包括可信设备、安装程序更新验证、移动端文件传输、客户端系统兼容与网络设置。现有功能优先使用官方协议与客户端选项。

**完成条件：** 官方发布版本更新时，补充版本、相关功能、使用场景、修改位置和测试要求，必要时创建新开发任务。版本监测属于持续维护。

### T022：中文、其他语言及移动端使用

**当前状态：中文管理界面以及桌面、平板、手机交互检查已通过，手机顶部导航布局已验证；其他语言覆盖与不同系统的无障碍操作仍需完善。**

当前文档已包括中文版 API、部署、测试、备份、设置中心及客户端安装说明；Chromium 已测试设备管理、设置中心、邮件通知和安装包构建界面。

**完成条件：** 中文界面及文档清晰、移动端可执行主要管理操作；根据实际用户需求完成所支持的其他语言翻译并对相应界面进行检查。

## 当前仍依赖实际环境的事项

| 必需资源 | 对应任务 | 可获得的检查结果 |
| --- | --- | --- |
| Windows、Linux、macOS 与可用的移动设备，至少两台能够互相连接的设备 | T001、T002、T003、T004、T009、T010、T013、T014 | 实际安装、登录、部署、会话权限及远程控制 |
| 至少两个不同地区的公网中继节点和已获授权的正式 GeoLite2 Country 数据 | T005 | 地区中继选择、重载与故障替代 |
| 配置好的实际 SMTP 服务、测试域名和可接收邮件的账户 | T007、T019 | 真实发件、收件、邀请及密码恢复 |
| 业务数据的受保护备份、独立恢复环境以及部署机器的运行权限 | T006、T008 | 实际数据库和安装文件恢复、系统重启后数据保持 |

相关资源尚未提供给此开发会话时，仍可继续修正代码、扩充自动化检查和维护台账；上述项目的实际环境结果必须在执行之后登记，不能预先写为完成。

## 新增检查结果的登记格式

以后每次执行 P0 任务，至少登记：**任务编号、执行日期、API 提交、Server 提交、客户端版本及操作系统、测试网络、启用配置、执行步骤、预期结果、实际结果、关联的 GitHub Actions 或测试截图、是否需要修改代码及复查状态**。敏感信息一律删除或遮蔽。

新增自动化测试通过后更新本文件和统一实施台账；功能代码发生变化时，在相应分支完成修改和复查。以后创建新的正式发行版本仍需维护者批准。

## 2026 年 11 月 11 日：正式发行与设置中心

本次发行采用 [API 提交 a98f539](https://github.com/HSOS6Miyabi/rustdesk-api/commit/a98f539a060698d1dcef7122573d4f3574f47de4)，数据库迁移版本为 **290**。

- [正式发行工作流 #42978795619](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42978795619)：九项任务均成功，完成 Go、浏览器、三平台编译、Release 和双架构 Docker 发布。
- [GitHub Release v3.0.0-api.107](https://github.com/HSOS6Miyabi/rustdesk-api/releases/tag/v3.0.0-api.107)：三个平台的安装文件均可获取。
- 设置中心已通过配置优先级、密钥原文保护、重启后恢复、操作权限、失败通知重试和移动端交互检查。
- 通知任务恢复、多个实例之间的任务认领、安装包长任务和缓存管理均已纳入自动化检查。
- 需要真实 SMTP、远程连接、跨地区中继与生产数据库的检查项目继续保留在 T001–T022 的对应条目中。
