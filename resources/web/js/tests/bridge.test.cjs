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

test("RustDesk 1.5.3 Web startup has a registered shortcut reload hook",()=>{
  assert.match(source,/window\.reloadShortcuts\s*=\s*\(\)\s*=>/);
});


test("RustDesk mouse, keyboard and display commands use the native connection API",()=>{
  const begin=source.indexOf("window.setByName = (name, value) => {");
  const finish=source.indexOf("window.getByName = (name, arg) => {",begin);
  assert.ok(begin>=0&&finish>begin);
  const calls=[];
  const conn={
    inputMouse(...args){calls.push(["mouse",...args]);},
    inputKey(...args){calls.push(["key",...args]);},
    switchDisplay(display){calls.push(["display",display]);}
  };
  const context={window:{},curConn:conn,getConn:()=>conn,console};
  vm.runInNewContext(source.slice(begin,finish),context);
  context.window.setByName("send_mouse",JSON.stringify({type:"down",buttons:"left",x:"320",y:"200",shift:"true"}));
  context.window.setByName("input_key",JSON.stringify({name:"VK_RETURN",down:"true",press:"false",ctrl:"true"}));
  context.window.setByName("switch_display","2");
  assert.deepEqual(calls[0],["mouse",9,320,200,false,false,true,false]);
  assert.deepEqual(calls[1],["key","VK_RETURN",true,false,false,true,false,false]);
  assert.deepEqual(calls[2],["display","2"]);
});


test("RustDesk Flutter starts exactly once after the JavaScript decoder is ready",async()=>{
  const begin=source.indexOf("window.init = async () => {");
  const finish=source.indexOf("export function getPeers()",begin);
  assert.ok(begin>=0&&finish>begin);
  const calls=[];
  const window={onInitFinished(){calls.push("ready");}};
  const context={
    window,yuvWorker:null,opusWorker:{},pcmPlayer:{feed(){}},
    loadVp9(){calls.push("video");},
    async initZstd(){calls.push("zstd");},
    console:{log(){}}
  };
  vm.runInNewContext(source.slice(begin,finish),context);
  await context.window.init();
  assert.deepEqual(calls,["video","zstd","ready"]);
  await context.window.init();
  assert.deepEqual(calls,["video","zstd","ready","video","zstd"]);
});


test("Flutter receives correctly typed desktop, file and terminal event fields",()=>{
  const events=[];
  const bridge=createBridge({onGlobalEvent:raw=>events.push(JSON.parse(raw))});
  const directory={id:0,path:"/home/user",entries:[{entry_type:4,name:"file.txt",size:99,modified_time:100}]};
  bridge.pushEvent("file_dir",{value:JSON.stringify(directory),is_local:false});
  assert.equal(events[0].value,JSON.stringify(directory));
  assert.equal(events[0].is_local,"false");
  assert.equal(JSON.parse(events[0].value).entries[0].name,"file.txt");
  bridge.pushEvent("terminal_response",{
    type:"opened",terminal_id:3,success:true,persistent_sessions:[2,4]
  });
  assert.equal(events[1].type,"opened");
  assert.equal(events[1].terminal_id,"3");
  assert.equal(events[1].success,"true");
  assert.deepEqual(Array.from(events[1].persistent_sessions),[2,4]);
  bridge.pushEvent("terminal_response",{type:"data",terminal_id:3,data:"aGVsbG8="});
  assert.equal(events[2].type,"data");
  assert.equal(events[2].data,"aGVsbG8=");
  bridge.pushEvent("peer_info",{hostname:"desk",platform:"Windows",displays:[{width:1920,height:1080}]});
  assert.equal(events[3].hostname,"desk");
  assert.equal(events[3].platform,"Windows");
  assert.equal(JSON.parse(events[3].displays)[0].width,1920);
});
