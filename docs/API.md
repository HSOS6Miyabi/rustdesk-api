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
