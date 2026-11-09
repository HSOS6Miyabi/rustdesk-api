# API 使用说明

HTTP 响应中的 `code: 0` 表示操作成功。管理员接口统一以 `/api/admin` 开头。登录后，在请求头中设置 `api-token`，值为登录返回的 `data.token`。

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
  "trust_duration_days": 30
}
```

为已登记的设备分配策略：

```json
{"device_id":"123456789","policy_id":1}
```

`trust_duration_days` 可填写 `0`、`1`、`7`、`30` 或 `90`，其中 `0` 表示不限期限。

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

`user_id` 可选，默认使用当前管理员账户。允许的操作是 `devices.assign` 和 `devices.deploy`，可只指定其中一项。有效天数为 `0` 时长期有效；有效天数最大为 3650。接口返回令牌原文后，数据库只保存摘要。撤销、过期或所属账户停用后，令牌立即失效。

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
