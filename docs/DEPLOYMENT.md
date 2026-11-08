# 部署与配置

## 1. 运行 API

API 默认监听 `21114`。在 `conf/config.yaml` 中修改 `gin.api-addr`，可以指定其他监听地址。SQLite 默认保存在 `data/rustdeskapi.db`，请将整个 `data` 目录持久保存。

执行：

```bash
go mod download
go build -o apimain ./cmd/apimain.go
mkdir -p data runtime
./apimain
```

首次运行后，打开 `http://服务器地址:21114/_admin/`，使用 `admin` 和启动日志中的初始密码登录。建议在反向代理服务器上配置 HTTPS。

## 2. 配置 RustDesk 服务器

在 `conf/config.yaml` 中填写：

```yaml
rustdesk:
  id-server: "desk.example.com:21116"
  relay-server: "desk.example.com:21117"
  api-server: "https://api.example.com"
  key: "这里填写服务器公钥"
  personal: 1
jwt:
  key: "请填写至少32位随机字符串"
```

也可使用环境变量 `RUSTDESK_API_RUSTDESK_ID_SERVER`、`RUSTDESK_API_RUSTDESK_RELAY_SERVER`、`RUSTDESK_API_RUSTDESK_API_SERVER` 和 `RUSTDESK_API_JWT_KEY`。

使用配套 RustDesk Server 时，将相同的 `RUSTDESK_API_JWT_KEY` 配置给服务端，使服务器能够验证用户登录令牌。可通过 `MUST_LOGIN=Y` 要求设备连接前登录。默认 `MUST_LOGIN=N`。

## 3. Docker

使用仓库自带的 Dockerfile：

```bash
docker build -t rustdesk-api .
docker run --name rustdesk-api -d \
  -p 21114:21114 \
  -e RUSTDESK_API_RUSTDESK_API_SERVER=https://api.example.com \
  -v "$PWD/data:/app/data" \
  --restart unless-stopped rustdesk-api
```

也可从 GHCR 获取发行镜像。管理员密码和数据库保存在挂载的数据目录中。升级容器时继续挂载同一个目录。

## 4. 群组和联系人

管理员在“用户分组”页面创建普通组、共享组或公共锚点组。创建公共锚点组后，打开“公共成员”选择需要对组员公开的账户。用户可通过主要分组或附加分组加入公共锚点组。

“批量管理成员”支持添加、移除及替换，成员变更后会影响下一次客户端请求。属于公共锚点组的管理员在 RustDesk 客户端也只看到自己和选中的公共成员及其设备。后台的管理员设备管理页面仍可查看全部设备。

任何用户都可以在“联系人”页面填写其他用户的用户名。添加后，自己的设备会展示在对方的“共享设备”页面。对方不需要接受邀请；撤销联系人后，相关设备从共享列表消失。通过其他方式得到设备 ID 的用户仍可按照原有 RustDesk 连接规则发起连接。

## 5. OIDC / SSO

管理员在“OIDC / SSO”页面设置身份认证服务的 Client ID、Client Secret、Issuer、Scopes，并在“自动加入的分组 ID”中填写用逗号分隔的群组编号，例如 `2,3`。

每次通过该身份认证服务成功登录时，系统会检查并添加相应成员关系，且不会重复添加。一个用户可以属于多个分组。管理员修改自动加入的群组后，新规则会在下次登录时应用。

## 6. 地址簿

控制台“地址簿”页面有“设备条目”和“地址簿分类”两个视图。每个分类可以包含多个设备条目。已有数据库中的分类及条目会继续保留，RustDesk 客户端原有的地址簿接口也可继续使用。

## 7. 升级与备份

升级前先备份数据库。SQLite 使用 `data/rustdeskapi.db`；MySQL 和 PostgreSQL 请使用相应数据库管理工具。更新代码或镜像后重新启动 API。程序启动时自动检查数据库结构并添加新字段和表。
