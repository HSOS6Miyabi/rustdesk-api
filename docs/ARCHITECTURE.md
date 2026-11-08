# 系统结构与设备命令接口

## 服务职责

- `rustdesk-api`：Gin HTTP API、账户、设备记录、群组、地址簿、集中策略及后台管理。
- `rustdesk-server`：`hbbs` 注册与连接协商；`hbbr` 中继。
- 数据库：API 支持 SQLite、MySQL、PostgreSQL；Server 保存现有设备注册信息。

## 官方客户端设备命令

`POST /api/devices/cli` 使用 `Authorization: Bearer <登录令牌>`，接收客户端本机 `id`、`uuid`，以及可选的 `user_name`、`strategy_name`、`address_book_name`、`address_book_tag`、`address_book_alias`、`address_book_password`、`address_book_note`、`device_group_name`、`note`、`device_username`、`device_name`。

只能修改登记身份一致的设备；管理员可调整设备所有者，普通用户只能修改自己的设备。关联的用户、设备组和策略必须已存在。地址簿分类不存在时自动为目标用户创建。修改在数据库事务中执行。

`POST /api/devices/deploy` 使用同样的令牌，接收 `id`、Base64 格式的 `uuid` 和 `pk`。启用严格设备登记后，返回 `OK`、`INVALID_INPUT` 或 `ID_TAKEN`。默认未启用时返回 `NOT_ENABLED`，官方客户端据此停止要求部署。

## 内部设备登记

`POST /internal/v1/device/authorize` 仅供 `hbbs` 使用，在请求头中提供 `X-RustDesk-Internal-Key`。请求体包含 `id`、`uuid` 和 `pk`。API 根据 `DeviceDeployment` 记录返回 `OK`、`NOT_DEPLOYED` 或 `ID_TAKEN`。

Server 启用 `REQUIRE_DEVICE_DEPLOYMENT=Y` 后，在处理 `RegisterPk` 时调用该接口。默认地址为 `http://127.0.0.1:21114/internal/v1/device/authorize`，可通过 `RUSTDESK_API_INTERNAL_URL` 修改。请求最长等待 3 秒；API 不可用时拒绝登记，避免意外开放严格登记模式。

必须限制内部接口的网络访问。跨机器部署时应使用受信任网络与 HTTPS。

## 连接登录

`MUST_LOGIN=Y` 时，`hbbs` 必须设置 `RUSTDESK_API_JWT_KEY`，并校验 `PunchHoleRequest` 与 `RequestRelay` 的令牌。默认 `MUST_LOGIN=N` 保留原有连接方式。
