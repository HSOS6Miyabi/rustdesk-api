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
cargo test --locked --lib --tests
cargo build --locked --bins
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

最新代码的综合 CI：[GitHub Actions](https://github.com/HSOS6Miyabi/rustdesk-api/actions/runs/42560569953)。

正式投入使用前，应使用实际 SMTP 服务提供商测试发送、接收、TLS 连接与外部可访问的邀请链接。
