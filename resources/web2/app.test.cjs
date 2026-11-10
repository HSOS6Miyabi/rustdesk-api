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
    id, listeners, value: "", textContent: "", hidden: false, children: [],
    className: "", disabled: false, attributes:{}, classList: { add() {} },
    setAttribute(name,value){this.attributes[name]=String(value);},
    addEventListener(type, fn) { listeners[type] = fn; },
    append(...items) { this.children.push(...items); },
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
    "v2-refresh", "v2-captcha", "v2-captcha-image", "v2-webrtc", "app",
    "host", "key"
  ];
  const elements = new Map(ids.map(id => [id, createElement(id)]));
  elements.set("native-id", createElement("native-id"));
  const server = {
    id_server: "desk.example.com:21116",
    key: "test-server-public-key"
  };
  const apiResults = {
    "/config/server": server,
    "/geoip/lookup": {device_status:{"123456789":options.online??true},locations:{},database_ready:false},
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
  let connectionCount = 0;
  const window = {
    connect(){connectionCount++;},
    cancel(){},
    scrollTo(){},
    location:{assign(url){assigned.push(url);}}
  };
  const document = {
    getElementById(id) {return elements.get(id) || null;},
    querySelector(selector) {
      if(selector==="#app #connect")return createElement();
      if(selector==="#app #id")return elements.get("native-id");
      return null;
    },
    createElement(type){return createElement(type);},
    createTextNode(content){return {textContent:String(content)};}
  };
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
  return {elements,sessionStorage,localStorage,apiCalls,assigned,
    settle,connectCount:()=>connectionCount};
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
  app.elements.get("shared-items").children[0].listeners.click();
  assert.equal(app.elements.get("v2-id").value,"123456789");
});

test("WebClient v2 initiates a connection without storing access tokens permanently",async()=>{
  const app=fixture({token:"session-token"});
  await app.settle();
  app.elements.get("v2-id").value="123456789";
  app.elements.get("connect-form").listeners.submit({preventDefault(){}});
  assert.equal(app.connectCount(),1);
  assert.equal(app.elements.get("native-id").value,"123456789");
  assert.equal(app.elements.get("host").value,"desk.example.com:21116");
  assert.equal(app.sessionStorage.getItem("rustdesk.console.token"),"session-token");
  assert.equal(app.localStorage.getItem("access_token"),null);
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
