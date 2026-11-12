# 个人账户与 Passkey 登录

登录 `/_admin/`，在“账户 → 个人设置”中，可以查看账户名称、昵称和邮箱，并管理密码、双重认证、邮箱登录验证以及 Passkey。桌面端按窗口宽度排列多列卡片，手机端自动使用单列。

## Passkey 注册与登录

Passkey 使用 WebAuthn 标准，通过浏览器和操作系统提供的指纹、面容或屏幕锁完成身份验证。

首次添加时，先使用现有密码或单点登录进入个人设置，在“Passkey 通行密钥”中填写设备名称，点击“添加 Passkey”，然后按照浏览器提示完成验证。注册成功后，在登录页面点击“使用 Passkey 登录”即可使用已经保存的凭证。

每个账户最多注册 25 个 Passkey。个人设置会显示名称、创建时间和最近使用时间，并允许用户移除自己不再使用的凭证。移除后，该凭证无法再用于登录。

Passkey 需要通过 HTTPS 使用。本机开发时允许通过 `localhost` 或 `127.0.0.1` 的 HTTP 访问。Passkey 与注册时的域名绑定，因此更换域名后需要重新注册。通过反向代理部署时，应将浏览器访问的 Host 保留并传递给 API。

注册和登录请求只在短时间内有效，每次只能使用一次。服务端校验网站来源、认证器签名和用户验证状态。系统只保存公开的认证器数据，不接收指纹或面容数据。

## 管理员帮助用户管理账户

管理员可在“用户账户”中选择目标用户，打开“编辑用户信息”，根据自己的管理权限执行下列操作：

- **编辑用户资料**：修改昵称、邮箱、备注及其他允许修改的资料字段。
- **重设密码**：为目标账户设置 8 至 72 个字符的新密码；保存后结束目标账户现有的登录会话。
- **清除双重认证**：清除用户已登记的 TOTP 身份验证器、恢复码和邮箱登录验证配置，同时结束其现有会话。用户可以在重新登录后自行启用双重认证。

管理员操作会检查权限，并在需要时显示确认信息。普通用户只能管理本人的 Passkey 和个人设置。

## HTTP 接口

| 方法 | 接口 | 用途 |
| --- | --- | --- |
| GET | `/api/admin/passkey/list` | 列出当前用户的 Passkey |
| POST | `/api/admin/passkey/register/begin` | 开始注册 |
| POST | `/api/admin/passkey/register/finish` | 验证并完成注册 |
| POST | `/api/admin/passkey/revoke` | 移除本人 Passkey |
| POST | `/api/admin/passkey/login/begin` | 获取登录认证请求 |
| POST | `/api/admin/passkey/login/finish` | 验证 Passkey 并登录 |
| POST | `/api/admin/user/update` | 管理员修改用户资料 |
| POST | `/api/admin/user/changePwd` | 管理员重设用户密码 |
| POST | `/api/admin/user/mfa/reset` | 管理员清除用户双重认证 |

Passkey 的 `begin` 接口返回 `options` 和 `session_id`。浏览器通过 `navigator.credentials.create()` 或 `navigator.credentials.get()` 与认证器交互，再将 `credential` 和 `session_id` 提交到对应的 `finish` 接口。注册完成接口还接受 `name`。

## 升级和检查

数据库迁移版本更新为 291，并新增 Passkey 凭证、账户句柄与短期认证请求的存储表。原有账户、密码和会话功能继续使用。升级前请备份数据库及配置。

自动化测试包含认证请求有效期限、重复使用检查、账户隔离、Passkey 注册与签名登录、凭证撤销，以及管理员重设密码和清除双重认证后的旧会话失效检查。页面布局将通过管理员及普通用户在不同屏幕尺寸下的逐页截图检查。
