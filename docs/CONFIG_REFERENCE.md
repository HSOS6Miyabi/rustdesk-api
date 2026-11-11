# 全局配置参考

本文列出 RustDesk API 设置中心登记的全部全局配置。程序启动时读取 YAML、环境变量和已启用的网页覆盖值。以下“模板值”来自仓库的 `conf/config.yaml`，仅作为部署参考。实际运行值、来源和待重启状态请在管理控制台“系统管理 → 设置中心”查看。

设置中心列出的项目涵盖 API 自身配置；SMTP、个人通知、设备策略、管理角色、会话访问规则和中继节点已有独立页面。设置中心提供这些页面的入口，相关数据仍按各自的业务接口保存。

## 配置读取顺序

每个配置项目分别按以下顺序计算实际值：

1. 已启用的网页覆盖设置。
2. 环境变量。兼容变量 `REQUIRE_DEVICE_DEPLOYMENT` 与 `RUSTDESK_API_INTERNAL_KEY` 继续保留。
3. `conf/config.yaml`。
4. 代码默认值。

网页明确设置的 `false`、`0` 和空字符串都具有覆盖效力。关闭网页覆盖后，下次重新启动时会恢复使用环境变量或配置文件的值。当前版本所有全局设置修改均需重新启动 API，管理界面会分别显示进程当前运行值和未生效的修改。

## 全局配置项目

| 配置标识 | 名称和作用 | 配置文件模板值 | 环境变量 | 网页设置保存位置 |
| --- | --- | --- | --- | --- |
| **基础设置** | | | | |
| `lang` | 语言。控制台与服务接口使用的语言。 | `zh-CN` | `RUSTDESK_API_LANG` | 数据库 |
| **账户与安全** | | | | |
| `app.web-client` | 网页客户端。控制网页客户端入口是否启用。 | `1` | `RUSTDESK_API_APP_WEB_CLIENT` | 数据库 |
| `app.register` | 开放注册。允许新用户自行申请账户。 | `false` | `RUSTDESK_API_APP_REGISTER` | 数据库 |
| `app.register-status` | 新账户状态。选择新注册账户的初始状态：1 启用、2 禁用。 | `1` | `RUSTDESK_API_APP_REGISTER_STATUS` | 数据库 |
| `app.show-swagger` | 接口文档。是否公开管理接口的 Swagger 页面。 | `0` | `RUSTDESK_API_APP_SHOW_SWAGGER` | 数据库 |
| `app.token-expire` | 登录有效期。普通账户登录令牌的有效时间。 | `168h` | `RUSTDESK_API_APP_TOKEN_EXPIRE` | 数据库 |
| `app.web-sso` | 网页登录单点登录。控制网页中的单点登录入口。 | `true` | `RUSTDESK_API_APP_WEB_SSO` | 数据库 |
| `app.disable-pwd-login` | 停用密码登录。阻止账户使用密码登录，启用前应确认其他登录方式可用。 | `false` | `RUSTDESK_API_APP_DISABLE_PWD_LOGIN` | 数据库 |
| `app.captcha-threshold` | 验证码阈值。登录失败达到多少次后要求填写验证码，负数表示关闭。 | `3` | `RUSTDESK_API_APP_CAPTCHA_THRESHOLD` | 数据库 |
| `app.ban-threshold` | IP 封禁阈值。登录失败达到多少次后封禁 IP 地址，0 表示关闭。 | `0` | `RUSTDESK_API_APP_BAN_THRESHOLD` | 数据库 |
| `app.require-device-deployment` | 设备审批登记。启用后要求设备通过部署审批才能登记。 | `false` | `RUSTDESK_API_APP_REQUIRE_DEVICE_DEPLOYMENT` | 数据库 |
| `app.internal-key` | 内部通信密钥。API 与 hbbs 共用的内部接口认证密钥。 | 空（敏感值） | `RUSTDESK_API_APP_INTERNAL_KEY` | 本机启动文件 |
| **管理控制台** | | | | |
| `admin.title` | 控制台标题。设置管理控制台显示的标题。 | `RustDesk API Admin` | `RUSTDESK_API_ADMIN_TITLE` | 数据库 |
| `admin.hello` | 欢迎文本。设置账户登录后显示的欢迎说明。 | （空字符串） | `RUSTDESK_API_ADMIN_HELLO` | 数据库 |
| `admin.hello-file` | 欢迎文本文件。读取欢迎说明的文件路径。 | `./conf/admin/hello.html` | `RUSTDESK_API_ADMIN_HELLO_FILE` | 本机启动文件 |
| `admin.id-server-port` | ID 服务管理端口。发送 RustDesk ID 服务器管理命令的端口。 | `21116` | `RUSTDESK_API_ADMIN_ID_SERVER_PORT` | 数据库 |
| `admin.relay-server-port` | 中继服务管理端口。发送 RustDesk 中继管理命令的端口。 | `21117` | `RUSTDESK_API_ADMIN_RELAY_SERVER_PORT` | 数据库 |
| **HTTP** | | | | |
| `gin.api-addr` | HTTP 监听地址。API 监听的 IP 地址和端口。 | `0.0.0.0:21114` | `RUSTDESK_API_GIN_API_ADDR` | 本机启动文件 |
| `gin.admin-addr` | 管理端监听地址。保留配置项，当前版本未单独启用管理监听端口。 | 未在配置模板列出 | `RUSTDESK_API_GIN_ADMIN_ADDR` | 本机启动文件（当前未启用） |
| `gin.mode` | 服务运行模式。选择 release、debug 或 test。 | `release` | `RUSTDESK_API_GIN_MODE` | 本机启动文件 |
| `gin.resources-path` | 静态资源目录。控制台网页和静态文件所在的目录。 | `resources` | `RUSTDESK_API_GIN_RESOURCES_PATH` | 本机启动文件 |
| `gin.trust-proxy` | 可信代理。设置可信反向代理地址，影响真实客户端 IP 的识别。 | （空字符串） | `RUSTDESK_API_GIN_TRUST_PROXY` | 本机启动文件 |
| **数据库** | | | | |
| `gorm.type` | 数据库类型。选择 sqlite、mysql 或 postgresql。 | `sqlite` | `RUSTDESK_API_GORM_TYPE` | 本机启动文件 |
| `gorm.max-idle-conns` | 空闲数据库连接数。连接池允许保留的空闲连接数。 | `10` | `RUSTDESK_API_GORM_MAX_IDLE_CONNS` | 本机启动文件 |
| `gorm.max-open-conns` | 数据库连接数上限。连接池的最大连接数，SQLite 实际最多四个。 | `100` | `RUSTDESK_API_GORM_MAX_OPEN_CONNS` | 本机启动文件 |
| `mysql.addr` | MySQL 地址。MySQL 服务器地址和端口。 | （空字符串） | `RUSTDESK_API_MYSQL_ADDR` | 本机启动文件 |
| `mysql.username` | MySQL 用户名。连接 MySQL 时使用的账户。 | （空字符串） | `RUSTDESK_API_MYSQL_USERNAME` | 本机启动文件 |
| `mysql.password` | MySQL 密码。连接 MySQL 时使用的密码。 | 空（敏感值） | `RUSTDESK_API_MYSQL_PASSWORD` | 本机启动文件 |
| `mysql.dbname` | MySQL 数据库名称。保存 RustDesk 管理数据的 MySQL 数据库。 | （空字符串） | `RUSTDESK_API_MYSQL_DBNAME` | 本机启动文件 |
| `mysql.tls` | MySQL TLS 模式。设置 MySQL 连接的证书验证方式。 | `false` | `RUSTDESK_API_MYSQL_TLS` | 本机启动文件 |
| `postgresql.host` | PostgreSQL 地址。PostgreSQL 服务器主机名。 | `127.0.0.1` | `RUSTDESK_API_POSTGRESQL_HOST` | 本机启动文件 |
| `postgresql.port` | PostgreSQL 端口。PostgreSQL 服务监听端口。 | `5432` | `RUSTDESK_API_POSTGRESQL_PORT` | 本机启动文件 |
| `postgresql.user` | PostgreSQL 用户名。连接 PostgreSQL 时使用的账户。 | （空字符串） | `RUSTDESK_API_POSTGRESQL_USER` | 本机启动文件 |
| `postgresql.password` | PostgreSQL 密码。连接 PostgreSQL 时使用的密码。 | 空（敏感值） | `RUSTDESK_API_POSTGRESQL_PASSWORD` | 本机启动文件 |
| `postgresql.dbname` | PostgreSQL 数据库名称。保存管理数据的 PostgreSQL 数据库。 | `postgres` | `RUSTDESK_API_POSTGRESQL_DBNAME` | 本机启动文件 |
| `postgresql.sslmode` | PostgreSQL TLS 模式。指定 PostgreSQL TLS 验证方式。 | `disable` | `RUSTDESK_API_POSTGRESQL_SSLMODE` | 本机启动文件 |
| `postgresql.time-zone` | PostgreSQL 时区。数据库连接使用的会话时区。 | `Asia/Shanghai` | `RUSTDESK_API_POSTGRESQL_TIME_ZONE` | 本机启动文件 |
| **RustDesk 服务** | | | | |
| `rustdesk.id-server` | ID 服务器地址。客户端连接的 RustDesk ID 服务器地址。 | `192.168.1.66:21116` | `RUSTDESK_API_RUSTDESK_ID_SERVER` | 数据库 |
| `rustdesk.relay-server` | 中继服务器地址。客户端连接的 RustDesk 中继服务器地址。 | `192.168.1.66:21117` | `RUSTDESK_API_RUSTDESK_RELAY_SERVER` | 数据库 |
| `rustdesk.api-server` | 公开 API 地址。客户端使用的 RustDesk API URL。 | `http://127.0.0.1:21114` | `RUSTDESK_API_RUSTDESK_API_SERVER` | 数据库 |
| `rustdesk.key` | RustDesk 公钥。RustDesk 服务端的公开身份密钥。 | （空字符串） | `RUSTDESK_API_RUSTDESK_KEY` | 数据库 |
| `rustdesk.key-file` | RustDesk 公钥文件。从指定文件读取 RustDesk 服务端公钥。 | `/data/id_ed25519.pub` | `RUSTDESK_API_RUSTDESK_KEY_FILE` | 本机启动文件 |
| `rustdesk.personal` | 个人模式。原项目提供的个人模式控制选项。 | `1` | `RUSTDESK_API_RUSTDESK_PERSONAL` | 数据库 |
| `rustdesk.webclient-magic-queryonline` | 网页在线查询模式。网页客户端特定的在线查询行为。 | `0` | `RUSTDESK_API_RUSTDESK_WEBCLIENT_MAGIC_QUERYONLINE` | 数据库 |
| `rustdesk.ws-host` | WebSocket 公开地址。浏览器客户端的 WebSocket 代理地址。 | （空字符串） | `RUSTDESK_API_RUSTDESK_WS_HOST` | 数据库 |
| **日志** | | | | |
| `logger.path` | 日志文件。API 日志文件存放位置。 | `./runtime/log.txt` | `RUSTDESK_API_LOGGER_PATH` | 本机启动文件 |
| `logger.level` | 日志级别。控制调试、信息和错误日志的详细程度。 | `info` | `RUSTDESK_API_LOGGER_LEVEL` | 本机启动文件 |
| `logger.report-caller` | 记录调用位置。是否在日志中记录产生记录的源代码位置。 | `true` | `RUSTDESK_API_LOGGER_REPORT_CALLER` | 本机启动文件 |
| **网络代理** | | | | |
| `proxy.enable` | 启用网络代理。是否使用设置的 HTTP 网络代理。 | `false` | `RUSTDESK_API_PROXY_ENABLE` | 数据库 |
| `proxy.host` | 网络代理地址。代理服务器 URL。 | `http://127.0.0.1:1080` | `RUSTDESK_API_PROXY_HOST` | 数据库 |
| **令牌** | | | | |
| `jwt.key` | JWT 签名密钥。用于签名账户访问令牌的密钥，变更可能使已有会话失效。 | 空（敏感值） | `RUSTDESK_API_JWT_KEY` | 本机启动文件 |
| `jwt.expire-duration` | JWT 有效时间。JWT 签名令牌的有效时间。 | `168h` | `RUSTDESK_API_JWT_EXPIRE_DURATION` | 本机启动文件 |
| **LDAP** | | | | |
| `ldap.enable` | 启用 LDAP。允许使用 LDAP 目录进行账户身份验证。 | `false` | `RUSTDESK_API_LDAP_ENABLE` | 数据库 |
| `ldap.url` | LDAP 地址。LDAP 服务连接地址。 | `ldap://ldap.example.com:389` | `RUSTDESK_API_LDAP_URL` | 数据库 |
| `ldap.tls-ca-file` | LDAP CA 文件。LDAP TLS 服务器证书签发机构文件。 | （空字符串） | `RUSTDESK_API_LDAP_TLS_CA_FILE` | 数据库 |
| `ldap.tls-verify` | 验证 LDAP TLS。是否检查 LDAP 服务器的证书。 | `false` | `RUSTDESK_API_LDAP_TLS_VERIFY` | 数据库 |
| `ldap.base-dn` | LDAP 基础 DN。LDAP 用户目录的基础查询位置。 | `dc=example,dc=com` | `RUSTDESK_API_LDAP_BASE_DN` | 数据库 |
| `ldap.bind-dn` | LDAP 绑定 DN。API 查询 LDAP 使用的账户 DN。 | `cn=admin,dc=example,dc=com` | `RUSTDESK_API_LDAP_BIND_DN` | 数据库 |
| `ldap.bind-password` | LDAP 绑定密码。API 查询 LDAP 使用的账户密码。 | 已在模板填写示例值（不显示） | `RUSTDESK_API_LDAP_BIND_PASSWORD` | 本机启动文件 |
| `ldap.user.base-dn` | LDAP 用户目录。用户条目所在的目录位置。 | `ou=users,dc=example,dc=com` | `RUSTDESK_API_LDAP_USER_BASE_DN` | 数据库 |
| `ldap.user.enable-attr` | LDAP 启用属性。读取账户启用状态时检查的属性名。 | （空字符串） | `RUSTDESK_API_LDAP_USER_ENABLE_ATTR` | 数据库 |
| `ldap.user.enable-attr-value` | LDAP 属性值。代表账户启用状态的属性值。 | （空字符串） | `RUSTDESK_API_LDAP_USER_ENABLE_ATTR_VALUE` | 数据库 |
| `ldap.user.filter` | LDAP 查询表达式。LDAP 用户搜索筛选条件。 | `(cn=*)` | `RUSTDESK_API_LDAP_USER_FILTER` | 数据库 |
| `ldap.user.username` | 用户名属性。LDAP 用户名对应的字段。 | `uid` | `RUSTDESK_API_LDAP_USER_USERNAME` | 数据库 |
| `ldap.user.email` | 邮箱属性。LDAP 邮箱对应的字段。 | `mail` | `RUSTDESK_API_LDAP_USER_EMAIL` | 数据库 |
| `ldap.user.first-name` | 名字属性。LDAP 名字对应的字段。 | `givenName` | `RUSTDESK_API_LDAP_USER_FIRST_NAME` | 数据库 |
| `ldap.user.last-name` | 姓氏属性。LDAP 姓氏对应的字段。 | `sn` | `RUSTDESK_API_LDAP_USER_LAST_NAME` | 数据库 |
| `ldap.user.sync` | 同步账户资料。登录或创建用户时是否同步 LDAP 用户资料。 | `false` | `RUSTDESK_API_LDAP_USER_SYNC` | 数据库 |
| `ldap.user.admin-group` | LDAP 管理员组。标识 LDAP 管理员的用户组 DN。 | `cn=admin,dc=example,dc=com` | `RUSTDESK_API_LDAP_USER_ADMIN_GROUP` | 数据库 |
| `ldap.user.allow-group` | LDAP 允许登录组。允许使用 LDAP 登录的用户组 DN。 | `cn=users,dc=example,dc=com` | `RUSTDESK_API_LDAP_USER_ALLOW_GROUP` | 数据库 |
| **缓存** | | | | |
| `redis.addr` | Redis 地址。兼容现有组件使用的 Redis 服务地址。 | 未在配置模板列出 | `RUSTDESK_API_REDIS_ADDR` | 本机启动文件 |
| `redis.password` | Redis 密码。Redis 连接认证密码。 | 空（敏感值） | `RUSTDESK_API_REDIS_PASSWORD` | 本机启动文件 |
| `redis.db` | Redis 数据库编号。Redis 逻辑数据库编号。 | 未在配置模板列出 | `RUSTDESK_API_REDIS_DB` | 本机启动文件 |
| `cache.type` | 缓存方式。选择内存、文件或 Redis 缓存。 | 未在配置模板列出 | `RUSTDESK_API_CACHE_TYPE` | 本机启动文件 |
| `cache.redis-addr` | 缓存 Redis 地址。Redis 缓存使用的服务器地址。 | 未在配置模板列出 | `RUSTDESK_API_CACHE_REDIS_ADDR` | 本机启动文件 |
| `cache.redis-pwd` | 缓存 Redis 密码。Redis 缓存的连接密码。 | 空（敏感值） | `RUSTDESK_API_CACHE_REDIS_PWD` | 本机启动文件 |
| `cache.redis-db` | 缓存 Redis 编号。Redis 缓存使用的逻辑数据库。 | 未在配置模板列出 | `RUSTDESK_API_CACHE_REDIS_DB` | 本机启动文件 |
| `cache.file-dir` | 文件缓存目录。本地文件缓存的保存目录。 | 未在配置模板列出 | `RUSTDESK_API_CACHE_FILE_DIR` | 本机启动文件 |
| **文件上传** | | | | |
| `oss.access-key-id` | 对象存储访问标识。对象存储服务使用的访问标识。 | 未在配置模板列出 | `RUSTDESK_API_OSS_ACCESS_KEY_ID` | 本机启动文件 |
| `oss.access-key-secret` | 对象存储访问密钥。对象存储服务的私密访问凭据。 | 空（敏感值） | `RUSTDESK_API_OSS_ACCESS_KEY_SECRET` | 本机启动文件 |
| `oss.host` | 对象存储服务地址。对象存储服务访问地址。 | 未在配置模板列出 | `RUSTDESK_API_OSS_HOST` | 本机启动文件 |
| `oss.callback-url` | 对象存储回调地址。上传完成后返回的回调地址。 | 未在配置模板列出 | `RUSTDESK_API_OSS_CALLBACK_URL` | 本机启动文件 |
| `oss.expire-time` | 对象存储链接期限。上传访问链接允许的有效时间。 | 未在配置模板列出 | `RUSTDESK_API_OSS_EXPIRE_TIME` | 本机启动文件 |
| `oss.max-byte` | 上传文件大小上限。限制上传对象的字节数。 | 未在配置模板列出 | `RUSTDESK_API_OSS_MAX_BYTE` | 本机启动文件 |
| **运行附加服务** | | | | |
| `runtime.shared-event-workers` | 共享通知任务。多个 API 实例使用同一数据库时启用共享任务认领。 | 未在配置模板列出 | `RUSTDESK_API_SHARED_EVENT_WORKERS` | 本机启动文件 |
| `runtime.geoip-db` | GeoIP 城市数据库。本地 GeoLite2 City MMDB 数据库文件路径。 | 未在配置模板列出 | `RUSTDESK_API_GEOIP_DB` | 本机启动文件 |
| `runtime.push-geoip-db` | 兼容 GeoIP 文件路径。仅在主要 GeoIP 路径未填写时读取的旧环境变量。 | 未在配置模板列出 | `RUSTDESK_API_PUSH_GEOIP_DB` | 本机启动文件 |
| `runtime.push-encryption-key` | 推送加密密钥。用于加密个人消息推送渠道的凭据。 | 空（敏感值） | `RUSTDESK_API_PUSH_ENCRYPTION_KEY` | 本机启动文件 |
| `runtime.mail-encryption-key` | 邮件加密密钥。用于加密 SMTP、双重认证等持久保存的秘密。 | 空（敏感值） | `RUSTDESK_API_MAIL_ENCRYPTION_KEY` | 本机启动文件 |
| `runtime.push-allow-private-urls` | 允许内部推送地址。允许推送访问内部网络 HTTP 地址，启用会增加安全风险。 | 未在配置模板列出 | `RUSTDESK_API_PUSH_ALLOW_PRIVATE_URLS` | 本机启动文件 |

配置表中若某项没有列在 `conf/config.yaml`，请在管理控制台查看其实际来源。尚未启用的 `gin.admin-addr` 仅展示说明，当前无法通过它额外启动一个管理监听服务。

## 其他配置来源

- `RUSTDESK_API_GEOIP_DB` 优先于兼容变量 `RUSTDESK_API_PUSH_GEOIP_DB`。数据库文件须在 API 所在机器或容器内可读取。
- `RUSTDESK_API_SHARED_EVENT_WORKERS` 只适用于多个 API 实例共享数据库的部署方式。单实例使用事件唤醒，无需启用共享认领。
- `RUSTDESK_API_MAIL_ENCRYPTION_KEY` 和 `RUSTDESK_API_PUSH_ENCRYPTION_KEY` 影响历史密文的读取。更换之前应制定密钥迁移和恢复步骤。
- 数据库密码、JWT 密钥、LDAP 密码和 API 内部密钥不通过设置查询接口返回原文。
- PostgreSQL、MySQL 的数据库连接设置与缓存种类属于启动设置。修改它们前应检查目标服务的连接参数，备份旧数据库和持久化配置。

## 普通服务器与 Docker

普通服务器应将 `data/` 目录作为需要备份的数据目录。Docker 至少挂载 `/app/data`，保证 `/app/data/settings-bootstrap.json` 与 SQLite 数据库在容器重建后仍然存在。所有实例都要保留自己使用的启动文件；共享数据库中的应用设置则由多个实例共同读取。

有关管理页面操作、权限和重启流程，参见 [设置中心使用说明](SETTINGS_CENTER.md)。部署前请阅读 [数据库备份与恢复说明](DATABASE_BACKUP_RESTORE.md) 和 [部署说明](DEPLOYMENT.md)。

## 多实例缓存更新

设置 `RUSTDESK_API_MULTI_INSTANCE_CACHE=1` 或在设置中心启用 `runtime.multi-instance-cache`，重启每个 API 实例后，设备策略分配和通知订阅缓存的有效时间缩短为约 5 秒。单实例默认使用约 30 秒的缓存有效时间，且本实例修改相关配置后会主动清除缓存。多实例设置只在有访问请求或设备心跳时检查数据库，不增加空闲状态下的周期查询。身份验证和严格连接许可仍由原有接口执行。

多实例缓存刷新时间还会受到客户端心跳间隔影响。远程连接许可更改应通过配套 RustDesk Server 的连接授权接口处理。
