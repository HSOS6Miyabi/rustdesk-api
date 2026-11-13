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

test("file-manager commands reach the browser transfer implementation",()=>{
  const begin=source.indexOf("window.setByName = (name, value) => {");
  const finish=source.indexOf("window.getByName = (name, arg) => {",begin);
  assert.ok(begin>=0&&finish>begin);
  const calls=[];
  const transfer={
    startUpload(value){calls.push(["upload",value]);},
    startDownload(value){calls.push(["download",value]);},
    cancel(value){calls.push(["cancel",value]);}
  };
  const conn={
    sendFileMessage(value){calls.push(["send",value]);},
    readRemoteDir(path,hidden){calls.push(["read",path,hidden]);},
    cancelFileJob(id){calls.push(["cancel-remote",id]);}
  };
  const context={
    window:{},
    getConn:()=>conn,
    currentBrowserTransfer:transfer,
    withFileTransfer:(_request,callback)=>callback(conn,transfer),
    openLocalFilePicker:folder=>calls.push(["pick",folder]),
    console
  };
  vm.runInNewContext(source.slice(begin,finish),context);
  const run=(key,value)=>context.window.setByName(key,JSON.stringify(value));
  context.window.setByName("select_files",true);
  run("send_local_files",{id:5,handle_index:2,to:"C:\\Users\\User\\file.txt"});
  run("send_files",{id:9,path:"C:\\Users\\User\\report.txt",is_remote:true});
  run("read_remote_dir",{path:"C:\\Users",include_hidden:false});
  run("rename_file",{id:8,path:"C:\\old.txt",new_name:"new.txt"});
  run("create_dir",{id:7,path:"C:\\folder"});
  context.window.setByName("cancel_job","9");
  assert.equal(calls[0][0],"pick");
  assert.equal(calls[1][0],"upload");
  assert.equal(calls[2][0],"download");
  assert.equal(calls[3][0],"read");
  assert.equal(calls[4][1].file_action.rename.new_name,"new.txt");
  assert.equal(calls[5][1].file_action.create.id,7);
  assert.equal(calls[6][0],"cancel");
  assert.equal(calls[7][0],"cancel-remote");
});

test("Flutter receives a valid scroll style before the first remote frame",()=>{
  const begin=source.indexOf("window.getByName = (name, arg) => {");
  const finish=source.indexOf("let opusWorker =",begin);
  assert.ok(begin>=0&&finish>begin);
  const values=new Map();
  const context={
    window:{},
    localStorage:{
      getItem(key){return values.get(key)||"";}
    },
    getConn:()=>({getOption(key){return values.get("conn."+key)||""}}),
    JSON, version:"1.5.3",console
  };
  vm.runInNewContext(source.slice(begin,finish),context);
  assert.equal(context.window.getByName("option:session","scroll_style"),"scrollauto");
  values.set("rustdesk.session.scroll_style","scrollbar");
  assert.equal(context.window.getByName("option:session","scroll_style"),"scrollbar");
  values.set("rustdesk.session.scroll_style","unsupported");
  assert.equal(context.window.getByName("option:session","scroll_style"),"scrollauto");
});

test("native RustDesk About version reflects the integrated 1.5.3 client",()=>{
  const build=fs.readFileSync(path.join(__dirname,"../src/gen_js_from_hbb.ts"),"utf8");
  assert.match(build,/export const version = "1\.5\.3"/);
  assert.doesNotMatch(build,/export const version = "1\.1\.10"/);
});
test("Flutter fullscreen bridge passes native toolbar requests to the parent workspace",async()=>{
  const begin=source.indexOf("window.setByName = (name, value) => {");
  const finish=source.indexOf("window.getByName = (name, arg) => {",begin);
  const calls=[];
  const parent={__rustdeskSetFullscreen(value,frame){
    calls.push([value,frame]);return Promise.resolve();
  },__rustdeskIsFullscreen(){return calls.at(-1)?.[0]===true}};
  const window={parent,onFullscreenChanged(v){calls.push(["notify",v]);}};
  const context={window,document:{},Promise,JSON,console};
  vm.runInNewContext(source.slice(begin,finish),context);
  assert.equal(window.setByName("fullscreen","Y"),"");
  assert.equal(window.setByName("fullscreen","N"),"");
  await new Promise(resolve=>setImmediate(resolve));
  assert.equal(calls[0][0],true);
  assert.equal(calls[1][0],false);
  assert.equal(calls[0][1],window);
});
