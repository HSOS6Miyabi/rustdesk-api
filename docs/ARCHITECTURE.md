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

## 邮箱验证码登录

邮箱验证设置存储在 `EmailLoginSetting`，按用户记录已验证的邮箱。临时验证请求存储在 `EmailCodeChallenge`，只保存一次性凭据和验证码摘要。用户修改邮箱后，旧的验证状态不再适用。

密码登录通过后，若用户启用了 TOTP，优先创建 TOTP 登录请求；否则如已启用邮箱验证，向账户邮箱发送一次性验证码。验证码通过后，才向客户端或 Web 控制台签发登录令牌。

邮件发送使用管理员配置的 SMTP 服务。邮箱验证码申请具有有效期、尝试次数和发送频率限制。

## 多中继节点与动态配置

`RelayNode` 保存中继名称、公开地址、优先级、启用状态及 CIDR 网络范围。内部接口 `GET /internal/v1/relay/nodes` 经过服务密钥验证后返回已启用节点列表。

`hbbs` 的 `relay_policy` 模块定期读取内部配置，并将节点列表交给已有的中继连通性检查程序。连接双方 IP 所属网络用于选择优先中继；没有可适用的规则时使用活动节点轮流选择。托管配置全部取消后，恢复启动时的 `--relay-servers` 设置。

普通安装和 Docker 安装都可以使用。动态同步依赖 Server 进程中的 `RUSTDESK_API_RELAY_NODES_URL` 环境变量；该地址缺失时继续使用原有模式。

## GeoLite2 Country IP 地理位置查询

管理员通过 API 的 `RelayNode.Countries` 保存国家代码。hbbs 将 GeoLite2 Country CSV 解析为 IPv4、IPv6 网络与两位国家代码的对应关系，按前缀长度从长到短寻找最精确的匹配；对应国家信息只保留在本地内存中。

中继选择先考虑节点是否在线，再根据精确 CIDR 范围或国家代码与连接双方 IP 地址的匹配程度进行选择。管理员直接输入的 CIDR 范围优先于自动推断的国家。国家信息缺失时继续执行已有节点选择流程。

GeoLite2 Country CSV 文件由服务器管理员获取，hbbs 可通过 `RUSTDESK_GEOIP_CSV_DIR` 指定文件目录。

## 远程控制角色协议

`ControlRole` 保存官方 RustDesk 的 14 项会话权限，每项使用两个二进制位，保留状态为 0、1、2。`ControlRoleAssignment` 保存角色及设备、用户或设备组的对应关系。

当连接客户端经 hbbs 请求目标设备时，Server 向 API 的内部控制权限接口查询目标设备生效角色，并将其编码写入 `PunchHole.control_permissions`、`FetchLocalAddr.control_permissions` 或 `RequestRelay.control_permissions`。官方被控客户端按对应权限处理远程操作。无角色时不设置这些字段，保留原有设备权限。

Server 可以将角色查询结果按设备短暂缓存，默认 5 秒。角色管理和分配由管理员 Web 控制台操作。

## 管理员角色与委派权限

`AdminRole` 保存角色名称、授权操作、适用范围和启用状态。`AdminRoleAssignment` 将角色与成员账户关联，一个账户可以拥有多个角色。

后台请求统一通过 `AdminPrivilege()` 检查：系统管理员直接通过，普通成员仅能访问明确映射到已授权权限的管理接口。角色管理接口本身仅允许系统管理员操作。管理权限由 `service/admin_role_routes.go` 提供明确的 HTTP 方法与路径匹配，未列出的管理接口不会被受委派角色自动开放。

对 `users.view` 和 `devices.view`，全局角色允许查看全部数据。指定账户和用户组的角色计算允许的用户编号集合，用户列表及设备列表使用 SQL 限制查询范围，详情接口再次检查目标归属。其他修改管理数据的操作目前只允许全局角色。

登录和会话信息的 `route_names` 提供可显示的管理权限名称，网页根据服务器返回的名称显示用户可用的功能；实际操作继续由服务端独立检查权限。

## 服务器连接访问规则

`AccessRule` 描述来源账户、用户组及目标设备或设备组之间的访问关系。服务器维护自己的账户令牌和设备登记资料，判断某个用户是否可以请求连接指定设备。

hbbs 通过 `RUSTDESK_API_CONNECTION_ACCESS_URL` 启用连接授权。每次普通连接或中继请求携带客户端的访问令牌，Server 将令牌和目标设备 ID 发送到受内部密钥保护的 API，API 根据已启用规则返回许可或拒绝。

授权在每次请求时重新计算，以反映设备所有者、群组成员和管理规则的更改。未启用严格模式时不调用该内部接口。严格模式启用后，API 无法访问时拒绝新的连接请求。

该模块覆盖 hbbs 负责协商的连接。由用户直接使用 IP 地址连接目标设备的操作仍按官方客户端原有方式处理。
