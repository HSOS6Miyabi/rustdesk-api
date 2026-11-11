# 测试说明

## API

```bash
go test ./... -count=1
go build -o /tmp/rustdesk-api ./cmd/apimain.go
node --check resources/admin/app.js
node --test resources/admin/app.test.cjs
```

设备命令测试位于 `http/controller/api/device_commands_test.go`、`device_commands_integration_test.go`。测试检查设备 UUID、管理员分配、地址簿、策略、部署重复提交和内部接口身份验证。使用 SQLite 临时数据库。

浏览器测试：

```bash
npm --prefix resources/admin install --no-save --no-package-lock playwright@1.56.0
node --test resources/admin/browser.e2e.cjs
```

## Server

```bash
git submodule update --init --recursive
cargo test --lib --tests
cargo build --bins
python3 scripts/test_runtime.py target/debug/hbbs target/debug/hbbr
python3 scripts/test_webrtc_rendezvous.py
```

## 实际设备配合测试

1. 不启用严格设备登记：原有 RustDesk 设备连接和注册成功。
2. 同时启用 API 和 Server 的严格登记：未部署设备不能通过 `RegisterPk`。
3. 使用登录令牌执行 `rustdesk --deploy --token ...`：设备被登记，随后注册成功。
4. 使用 `rustdesk --assign --token ...`：用户、设备组、策略和地址簿记录正常变化。
5. 测试相同 ID 的不同 UUID 或公钥被拒绝；重复提交同一身份可正常处理。
6. 启用 `MUST_LOGIN=Y`：未登录连接、错误令牌与中继请求被拒绝；合法登录仍能连接。
7. 断开 API：严格设备登记被拒绝，其他未启用该功能的服务保持原有行为。

GitHub Actions：API [PR #6](https://github.com/HSOS6Miyabi/rustdesk-api/pull/6)，Server [PR #4](https://github.com/HSOS6Miyabi/rustdesk-server/pull/4)。运行结果以实际检查记录为准。

## 三级策略和活动连接

API 新增以下测试：

```bash
go test ./service -run 'TestDevicePolicyAssignmentPriorityAndFallback|TestDevicePolicyTargetReassignmentAndCleanup' -count=1
go test ./http/controller/api -run 'TestActiveSessionsAndHeartbeatDisconnect' -count=1
go test ./service -run 'TestSessionManagementRejectsStaleAndInvalidRequests' -count=1
```

## API 令牌

```bash
go test ./service -run 'TestAPITokenIssuancePermissionsAndRevocation|TestAPITokenDisabledAccountCannotUseToken' -count=1
go test ./http/controller/api -run 'TestOfficialDeviceCLIWithScopedAPIToken' -count=1
```

管理控制台浏览器测试包含设备分组策略、活动连接及令牌创建。测试人员还应在正式发布前使用官方 RustDesk 客户端实际执行 `--assign` 和 `--deploy`。

## 共享地址簿和多群组成员

```bash
go test ./service -run 'TestAddressBookSharingAdditionalGroupsAndEveryone|TestAddressBookShareRuleUpdateCanSelectEveryone' -count=1
go test ./http/controller/api -run TestSharedAddressBookProfilesRespectIndividualAndAdditionalGroupPermissions -count=1
node --test resources/admin/app.test.cjs
```

测试覆盖用户的主要群组和附加群组、不同共享对象的权限优先级、成员退出群组后权限变化，以及 RustDesk 原生 `/api/ab/shared/profiles` 响应的最终权限。

## 地址簿 CSV 与回收站

```bash
go test ./service -run 'TestAddressBookCSV|TestAddressBookTrash|TestDeletingCollection' -count=1
go test ./http/controller/admin -run 'TestAddressBookCSVController|TestAddressBookTrashHTTP' -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

CSV 测试覆盖中文、数据导出、公式字符处理、跨用户导入限制、重复导入和事务提交。回收站测试验证恢复原始设备设置、拒绝重复恢复、永久删除和删除分类时的清理。

## SMTP、密码恢复与成员邀请

```bash
go test ./service -run 'TestSMTP|TestPasswordRecovery|TestTeamInvitation' -count=1
go test ./http/controller/admin -run 'TestPublicPasswordRecovery|TestInvitationPublicRegistration' -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

SMTP 测试在本地启动临时邮件服务并检查实际发送内容。密码恢复测试验证到期、重复使用、密码修改和原有登录会话撤销。邀请测试覆盖邮件链接、群组归属、用户注册、撤销和重复使用。

当前客户端构建和邮件通知等功能的综合 CI：[GitHub Actions #42620320903](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620320903)。

正式投入使用前，应使用实际 SMTP 服务提供商测试发送、接收、TLS 连接与外部可访问的邀请链接。

## 远程连接审计备注

```bash
go test ./service -run TestConnectionAuditNotes -count=1
go test ./http/controller/admin -run TestConnectionAuditNoteHTTPAndCSV -count=1
node --test resources/admin/browser.e2e.cjs
```

检查备注添加、修改、清除、长度限制、管理员界面保存以及历史连接 CSV 导出。测试同时确认原始连接编号和远程设备信息保留。

## TOTP 双重认证及恢复码

```bash
go test ./service -run 'TestTOTP|TestDeleteUser' -count=1
go test ./http/controller/api -run TestNativeRustDeskLoginRequiresTOTP -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

检查 RFC 6238 时间步进计算、绑定、恢复码单次使用、登录挑战有效期和最大尝试次数、官方客户端 HTTP 响应、网页两步登录及账户删除后的认证数据清理。

## 邮箱验证码登录

```bash
go test ./service -run 'TestEmailLogin|TestEmailCode' -count=1
go test ./http/controller/api -run TestNativeRustDeskEmailCodeLogin -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

测试覆盖邮件验证码生成与发送、用户邮箱绑定、验证码单次使用、有效期和验证尝试次数、账户邮箱改变后的处理，以及官方客户端和网页双阶段登录的实际 HTTP 响应。正式部署前还需要使用实际邮件服务商和 RustDesk 客户端进行网络测试。

## 多中继节点管理与动态选择

API：

```bash
go test ./service -run TestRelayNodeManagementAndSubnetValidation -count=1
go test ./http/controller/api -run TestInternalRelayNodeListingRequiresSharedKey -count=1
node --test resources/admin/browser.e2e.cjs
```

Server：

```bash
cargo test --lib --tests
cargo build --bins
python3 scripts/test_managed_relays.py target/debug/hbbs
```

实际服务测试将启动临时 API 与中继节点，检查 IP 网络偏好、动态刷新、节点断线后的替代选择以及恢复启动时配置的中继地址。

## GeoLite2 中继国家识别

API：

```bash
go test ./service -run TestRelayNodeManagementAndSubnetValidation -count=1
go test ./http/controller/api -run TestInternalRelayNodeListingRequiresSharedKey -count=1
node --test resources/admin/browser.e2e.cjs
```

Server：

```bash
cargo test --lib --tests
cargo build --bins
python3 scripts/test_managed_relays.py target/debug/hbbs
```

测试程序创建本地 GeoLite2 Country CSV 示例文件，并使用真实 hbbs 和两个模拟中继进行 IP 国家选择与数据重新载入验证。实际部署需使用管理员自行取得的最新 GeoLite2 Country CSV 文件。

## 远程控制角色

```bash
go test ./service -run 'TestControlPermissionsAndAssignmentPriority|TestControlRoleAssignmentUpdatesAndGroupDeletion' -count=1
go test ./http/controller/api -run TestInternalControlPermissionsUseEffectiveRole -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

验证权限位编码、14 项客户端权限配置、按设备和分组的优先级、删除后的数据处理、内部 API 访问验证和浏览器中的角色编辑。Server 使用 `scripts/test_control_roles.py` 启动真实 hbbs 进程，检查远程会话协议中的权限字段。

## 管理员角色与限定范围访问

```bash
go test ./service -run 'TestAdministrator|TestDelegated|TestAdminResetSecondFactors' -count=1
go test ./http/controller/admin -run 'TestDelegatedAdmin|TestAdminRoles' -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

测试检查同一用户的多个角色合并、全局权限、指定账户与用户组的查看范围、拒绝查看其他账户设备、角色启停和删除、管理员角色创建页面及已获授权成员的导航。用户双重认证重置测试还检查现有会话是否撤销。

正式部署前需要使用拥有不同角色的真实账户检查管理页面以及不同设备归属情况下的数据范围。

## 服务器执行的连接访问规则

API：

```bash
go test ./service -run 'TestConnectionAccess' -count=1
go test ./http/controller/api -run TestInternalConnectionAuthorizationChecksRuleAndToken -count=1
node --test resources/admin/app.test.cjs
node --test resources/admin/browser.e2e.cjs
```

Server：

```bash
cargo test --lib --tests
cargo build --bins
python3 scripts/test_connection_access.py target/debug/hbbs
```

API 测试覆盖设备所有者、指定用户、主要群组和附加群组、允许与禁止规则的优先级、无效登录令牌、未知设备和删除数据后的关联规则清理。

Server 测试启动真实 hbbs 和模拟 API，验证已授权的 `PunchHoleRequest`、`RequestRelay` 能够发送到被控设备；不符合规则的请求不会转发，API 无法访问时也不会允许新连接。


## 客户端构建器和独立 API 令牌

```bash
go test ./service -run 'TestClientBuild|TestOfficialLinuxAndMac|TestClientSetupScript|TestInterruptedClient' -count=1
go test ./http/middleware -run 'TestAutomationTokenRequiresScopeAndAdministrator' -count=1
go test ./http/router -run 'TestAutomationDeviceInventoryAndRouteIsolation' -count=1
node --test resources/admin/app.test.cjs resources/admin/browser.e2e.cjs
python3 scripts/verify_upstream_client_commands.py
```

`.github/workflows/verify-client-build.yml` 实际下载 RustDesk 1.5.1 的 Windows EXE、Linux DEB 和 macOS DMG（x64 与 ARM64），检查大小、SHA-256、文件类型和原始安装程序内容保持不变。Linux/macOS 的 ZIP 及配置脚本还通过 Go 单元测试和 Linux 模拟安装命令检查，确认客户端设置能够写入及读取，保存未生效时正确停止。管理界面的浏览器测试检查安装配置保存、构建历史、桌面和移动端页面。生成的截图由 GitHub Actions 的 `rustdesk-admin-browser-screenshots` 文件提供。

## 邮件事件通知、设备策略与导出

```bash
go test ./service -run 'TestNotification|TestClientSecurityPolicyOptions' -count=1
go test ./http/controller/admin -run 'TestDeviceExport|TestAuditExport' -count=1
node --test resources/admin/app.test.cjs resources/admin/browser.e2e.cjs
```

邮件通知测试使用模拟 SMTP，检查收件人、事件过滤、重复通知、重试和记录保存。策略测试检查访问模式、键盘鼠标、剪贴板、文件传输与连接确认设置的发送和撤销。导出测试检查设备归属、日期、来源和 CSV 内容。

## SQLite、MySQL 和 PostgreSQL 数据升级

```bash
go test ./service -run '^TestDatabaseMigrationAcrossDrivers$' -count=1 -v
```

本地仅配置 SQLite 时会运行 SQLite 测试。在 CI 的 `.github/workflows/verify-database-upgrades.yml` 中，会同时启动 MySQL 8.4 和 PostgreSQL 16 临时数据库，并设置 `RUSTDESK_TEST_MYSQL_DSN`、`RUSTDESK_TEST_POSTGRES_DSN`，验证旧结构迁移到版本 282、现有数据保存及重复迁移行为。对应运行：[数据库检查 #42620320922](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42620320922)。

## Server 普通进程与 Docker 双架构部署

配套 Server 仓库的 `.github/workflows/milestone-preflight.yml` 在 AMD64、ARM64 上分别编译两个项目。检查脚本 `scripts/test_native_stack.py` 运行独立 API、`hbbs`、`hbbr` 并重启两次，检查管理页面、WebClient、通信端口和数据保存；`scripts/test_persistent_container.py` 验证 Docker 容器重新创建后账户、数据库和公钥保持一致。

对应运行：[双架构部署检查 #42618003889](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42618003889)，已通过。正式公网与真实客户端连接仍需另行完成。


### 数据库备份恢复检查

`service/database_restore_test.go` 在停止数据库连接后执行 SQLite 文件备份和恢复。`scripts/test_database_restore.sh` 由 GitHub Actions 在临时 MySQL 8.4、PostgreSQL 16 服务中调用数据库原生导出工具，在独立数据库恢复后验证历史记录。对应 [CI #42627684759](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42627684759)，结果为成功。

测试环境由工作流自行建立，相关命令及生产备份注意事项见 [数据库备份恢复说明](DATABASE_BACKUP_RESTORE.md)。正式数据库恢复应当另行进行，并保留数据库、Server 身份文件和客户端构建文件。


## 官方 Linux 客户端与实际 API

GitHub Actions 的 `.github/workflows/verify-native-linux-commands.yml` 使用已核对 SHA-256 的 RustDesk 1.5.1 Linux 官方安装文件，启动系统服务，编译本仓库 `cmd/apimain.go`，再运行：

```bash
python3 -m py_compile scripts/test_official_linux_real_api.py
sudo -E python3 scripts/test_official_linux_real_api.py /path/to/apimain
```

此检查在独立的临时 API 数据库中创建短期测试会话，验证 `--deploy`、重复部署、紧接着执行 `--assign`、设备所有者与身份信息、`/api/sysinfo` 以及 SQLite 保存结果。

已完成的测试：[GitHub Actions #42646358750](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42646358750)。Server 的双架构联合检查见 [Server #42647450607](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42647450607)。


## 设备所有者转移与登录会话

在同一测试环境运行下列命令，检查命令行设备分配、管理员转移设备所有者，以及设备登录会话的保存与撤销：

```bash
go test ./service -run '^TestRevokeDeviceSessionsPreservesUnrelatedEmptyUUID$' -count=1
go test ./http/controller/api -run '^TestManagedDeploymentCreatesOwnedPeerAndProtectsSysInfo$' -count=1
go test ./http/controller/admin -run '^TestScopedAdministratorsCanModifyOnlyAssignedAccountsAndDevices$' -count=1
```

测试覆盖设备所有者变化、与设备编号或 UUID 关联的会话撤销、其他设备会话保留，以及同一所有者重复分配的结果。最新综合测试记录：[GitHub Actions #42652681801](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42652681801)。


## 严格登记模式的设备心跳

```bash
go test ./http/controller/api -run '^TestManagedHeartbeatChecksApprovedDeployment$' -count=1
go test ./http/controller/api -run '^TestDeviceHeartbeatReturnsNativeRustDeskStrategyOnlyForMatchingUUID$' -count=1
```

测试使用临时 SQLite，验证批准设备策略下发、未登记和撤销许可后的空响应、UUID 身份核查，以及未启用严格登记时的已有心跳行为。

[GitHub Actions #42655936250](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42655936250) 已完成 Go、WebClient 和浏览器测试。


## JWT 唯一性与实际 API 登录

```bash
go test ./lib/jwt -run '^TestLoginTokensAreUniqueAndSigned$' -count=1
node --test resources/admin/browser.e2e.cjs
```

API 签发 JWT 时生成独立随机标识。上述 Go 测试连续生成并验证多个令牌。Server 的 `scripts/test_real_api_strict_combination.py` 启动真实 API 和 `hbbs`，完成密码登录、JWT 验证、登出、再次登录、旧令牌拒绝、访问规则与 WebRTC SDP/ICE 信令检查。

管理控制台的 Chromium 测试同时检查手机端顶部导航保持单行显示，并保存桌面、平板与手机截图。最新成功记录：[API #42666929269](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42666929269)、[Server #42666031144](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42666031144)。


## 严格登记设备的连接授权

```bash
go test ./service -run '^TestConnectionAccessChecksManagedDeviceDeployment$' -count=1
go test ./http/controller/api -run '^TestManagedHeartbeatChecksApprovedDeployment$' -count=1
```

测试包含未登记、已批准、撤销许可、UUID 不匹配和所有者不一致等情况。Server 配套测试 `scripts/test_real_api_strict_combination.py` 在实际 API 与 `hbbs` 中复核撤销后的新连接拒绝。已通过 [API #42670400984](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42670400984) 和 [Server #42669744598](https://github.com/HSOS6Miyabi/rustdesk-server/actions/runs/42669744598)。

## 设置中心与运行可靠性

更新设置中心后，先运行 Go、JavaScript 与真实 API 进程测试：

```bash
go test ./... -count=1
go test -race ./service ./lib/cache ./http/... -count=1
node --test resources/admin/app.test.cjs
go build -o /tmp/rustdesk-api ./cmd/apimain.go
bash scripts/test_settings_startup.sh /tmp/rustdesk-api
```

其中真实 API 进程测试会检查 SQLite 初次创建、环境变量与数据库覆盖的优先级、关闭覆盖后的恢复，以及 SIGTERM 正常退出。

已通过 Chromium 检查设置中心的桌面、平板与手机操作；Go 测试覆盖管理员权限、数据库覆盖版本冲突、敏感值保护、通知失败重试及多实例任务认领。正式发行流程为 [#42978795619](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42978795619)，数据库升级与恢复检查为 [#42975509940](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42975509940)。实际 SMTP 服务商收发、两台设备远程控制和生产数据恢复仍需在部署环境执行。

普通服务器与 Docker 分别通过持久化目录保存启动设置。验证 Docker 时，将数据卷挂载到 `/app/data`，重新创建容器并确认已保存设置能够恢复。


## 检查管理控制台的页面布局

管理控制台使用 Playwright 和 Chromium 检查实际页面。测试服务会返回示例数据，无须连接生产数据库。

安装浏览器测试依赖后，运行全部浏览器交互测试：

~~~bash
npm --prefix resources/admin install --no-save --no-package-lock --no-audit --no-fund playwright@1.56.0
resources/admin/node_modules/.bin/playwright install chromium
node --test resources/admin/browser.e2e.cjs
~~~

需要逐页核查布局时，可以启用完整的截图检查：

~~~bash
RUSTDESK_VISUAL_AUDIT=1 node --test --test-name-pattern="visual audit:" resources/admin/browser.e2e.cjs
~~~

此检查会分别遍历管理员桌面、平板、手机界面及普通用户手机界面的所有导航页面。截图和页面宽度检查结果保存在 resources/admin/test-screenshots/audit/。每次更改管理控制台的页面代码后，GitHub Actions 会重新生成 README 中的界面截图。
