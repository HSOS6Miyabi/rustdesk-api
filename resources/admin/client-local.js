"use strict";
(function(root, factory) {
  const builder = factory();
  if (typeof module === "object" && module.exports) module.exports = builder;
  if (root) root.RustDeskLocalBuilder = builder;
})(typeof globalThis === "object" ? globalThis : undefined, function() {
  const encoder = new TextEncoder();
  const MAX_FILE_SIZE = 100 * 1024 * 1024;
  const HOST_RE = /^[a-zA-Z0-9][a-zA-Z0-9.\-:\[\]]*$/;

  function assertHost(value, required) {
    if (!value && !required) return;
    if (!value || value.length > 180 || !HOST_RE.test(value) ||
        value.includes("..") || /[\s,\/\\]/.test(value)) {
      throw Error("请填写有效的 RustDesk 服务器地址");
    }
  }
  function quote(value) {
    return "'" + String(value).replaceAll("'", "'\\''") + "'";
  }
  function validateSettings(input) {
    const host = String(input.host || "").trim();
    const key = String(input.public_key || "").trim();
    const api = String(input.api_server || "").trim().replace(/\/$/, "");
    const relay = String(input.relay_server || "").trim();
    assertHost(host, true);
    assertHost(relay, false);
    let keyBytes;
    try {
      keyBytes = atob(key);
    } catch (_) {
      throw Error("服务器公钥必须使用 Base64 编码");
    }
    if (keyBytes.length !== 32) throw Error("服务器公钥解码后必须为 32 字节");
    if (api) {
      let value;
      try { value = new URL(api); } catch (_) { throw Error("API 地址无效"); }
      if (!["http:", "https:"].includes(value.protocol) || !value.hostname ||
          value.username || value.password || value.search || value.hash ||
          (value.pathname !== "/" && value.pathname !== "") || api.length > 200) {
        throw Error("请输入完整且有效的 HTTP 或 HTTPS API 地址");
      }
    }
    return {host, key, api, relay};
  }
  function validateFile(file, platform) {
    if (!file || typeof file.slice !== "function" || typeof file.size !== "number") {
      throw Error("请先选择本地 RustDesk 安装文件");
    }
    if (file.size < 64 || file.size > MAX_FILE_SIZE) {
      throw Error("本地安装文件大小应为 64 字节至 100 MiB");
    }
    const suffix = platform.startsWith("windows-") ? ".exe" :
      platform.startsWith("linux-") ? ".deb" :
      platform.startsWith("macos-") ? ".dmg" : null;
    if (!suffix) throw Error("不支持所选的客户端平台");
    const name = String(file.name || "");
    if (!name.toLowerCase().endsWith(suffix) ||
        !/^[a-zA-Z0-9][a-zA-Z0-9._+ -]*$/.test(name) ||
        name.length > 180 || name.includes("..")) {
      throw Error("所选平台需要合法的 " + suffix + " 安装文件名");
    }
    return {name, suffix};
  }
  function windowsName(settings) {
    const config = {host:settings.host, key:settings.key, api:settings.api, relay:settings.relay};
    const json = JSON.stringify(config);
    const raw = encoder.encode(json);
    let text = "";
    for (let i = 0; i < raw.length; ++i) text += String.fromCharCode(raw[i]);
    const encoded = btoa(text).replaceAll("+", "-").replaceAll("/", "_").replace(/=+$/, "");
    const filename = "rustdesk-licensed-" + encoded.split("").reverse().join("") + ".exe";
    if (filename.length > 240) throw Error("服务器配置太长，请缩短服务器或 API 地址");
    return filename;
  }
  function shellScript(platform, name, settings) {
    const linux = platform.startsWith("linux-");
    let script = "#!/bin/sh\nset -eu\n";
    if (linux) {
      script += 'cd "$(dirname "$0")"\n' +
        "if ! command -v apt-get >/dev/null 2>&1; then\n" +
        "  echo '该安装文件适用于 Debian 和 Ubuntu。' >&2\n  exit 1\nfi\n" +
        "sudo apt-get install -y -- " + quote("./" + name) + "\n" +
        "RUSTDESK_BIN=$(command -v rustdesk || true)\n" +
        'if [ -z "$RUSTDESK_BIN" ]; then\n' +
        "  echo '找不到已安装的 rustdesk 命令。' >&2\n  exit 1\nfi\n";
    } else {
      script += "RUSTDESK_BIN=/Applications/RustDesk.app/Contents/MacOS/RustDesk\n" +
        'if [ ! -x "$RUSTDESK_BIN" ]; then\n' +
        "  echo '请先通过 DMG 将 RustDesk.app 安装至应用程序目录。' >&2\n  exit 1\nfi\n";
    }
    for (const [option, value] of [
      ["custom-rendezvous-server", settings.host],
      ["key", settings.key],
      ["api-server", settings.api],
      ["relay-server", settings.relay],
    ]) {
      if (!value) continue;
      script += 'sudo "$RUSTDESK_BIN" --option ' + option + " " + quote(value) + "\n" +
        'ACTUAL=$(sudo "$RUSTDESK_BIN" --option ' + option + ")\n" +
        'if [ "$ACTUAL" != ' + quote(value) + ' ]; then\n' +
        "  echo '无法确认配置 " + option + " 已保存。' >&2\n  exit 1\nfi\n";
    }
    return script + "echo 'RustDesk 服务器设置已保存并已核对。'\n";
  }

  const TABLE = (() => {
    const result = new Uint32Array(256);
    for (let i = 0; i < result.length; i++) {
      let c = i;
      for (let j = 0; j < 8; ++j) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1;
      result[i] = c >>> 0;
    }
    return result;
  })();
  async function crc32(blob) {
    let crc = 0xffffffff;
    for (let position = 0; position < blob.size; position += 1024 * 1024) {
      const slice = new Uint8Array(await blob.slice(position, position + 1024 * 1024).arrayBuffer());
      for (const b of slice) crc = TABLE[(crc ^ b) & 255] ^ (crc >>> 8);
    }
    return (crc ^ 0xffffffff) >>> 0;
  }
  function putName(bytes, offset, text) { bytes.set(text, offset); }
  async function createZip(files) {
    if (files.length < 1 || files.length > 16) throw Error("压缩文件数量无效");
    const parts = [], entries = [];
    let position = 0;
    for (const file of files) {
      const name = encoder.encode(file.name);
      if (name.length > 500 || file.blob.size > 0xffffffff) throw Error("压缩包文件信息无效");
      const checksum = await crc32(file.blob); // ZIP 文件格式要求 CRC-32，不计算 SHA。
      const h = new Uint8Array(30 + name.length);
      const dv = new DataView(h.buffer);
      dv.setUint32(0, 0x04034b50, true);
      dv.setUint16(4, 20, true);
      dv.setUint16(6, 0x0800, true);
      dv.setUint16(8, 0, true);
      dv.setUint32(14, checksum, true);
      dv.setUint32(18, file.blob.size, true);
      dv.setUint32(22, file.blob.size, true);
      dv.setUint16(26, name.length, true);
      putName(h, 30, name);
      parts.push(h, file.blob);
      entries.push({name, checksum, size:file.blob.size, position, mode:file.mode || 0o644});
      position += h.length + file.blob.size;
    }
    const centralStart = position;
    for (const entry of entries) {
      const h = new Uint8Array(46 + entry.name.length);
      const dv = new DataView(h.buffer);
      dv.setUint32(0, 0x02014b50, true);
      dv.setUint16(4, (3 << 8) | 20, true);
      dv.setUint16(6, 20, true);
      dv.setUint16(8, 0x0800, true);
      dv.setUint32(16, entry.checksum, true);
      dv.setUint32(20, entry.size, true);
      dv.setUint32(24, entry.size, true);
      dv.setUint16(28, entry.name.length, true);
      dv.setUint32(38, ((0x8000 | entry.mode) << 16) >>> 0, true);
      dv.setUint32(42, entry.position, true);
      putName(h, 46, entry.name);
      parts.push(h);
      position += h.length;
    }
    const ending = new Uint8Array(22);
    const dv = new DataView(ending.buffer);
    dv.setUint32(0, 0x06054b50, true);
    dv.setUint16(8, entries.length, true);
    dv.setUint16(10, entries.length, true);
    dv.setUint32(12, position - centralStart, true);
    dv.setUint32(16, centralStart, true);
    parts.push(ending);
    return new Blob(parts, {type:"application/zip"});
  }
  async function build(file, platform, input) {
    const settings = validateSettings(input);
    const {name, suffix} = validateFile(file, platform);
    const first = new Uint8Array(await file.slice(0, 8).arrayBuffer());
    if (suffix === ".exe" && (first[0] !== 77 || first[1] !== 90)) {
      throw Error("所选文件缺少 Windows EXE 文件标识");
    }
    if (suffix === ".deb" && String.fromCharCode(...first) !== "!<arch>\n") {
      throw Error("所选文件不是有效的 Debian 安装包");
    }
    if (suffix === ".exe") return {name:windowsName(settings), blob:file, archive:false};
    const linux = suffix === ".deb";
    const scriptName = linux ? "install-and-configure.sh" : "configure-rustdesk.command";
    const script = shellScript(platform, name, settings);
    const readme = linux ?
      "RustDesk Linux 安装文件和服务器配置\n\n在 Debian/Ubuntu 上运行 sh ./install-and-configure.sh。\n" :
      "RustDesk macOS 安装文件和服务器配置\n\n先从 DMG 安装 RustDesk.app，再运行 sh ./configure-rustdesk.command。\n";
    const note = "\n此压缩包由浏览器本地生成，所选安装文件内容未改变。\n" +
      "浏览器没有上传安装文件，也没有计算 SHA-256。\n" +
      "配置脚本通过已安装客户端的 --option 命令保存设置，并再次读取结果。\n" +
      "请检查安装文件来源、数字签名、客户端版本和服务器连接。\n";
    const zip = await createZip([
      {name, blob:file, mode:0o644},
      {name:scriptName, blob:new Blob([encoder.encode(script)]), mode:0o755},
      {name:"README.txt", blob:new Blob([encoder.encode(readme + note)]), mode:0o644},
    ]);
    return {name:"rustdesk-local-" + platform + "-configured.zip",blob:zip,archive:true};
  }
  return {build, validateSettings, validateFile, windowsName, shellScript, createZip, crc32};
});
