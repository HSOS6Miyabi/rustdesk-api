"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const vm = require("node:vm");
const fs = require("node:fs");
const path = require("node:path");

const source = fs.readFileSync(path.join(__dirname, "app.js"), "utf8");

function storage(initial = {}) {
  const values = new Map(Object.entries(initial));
  return {
    getItem(key) { return values.get(key) || null; },
    setItem(key, value) { values.set(key, String(value)); },
    removeItem(key) { values.delete(key); }
  };
}

function createElement(id = "") {
  const listeners = {};
  return {
    id, listeners, value: "", textContent: "", hidden: false, children: [], src: "", title: "",
    className: "", disabled: false, attributes:{}, classList: { add() {},toggle() {} },
    setAttribute(name,value){this.attributes[name]=String(value);},
    addEventListener(type, fn) { listeners[type] = fn; },
    append(...items) { this.children.push(...items); },
    remove() { this.removed = true; },
    replaceChildren(...items) { this.children = items; },
    focus() {}, scrollIntoView() {},
    querySelector() { return createElement(); }
  };
}

function fixture(options = {}) {
  const ids = [
    "v2-host", "v2-key", "v2-id", "v2-message",
    "engine-state", "shared-info", "shared-items", "shared-login",
    "shared-login-form", "sso-login-link", "connect-form", "v2-disconnect",
    "v2-refresh", "v2-captcha", "v2-captcha-image", "v2-webrtc",
    "v2-page-fullscreen", "v2-screen-fullscreen", "v2-fullscreen-exit", "v2-native-client", "v2-clear-password",
    "v2-session-tabs", "v2-session-panels", "v2-empty-session",
    "v2-mode-desktop", "v2-mode-file", "v2-mode-camera", "v2-mode-terminal", "v2-mode-adminTerminal", "v2-device-query", "v2-search-device", "v2-device-results"
  ];
  const elements = new Map(ids.map(id => [id, createElement(id)]));
  const stage=createElement("stage");
  const docEvents={};
  const frameEvents={};
  const bridgeCommands=[];
  elements.get("v2-native-client").contentWindow={
    addEventListener(type,listener){frameEvents[type]=listener;},
    setByName(name,value){bridgeCommands.push([name,value]);}
  };
  const server = {
    id_server: "desk.example.com:21116",
    key: "test-server-public-key"
  };
  const apiResults = {
    "/config/server": server,
    "/my/peer/list":{list:[{id:"owned-01",hostname:"Owned Workstation",alias:"Office owned",os:"Windows 11"}],total:1},
    "/geoip/lookup": {device_status:{"123456789":options.online??true,"owned-01":true},locations:{},database_ready:false},
    "/shared_devices/list": {
      total: 1,
      list: [{
        id: "123456789", hostname: "Remote Workstation",
        owner_username: "alice", sources: [{type:"contact",name:"alice"}]
      }]
    },
    "/login-options": { ops: ["company-oidc"], disable_pwd: options.disablePwd ?? true, need_captcha:false }
  };
  const apiCalls = [];
  const localStorage = storage({access_token:"stale-persistent-token"});
  const sessionStorage = storage(options.token ? {"rustdesk.console.token":options.token} : {});
  const assigned = [];
  const window = {
    addEventListener(){},
    scrollTo(){},
    location:{search:options.search||"",assign(url){assigned.push(url);}}
  };
  const document = {
    fullscreenElement:null,
    body:{classList:{toggle(){}}},
    addEventListener(type,listener){docEvents[type]=listener;},
    async exitFullscreen(){this.fullscreenElement=null;docEvents.fullscreenchange?.();},
    getElementById(id) {return elements.get(id) || null;},
    querySelector(selector) {
      if(selector===".stage")return stage;
      return null;
    },
    createElement(type){return createElement(type);},
    createTextNode(content){return {textContent:String(content)};}
  };
  stage.requestFullscreen=async()=>{document.fullscreenElement=stage;docEvents.fullscreenchange?.();};
  const fetch = async (url) => {
    const u = new URL(String(url), "https://example.test");
    const endpoint = u.pathname.replace(/^\/api\/admin/, "");
    apiCalls.push(endpoint);
    if(endpoint==="/shared_devices/list"&&!options.token){
      return {ok:false,status:403,async json(){return {message:"Unauthenticated"};}};
    }
    if(!Object.prototype.hasOwnProperty.call(apiResults,endpoint)){
      return {ok:false,status:404,async json(){return {message:"Unexpected path "+endpoint};}};
    }
    return {ok:true,status:200,async json(){return {code:0,data:apiResults[endpoint]};}};
  };
  vm.runInNewContext(source,{
    window, document, fetch, sessionStorage, localStorage, URL, URLSearchParams,
    MutationObserver: class {observe(){}disconnect(){}},
    console
  }, {filename:"resources/web2/app.js"});
  const settle=async()=>{await new Promise(resolve=>setImmediate(resolve));await new Promise(resolve=>setImmediate(resolve));};
  return {elements,sessionStorage,localStorage,apiCalls,assigned,document,stage,docEvents,frameEvents,bridgeCommands,window,settle};
}

test("WebClient v2 loads its own server settings and shared device list",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  assert.equal(app.elements.get("v2-host").value,"desk.example.com:21116");
  assert.equal(app.elements.get("v2-key").value,"test-server-public-key");
  assert.ok(app.apiCalls.includes("/config/server"));
  assert.ok(app.apiCalls.includes("/shared_devices/list"));
  assert.match(app.elements.get("shared-info").textContent,/1/);
  assert.equal(app.elements.get("shared-items").children.length,1);
  assert.ok(app.apiCalls.includes("/geoip/lookup"));
  const shared=app.elements.get("shared-items").children[0];
  const dot=shared.children[0].children[0].children[0];
  assert.equal(dot.className,"device-status-dot online");
  assert.equal(dot.attributes["aria-label"],"在线");
  app.elements.get("shared-items").children[0].children[1].listeners.click();
  assert.equal(app.elements.get("v2-id").value,"123456789");
});

test("WebClient v2 launches the original RustDesk client with the selected device",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  app.elements.get("v2-id").value="123456789";
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  assert.equal(app.elements.get("v2-native-client").src,"/webclient/#/connect/123456789");
  assert.equal(app.localStorage.getItem("custom-rendezvous-server"),"desk.example.com:21116");
  assert.equal(app.sessionStorage.getItem("rustdesk.console.token"),"session-token");
  assert.equal(app.localStorage.getItem("access_token"),null);
  app.elements.get("v2-disconnect").listeners.click();
  assert.equal(app.elements.get("v2-native-client").src,"about:blank");
});

test("WebClient v2 offers SSO when password login is disabled",async()=>{
  const app=fixture({disablePwd:true});
  await app.settle();
  const link=app.elements.get("sso-login-link");
  assert.equal(link.hidden,false);
  assert.equal(app.elements.get("shared-login-form").hidden,true);
  link.listeners.click();
  assert.equal(app.sessionStorage.getItem("rustdesk.console.return"),"/webclient2/");
});

test("WebClient v2 supports password login alongside SSO when allowed",async()=>{
  const app=fixture({disablePwd:false});
  await app.settle();
  assert.equal(app.elements.get("shared-login-form").hidden,false);
  assert.equal(app.elements.get("sso-login-link").hidden,false);
});

test("WebClient v2 enables direct WebRTC only after the user selects it",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const toggle=app.elements.get("v2-webrtc");
  assert.equal(toggle.checked,false);
  toggle.checked=true;
  app.elements.get("v2-id").value="123456789";
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  assert.equal(app.sessionStorage.getItem("rustdesk.webclient.webrtc"),"Y");
  toggle.checked=false;
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  assert.equal(app.sessionStorage.getItem("rustdesk.webclient.webrtc"),"N");
});


test("WebClient v2 displays offline devices in red without periodic requests",async()=>{
  const app=fixture({token:"session-token",online:false});
  await app.settle();
  const shared=app.elements.get("shared-items").children[0];
  const dot=shared.children[0].children[0].children[0];
  assert.equal(dot.className,"device-status-dot offline");
  assert.equal(dot.attributes["aria-label"],"离线");
  assert.equal(app.apiCalls.filter(path=>path==="/geoip/lookup").length,1);
});

test("Fullscreen uses a compact exit button beside the native toolbar",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const page=app.elements.get("v2-page-fullscreen");
  const screen=app.elements.get("v2-screen-fullscreen");
  const exit=app.elements.get("v2-fullscreen-exit");
  assert.equal(exit.hidden,true);
  await page.listeners.click();
  assert.equal(page.attributes["aria-pressed"],"true");
  assert.equal(exit.hidden,false);
  app.docEvents.keydown({key:"Escape"});
  assert.equal(exit.hidden,true);
  await screen.listeners.click();
  assert.equal(app.document.fullscreenElement,app.stage);
  assert.equal(exit.hidden,false);
  await exit.listeners.click();
  assert.equal(app.document.fullscreenElement,null);
  assert.equal(exit.hidden,true);
  await page.listeners.click();
  app.elements.get("v2-native-client").listeners.load();
  app.frameEvents.keydown({key:"Escape"});
  assert.equal(exit.hidden,true);
});

test("Shared device names and owner names link to the corresponding control-console details",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const device=app.elements.get("shared-items").children[0];
  const details=device.children[0].children[0];
  const owner=device.children[0].children[1].children[1];
  assert.match(details.href,/entity=device&value=123456789/);
  assert.match(owner.href,/entity=user&value=alice/);
  assert.equal(device.children[1].textContent,"选中连接");
});

test("WebClient v2 searches owned and shared devices by their details",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  app.elements.get("v2-device-query").value="workstation";
  await app.elements.get("v2-search-device").listeners.click();
  await app.settle();
  assert.ok(app.apiCalls.includes("/my/peer/list"));
  assert.ok(app.apiCalls.includes("/shared_devices/list"));
  const results=app.elements.get("v2-device-results");
  assert.equal(results.children.length,2);
  const ownerResult=results.children[0];
  assert.match(ownerResult.children[0].children[0].href,/entity=device&value=owned-01/);
  assert.match(ownerResult.children[0].children[0].children[0].className,/device-status-dot online/);
  ownerResult.children[1].listeners.click();
  assert.equal(app.elements.get("v2-id").value,"owned-01");
});

test("Linked devices start the existing RustDesk client using the original toolbar",async()=>{
  const app=fixture({token:"session-token",search:"?id=owned-01&name=Office%20desktop"});
  await app.settle();
  assert.equal(app.elements.get("v2-id").value,"owned-01");
  assert.equal(app.elements.get("v2-native-client").title,"Office desktop · 桌面 · RustDesk");
  assert.equal(app.elements.get("v2-native-client").src,"/webclient/#/connect/owned-01");
  app.elements.get("v2-native-client").listeners.load();
  assert.match(app.elements.get("engine-state").textContent,/RustDesk 操作界面/);
});
test("Selecting a shared device keeps its name as the iframe accessibility title",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const item=app.elements.get("shared-items").children[0];
  item.children[1].listeners.click();
  assert.equal(app.elements.get("v2-native-client").title,"Remote Workstation");
  assert.equal(app.elements.get("v2-id").value,"123456789");
  app.elements.get("v2-id").value="manual-02";
  app.elements.get("v2-id").listeners.input();
  assert.equal(app.elements.get("v2-native-client").title,"RustDesk 远程桌面");
});
test("The original Flutter client renders the operation bar without an additional toolbar",()=>{
  const html=fs.readFileSync(path.join(__dirname,"index.html"),"utf8");
  const css=fs.readFileSync(path.join(__dirname,"style.css"),"utf8");
  const original=fs.readFileSync(path.join(__dirname,"../web/index.html"),"utf8");
  assert.match(html,/<iframe id="v2-native-client"/);
  assert.match(html,/id="v2-session-tabs"/);
  assert.match(html,/id="v2-session-panels"/);
  assert.match(html,/id="v2-fullscreen-exit"/);
  assert.doesNotMatch(html,/class="stage-toolbar"|id="v2-active-device-id"|id="app"/);
  assert.match(original,/flutter_bootstrap\.js/);
  assert.match(css,/\.native-client\{/);
  assert.match(css,/\.stage:fullscreen/);
});

test("Switching remote-session tabs preserves existing iframe connections",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const id=app.elements.get("v2-id");
  const connect=app.elements.get("connect-form").listeners.submit;
  const first=app.elements.get("v2-native-client");
  id.value="first-device";
  connect({preventDefault(){}});
  const firstAddress=first.src;
  assert.equal(app.elements.get("v2-session-tabs").children.length,1);
  id.value="second-device";
  id.listeners.input();
  connect({preventDefault(){}});
  const tabs=app.elements.get("v2-session-tabs");
  const panels=app.elements.get("v2-session-panels");
  assert.equal(tabs.children.length,2);
  assert.equal(panels.children.length,1);
  const second=panels.children[0];
  assert.equal(first.src,firstAddress);
  assert.equal(second.src,"/webclient/#/connect/second-device");
  assert.equal(tabs.children[1].children[0].attributes["aria-selected"],"true");
  tabs.children[0].children[0].listeners.click();
  assert.equal(first.src,firstAddress);
  assert.equal(second.src,"/webclient/#/connect/second-device");
  assert.equal(tabs.children[0].children[0].attributes["aria-selected"],"true");
  assert.equal(app.elements.get("v2-id").value,"first-device");
  app.elements.get("v2-disconnect").listeners.click();
  assert.equal(first.src,"about:blank");
  assert.equal(second.removed,undefined);
  assert.equal(tabs.children.length,1);
  assert.equal(tabs.children[0].children[0].attributes["aria-selected"],"true");
});

test("Function tabs keep the remote desktop mounted while opening new RustDesk modes",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  app.elements.get("v2-id").value="desk-01";
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  const desktop=app.elements.get("v2-native-client");
  const address=desktop.src;
  assert.equal(app.elements.get("v2-mode-desktop").attributes["aria-selected"],"true");
  app.elements.get("v2-mode-file").listeners.click();
  assert.equal(app.elements.get("v2-mode-file").attributes["aria-selected"],"true");
  assert.equal(app.elements.get("v2-session-panels").children[0].src,"/webclient/#/file-transfer/desk-01");
  assert.equal(desktop.src,address);
  app.elements.get("v2-mode-desktop").listeners.click();
  assert.equal(desktop.src,address);
  assert.equal(app.elements.get("v2-session-panels").children.length,1);
  assert.equal(app.elements.get("v2-mode-desktop").attributes["aria-selected"],"true");
  app.elements.get("v2-mode-camera").listeners.click();
  assert.equal(app.elements.get("v2-session-panels").children[1].src,"/webclient/#/view-camera/desk-01");
  app.elements.get("v2-mode-terminal").listeners.click();
  assert.equal(app.elements.get("v2-session-panels").children[2].src,"/webclient/#/terminal/desk-01");
  app.elements.get("v2-mode-adminTerminal").listeners.click();
  assert.equal(app.elements.get("v2-session-panels").children[3].src,"/webclient/#/terminal-admin/desk-01");
  assert.equal(desktop.src,address);
});

test("Remembered password deletion is scoped to one server and device without closing its session",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  const firstId="desk-01",otherId="desk-02",host="desk.example.com:21116";
  const savedKey="rustdesk.webclient.credentials.v1:"+JSON.stringify([host,firstId]);
  const otherKey="rustdesk.webclient.credentials.v1:"+JSON.stringify([host,otherId]);
  app.localStorage.setItem(savedKey,"[1,2,3]");
  app.localStorage.setItem(otherKey,"[4,5,6]");
  app.localStorage.setItem("peers",JSON.stringify({[firstId]:{remember:true,password:"legacy"}}));
  app.elements.get("v2-id").value=firstId;
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  const frame=app.elements.get("v2-native-client"),existing=frame.src;
  app.elements.get("v2-clear-password").listeners.click();
  assert.equal(app.localStorage.getItem(savedKey),null);
  assert.equal(app.localStorage.getItem(otherKey),"[4,5,6]");
  assert.equal(JSON.parse(app.localStorage.getItem("peers"))[firstId].remember,false);
  assert.equal(JSON.parse(app.localStorage.getItem("peers"))[firstId].password,undefined);
  assert.deepEqual(app.bridgeCommands.map(command=>command[0]),["forget_password"]);
  assert.equal(frame.src,existing);
  assert.equal(app.elements.get("v2-session-tabs").children.length,1);
  assert.match(app.elements.get("v2-message").textContent,/已清除/);
});
