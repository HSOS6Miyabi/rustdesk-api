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

## 三级策略

API 数据库保留原来的设备直接分配表，新增用户与设备组分配表。客户端心跳时按设备、用户、设备组的顺序选择有效策略。停用或删除高优先级策略后可以使用下一级策略，没有适用策略时返回空设置以清除以前下发的管理配置。

## 活动连接与结束指令

客户端 `/api/heartbeat` 请求上报 `id`、`uuid`、`conns` 和 `modified_at`。服务端验证设备 ID 和 UUID 后，更新活动连接记录。

管理员请求结束连接时，API 确认目标连接近期存在，再保存具有有效期的结束请求。客户端下一次心跳收到 `disconnect: [连接编号]`。客户端此后停止上报该连接编号时，API 将请求标记为已经完成。

## 管理令牌

管理令牌与普通 RustDesk 登录令牌分别保存。API 令牌格式为 `rdapi_` 加随机字符，数据库仅保存 SHA-256 摘要。管理令牌仅允许访问带有相应操作权限的 `/api/devices/cli` 和 `/api/devices/deploy`。其余客户端接口继续使用普通登录令牌。

管理令牌创建后只显示一次；删除操作保留撤销记录。过期、撤销和停用所属账户后，令牌不再有效。

## 共享地址簿权限

地址簿共享规则存放在 `AddressBookCollectionRule`，由 `UserId` 指明所有者，`CollectionId` 指明地址簿。分享对象类型：

- `1`：指定用户。
- `2`：指定群组，用户的主要群组及附加群组均参与计算。
- `3`：所有用户，此时 `ToId=0`。

最终权限按指定用户、群组、所有用户的顺序选择。用户同时属于多个群组时，选用群组中的最高权限。用户本人的地址簿始终具有完全控制权限。

`POST /api/ab/shared/profiles` 根据上述规则生成客户端列表，再按公共锚点组的用户可见范围进行过滤。

## 地址簿 CSV 与回收站

地址簿导入由 `service/address_book_csv.go` 完成，校验所有 CSV 数据和分类归属后统一保存。导出只包含常用设备信息，不包含连接密码和认证数据。回收站通过 `address_books.deleted_at` 标记被删除的条目，恢复时检查原始用户、分类和设备 ID 是否存在冲突。

删除整个地址簿分类会彻底清除分类下的活动和已删除记录。删除用户账户也会清除其历史地址簿数据。

## SMTP 与账户邮件

SMTP 配置存储在 `smtp_settings` 中。SMTP 密码使用 `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 派生的 AES-256-GCM 密钥加密。管理端查看 SMTP 配置时只返回是否设置了密码。可选择 STARTTLS、TLS 或直接 SMTP 连接。

密码恢复和成员邀请分别使用 `password_reset_tokens`、`user_invitations` 数据表，链接中的随机凭据只通过邮箱发送，数据库保存其 SHA-256 摘要。账户恢复与邀请接受过程使用数据库事务提交。密码恢复完成后撤销账户原有登录会话。

密码恢复链接有效期为 15 分钟。成员邀请有效期由管理员选择，范围为 1 至 30 天。所有邮件链接均基于 SMTP 设置中的公开控制台 URL。

## 连接记录备注

备注存储在原有 `audit_conns` 表中，使用 `note` 和 `note_updated_by` 字段。仅管理员可以修改备注，管理操作经已有 HTTP 操作记录组件记录。导出连接日志时将备注及更新人编号追加至 CSV 字段中。

## TOTP 与官方客户端登录

TOTP 密钥保存在 `UserTOTP`，恢复码哈希保存在 `TOTPRecoveryCode`。密码验证成功后，如果用户启用了 TOTP，API 生成有效期五分钟的 `TOTPLoginChallenge`，暂不创建登录会话。用户提交有效动态验证码或一次性恢复码后，服务器才签发原有登录令牌。

客户端使用公开 RustDesk 登录协议中的 `type`、`tfa_type`、`secret` 和 `tfaCode`。管理控制台通过个人设置接口完成绑定、解除和恢复码重新生成。
