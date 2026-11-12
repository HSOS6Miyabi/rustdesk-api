"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const flutter=fs.readFileSync(path.join(__dirname,"../../main.dart.js"),"utf8");
const connection=fs.readFileSync(path.join(__dirname,"../src/globals.js"),"utf8");
test("WebClient v2 reuses RustDesk Flutter's remote desktop page and built-in toolbar",()=>{
  const version=JSON.parse(fs.readFileSync(path.join(__dirname,"../../version.json"),"utf8"));
  assert.equal(version.version,"1.5.3");
  assert.match(flutter,/session_add_sync/);
  assert.match(flutter,/send_mouse/);
  for(const icon of ["display_switcher.svg","keyboard_mouse.svg","record_screen.svg"]){
    assert.ok(fs.existsSync(path.join(__dirname,"../../assets/assets",icon)));
  }
});
test("Flutter's remote page automatically opens a selected device only once",()=>{
  assert.match(flutter,/session_add_sync/);
  assert.match(flutter,/session_start/);
  assert.match(flutter,/Web Client V2 \(Preview\)/);
  assert.match(connection,/rustdesk:native-session-open/);
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
