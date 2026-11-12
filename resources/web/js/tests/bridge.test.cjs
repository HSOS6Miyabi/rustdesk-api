"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const vm=require("node:vm");
const source=fs.readFileSync(path.join(__dirname,"../src/globals.js"),"utf8");
const start=source.indexOf("export function msgbox("),end=source.indexOf("let yuvWorker;",start);
assert.ok(start>=0&&end>start);
const functions=source.slice(start,end).replace(/export function/g,"function");
function createBridge(window){
  const context={window,checkIfRetry:()=>false,Uint8Array,console};
  vm.runInNewContext(functions+";this.msgbox=msgbox;this.pushEvent=pushEvent;",context);
  return context;
}
test("native WebClient can emit connection events without a Flutter callback",()=>{
  const bridge=createBridge({});
  assert.doesNotThrow(()=>bridge.msgbox("error","Connection Error","server unavailable"));
  assert.doesNotThrow(()=>bridge.pushEvent("peer-info",{id:"host-1"}));
});
test("Flutter event callbacks remain supported when provided",()=>{
  const events=[];
  const bridge=createBridge({onGlobalEvent:event=>events.push(JSON.parse(event))});
  bridge.msgbox("error","Connection Error","offline");
  bridge.pushEvent("peer-info",{id:"host-1"});
  assert.equal(events.length,2);
  assert.equal(events[0].name,"msgbox");
  assert.equal(events[1].name,"peer-info");
});
test("native rendering avoids the Flutter RGBA callback and production bundle includes the same fix",()=>{
  assert.match(source,/if \(typeof window\.onRgba !== "function"\) return;/);
  const built=fs.readFileSync(path.join(__dirname,"../dist/index.js"),"utf8");
  assert.match(built,/typeof window\.onRgba/);
  assert.match(built,/typeof window\.onGlobalEvent/);
  assert.match(built,/rustdesk:renderer-ready/);
});
