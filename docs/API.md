# API 使用说明

HTTP 响应中的 `code: 0` 表示操作成功。管理员接口统一以 `/api/admin` 开头。登录后，在请求头中设置 `api-token`，值为登录返回的 `data.token`。结构化服务器管理与个人会话接口详见 [控制台与服务器管理](CONSOLE_MANAGEMENT.md)。

以下示例使用占位符 `TOKEN` 表示当前用户的令牌。

## 联系人

### 查看联系人

`GET /api/admin/contact/list`

响应数据包含 `outgoing` 和 `incoming`。其中 `outgoing` 表示您主动向哪些用户公开设备，`incoming` 表示哪些用户主动向您公开设备。

### 添加联系人

`POST /api/admin/contact/add`

```json
{"username":"alice"}
```

也可以使用 `{"user_id": 12}`。成功后，对方即获得查看您名下设备列表的权限。

### 删除联系人

`POST /api/admin/contact/delete`

```json
{"username":"alice"}
```

删除后，对方将无法再通过联系人共享列表看到您的设备。

### 批量添加或撤销联系人

`POST /api/admin/contact/batch`，例如：

```json
{"user_ids":[12,13,14],"action":"add"}
```

将 `action` 改成 `remove` 即可批量撤销。一次最多处理 100 个不同账户。批量添加时所有目标账户必须有效且已启用；所有修改使用同一数据库事务。

## 共享设备

`GET /api/admin/shared_devices/list?page=1&page_size=20&source_type=all&source_id=0&search=`

`source_type` 可以填写 `all`、`contact` 或 `group`。`source_id` 可以进一步指定某位联系人或某个群组的编号。结果的每条设备记录包含 `sources` 数组，用于说明该设备从哪些联系人或群组共享而来。

`GET /api/admin/shared_devices/sources`

获取当前用户可以选择的共享来源，便于构建过滤菜单。

## 公共锚点组

管理员可以创建 `type: 3` 的用户分组。管理员接口必须通过权限验证。

`GET /api/admin/group/anchors?group_id=3`

查看当前公共成员。

`POST /api/admin/group/anchors`

```json
{"group_id":3,"user_ids":[12,25,31]}
```

一次替换该组的全部公共成员。允许使用空数组移除所有公共成员。

`GET /api/admin/group/members?group_id=3`

获取成员列表，包括主要分组成员和附加成员。

`POST /api/admin/group/members/batch`

```json
{"group_id":3,"user_ids":[12,25,31],"action":"add"}
```

`action` 支持 `add`、`remove`、`replace`。用户会保留主要分组；需要移除主要分组成员时，系统会将其移动到默认分组。

## 个人群组管理

个人群组与管理员群组使用独立接口。所有个人群组接口都要求已登录，且在服务端检查当前用户是否是群组所有者。

- `GET /api/admin/my/group/list`：列出自己创建的群组。
- `GET /api/admin/my/group/detail/{id}`：查询自己的群组详情。
- `POST /api/admin/my/group/create`：创建群组，如 `{"name":"家庭设备","type":3}`。
- `POST /api/admin/my/group/update`：修改自己创建的群组，如 `{"id":7,"name":"家庭设备","type":3}`。
- `POST /api/admin/my/group/delete`：删除自己创建的群组，如 `{"id":7}`。
- `GET /api/admin/my/group/members?group_id=7`：查看群组成员。
- `POST /api/admin/my/group/members/batch`：批量添加、移除或替换群组成员，参数与管理员接口相同。
- `GET /api/admin/my/group/anchors?group_id=7`：查看公共成员。
- `POST /api/admin/my/group/anchors`：替换公共成员。
- `GET /api/admin/my/group/user/lookup?username=alice`：通过完整用户名查找用户，返回可用于添加群组成员的基本信息。

普通用户只能操作自己创建的群组。创建者必须保留群组成员身份，普通用户不得通过这些接口修改管理员的成员关系。管理员仍使用 `/api/admin/group` 系列接口管理所有群组。

## 我的审计记录

`GET /api/admin/my/audit_conn/list?page=1&page_size=20`

返回当前用户参与的连接日志，包含：

- `from_peer`、`from_device_name`、`from_user_id`、`from_account`：连接方的设备和账户信息。
- `peer_id`、`to_device_name`、`to_user_id`、`to_account`：被连接方的设备和账户信息。
- `direction`：`outgoing` 表示本人主动发起连接，`incoming` 表示其他设备连接本人的设备，`both` 表示双方设备均属于本人。
- `from_name`、`ip`、`created_at`、`close_time`：连接方的设备登录用户名、来源 IP、开始与结束时间。

`GET /api/admin/my/audit_file/list?page=1&page_size=20`

返回当前用户设备参与的文件传输记录，包含连接双方设备与账户信息、文件路径、时间和 `direction`。

对于已经存在的历史记录，系统会利用保存的设备 ID 获取当前仍可识别的账户和设备名称。后续产生的新记录会在创建时保存双方已知的信息，并在收到补充事件时更新记录。个人接口只允许查询与当前用户或其设备有关的记录；管理员可以继续通过 `/api/admin/audit_conn/list` 和 `/api/admin/audit_file/list` 查询全部记录。

## OIDC 自动入组

创建或更新身份认证服务时，设置 `auto_group_ids`，例如 `"2,3"`。此字段与现有 `oauth_type`、`issuer`、`client_id` 等字段一起提交到 `/api/admin/oauth/create` 或 `/api/admin/oauth/update`。

为保护密钥，身份认证服务的查询接口不会返回 Client Secret。更新时将 `client_secret` 留空即可保留原来的密钥。

## 设备可见范围

RustDesk 客户端请求 `GET /api/users` 或 `GET /api/peers` 时，会依据当前用户所在群组、公共成员和联系人共享关系返回可见内容。公共锚点组中的管理员同样遵守该范围。此机制不拦截用户凭设备 ID 发起的连接。


## RustDesk 1.5.x 连接审计

RustDesk 客户端使用以下接口提交审计事件：

- `POST /api/audit/conn`：连接记录，支持 `id`、`peer`、`conn_id`、`session_id`、`action`、`primary_auth`、`two_factor`、`conn_audit_ref` 和 `nonce`。
- `POST /api/audit/file`：文件传输记录，支持 `id`、`peer_id`、`conn_id`、`path`、`info` 和 `nonce`。
- `POST /api/audit/alarm`：安全告警，支持 `id`、`typ`、`info`、`conn_id`、`conn_audit_ref`、`uuid` 和 `nonce`。

`primary_auth` 中，`1` 表示点击授权，`2` 表示临时密码，`3` 表示永久密码，`4` 表示切换控制方向。`two_factor` 中，`1` 表示 TOTP 动态验证码，`2` 表示可信设备。未提供的数据记为 `0`。

`session_id` 可以是 JSON 整数或字符串。服务器保存原始十进制数值，不会丢失大整数的精度。每次提交的 `nonce` 用于识别重试事件，同一类型、相同 `nonce` 的事件只保存一次。

安全告警的 `typ` 值包括：`0` IP 白名单拒绝、`1` 密码尝试次数过多、`2` 短时间连续登录失败、`6` IPv6 地址段尝试过多、`7` 系统账户登录暂缓、`8` 系统账户登录并发限制、`9` 会话权限违规、`10` 设备 ID 白名单拒绝。

## 安全告警及管理操作

登录用户通过 `api-token` 请求头访问：

- `GET /api/admin/my/audit_alarm/list?page=1&page_size=20`：查询与当前账户有关的安全告警。
- `GET /api/admin/audit_alarm/list?page=1&page_size=20`：管理员查询所有告警。
- `GET /api/admin/audit_operation/list?page=1&page_size=20`：管理员查询操作记录。
- `GET /api/admin/audit/alarms?peer_id=123456&typ=10`：管理员筛选特定设备和告警类别。
- `GET /api/admin/audit/operations?user_id=3&method=POST`：管理员按操作账户和请求方法筛选记录。

管理操作记录保存操作者、接口、请求方法、来源地址与操作结果。系统不记录密码、访问令牌或请求正文。

## 日志保留期限及导出

`GET /api/admin/audit/retention` 查询当前设置；`POST /api/admin/audit/retention` 保存设置：

```json
{
  "connection_days": 90,
  "file_days": 90,
  "alarm_days": 180,
  "login_days": 90,
  "operation_days": 180
}
```

保留期限的单位是天。`0` 表示永久保留，最大值为 `3650`。服务启动时以及此后每 24 小时检查一次需要删除的记录，初始设置为永久保留。

`GET /api/admin/audit/export?kind=audit_conn` 导出 CSV。`kind` 支持 `audit_conn`、`audit_file`、`audit_alarm`、`audit_operation` 和 `login_log`。每次最多导出 10000 条记录；`from_id=记录编号` 用于继续导出较早的记录。必须在 HTTP 请求头中发送管理员令牌。

## 设备安全策略

管理员可以使用以下接口管理策略：

- `GET /api/admin/device_policy/list`
- `POST /api/admin/device_policy/create`
- `POST /api/admin/device_policy/update`
- `POST /api/admin/device_policy/delete`
- `GET /api/admin/device_policy/assignment/list`
- `POST /api/admin/device_policy/assignment/create`
- `POST /api/admin/device_policy/assignment/update`
- `POST /api/admin/device_policy/assignment/delete`

新建策略的示例：

```json
{
  "name": "办公设备",
  "id_whitelist": "123456,789*",
  "ip_whitelist": "192.0.2.0/24",
  "enable_webrtc": true,
  "trust_duration_days": 30,
  "access_mode": "view",
  "enable_keyboard": "N",
  "enable_clipboard": "N",
  "enable_file_transfer": "N",
  "approval_mode": "password-click"
}
```

为已登记的设备分配策略：

```json
{"device_id":"123456789","policy_id":1}
```

`trust_duration_days` 可填写 `0`、`1`、`7`、`30` 或 `90`，其中 `0` 表示不限期限。

新增的远程安全设置遵循 RustDesk 官方客户端设置名称。访问模式可填写 `custom`、`full` 或 `view`；键盘鼠标、剪贴板、文件传输分别使用 `Y`（允许）、`N`（禁止）；连接确认方式可填写 `password-click`、`password` 或 `click`。留空表示沿用客户端设置。服务器保存此前由策略管理的选项，管理员将其改为空值时，后续心跳会发送清除指令。尚未通过策略管理的选项保持原有处理方式。

设置名称及适用平台见 [RustDesk 官方高级设置说明](https://rustdesk.com/docs/en/self-host/client-configuration/advanced-settings/)。控制角色在连接协商期间处理会话权限，这里的策略通过客户端心跳更新本地设置。

`GET /api/device-policy/current?device_id=123456789` 供已登录的设备所有者及管理员查询当前策略。

### 自动应用到 RustDesk 客户端

RustDesk 1.5.x 客户端会向 `POST /api/heartbeat` 发送设备 ID、设备 UUID 和当前策略更新时间。API 确认 ID 和登记 UUID 一致后，返回原生策略数据：

```json
{
  "modified_at": 1800000000000000000,
  "strategy": {
    "config_options": {
      "id-whitelist": "123456,789*",
      "whitelist": "192.0.2.0/24",
      "trusted-device-expiration-days": "30"
    }
  }
}
```

`modified_at` 是示例值，实际由服务器生成。没有变更时返回空对象。管理员撤销策略分配后，客户端在下一次心跳中清除之前下发的设置。

官方 RustDesk 客户端通过本地配置控制 `enable-webrtc`。策略中保存的同名设置提供管理参考，无法通过心跳直接改变官方客户端的本地 WebRTC 开关。

## 官方客户端设备部署和分配

两个接口都使用 `Authorization: Bearer <TOKEN>`。TOKEN 可以是普通登录令牌，也可以是具有对应权限的独立 API 管理令牌。

**设备分配**：`POST /api/devices/cli`

```json
{
  "id": "desk-123456",
  "uuid": "已登记设备的 UUID",
  "user_name": "alice",
  "device_group_name": "办公设备",
  "strategy_name": "办公策略",
  "note": "前台电脑",
  "address_book_name": "办公室",
  "address_book_alias": "前台"
}
```

还支持 `address_book_tag`、`address_book_password`、`address_book_note`、`device_username` 和 `device_name`。UUID 必须与已登记的设备一致。普通用户只能修改自己的设备，管理员可以分配设备所有者。成功返回 HTTP 200。

**设备登记**：`POST /api/devices/deploy`

```json
{"id":"desk-123456","uuid":"Base64 UUID","pk":"Base64 公钥"}
```

启用可选设备登记后，接口会验证设备身份并记录部署许可。响应 `result` 的值包括 `OK`、`INVALID_INPUT` 和 `ID_TAKEN`。默认未启用严格登记时返回 `NOT_ENABLED`。

## 独立 API 管理令牌

管理员在控制台“API 管理令牌”页面创建令牌，令牌原文仅显示一次。

- `GET /api/admin/api_token/list?page=1&page_size=20`：查看令牌名称、前缀、权限、有效期和最近使用时间。
- `POST /api/admin/api_token/create`：创建令牌。
- `POST /api/admin/api_token/delete`：撤销令牌，正文例如 `{"id":3}`。

创建示例：

```json
{
  "name": "自动部署电脑",
  "permissions": "devices.assign,devices.deploy",
  "expires_in_days": 365
}
```

`user_id` 可选，默认使用当前管理员账户。有效天数为 `0` 时长期有效；有效天数最大为 3650。接口返回令牌原文后，数据库只保存摘要。撤销、过期或所属账户停用后，令牌立即失效。

除了官方客户端使用的 `devices.assign`、`devices.deploy`，还可分别授权以下自动化操作：

| 权限 | 允许访问的管理数据 |
| --- | --- |
| `devices.read` / `devices.manage` | 查看设备；创建、修改和删除设备 |
| `deployments.read` / `deployments.manage` | 查看设备登记；批准及撤销登记 |
| `relays.read` / `relays.manage` | 查看和修改中继节点 |
| `sessions.read` / `sessions.disconnect` | 查看活动连接与处理记录；请求结束连接 |
| `control_roles.read` / `control_roles.manage` | 查看和修改远程控制角色及分配 |
| `access_rules.read` / `access_rules.manage` | 查看和修改集中连接访问规则 |
| `policies.read` / `policies.manage` | 查看和修改设备策略 |
| `audits.read` | 查看连接及文件传输记录 |

自动化接口统一使用 `Authorization: Bearer rdapi_...`，位于 `/api/automation/v1`。该接口只接受独立 API 令牌；拥有相应权限的令牌还要求所属账户为当前启用的系统管理员。此限制在每次请求时检查，因此撤销管理员身份后相关管理接口立即不可用。常规 RustDesk 登录令牌不能调用这些接口。

示例：

```bash
curl -H "Authorization: Bearer ${RUSTDESK_AUTOMATION_TOKEN}" \
  "https://desk.example.com/api/automation/v1/devices?page=1&page_size=20"

curl -X POST -H "Authorization: Bearer ${RUSTDESK_AUTOMATION_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{"id":3,"status":"revoked"}' \
  "https://desk.example.com/api/automation/v1/deployments/status"
```

自动化路由与现有控制台使用相同的请求字段及响应格式。可用资源路径：

- `GET /devices`、`GET /devices/:id`；`POST /devices/create`、`/devices/update`、`/devices/transfer`、`/devices/delete`、`/devices/batch-delete`
- `GET /deployments`；`POST /deployments/status`、`/deployments/batch-status`
- `GET /relays`；`POST /relays/create`、`/relays/update`、`/relays/delete`
- `GET /sessions`、`/sessions/disconnects`；`POST /sessions/disconnect`
- `GET /control-roles`、`/control-assignments`、`/control-roles/effective`；对应的 `/create`、`/update`、`/delete` 使用 `POST`
- `GET /access-rules`、`/policies`、`/policies/assignments`、`/policies/user-assignments`、`/policies/group-assignments`；资源修改使用 `POST`
- `GET /audits/connections`、`/audits/files`

上述路径均以 `/api/automation/v1` 为前缀。创建令牌时建议只选择实际需要的权限，例如只读监控使用 `devices.read,sessions.read`。原有设备分配与部署接口及令牌格式保持兼容。

## 用户和设备组策略

管理员继续使用原有 `/api/admin/device_policy` 系列接口，并新增：

- `GET /api/admin/device_policy/user_assignment/list`
- `POST /api/admin/device_policy/user_assignment/create`
- `POST /api/admin/device_policy/user_assignment/update`
- `POST /api/admin/device_policy/user_assignment/delete`
- `GET /api/admin/device_policy/group_assignment/list`
- `POST /api/admin/device_policy/group_assignment/create`
- `POST /api/admin/device_policy/group_assignment/update`
- `POST /api/admin/device_policy/group_assignment/delete`

按用户分配：`{"user_id":5,"policy_id":2}`。按设备组分配：`{"group_id":3,"policy_id":2}`。更新或删除时传入记录的 `id`。

策略优先顺序为：设备直接分配、所属用户、设备组。策略 `status:1` 表示启用，`status:2` 表示停用。停用高优先级策略后自动查找下一层有效策略。

`GET /api/device-policy/current?device_id=desk-123456` 现在还返回 `source` 和 `policy_id`，便于查看配置来源。客户端心跳仍使用原生 `modified_at`、`strategy.config_options` 格式。

## 活动远程连接与结束会话

管理员可以查看设备最近报告的活动连接：

- `GET /api/admin/session/list?page=1&page_size=20`：查看活动连接、设备 ID、连接编号和最后一次心跳。
- `POST /api/admin/session/disconnect`：请求结束某个连接。
- `GET /api/admin/session/disconnect/list`：查看结束连接请求及执行状态。

结束连接示例：

```json
{"device_id":"desk-123456","conn_id":41}
```

设备通过 `POST /api/heartbeat` 上传 `conns` 数组。API 在下一次心跳中通过原生 `disconnect` 数组发送结束指令。

处理状态有 `pending`（等待通知）、`delivered`（已经通知）、`completed`（之后的心跳已确认连接消失）和 `expired`（请求过期）。客户端停止通信时，系统会保留真实的最后处理状态。

## 地址簿共享规则及附加群组

地址簿所有者可以在“我的地址簿共享规则”页面管理自己创建的地址簿共享方式，也可以通过以下接口管理：

- `GET /api/admin/my/address_book_collection_rule/list`
- `POST /api/admin/my/address_book_collection_rule/create`
- `POST /api/admin/my/address_book_collection_rule/update`
- `POST /api/admin/my/address_book_collection_rule/delete`
- `POST /api/admin/my/address_book_collection_rule/batch`：批量向指定账户共享地址簿或撤销共享。

共享到所有用户时：

```json
{
  "collection_id": 12,
  "type": 3,
  "to_id": 0,
  "rule": 1
}
```

`type`：`1` 表示指定用户，`2` 表示指定用户组，`3` 表示所有用户。`type=3` 时 `to_id` 必须为 `0`。

`rule`：`1` 为只读，`2` 为读写，`3` 为完全控制。

批量共享示例：

```json
{"collection_id":12,"target_user_ids":[21,22],"action":"share","rule":2}
```

撤销共享使用 `{"collection_id":12,"target_user_ids":[21,22],"action":"revoke"}`。每次最多指定 100 位用户，必须拥有相关地址簿；若有任何目标账户无效，整批共享不会执行。

当用户同时满足多个规则时，优先使用指定用户的规则；没有指定用户的规则时，使用其所有所属群组中权限最高的规则；仍无群组规则时使用所有用户的规则。用户的主要群组和附加群组都参与计算。

RustDesk 客户端通过 `POST /api/ab/shared/profiles` 获取共享地址簿和最终权限。公共锚点组已有的账户可见范围限制仍然生效。

## CSV 地址簿导入与导出

普通成员管理自己的地址簿：

- `GET /api/admin/my/address_book/export`
- `POST /api/admin/my/address_book/import`

系统管理员管理自己或指定账户的地址簿：

- `GET /api/admin/address_book/export?user_id=5`
- `POST /api/admin/address_book/import?user_id=5`

使用后台登录会话的 `api-token` 请求头。导出返回带 UTF-8 标记的 CSV 文件。导入请求正文是 UTF-8 CSV，Content-Type 为 `text/csv`。

CSV 标题支持以下字段：

```csv
id,alias,hostname,username,platform,collection_id,tags,note,force_always_relay
desk-123456,办公室电脑,office-pc,staff,Windows,0,work;remote,远程协助,true
```

`id` 必填，`collection_id` 默认为 0，标签使用分号分隔。CSV 不能超过 4 MB，最多 5000 条。已有的相同设备 ID 和地址簿编号会更新常用信息，同时保留连接密码、认证信息及其他客户端设置。

导入成功返回 `data.created` 和 `data.updated`。如果有格式错误或地址簿不属于指定账户，全部导入操作取消。

## 地址簿回收站

普通账户：

- `GET /api/admin/my/address_book/trash/list`
- `POST /api/admin/my/address_book/trash/restore`
- `POST /api/admin/my/address_book/trash/purge`

系统管理员也可使用 `/api/admin/address_book/trash/...`，通过查询参数或请求正文的 `user_id` 指定账户。

恢复或永久删除时提交：

```json
{"row_id":12}
```

删除单个条目后，它会出现在回收站。恢复后原有密码及设备设置仍被保留。已有同名条目时会拒绝恢复。删除整个地址簿分类会永久删除该分类的所有条目，包括此前放入回收站的记录。

## 邮件服务

以下管理接口需要管理员登录：

- `GET /api/admin/mail/settings`：查看 SMTP 配置，返回 `password_configured` 表明是否设置密码。
- `POST /api/admin/mail/settings`：保存邮件服务设置。
- `POST /api/admin/mail/test`：向指定邮箱发送测试邮件。

示例：

```json
{
  "enabled": true,
  "host": "smtp.example.com",
  "port": 587,
  "security": "starttls",
  "username": "helpdesk@example.com",
  "password": "通过管理后台输入 SMTP 密码",
  "sender_email": "helpdesk@example.com",
  "sender_name": "RustDesk 团队",
  "base_url": "https://remote.example.com"
}
```

`security` 支持 `starttls`、`tls`、`none`。修改配置时 `password` 留空可保留已有密码，`clear_password: true` 用于清除已保存密码。密码原文不会在查询接口中返回。

发送测试邮件的请求正文为 `{"to":"member@example.com"}`。

## 密码恢复

无需登录：

- `POST /api/admin/password/forgot`，正文 `{"email":"member@example.com"}`
- `POST /api/admin/password/reset`，正文 `{"token":"邮件链接中的凭据","password":"新密码"}`

恢复链接有效期为 15 分钟，每个链接使用一次。完成后会撤销该账户现有登录会话。邮箱不存在时，申请接口仍返回同一提示，避免暴露账户信息。

## 成员邀请

管理员接口：

- `GET /api/admin/invite/list`：查看邀请列表。
- `POST /api/admin/invite/create`：发送邀请邮件。
- `POST /api/admin/invite/revoke`：撤销邀请。

邀请示例：

```json
{"email":"new.member@example.com","group_id":2,"expires_in_days":7}
```

有效天数范围为 1 至 30，默认 7。收件人打开 `/_admin/#/invite?token=...` 创建账户。无需登录的接受接口是 `POST /api/admin/invite/accept`：

```json
{
  "token": "邀请链接中的凭据",
  "username": "new.member",
  "nickname": "新成员",
  "password": "新账号密码"
}
```

邀请仅能使用一次，注册的账户被加入邀请时指定的用户组。

## 历史远程连接备注

管理员可以在“全部连接记录”中选择某条连接，编辑备注。接口为：

```http
POST /api/admin/audit_conn/note
```

请求正文：

```json
{"id":123,"note":"帮助成员安装办公软件，连接完成"}
```

备注最多 2000 个字符，提交空字符串可清除备注。仅修改备注与更新人编号，保留原始连接时间、认证信息及会话编号。

查询 `GET /api/admin/audit_conn/list` 返回 `note`、`note_updated_by`；`GET /api/admin/audit/export?kind=audit_conn` 下载的 CSV 也包含这两个字段。

## TOTP 双重认证

双重认证默认关闭。用户登录管理控制台后，可以通过以下接口管理自己的身份验证器：

- `GET /api/admin/mfa/status`：启用状态、待完成绑定以及剩余恢复码数量。
- `POST /api/admin/mfa/setup`：生成身份验证器密钥和 `otpauth://` 配置链接，绑定请求有效期 10 分钟。
- `POST /api/admin/mfa/enable`：验证当前六位验证码并启用双重认证，生成 6 个一次性恢复码。
- `POST /api/admin/mfa/recovery/regenerate`：提交有效验证码或恢复码，重新生成 6 个恢复码。
- `POST /api/admin/mfa/disable`：提交有效验证码或恢复码，解除双重认证。

启用、解除及重新生成恢复码的请求正文：

```json
{"code":"123456"}
```

启用成功后响应 `data.recovery_codes` 列出 6 个恢复码，仅显示一次。后续 `/mfa/status` 只返回恢复码数量。

官方客户端 `POST /api/login` 在已启用 TOTP 的用户输入正确账号与密码后，先返回：

```json
{"type":"email_check","tfa_type":"tfa_check","secret":"一次性登录凭据","user":{"name":"team-member"}}
```

然后客户端提交 `type=email_code` 或 `tfa_code`、`secret`、`tfaCode`、`username` 完成验证，成功后返回普通 `access_token` 登录结果。

网页管理控制台 `POST /api/admin/login` 会返回 `data.type=tfa_check`、`secret` 和 `data.user`，然后以 `type=tfa_code` 及验证码再次提交。单次挑战有效期为 5 分钟，每次最多验证 5 次。

## 邮箱验证码登录

用户可以在 Web 管理控制台“个人设置”中启用或关闭邮箱验证码登录。该功能默认关闭，管理员需要先配置 SMTP 邮件服务，用户需要具有正确的账户邮箱。

已登录用户的接口：

- `GET /api/admin/mfa/email/status`：查看邮箱地址、验证码登录启用状态，以及 SMTP 是否已配置。
- `POST /api/admin/mfa/email/setup`：向账户邮箱发送 6 位绑定验证码，返回一次性 `secret`。
- `POST /api/admin/mfa/email/enable`：提交 `secret` 和收到的 6 位验证码，启用邮箱登录验证。
- `POST /api/admin/mfa/email/disable`：关闭当前账户的邮箱验证码登录。

验证绑定时的请求格式：

```json
{"secret":"绑定请求返回的凭据","code":"123456"}
```

绑定成功之后，客户端密码登录第一步返回：

```json
{
  "type":"email_check",
  "tfa_type":"email_check",
  "secret":"当前登录请求凭据",
  "user":{"name":"team-member","email":"team@example.com"}
}
```

客户端收到邮件后，向 `POST /api/login` 提交：

```json
{
  "type":"email_code",
  "username":"team-member",
  "secret":"当前登录请求凭据",
  "verificationCode":"123456"
}
```

成功后才会签发普通 `access_token`。网页 `/api/admin/login` 采用相同的双阶段流程。每个验证码有效期 10 分钟，尝试次数和申请频率受限制。TOTP 已启用的账户优先使用 TOTP。

## 多中继节点配置

系统管理员在“中继节点管理”中设置节点，提供以下接口：

- `GET /api/admin/relay/list`：查看中继节点。
- `POST /api/admin/relay/create`：增加节点。
- `POST /api/admin/relay/update`：修改节点，正文包含 `id`。
- `POST /api/admin/relay/delete`：删除节点，正文例如 `{"id":7}`。

创建示例：

```json
{
  "name":"办公区域 A",
  "address":"relay-a.example.com:21117",
  "subnets":"203.0.113.0/24\n2001:db8:1::/48",
  "countries":"JP,US",
  "priority":10,
  "enabled":true
}
```

`address` 为公开的中继地址，支持主机名、IP 地址或主机名与端口组合。IPv6 地址和端口组合使用方括号形式，例如 `[2001:db8::1]:21117`。

`subnets` 为精确 IP 网络范围，可用换行或逗号分隔；留空表示不使用此条件。`countries` 为两位 ISO 3166-1 国家代码，可用逗号或换行分隔，输入时不区分大小写。例如 `JP,US,CA`。每项最多 64 个，系统自动删除重复代码。`priority` 允许 0 至 10000，数值较小的节点优先。GeoLite2 Country CSV 提供国家识别后，hbbs 可根据客户端 IP 地址自动匹配。

受信任的 `hbbs` 使用内部接口：

```http
GET /internal/v1/relay/nodes
X-RustDesk-Internal-Key: <内部通信密钥>
```

返回：

```json
{"nodes":[{"address":"relay-a.example.com:21117","priority":10,"subnets":["203.0.113.0/24","2001:db8:1::/48"],"countries":["JP","US"]}]}
```

该接口只返回已启用节点，供 `hbbs` 定期同步，不提供公共访问。

GeoLite2 数据在 hbbs 的本地文件目录中读取，API 只向 hbbs 传输中继节点偏好，内部接口不会上传 GeoLite2 数据文件。

## 远程控制角色

控制角色用于集中管理远程会话的 14 项权限。系统管理员可以使用以下接口：

- `GET /api/admin/control_role/list`、`POST /api/admin/control_role/create`、`POST /api/admin/control_role/update`、`POST /api/admin/control_role/delete`：角色管理。
- `GET /api/admin/control_assignment/list`、`POST /api/admin/control_assignment/create`、`POST /api/admin/control_assignment/update`、`POST /api/admin/control_assignment/delete`：角色分配管理。
- `GET /api/admin/control_role/effective?device_id=desk-123456`：查看设备实际生效角色及权限位编码。

创建角色示例：

```json
{
  "name":"客服只读协助",
  "keyboard":1,
  "remote_printer":0,
  "clipboard":2,
  "file":1,
  "audio":0,
  "camera":0,
  "terminal":1,
  "tunnel":1,
  "restart":0,
  "recording":0,
  "block_input":0,
  "remote_modify":1,
  "privacy_mode":0,
  "usb":1
}
```

每项权限采用相同状态：`0` 为沿用设备设置，`1` 为禁止，`2` 为允许。官方客户端的 `ControlPermissions` 定义了上述字段顺序，每项状态占两个二进制位。

为设备指定角色：

```json
{"target_type":"device","target_id":"desk-123456","role_id":3}
```

也可以把 `target_type` 设置为 `user` 或 `device_group`，此时 `target_id` 使用用户编号或设备组编号。分配的优先顺序是设备、设备所有者、设备组。

hbbs 使用内部接口读取有效角色：

```http
GET /internal/v1/control/permissions?device_id=desk-123456
X-RustDesk-Internal-Key: <内部通信密钥>
```

响应例如：

```json
{"assigned":true,"permissions":33817409,"source":"device"}
```

具体权限数字以角色配置计算的实际值为准。没有分配角色时，返回 `assigned:false`，hbbs 保留客户端原有的权限处理方式。

## 管理员分级角色

系统管理员可以使用管理角色，把指定的后台功能交给其他团队成员维护。被委派的成员可以保持 `is_admin=false`。

管理角色接口：

- `GET /api/admin/admin_role/catalog`：获取服务器支持的管理权限清单。
- `GET /api/admin/admin_role/list`：列出角色。
- `POST /api/admin/admin_role/create`：创建角色。
- `POST /api/admin/admin_role/update`：修改角色。
- `POST /api/admin/admin_role/delete`：删除角色，正文 `{"id":3}`。

角色示例：

```json
{
  "name":"客服设备查看",
  "permissions":"users.view,devices.view",
  "scope_type":"group",
  "scope_id":2,
  "enabled":true
}
```

`scope_type` 为 `global`、`group` 或 `individual`。全局角色的 `scope_id` 使用 0；用户组角色填写用户组编号，指定账户角色填写用户编号。

限定用户组或指定账户的角色支持 `users.view`、`users.edit`、`users.delete`、`devices.view`、`devices.edit_info`、`devices.delete` 和 `devices.transfer`。列表与详情仅显示允许查看的账户及设备。修改账户时不得变更其所属用户组或管理员身份，删除账户时不得删除系统管理员；编辑和删除设备时检查当前所有者是否属于授权范围。批量删除设备会先检查所有目标的归属，有任何无权限的设备则不执行删除。

转移设备所有者时，需要同时有权管理原所有者和目标所有者，客户端登记记录中的所有者也会同步修改。控制台“设备管理”可直接执行转移；还可调用：

```http
POST /api/admin/peer/transfer
api-token: <后台登录令牌>
Content-Type: application/json

{"row_id":18,"user_id":12}
```

单台设备的 `row_id` 是设备数据库记录编号，`user_id` 是目标用户编号。适用于全局管理员和具备 `devices.transfer` 权限的指定范围管理员。其他修改、删除接口保持原有数据格式，具体的目标归属由服务器检查。

角色分配接口：

- `GET /api/admin/admin_role_assignment/list`
- `POST /api/admin/admin_role_assignment/create`
- `POST /api/admin/admin_role_assignment/update`
- `POST /api/admin/admin_role_assignment/delete`

创建分配：

```json
{"user_id":12,"role_id":3}
```

成员可以拥有多个角色，服务器会合并这些角色允许的管理操作。角色被禁用或删除后，对应操作权限不再有效。后台登录或查询 `GET /api/admin/user/current` 时，`route_names` 会列出普通成员已获授权的管理权限名称。系统管理员仍以 `["*"]` 表示全部管理权限。

支持的权限名称由 `/admin_role/catalog` 返回，涵盖用户、设备、分组、设备策略、远程控制角色、审计、SMTP、中继节点、API 令牌和活动连接等类别。创建角色时服务端验证所有权限名称。

如果成员丢失身份验证器或无法访问原邮箱，具备全局 `users.manage_2fa` 权限的管理员可以调用：

```http
POST /api/admin/user/mfa/reset
```

请求正文为 `{"id":12}`。此操作清除目标账户的 TOTP、恢复码和邮箱登录验证设置，同时撤销其原有登录会话。成员随后可通过仍可使用的登录方式重新登录并设置双重认证。

## 服务器连接访问规则

管理员可以在“连接访问规则”页面建立访问许可。接口需要相应的管理权限：

- `GET /api/admin/access_rule/list`：列出全部规则。
- `POST /api/admin/access_rule/create`：创建规则。
- `POST /api/admin/access_rule/update`：修改规则，附加 `id`。
- `POST /api/admin/access_rule/delete`：删除规则，提交 `{"id":5}`。

示例：允许指定用户组中的成员访问一个设备组：

```json
{
  "name":"技术支持访问办公室设备",
  "source_type":"group",
  "source_id":2,
  "target_type":"device_group",
  "target_id":"3",
  "effect":"allow",
  "enabled":true
}
```

来源 `source_type` 支持 `all`、`user`、`group`。选择 `all` 时 `source_id=0`；选择用户或群组时填写对应编号。

目标 `target_type` 支持 `all`、`device`、`user`、`device_group`。选择单台设备时在 `target_id` 写设备 ID，选择设备所有者或设备组时填写其编号，选择所有设备时 `target_id` 留空。

`effect` 可以是 `allow`（允许）或 `deny`（禁止）。若同一连接符合多条规则，禁止规则优先。设备所有者始终可以连接自己已登记的设备。没有匹配允许规则时，启用严格模式的 hbbs 会拒绝连接。

hbbs 使用受内部密钥保护的接口判断每次新连接：

```http
POST /internal/v1/connection/authorize
X-RustDesk-Internal-Key: <内部通信密钥>
Content-Type: application/json

{"device_id":"desk-123456","token":"客户端当前账户登录令牌"}
```

响应例如：

```json
{"allow":true,"reason":"matching_allow","rule_id":5}
```

普通客户端不需要直接调用此内部接口。必须通过 hbbs 配置 `RUSTDESK_API_CONNECTION_ACCESS_URL` 才会启用服务器连接访问检查。

## 批量设备部署许可

管理员可使用 `POST /api/admin/device_deployment/batchStatus` 修改最多 200 条设备部署记录：

```json
{"ids":[4,7,11],"status":"revoked"}
```

`status` 支持 `approved`（批准）和 `revoked`（撤销）。服务器在同一数据库事务中检查全部记录，任一编号不存在时不修改任何记录。持有 `deployments.manage` 权限的独立 API 令牌可调用 `POST /api/automation/v1/deployments/batch-status`，请求内容相同。

## 批量恢复及永久删除地址簿记录

在“地址簿回收站”页面可执行整批操作，对应接口为：

- `POST /api/admin/my/address_book/trash/batchRestore`
- `POST /api/admin/my/address_book/trash/batchPurge`
- `POST /api/admin/address_book/trash/batchRestore` 与 `batchPurge`（管理员）

请求示例：

```json
{"row_ids":[14,17,23]}
```

一次可处理最多 200 条记录。个人接口只处理当前账户的回收站。管理员接口可通过 `user_id` 指定所有者。恢复前检查原分类、设备 ID 是否冲突以及记录归属，任何不满足条件的记录都会取消整批恢复。批量永久删除会检查所有目标记录已处于回收站，删除操作不可恢复。


## 邮件事件通知

管理员可以在“邮件事件通知”页面选择需要发送的事件及通知收件邮箱。数据库迁移版本为 281，初始状态不启用通知。建议先在“SMTP 邮件服务”页面配置发送服务器并发送测试邮件。

- `GET /api/admin/notification/settings`：查看通知状态及支持的事件。
- `POST /api/admin/notification/settings`：保存配置。
- `GET /api/admin/notification/list?page=1&page_size=20`：查看邮件发送记录与失败原因。
- `POST /api/admin/notification/retry`：重新提交已失败的发送记录，请求正文为 `{"id":8}`。

示例：

```json
{
  "enabled": true,
  "recipients": "admin@example.com,support@example.com",
  "events": "login.failure,deployment.changed,session.disconnect_requested,session.ended,audit.file"
}
```

一次最多配置 5 个收件邮箱。登录异常会在同一来源、账户与 15 分钟时间范围内合并通知，降低重复邮件数量。发送任务与业务操作分开执行，发送失败会保留状态及失败原因；系统最多自动尝试三次，管理员可以重新提交失败的邮件。通知邮件不包含登录密码、设备密钥或 API 令牌。

## 设备清单及审计 CSV 筛选

具有相应管理权限的用户可通过 `GET /api/admin/peer/export` 下载设备清单。参数支持 `user_id`（所有者编号）与 `search`（设备 ID、主机名或别名）。服务器会进一步限制受委派管理员可读取的用户范围。具有 `devices.read` 权限的独立 API 令牌也可以访问 `GET /api/automation/v1/devices/export`，令牌所属账户必须是启用的系统管理员。

`GET /api/admin/audit/export` 在原有 `kind` 与 `from_id` 参数之外，还支持 `from_date`、`to_date`（日期格式 `YYYY-MM-DD`），以及适用类别中的 `peer_id`、`from_peer`、`from_user_id`、`action`。日期筛选使用 UTC 日期，数据下载最多 10000 条。对 CSV 的文本字段进行电子表格安全处理。


## 官方客户端安装文件制作

管理员可以创建 RustDesk 1.5.1 的 Windows、Linux、macOS x64/ARM64 客户端配置，下载经过 SHA-256 验证的官方安装程序，并查看任务历史。Windows 使用包含配置的安装文件名；Linux/macOS 提供原始 DEB/DMG 与单独的配置脚本，打包成 ZIP。管理接口统一使用 `/api/admin/client_builder` 前缀：

- `GET /profiles`、`POST /profiles`、`POST /profiles/delete`：查看、创建、更新和删除服务器配置。
- `GET /tasks`、`POST /tasks`：查看任务或根据 `{"profile_id":12}` 创建任务。
- `POST /tasks/retry`：使用 `{"id":8}` 重新提交失败任务。
- `GET /tasks/:id/download`：下载已经准备好的安装程序。

服务器会保留官方 EXE、DEB 或 DMG 文件内容；Windows 通过文件名传递服务器配置，Linux 和 macOS 通过单独的脚本保存服务器配置。下载前再次验证最终文件的 SHA-256。配置需要系统管理员权限。详细的填写方法、状态说明和普通服务器及 Docker 数据保存方式见 [客户端构建器说明](CLIENT_BUILDER.md)。

## 系统设置中心

系统管理员通过 `/api/admin/settings` 管理应用全局设置。每个配置项目拥有独立的网页覆盖开关，启用后其值优先于环境变量。当前版本的全局设置修改后，需要重新启动 API。

| 请求 | 用途 |
| --- | --- |
| `GET /schema` | 查看配置名称、用途、类型、环境变量和生效方式 |
| `GET /effective` | 查看当前运行值、来源、覆盖开关和待重启状态；敏感值只显示是否已配置 |
| `POST /validate` | 检查修改内容，不保存 |
| `PATCH /overrides` | 保存一组覆盖值 |
| `POST /overrides/reset` | 关闭指定项目的网页覆盖 |
| `GET /history` | 查看设置修改记录 |
| `GET /overview` | 查看专项设置与通知处理数量 |
| `GET /outbox` | 查看失败通知事件 |
| `POST /outbox/retry` | 重新提交指定失败事件 |

保存接口需要提交 `bootstrap_version`、`database_version` 和 `changes`。每个 `changes` 项包含 `key`、`enabled`，启用新覆盖时填写字符串形式的 `value`。每次最多修改 50 项；同一请求仅能修改一种保存位置的设置。旧配置版本会收到冲突提示，需要重新读取后保存。

完整请求示例、设置来源和权限说明见 [设置中心使用说明](SETTINGS_CENTER.md)。


## 事件通知接口：SSE 与 HTTP JSON

事件通知可以通过 `GET /api/events/v1/stream` 持续接收。普通用户在控制台的 **账户 → 消息推送 → SSE 实时订阅** 中生成带 `events.read` 权限的个人 API Key，并在 `Authorization: Bearer <API_KEY>` 请求头中发送。可通过 `events` 参数筛选事件，例如 `?events=connection.started,connection.ended`。

用户也可以添加 `kind: "webhook"` 推送渠道，服务器将事件转换为 JSON，使用 `POST` 发送到用户配置的 HTTPS 地址；可选 Bearer 密钥，并保留失败重试和发送记录。`connection.started` 表示远程连接开始建立；`connection.authenticated` 表示该连接已经通过身份认证。

完整的事件代码、JSON 字段、密钥管理 API、curl 示例、发送重试、安全设置和多实例部署说明，请阅读 [消息推送与实时事件接口](PUSH_NOTIFICATIONS.md)。
