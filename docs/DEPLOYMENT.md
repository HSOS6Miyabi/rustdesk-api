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

## 3. 为 WebClient v2 配置 HTTPS 和 WebSocket

在浏览器中打开 `https://api.example.com/webclient2/` 后，可以使用设备 ID 连接 RustDesk 服务。网页登录后，页面会自动读取 ID 服务器、公钥和共享设备列表。

RustDesk 服务端使用 `21118` 端口提供 ID 服务器的 WebSocket，使用 `21119` 端口提供中继 WebSocket。浏览器通过 HTTPS 访问 API 时，WebSocket 也必须使用有效的 TLS 证书。可以在 Nginx 上设置两个 HTTPS WebSocket 代理入口：

```nginx
server {
    listen 443 ssl;
    server_name desk.example.com;
    ssl_certificate /etc/letsencrypt/live/desk.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/desk.example.com/privkey.pem;

    location = /ws/id {
        proxy_pass http://127.0.0.1:21118/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }

    location = /ws/relay {
        proxy_pass http://127.0.0.1:21119/;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_read_timeout 3600s;
    }
}
```

如果 RustDesk 服务端在另一台机器上运行，请将 `127.0.0.1` 替换为其内网地址，并确保 WebSocket 代理能够访问端口 `21118` 和 `21119`。证书路径应根据实际的域名与证书位置填写。

在 API 的 `conf/config.yaml` 中设置：

```yaml
rustdesk:
  id-server: "desk.example.com:21116"
  relay-server: "desk.example.com:21117"
  api-server: "https://api.example.com"
  ws-host: "wss://desk.example.com"
```

`ws-host` 用于生成 `wss://desk.example.com/ws/id` 和 `wss://desk.example.com/ws/relay` 两个连接地址。修改配置后重新启动 API。已有的设备 ID 与远程连接密码不受影响。

从源代码部署时，需要先构建网页连接组件：

```bash
cd resources/web/js
npm ci
npm run build
cd ../../..
```

GitHub Release 和 GHCR 镜像会自动包含已构建的文件。

## 4. Docker

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

## 5. 群组和联系人

管理员在“用户分组”页面创建普通组、共享组或公共锚点组。创建公共锚点组后，打开“公共成员”选择需要对组员公开的账户。用户可通过主要分组或附加分组加入公共锚点组。

“批量管理成员”支持添加、移除及替换，成员变更后会影响下一次客户端请求。属于公共锚点组的管理员在 RustDesk 客户端也只看到自己和选中的公共成员及其设备。后台的管理员设备管理页面仍可查看全部设备。

任何用户都可以在“联系人”页面填写其他用户的用户名。添加后，自己的设备会展示在对方的“共享设备”页面。对方不需要接受邀请；撤销联系人后，相关设备从共享列表消失。通过其他方式得到设备 ID 的用户仍可按照原有 RustDesk 连接规则发起连接。

## 6. 个人群组和审计记录

所有用户都可以在“我的群组”页面创建群组，管理群组名称、成员和公共锚点成员。个人群组的创建者会自动加入该群组，并保留原来的主要分组。群组所有者只能编辑自己创建的群组，管理员可在独立的“用户分组”页面管理所有群组。

添加成员时可以填写已知用户的完整用户名。为防止用户通过群组管理接口查看其他群组的成员，个人群组的查找功能仅按完整用户名查询。普通用户无法通过接口修改管理员的群组成员关系。

“我的连接记录”展示用户主动连接远程设备和其他人连接自己设备的日志。“我的文件传输”展示本人设备参与的文件传输。历史日志会利用设备 ID 查询现有归属信息；新日志保存连接双方的信息，在设备更换归属后仍能显示原有记录。

## 7. OIDC / SSO

管理员在“OIDC / SSO”页面设置身份认证服务的 Client ID、Client Secret、Issuer、Scopes，并在“自动加入的分组 ID”中填写用逗号分隔的群组编号，例如 `2,3`。

每次通过该身份认证服务成功登录时，系统会检查并添加相应成员关系，且不会重复添加。一个用户可以属于多个分组。管理员修改自动加入的群组后，新规则会在下次登录时应用。

## 8. 地址簿

控制台“地址簿”页面有“设备条目”和“地址簿分类”两个视图。每个分类可以包含多个设备条目。已有数据库中的分类及条目会继续保留，RustDesk 客户端原有的地址簿接口也可继续使用。

## 9. 升级与备份

升级前先备份数据库。SQLite 使用 `data/rustdeskapi.db`；MySQL 和 PostgreSQL 请使用相应数据库管理工具。更新代码或镜像后重新启动 API。程序启动时自动检查数据库结构并添加新字段和表。
