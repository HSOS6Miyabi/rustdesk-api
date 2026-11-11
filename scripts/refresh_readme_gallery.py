#!/usr/bin/env python3
"""Prepare compact README screenshots from the browser interaction tests."""

from pathlib import Path
import shutil
import struct

ROOT=Path(__file__).resolve().parents[1]
SOURCE=ROOT/"resources/admin/test-screenshots"
DESTINATION=ROOT/"docs/screenshots"

IMAGES={
    "overview-preview-desktop.png":"overview.png",
    "devices-preview-desktop.png":"devices.png",
    "settings-preview-desktop.png":"settings.png",
    "notifications-desktop.png":"notifications.png",
    "client-builder-desktop.png":"client-builder.png",
    "settings-preview-mobile.png":"settings-mobile.png",
}
DESTINATION.mkdir(parents=True,exist_ok=True)
for original,renamed in IMAGES.items():
    image=SOURCE/original
    data=image.read_bytes()
    if data[:8]!=b"\\x89PNG\\r\\n\\x1a\\n" or len(data)<10000:
        raise RuntimeError(f"Invalid screenshot: {image}")
    w,h=struct.unpack(">II",data[16:24])
    if w<300 or h<500:
        raise RuntimeError(f"Unexpected screenshot size: {image}: {w}x{h}")
    shutil.copyfile(image,DESTINATION/renamed)

gallery="""## 界面截图

以下截图来自管理控制台的 Chromium 浏览器测试，使用示例数据展示实际界面。

### 控制台概览

![RustDesk API 管理控制台概览](docs/screenshots/overview.png)

| 设备管理 | 设置中心 |
| :---: | :---: |
| <img src="docs/screenshots/devices.png" alt="设备列表与管理操作" width="600"> | <img src="docs/screenshots/settings.png" alt="全局设置中心与配置来源" width="600"> |

| 邮件事件通知 | 客户端安装包构建 |
| :---: | :---: |
| <img src="docs/screenshots/notifications.png" alt="邮件通知订阅与发送记录" width="600"> | <img src="docs/screenshots/client-builder.png" alt="客户端安装文件构建任务" width="600"> |

### 手机界面

<p align="center"><img src="docs/screenshots/settings-mobile.png" alt="手机端设置中心" width="320"></p>

"""
readme=ROOT/"README.md"
current=readme.read_text(encoding="utf-8")
main_heading="## 主要功能"
if current.count(main_heading)!=1:raise RuntimeError("README main function section not found")
if "## 界面截图" in current:
    lead,rest=current.split("## 界面截图",1)
    if main_heading not in rest:raise RuntimeError("README screenshot section could not be located")
    current=lead+rest[rest.index(main_heading):]
current=current.replace(main_heading,gallery+main_heading,1)
readme.write_text(current,encoding="utf-8")
print(f"Prepared {len(IMAGES)} real administration screenshots and README gallery")
