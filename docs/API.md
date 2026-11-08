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
