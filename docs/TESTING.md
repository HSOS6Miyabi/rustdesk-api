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
