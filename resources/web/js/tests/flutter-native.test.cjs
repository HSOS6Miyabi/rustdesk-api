"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const flutter=fs.readFileSync(path.join(__dirname,"../../main.dart.js"),"utf8");
const connection=fs.readFileSync(path.join(__dirname,"../src/globals.js"),"utf8");
test("WebClient v2 reuses RustDesk Flutter's remote desktop page and built-in toolbar",()=>{
  assert.match(flutter,/A\.rs\.prototype=\{/);
  assert.match(flutter,/A\.HU\.prototype=\{/);
  assert.match(flutter,/A\.aLN\(s\.a\.c,!1\)/);
  assert.match(flutter,/A\.Go\.prototype=\{/);
});
test("Flutter's remote page automatically opens a selected device only once",()=>{
  assert.match(flutter,/__rustdeskAutoOpened/);
  assert.match(flutter,/URLSearchParams\(window\.location\.search\)\.get\("connect"\)/);
  assert.match(flutter,/remoteHome\.HF\(remoteId\)/);
  assert.match(flutter,/rustdesk:native-session-open/);
});
test("original RustDesk remote UI retains mouse, keyboard and display-switch actions",()=>{
  assert.match(flutter,/"send_mouse"/);
  assert.match(flutter,/"input_key"/);
  assert.match(flutter,/"switch_display"/);
  assert.match(connection,/curConn\.inputMouse\(/);
  assert.match(connection,/curConn\.inputKey\(/);
  assert.match(connection,/curConn\.switchDisplay\(/);
  assert.match(connection,/window\.onGlobalEvent/);
});
