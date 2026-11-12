# 随软件提供的 IP 地理位置数据库

程序发行包和 Docker 镜像包含完整的 DB-IP City Lite 城市数据库，文件位置为 `resources/geoip/dbip-city-lite.mmdb`。启用后，控制台可直接显示公网 IP 对应的国家、省份及城市，无须上传 IP 地址，也无须配置外部查询服务。

本项目使用 DB-IP City Lite 2026 年 11 月数据，来源：https://db-ip.com/db/lite.php 。该数据库按照 **Creative Commons Attribution 4.0 International（CC BY 4.0）** 发布；使用者应保留以下署名，并在显示 IP 地理位置的网页提供来源链接。

**IP Geolocation by [DB-IP.com](https://db-ip.com)**

许可证：https://creativecommons.org/licenses/by/4.0/ 。

完整数据库为 126,873,350 字节，SHA-256 为 `5f64491f77c2c78b56cd83a88db4ea6c4666874322da38805498bbb754ccc735`。为避免每次修改代码都提交体积较大的二进制文件，项目使用 `scripts/package_geoip.py` 在构建发行包时下载、校验，并将整个 MMDB 文件放进发行包。Docker 镜像构建也会将其包含在内。安装完成后，查询 IP 归属地不需要联网。

使用源码构建时，先在仓库根目录运行：

```bash
python3 scripts/package_geoip.py
```

之后运行 `build.sh` 或 `build.bat`，数据库会随 `resources` 目录一起进入程序目录。普通服务器运行时，数据库位于可执行文件旁的 `resources/geoip` 目录；Docker 内部位置为 `/app/resources/geoip`。

管理员也可以通过 `RUSTDESK_API_GEOIP_DB` 指定其他 MMDB 文件。设置此变量后优先使用管理员指定的文件。

地理位置属于 IP 地址的估计位置，不能用于确认个人的实际位置。局域网、回环地址或数据库未收录的地址会分别显示适当的提示。

## 中文地名显示

当 DB-IP 返回中国大陆 IP 的位置时，程序会优先把省、市、区县名称显示为中文。例如，`221.11.67.246` 对应的地区可以显示为 `中国 陕西 西安 雁塔区`。其他国家和地区优先使用 MMDB 已提供的中文名称。无法确认中文对应名称时，继续显示数据库原有名称。

中文名称使用随软件编译的中国行政区划数据进行匹配，查询时不会发送 IP 地址到外部翻译服务。使用时会同时核对行政区划层级，避免将相同拼音的不同地区混淆。中国行政区划中文名称数据来自 [cn-divisions](https://github.com/roy-tian/cn-divisions)（Copyright © 2026 Roy Tian，MIT License），许可证全文保存在发行包的 `docs/licenses/cn-divisions-MIT.txt`。

中文地名之间使用空格分隔，避免占用不必要的横向空间。中文名称以原有 MMDB 数据库返回的地区为准。更新 DB-IP 或行政区划数据后，相关名称也可以随新版程序更新。
