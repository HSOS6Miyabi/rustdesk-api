"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const vm = require("node:vm");

const source = fs.readFileSync(path.join(__dirname, "app.js"), "utf8");

function storage(initial = {}) {
  const entries = new Map(Object.entries(initial));
  return {
    getItem: key => entries.get(key) || null,
    setItem: (key, value) => entries.set(key, String(value)),
    removeItem: key => entries.delete(key)
  };
}

function createConsole(isAdmin = false, overrides = {}, localValues = {}, behavior = {}) {
  const calls = [];
  const events = {};
  const windowEvents = {};
  const root = { innerHTML: "",querySelector(){return null;} };
  const assigned = [];
  const opened = [];
  const replaced = [];
  const browserLocation = {
    hash: behavior.initialHash || "#/overview",pathname:"/_admin/",search:"",
    assign(url) {assigned.push(url);}
  };
  const user = {
    username: "demo-user",
    nickname: "Demo",
    route_names: isAdmin ? ["*"] : [],
    token: "example-session"
  };
  const fixtures = {
    "/config/admin": { title: "RustDesk" },
    "/settings/effective": {bootstrap_version:0,database_version:1,items:[
      {key:"app.register",name:"开放注册",description:"允许新用户注册",category:"常规与登录",
        environment:"RUSTDESK_API_APP_REGISTER",value_type:"bool",source:"环境变量",
        current_value:"true",override_enabled:false,bootstrap:false,requires_restart:true,active:true},
      {key:"gorm.type",name:"数据库类型",description:"选择要连接的数据库",category:"数据库",
        environment:"RUSTDESK_API_GORM_TYPE",value_type:"string",source:"配置文件",
        current_value:"sqlite",override_enabled:false,bootstrap:true,active:true},
      {key:"jwt.key",name:"JWT 签名密钥",description:"登录令牌签名",category:"令牌",
        environment:"RUSTDESK_API_JWT_KEY",value_type:"string",source:"环境变量",
        configured:true,current_value:"",override_enabled:false,bootstrap:true,sensitive:true,requires_restart:true,active:true}
    ]},
    "/audit/retention": {connection_days:90,file_days:30,login_days:90},
    "/audit/cleanup/preview": {items:[{kind:"connection",name:"远程连接记录",eligible:2,retention_days:90}],total:2},
    "/audit/cleanup": {items:[{kind:"connection",name:"远程连接记录",deleted:2}],deleted:2},
    "/settings/overview": {summary:{
      smtp_settings:{enabled:true,host:"mail.example.test",port:587,sender_email:"team@example.test"},
      notification_settings:{enabled:false,events:""},
      audit_retention:{connection_days:90,login_days:30},
      device_policies:2,access_rules:0,relay_nodes:1,push_channels:1
    },outbox:{push:{pending:0,failed:0},notification:{pending:0,failed:0}}},
    "/settings/history": {list:[]},
    "/settings/outbox": {list:[]},
    "/settings/validate": {valid:true},
    "/settings/overrides": {saved:true},
    "/settings/overrides/reset": {saved:true},

    "/user/current": user,
    "/my/group/list": { list: [{ id: 7, name: "个人群组", type: 3, owner_user_id: 12 }], total: 1 },
    "/group/list": { list: [{ id: 1, name: "系统分组", type: 1 }], total: 1 },
    "/my/audit_conn/list": {
      list: [{
        id: 17, from_peer: "123456", peer_id: "654321", from_account: "demo-user",
        to_account: "another-user", direction: "outgoing", created_at: 1710000000
      }], total: 1
    },
    "/audit_conn/list": { list: [{ id:21, peer_id:"managed-desk", from_peer:"helpdesk", note:"现场支持" }], total:1 },
    "/my/audit_file/list": {
      list: [{ id: 18, from_peer: "123456", peer_id: "654321", from_account: "demo-user",
        to_account: "another-user", path: "/home/file", direction: "outgoing" }], total: 1
    },
    "/mail/settings": { enabled:false, host:"smtp.example.test",port:587,security:"starttls",sender_email:"team@example.test" },
    "/notification/settings": {settings:{enabled:true,recipients:"team@example.test",events:"login.failure,audit.file"},catalog:["login.failure","audit.file"]},
    "/notification/list": {list:[{id:4,event_type:"audit.file",recipient:"team@example.test",status:"failed",attempts:3,last_error:"SMTP unreachable"}],total:1},
    "/geoip/lookup": {locations:{"10.0.0.5":"局域网地址","127.0.0.1":"本地回环地址","192.0.2.1":"位置未知"},device_status:{"office":true,"home-remote":false,"123456":true,"654321":false,"helpdesk":true,"managed-desk":false,"support-123456":true,"deployed-10002":false},device_names:{"office":"Office","home-remote":"Home PC","123456":"Device A","654321":"Device B","helpdesk":"Helpdesk","managed-desk":"Managed desk","support-123456":"Support PC","deployed-10002":"Deployment"},database_ready:false},
    "/push/events": [{code:"device.online",name:"设备上线"},{code:"device.offline",name:"设备离线"},{code:"connection.started",name:"远程连接建立"}],
    "/push/channels": {list:[{id:11,kind:"messagepusher",name:"我的通知",enabled:true,events:["device.online","connection.started"],base_url:"https://push.example.test",api_key_configured:true}]},
    "/push/stream-keys": {list:[{id:13,name:"自动化订阅",prefix:"rdapi_example",permissions:"events.read",expires_at:0,revoked_at:0}]},
    "/push/history": {list:[{id:21,channel_id:11,title:"[RustDesk · 设备上线] 办公室电脑",event_type:"device.online",attempts:1,status:"sent",created_at:1710000000}],total:1},
    "/client_builder/profiles": {list:[{
      id:2,name:"Office installer",platform:"windows-x64",version:"1.5.1",
      host:"desk.example.com:21116",public_key:"AA==",api_server:"https://desk.example.com",
      relay_server:""
    }],total:1},
    "/client_builder/tasks": {list:[{
      id:15,profile_id:2,platform:"windows-x64",version:"1.5.1",
      status:"ready",filename:"rustdesk-licensed-verified.exe",sha256:"sample",bytes:26
    },{
      id:16,profile_id:2,platform:"windows-x64",version:"1.5.1",
      status:"failed",error:"download unavailable"
    }],total:2},


    "/access_rule/list": {list:[{id:8,name:"Office support",source_type:"group",source_id:2,target_type:"device_group",target_id:"3",effect:"allow",enabled:true}],total:1},
    "/admin_role/list": {list:[{id:12,name:"Read-only devices",permissions:"devices.view",scope_type:"global",scope_id:0,enabled:true}],total:1},
    "/admin_role_assignment/list": {list:[{id:4,user_id:15,role_id:12}],total:1},
    "/mfa/status": {enabled:false,pending:false,remaining_recovery_codes:0},
    "/mfa/email/status": {enabled:false,email:"owner@example.test",mail_configured:true},
    "/passkey/list": {list:[],total:0},
    "/passkey/revoke": {},
    "/my/login_log/list": { list: [], total: 0 },
    "/my/session/list": {list:[{id:10,device_id:"desk-owned",conn_id:501,last_seen_at:1794400000}],total:1},
    "/my/session/disconnect/list": {list:[{id:4,device_id:"desk-owned",conn_id:400,status:"completed"}],total:1},
    "/my/user_token/list": {list:[{id:50,device_id:"desk-owned",device_uuid:"uuid-owned",login_ip:"192.0.2.1",client:"webadmin",platform:"windows",username:"demo-user",active:true,current:false}],total:1},
    "/my/user_token/revoke": {},
    "/my/session/disconnect": {status:"pending"},
    "/user_token/list": {list:[{id:51,username:"demo-user",device_id:"desk-owned",device_uuid:"uuid-owned",active:true,current:false}],total:1},
    "/user_token/delete": {},
    "/user/list": { list: [], total: 0 },
    "/peer/list": { list: [], total: 0 },
    "/my/peer/list": { list: [], total: 0 },
    "/my/address_book/list": { list: [], total: 0 },
    "/contact/list": { outgoing: [], incoming: [] },
    "/shared_devices/list": { list: [], total: 0 },
    "/shared_devices/sources": { sources: [] },
    "/config/server":{id_server:"desk.example.test:21116",relay_server:"desk.example.test:21117",api_server:"https://api.example.test"},
    "/rustdesk/execute": {kind:"metric",summary:"服务器当前设置",value:10,unit:"Mb/s"},
    "/group/members": { user_ids: [], users: [] },
    "/my/group/members": { user_ids: [12], users: [{ id: 12, username: "demo-user" }] }
  };
  const fetch = async function(url, options) {
    const location = new URL(String(url), "https://example.test");
    const endpoint = location.pathname.replace(/^\/api\/admin/, "");
    calls.push({ endpoint, method: options?.method || "GET", query:location.search });
    if(endpoint==="/oidc/auth-query"){
      if(behavior.oidcNetworkError)throw Error("Network unavailable");
      if(behavior.oidcError)return {ok:true,status:200,async json(){
        return {code:101,message:behavior.oidcError};
      }};
      return {ok:true,status:200,async json(){
        return {code:0,data:behavior.oidcResult || {token:"mobile-sso-token"}};
      }};
    }
    if(endpoint==="/user/current"&&behavior.failCurrent)throw Error("Temporary API outage");
    if(endpoint==="/user/current"&&behavior.rejectCurrent){
      return {ok:true,status:200,async json(){return {code:403,message:"Please sign in"};}};
    }
    if (!Object.prototype.hasOwnProperty.call(fixtures, endpoint)) {
      return { ok: true, status: 200, async json() { return { code: 101, message: "Unexpected "+endpoint }; } };
    }
    return { ok: true, status: 200, async json() { return { code: 0, data: fixtures[endpoint] }; } };
  };
  const mockDocument = {
    documentElement: { dataset: {} },
    getElementById(id) { return id === "app" ? root : null; },
    addEventListener(name, handler) { events[name] = handler; },
    querySelector() { return null; },
    createElement() { return { append() {}, remove() {}, className:"", role:"", textContent:"" }; },
    body: { append() {} }
  };
  const mockWindow = {
    open(url,target){const tab={opener:{}};opened.push({url,target,tab});return tab;},
    addEventListener(name,handler) { windowEvents[name]=handler; },
    confirm() { return true; },
    history: {replaceState(_state,_title,url){
      replaced.push(url);
      browserLocation.hash=new URL(url,"https://example.test").hash;
    }},
    location: browserLocation
  };
  const session = storage({ "rustdesk.console.token": "example-session", ...overrides });
  const local = storage(localValues);
  const context = {
    document: mockDocument, window: mockWindow, location: browserLocation,
    sessionStorage: session, localStorage: local, fetch,
    AbortController, URLSearchParams,
    setTimeout: () => 1, clearTimeout() {},
    console
  };
  vm.runInNewContext(source, context, { filename: "resources/admin/app.js" });
  const settle = () => new Promise(resolve => setImmediate(resolve));
  return {
    root, calls, events, windowEvents, fixtures, settle, assigned, opened, replaced, document:mockDocument, session, local, behavior,
    async open(view) {
      const control = { dataset: { view } };
      await events.click({ target: { closest: () => control } });
      await settle();
      await settle();
    },
    async action(name, id) {
      const control = { dataset: { action: name, ...(id === undefined ? {} : { id: String(id) }) } };
      await events.click({ target: { closest: () => control } });
      await settle();
      await settle();
    }
  };
}

test("ordinary users see personal-group and personal-audit navigation", async () => {
  const app = createConsole();
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML, /我的群组/);
  assert.match(app.root.innerHTML, /我的连接记录/);
  assert.match(app.root.innerHTML, /我的文件传输/);
  assert.doesNotMatch(app.root.innerHTML, /用户分组<\/button>/);
  await app.open("mygroups");
  assert.match(app.root.innerHTML, /个人群组/);
  assert.ok(app.calls.some(r => r.endpoint === "/my/group/list"));
  assert.ok(!app.calls.some(r => r.endpoint === "/group/list"));
});

test("administrators have a separate global group entry", async () => {
  const app = createConsole(true);
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML, /我的群组/);
  assert.match(app.root.innerHTML, /用户分组/);
  await app.open("groups");
  assert.ok(app.calls.some(r => r.endpoint === "/group/list"));
});

test("personal connection audit presents both remote participants", async () => {
  const app = createConsole();
  await app.settle();
  await app.settle();
  await app.open("myconnections");
  assert.ok(app.calls.some(r => r.endpoint === "/my/audit_conn/list"));
  assert.match(app.root.innerHTML, /123456/);
  assert.match(app.root.innerHTML, /654321/);
  assert.match(app.root.innerHTML, /another-user/);
  assert.match(app.root.innerHTML, /我发起的连接/);
  assert.ok(!app.calls.some(r => r.endpoint === "/audit_conn/list"));
});

test("personal group form includes anchor type", async () => {
  const app = createConsole();
  await app.settle();
  await app.settle();
  await app.open("mygroups");
  await app.action("create");
  assert.match(app.root.innerHTML, /公共锚点组/);
  assert.match(app.root.innerHTML, /新增我的群组|新建我的群组/);
});

test("control console returns to WebClient after SSO login",async()=>{
  const app=createConsole(false, {"rustdesk.console.return":"/webclient2/"});
  await app.settle();
  await app.settle();
  assert.deepEqual(app.assigned, ["/webclient2/"]);
});

test("deployment records are visible only to administrators", async () => {
  const regular = createConsole();
  await regular.settle();
  assert.doesNotMatch(regular.root.innerHTML, /设备部署记录/);
  const admin = createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML, /设备部署记录/);
});

test("administrators can manage user and device-group policies", async () => {
  const regular = createConsole();
  await regular.settle();
  await regular.settle();
  assert.doesNotMatch(regular.root.innerHTML, /用户策略分配/);
  const admin = createConsole(true);
  await admin.settle();
  await admin.settle();
  assert.match(admin.root.innerHTML, /用户策略分配/);
  assert.match(admin.root.innerHTML, /设备组策略分配/);
  await admin.open("userPolicyAssignments");
  assert.ok(admin.calls.some(call => call.endpoint === "/device_policy/user_assignment/list"));
  await admin.open("groupPolicyAssignments");
  assert.ok(admin.calls.some(call => call.endpoint === "/device_policy/group_assignment/list"));
});

test("personal and administrator session records use separate API routes",async()=>{
  const owner=createConsole(false);
  await owner.settle();await owner.settle();
  assert.match(owner.root.innerHTML,/我的活动远程连接/);
  assert.match(owner.root.innerHTML,/我的结束连接记录/);
  assert.match(owner.root.innerHTML,/我的登录会话/);
  assert.doesNotMatch(owner.root.innerHTML,/全部登录会话/);
  await owner.open("myactiveSessions");
  assert.ok(owner.calls.some(x=>x.endpoint==="/my/session/list"));
  assert.match(owner.root.innerHTML,/data-action="session-disconnect"/);
  await owner.open("myDisconnectHistory");
  assert.ok(owner.calls.some(x=>x.endpoint==="/my/session/disconnect/list"));
  await owner.open("mysessions");
  assert.match(owner.root.innerHTML,/结束登录/);
  assert.match(owner.root.innerHTML,/当前会话/);
  await owner.action("session-revoke",50);
  assert.ok(owner.calls.some(x=>x.endpoint==="/my/user_token/revoke"&&x.method==="POST"));
  const admin=createConsole(true);
  await admin.settle();await admin.settle();
  assert.match(admin.root.innerHTML,/全部登录会话/);
  await admin.open("activeSessions");
  assert.ok(admin.calls.some(x=>x.endpoint==="/session/list"));
  await admin.open("sessions");
  assert.match(admin.root.innerHTML,/登录账户/);
  assert.ok(admin.calls.some(x=>x.endpoint==="/user_token/list"));
});

test("API management tokens are only shown to administrators", async () => {
  const owner = createConsole();
  await owner.settle();
  assert.doesNotMatch(owner.root.innerHTML, /API 管理令牌/);
  const admin = createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML, /API 管理令牌/);
  await admin.open("apiTokens");
  assert.ok(admin.calls.some(call => call.endpoint === "/api_token/list"));
});

test("users can manage personal address book sharing including everyone",async()=>{
  const app=createConsole();
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML,/我的地址簿共享规则/);
  await app.open("myRules");
  assert.ok(app.calls.some(call=>call.endpoint==="/my/address_book_collection_rule/list"));
  await app.action("create");
  assert.match(app.root.innerHTML,/所有用户/);
  assert.match(app.root.innerHTML,/对象 ID（所有用户填 0）/);
});

test("address book CSV actions are accessible to personal users",async()=>{
  const app=createConsole();
  await app.settle();
  await app.open("addressbooks");
  assert.match(app.root.innerHTML,/导入 CSV/);
  assert.match(app.root.innerHTML,/导出 CSV/);
  await app.action("csv-import");
  assert.match(app.root.innerHTML,/选择 UTF-8 CSV 文件/);
  assert.match(app.root.innerHTML,/5000 条设备/);
});

test("personal users can open the address book trash page",async()=>{
  const app=createConsole();
  await app.settle();
  await app.open("addressbooks");
  assert.match(app.root.innerHTML,/查看回收站/);
  await app.open("trash");
  assert.ok(app.calls.some(x=>x.endpoint==="/my/address_book/trash/list"));
});

test("mail configuration is visible to administrators",async()=>{
  const user=createConsole();
  await user.settle();
  assert.doesNotMatch(user.root.innerHTML,/SMTP 邮件服务/);
  const admin=createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML,/SMTP 邮件服务/);
  await admin.open("mail");
  assert.ok(admin.calls.some(x=>x.endpoint==="/mail/settings"));
  assert.match(admin.root.innerHTML,/SMTP 主机/);
  assert.match(admin.root.innerHTML,/STARTTLS/);
});

test("login screen includes password recovery by email",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""});
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML,/忘记密码/);
  await app.action("forgot-password");
  assert.match(app.root.innerHTML,/账户邮箱/);
  assert.match(app.root.innerHTML,/发送恢复邮件/);
});

test("team invitation management is limited to administrators",async()=>{
  const regular=createConsole();
  await regular.settle();
  assert.doesNotMatch(regular.root.innerHTML,/成员邮件邀请/);
  const admin=createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML,/成员邮件邀请/);
  await admin.open("invitations");
  assert.ok(admin.calls.some(x=>x.endpoint==="/invite/list"));
  await admin.action("create");
  assert.match(admin.root.innerHTML,/成员邮箱/);
  assert.match(admin.root.innerHTML,/发送邀请/);
});

test("administrator can edit connection notes in the audit list",async()=>{
  const admin=createConsole(true);
  await admin.settle();
  await admin.open("connections");
  assert.ok(admin.calls.some(x=>x.endpoint==="/audit_conn/list"));
  assert.match(admin.root.innerHTML,/连接备注/);
  assert.match(admin.root.innerHTML,/编辑备注/);
});

test("users can see optional authenticator setup in personal settings",async()=>{
  const app=createConsole();
  await app.settle();
  await app.open("profile");
  assert.ok(app.calls.some(x=>x.endpoint==="/mfa/status"));
  assert.match(app.root.innerHTML,/双重认证（TOTP）/);
  assert.match(app.root.innerHTML,/启用 TOTP 双重认证/);
});

test("personal profile shows optional emailed verification",async()=>{
  const app=createConsole();
  await app.settle();
  await app.open("profile");
  assert.ok(app.calls.some(x=>x.endpoint==="/mfa/email/status"));
  assert.match(app.root.innerHTML,/邮箱登录验证/);
  assert.match(app.root.innerHTML,/发送邮箱验证码并启用/);
});

test("relay node settings provide multi-server and IP range controls",async()=>{
  const ordinary=createConsole();
  await ordinary.settle();
  assert.doesNotMatch(ordinary.root.innerHTML,/中继节点管理/);
  const administrator=createConsole(true);
  await administrator.settle();
  assert.match(administrator.root.innerHTML,/中继节点管理/);
  await administrator.open("relayNodes");
  assert.ok(administrator.calls.some(x=>x.endpoint==="/relay/list"));
  await administrator.action("create");
  assert.match(administrator.root.innerHTML,/中继服务器地址/);
  assert.match(administrator.root.innerHTML,/IP 网络/);
  assert.match(administrator.root.innerHTML,/国家代码/);
});

test("control roles expose all 14 native connection permissions",async()=>{
  const regular=createConsole();
  await regular.settle();
  assert.doesNotMatch(regular.root.innerHTML,/远程控制角色/);
  const admin=createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML,/远程控制角色/);
  await admin.open("controlRoles");
  assert.ok(admin.calls.some(call=>call.endpoint==="/control_role/list"));
  await admin.action("create");
  for(const label of ["键盘和鼠标","剪贴板","文件传输","远程终端",
    "远程打印机","音频","摄像头","TCP 隧道","远程重启",
    "会话录制","禁止远程设备本地输入","修改远程设置","隐私模式","USB 设备"]){
    assert.match(admin.root.innerHTML,new RegExp(label));
  }
  assert.match(admin.root.innerHTML,/沿用设备设置/);
  assert.match(admin.root.innerHTML,/禁止/);
  assert.match(admin.root.innerHTML,/允许/);
  await admin.open("controlAssignments");
  assert.ok(admin.calls.some(call=>call.endpoint==="/control_assignment/list"));
});

test("full administrators can create scoped roles with a permission checklist",async()=>{
  const app=createConsole(true);
  await app.settle();
  await app.open("adminRoles");
  assert.ok(app.calls.some(x=>x.endpoint==="/admin_role/list"));
  assert.match(app.root.innerHTML,/Read-only devices/);
  await app.action("create");
  assert.match(app.root.innerHTML,/管理权限/);
  assert.match(app.root.innerHTML,/查看设备/);
  assert.match(app.root.innerHTML,/修改设备信息/);
  assert.match(app.root.innerHTML,/指定用户组/);
  assert.match(app.root.innerHTML,/指定用户/);
  await app.open("adminRoleAssignments");
  assert.ok(app.calls.some(x=>x.endpoint==="/admin_role_assignment/list"));
});
test("delegated viewers see authorized directory pages without edit buttons",async()=>{
  const app=createConsole();
  app.fixtures["/user/current"].route_names=["users.view","devices.view"];
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML,/用户账户/);
  assert.match(app.root.innerHTML,/设备管理/);
  assert.doesNotMatch(app.root.innerHTML,/管理员角色分配/);
  await app.open("users");
  assert.ok(app.calls.some(x=>x.endpoint==="/user/list"));
  assert.doesNotMatch(app.root.innerHTML,/新增用户账户/);
  await app.open("devices");
  assert.ok(app.calls.some(x=>x.endpoint==="/peer/list"));
  assert.ok(app.calls.some(x=>x.endpoint==="/peer/list"));
});

test("access rules expose source and target selectors without changing default behavior",async()=>{
  const ordinary=createConsole();
  await ordinary.settle();
  assert.doesNotMatch(ordinary.root.innerHTML,/连接访问规则/);
  const admin=createConsole(true);
  await admin.settle();
  await admin.open("accessRules");
  assert.ok(admin.calls.some(x=>x.endpoint==="/access_rule/list"));
  assert.match(admin.root.innerHTML,/Office support/);
  await admin.action("create");
  for(const text of ["所有已登录用户","指定用户组","指定所有者的设备","指定设备组","允许访问","禁止访问"]){
    assert.match(admin.root.innerHTML,new RegExp(text));
  }
});

test("device transfer control respects delegated permission and displays target account form", async () => {
  const readonly = createConsole();
  readonly.fixtures["/user/current"].route_names = ["devices.view"];
  readonly.fixtures["/peer/list"] = {list: [{row_id: 15, id: "desk-15", user_id: 9, hostname: "Lab"}], total: 1};
  await readonly.settle();
  await readonly.open("devices");
  assert.doesNotMatch(readonly.root.innerHTML, /转移设备/);

  const delegated = createConsole();
  delegated.fixtures["/user/current"].route_names = ["devices.view", "devices.transfer"];
  delegated.fixtures["/peer/list"] = {list: [{row_id: 15, id: "desk-15", user_id: 9, hostname: "Lab"}], total: 1};
  await delegated.settle();
  await delegated.open("devices");
  assert.match(delegated.root.innerHTML, /转移设备/);
  await delegated.action("transfer-device", 15);
  assert.match(delegated.root.innerHTML, /新所有者的用户编号/);
  assert.match(delegated.root.innerHTML, /desk-15/);
});

test("personal address book recycle bin supports batch restoration and deletion", async () => {
  const app = createConsole();
  app.fixtures["/my/address_book/trash/list"] = {
    list: [{row_id: 14, id: "old-device", alias: "Office"}], total: 1
  };
  await app.settle();
  await app.open("trash");
  assert.match(app.root.innerHTML, /批量恢复/);
  assert.match(app.root.innerHTML, /批量永久删除/);
  await app.action("trash-batch-restore");
  assert.match(app.root.innerHTML, /选择回收站记录/);
  assert.match(app.root.innerHTML, /整批检查完成后才会恢复记录/);
  await app.action("close-modal");
  await app.action("trash-batch-purge");
  assert.match(app.root.innerHTML, /删除后无法恢复/);
});

test("personal contact manager exposes batch sharing controls", async () => {
  const app = createConsole();
  await app.settle();
  await app.open("contacts");
  assert.match(app.root.innerHTML, /批量添加联系人/);
  assert.match(app.root.innerHTML, /批量撤销共享/);
  await app.action("contact-batch-add");
  assert.match(app.root.innerHTML, /搜索并选择联系人/);
  assert.match(app.root.innerHTML, /每次最多处理 100 位用户/);
  await app.action("close-modal");
  await app.action("contact-batch-remove");
  assert.match(app.root.innerHTML, /选中联系人将无法再通过联系人关系查看您的设备/);
});

test("deployment manager exposes batch approval and revocation", async () => {
  const app = createConsole(true);
  app.fixtures["/device_deployment/list"] = {
    list: [{id: 8, device_id: "desk-8", status: "approved"}], total: 1
  };
  await app.settle();
  await app.open("deployed");
  assert.match(app.root.innerHTML, /批量批准部署/);
  assert.match(app.root.innerHTML, /批量撤销部署/);
  await app.action("deployment-batch-approve");
  assert.match(app.root.innerHTML, /选择需要修改的设备部署记录/);
  await app.action("close-modal");
  await app.action("deployment-batch-revoke");
  assert.match(app.root.innerHTML, /撤销后设备将无法在严格登记模式下重新登记/);
});

test("personal address book sharing supports multiple account operations", async () => {
  const app = createConsole();
  app.fixtures["/my/address_book_collection_rule/list"] = {list: [], total: 0};
  await app.settle();
  await app.open("myRules");
  assert.match(app.root.innerHTML, /批量共享地址簿/);
  assert.match(app.root.innerHTML, /批量撤销共享/);
  await app.action("rules-batch-share");
  for (const label of ["选择需要共享的地址簿", "搜索并选择目标用户", "只读", "可读写", "完全控制"]) {
    assert.match(app.root.innerHTML, new RegExp(label));
  }
  await app.action("close-modal");
  await app.action("rules-batch-revoke");
  assert.match(app.root.innerHTML, /只能修改您本人拥有的地址簿/);
});

test("device list exposes filtered CSV export for authorized users", async () => {
  const app=createConsole();
  app.fixtures["/user/current"].route_names=["devices.view"];
  app.fixtures["/peer/list"]={list:[{row_id:11,id:"managed-desk",hostname:"Managed"}],total:1};
  await app.settle();
  await app.open("devices");
  assert.match(app.root.innerHTML,/导出设备清单 CSV/);
  const privateUser=createConsole();
  await privateUser.settle();
  await privateUser.open("devices");
  assert.doesNotMatch(privateUser.root.innerHTML,/导出设备清单 CSV/);
});

test("audit CSV filters expose date, device and connection event options",async()=>{
  const app=createConsole(true);
  await app.settle();
  await app.open("connections");
  assert.match(app.root.innerHTML,/筛选并导出 CSV/);
  await app.action("audit-export","audit_conn");
  for(const label of ["开始日期","结束日期","目标设备 ID","来源设备 ID","连接事件"]){
    assert.match(app.root.innerHTML,new RegExp(label));
  }
});

test("notification settings and delivery history display per-event controls",async()=>{
  const ordinary=createConsole();
  await ordinary.settle();
  assert.doesNotMatch(ordinary.root.innerHTML,/邮件事件通知/);
  const admin=createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML,/邮件事件通知/);
  await admin.open("notifications");
  assert.ok(admin.calls.some(x=>x.endpoint==="/notification/settings"));
  assert.ok(admin.calls.some(x=>x.endpoint==="/notification/list"));
  for(const name of ["启用邮件事件通知","通知收件邮箱","登录失败","设备登记状态变更",
    "远程连接已结束","文件传输","发送记录","重新发送"]){
    assert.match(admin.root.innerHTML,new RegExp(name));
  }
});

test("device strategy form offers supported access and approval policies",async()=>{
  const app=createConsole(true);
  await app.settle();
  await app.open("policies");
  await app.action("create");
  for(const label of ["允许的远程连接方式","允许远程键盘和鼠标","允许远程剪贴板",
    "允许远程文件传输","远程连接确认方式","自定义权限","仅查看",
    "密码或手动确认","仅密码","仅手动确认"]){
    assert.match(app.root.innerHTML,new RegExp(label));
  }
});

test("visual client builder shows official Windows configurations, tasks and downloads",async()=>{
  const ordinary=createConsole();
  await ordinary.settle();
  assert.doesNotMatch(ordinary.root.innerHTML,/客户端安装包构建/);
  const admin=createConsole(true);
  await admin.settle();
  assert.match(admin.root.innerHTML,/客户端安装包构建/);
  await admin.open("clientBuilder");
  for(const label of ["客户端预设配置","Office installer","Windows x64","Windows ARM64",
    "Linux x64","Linux ARM64","macOS x64","macOS ARM64",
    "服务器公钥","RustDesk 官方版本","创建安装包任务","安装包已准备好","下载安装程序","重新处理"]){
    assert.match(admin.root.innerHTML,new RegExp(label));
  }
  assert.ok(admin.calls.some(x=>x.endpoint==="/client_builder/profiles"));
  assert.ok(admin.calls.some(x=>x.endpoint==="/client_builder/tasks"));
  assert.ok(admin.calls.some(x=>x.endpoint==="/client_builder/versions"));
  await admin.action("client-edit",2);
  assert.match(admin.root.innerHTML,/desk.example.com:21116/);
  await admin.action("client-new");
  assert.match(admin.root.innerHTML,/新建配置/);
});


test("personal users can configure push channels, HTTP JSON and SSE credentials",async()=>{
  const app=createConsole();
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML,/消息推送/);
  await app.open("push");
  assert.ok(app.calls.some(x=>x.endpoint==="/push/channels"));
  assert.ok(app.calls.some(x=>x.endpoint==="/push/events"));
  assert.ok(app.calls.some(x=>x.endpoint==="/push/history"));
  assert.ok(app.calls.some(x=>x.endpoint==="/push/stream-keys"));
  assert.match(app.root.innerHTML,/HTTP JSON/);
  assert.match(app.root.innerHTML,/SSE 实时订阅/);
  assert.match(app.root.innerHTML,/自动化订阅/);
  assert.match(app.root.innerHTML,/events.read/);
  assert.match(app.root.innerHTML,/MessagePusher/);
  assert.match(app.root.innerHTML,/SMTP 邮件/);
  assert.match(app.root.innerHTML,/Server酱 Turbo/);
  assert.match(app.root.innerHTML,/设备上线/);
  assert.match(app.root.innerHTML,/连接建立/);
  assert.match(app.root.innerHTML,/我的通知/);
  assert.match(app.root.innerHTML,/发送记录/);
  await app.action("push-edit",11);
  assert.match(app.root.innerHTML,/留空保持不变/);
  assert.doesNotMatch(app.root.innerHTML,/mpk_example-secret/);
  await app.action("push-new");
  assert.match(app.root.innerHTML,/添加推送渠道/);
});

test("login history and device IP addresses show parenthetical GeoIP locations",async()=>{
  const app=createConsole();
  app.fixtures["/my/login_log/list"]={list:[{id:5,client:1,ip:"10.0.0.5",created_at:1710000000}],total:1};
  app.fixtures["/my/peer/list"]={list:[{id:"office",row_id:6,hostname:"办公室电脑",last_online_ip:"127.0.0.1"}],total:1};
  await app.settle();
  await app.settle();
  await app.open("myloginlogs");
  assert.ok(app.calls.some(x=>x.endpoint==="/geoip/lookup"&&x.method==="POST"));
  assert.match(app.root.innerHTML,/10\.0\.0\.5/);
  assert.match(app.root.innerHTML,/（局域网地址）/);
  await app.open("devices");
  assert.match(app.root.innerHTML,/最近上线 IP/);
  assert.match(app.root.innerHTML,/127\.0\.0\.1/);
  assert.match(app.root.innerHTML,/（本地回环地址）/);
});


test("device presence uses inline dots in user and administrator views",async()=>{
  const user=createConsole();
  user.fixtures["/my/peer/list"]={list:[
    {row_id:1,id:"office",hostname:"办公室设备"},
    {row_id:2,id:"home-remote",hostname:"家庭设备"}
  ],total:2};
  await user.settle();
  await user.settle();
  await user.open("devices");
  assert.match(user.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(user.root.innerHTML,/device-status-dot offline[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">home-remote<\/span>/);
  assert.ok(user.calls.some(x=>x.endpoint==="/geoip/lookup"&&x.method==="POST"));
  assert.doesNotMatch(user.root.innerHTML,/<th[^>]*>在线状态<\/th>/);
  await user.open("myconnections");
  assert.match(user.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">123456<\/span>/);
  assert.match(user.root.innerHTML,/device-status-dot offline[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">654321<\/span>/);

  const admin=createConsole(true);
  admin.fixtures["/peer/list"]={list:[{row_id:3,id:"office",hostname:"管理员设备"}],total:1};
  await admin.settle();
  await admin.settle();
  await admin.open("devices");
  assert.match(admin.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">office<\/span>/);
  await admin.open("connections");
  assert.match(admin.root.innerHTML,/device-status-dot offline[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">managed-desk<\/span>/);
  assert.match(admin.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">helpdesk<\/span>/);
});

test("device status also appears in shared devices and saved address books",async()=>{
  const app=createConsole();
  app.fixtures["/shared_devices/list"]={list:[
    {row_id:6,id:"office",hostname:"办公室电脑",owner_username:"owner",sources:[]},
    {row_id:7,id:"home-remote",hostname:"家用电脑",owner_username:"owner",sources:[]}
  ],total:2};
  app.fixtures["/my/address_book/list"]={list:[
    {row_id:8,id:"office",alias:"办公室电脑"},
    {row_id:9,id:"home-remote",alias:"家庭设备"}
  ],total:2};
  await app.settle();
  await app.settle();
  await app.open("shared");
  assert.match(app.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(app.root.innerHTML,/device-status-dot offline[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">home-remote<\/span>/);
  await app.open("addressbooks");
  assert.match(app.root.innerHTML,/device-status-dot online[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(app.root.innerHTML,/device-status-dot offline[^<]*<\/span><span class="entity-link-text"><span class="entity-link-name">[^<]+<\/span><span class="entity-link-id">home-remote<\/span>/);
});

test("settings center lists effective sources without revealing secrets", async () => {
  const user = createConsole(false);
  await user.settle();await user.settle();
  assert.doesNotMatch(user.root.innerHTML, /设置中心<\/button>/);
  const admin = createConsole(true);
  await admin.settle();await admin.settle();
  await admin.open("settings");
  assert.match(admin.root.innerHTML, /设置中心/);
  assert.match(admin.root.innerHTML, /data-action="settings-category"/);
  await admin.events.click({target:{closest:()=>({dataset:{action:"settings-category",category:"常规与登录"}})}});
  assert.match(admin.root.innerHTML, /开放注册/);
  assert.match(admin.root.innerHTML, /RUSTDESK_API_APP_REGISTER/);
  assert.match(admin.root.innerHTML, /环境变量/);
  await admin.events.click({target:{closest:()=>({dataset:{action:"settings-category",category:"令牌"}})}});
  assert.match(admin.root.innerHTML, /已配置（隐藏具体内容）/);
  assert.ok(admin.calls.some(call=>call.endpoint==="/settings/effective"));
});

test("settings center uses categories for system, account and overview settings",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();await app.open("settings");
  assert.match(app.root.innerHTML,/data-action="settings-tab"/);
  assert.match(app.root.innerHTML,/data-action="settings-category"/);
  await app.events.click({target:{closest:()=>({dataset:{action:"settings-category",category:"令牌"}})}});
  assert.match(app.root.innerHTML,/更换通信密钥前/);
  assert.doesNotMatch(app.root.innerHTML,/切换数据库前/);
  const switchTab=async tab=>{
    await app.events.click({target:{closest:()=>({dataset:{action:"settings-tab",tab}})}});
  };
  await switchTab("storage");
  await app.events.click({target:{closest:()=>({dataset:{action:"settings-category",category:"数据库"}})}});
  assert.match(app.root.innerHTML,/切换数据库前/);
  await switchTab("overview");
  assert.match(app.root.innerHTML,/mail.example.test/);
  assert.match(app.root.innerHTML,/连接日志 90 天/);
});

test("separates owned devices and administrator device inventory",async()=>{
  const ordinary=createConsole(false);
  await ordinary.settle();await ordinary.settle();
  assert.match(ordinary.root.innerHTML,/data-view="mydevices"/);
  assert.doesNotMatch(ordinary.root.innerHTML,/data-view="devices"/);
  await ordinary.open("mydevices");
  assert.ok(ordinary.calls.some(x=>x.endpoint==="/my/peer/list"));
  const admin=createConsole(true);
  await admin.settle();await admin.settle();
  assert.match(admin.root.innerHTML,/data-view="mydevices"/);
  assert.match(admin.root.innerHTML,/data-view="devices"/);
  await admin.open("devices");
  assert.ok(admin.calls.some(x=>x.endpoint==="/peer/list"));
});

test("all data table columns have sorting controls and proper action gaps",async()=>{
  const app=createConsole(true);
  app.fixtures["/group/list"]={list:[{id:11,name:"Alpha",type:1},{id:4,name:"Zulu",type:2}],total:2};
  await app.settle();await app.settle();
  await app.open("groups");
  assert.match(app.root.innerHTML,/data-action="sort-table"/);
  assert.match(app.root.innerHTML,/aria-sort="none"/);
  assert.match(app.root.innerHTML,/data-action="toggle-section"/);
  assert.match(app.root.innerHTML,/data-action="edit"/);
});

test("server administration uses navigation and structured results",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();
  await app.open("server");
  assert.match(app.root.innerHTML,/服务器概况/);
  assert.match(app.root.innerHTML,/data-action="server-tab"/);
  assert.match(app.root.innerHTML,/desk.example.test/);
  await app.events.click({target:{closest:()=>({dataset:{action:"server-tab",tab:"id"}})}});
  assert.match(app.root.innerHTML,/中继服务器列表设置/);
  assert.match(app.root.innerHTML,/data-action="server-select"/);
  await app.events.click({target:{closest:()=>({dataset:{action:"server-tab",tab:"relay"}})}});
  assert.match(app.root.innerHTML,/查看中继使用情况/);
  assert.match(app.root.innerHTML,/server-result/);
  assert.ok(app.calls.some(call=>call.endpoint==="/config/server"));
  assert.ok(!app.calls.some(call=>call.endpoint==="/rustdesk/cmdList"));
});

test("reference names and modal choices replace isolated numeric IDs",async()=>{
  const app=createConsole(true);
  app.fixtures["/group/list"]={list:[{id:3,name:"研究人员组",type:1}],total:1};
  app.fixtures["/user/list"]={list:[{id:9,username:"learner",group_id:3}],total:1};
  await app.settle();await app.settle();
  await app.open("users");
  assert.match(app.root.innerHTML,/研究人员组（编号 3）/);
  await app.action("edit",9);
  assert.match(app.root.innerHTML,/data-action="picker-open"/);
  await app.open("oauth");
  await app.action("create");
  assert.match(app.root.innerHTML,/data-picker="user_group" data-target="auto_group_ids"/);
});

test("ARGB tag color is rendered with a browser color picker",async()=>{
  const app=createConsole(true);
  await app.open("tags");
  assert.match(app.root.innerHTML,/设备标签/);
  assert.match(app.root.innerHTML,/data-action="create"/);
});


test("address book sharing displays group names and explicit global targets",async()=>{
  const app=createConsole(true);
  app.fixtures["/address_book_collection_rule/list"]={list:[
    {id:1,collection_id:4,type:2,to_id:3,rule:1},
    {id:2,collection_id:4,type:3,to_id:0,rule:1}
  ],total:2};
  app.fixtures["/group/list"]={list:[{id:3,name:"开发协作组",type:1}],total:1};
  await app.settle();await app.settle();
  await app.open("rules");
  assert.match(app.root.innerHTML,/开发协作组（编号 3）/);
  assert.match(app.root.innerHTML,/所有用户/);
});


test("account profile uses all desktop space with Passkey management",async()=>{
  const app=createConsole(true);
  app.fixtures["/passkey/list"]={list:[{id:19,name:"Work laptop",created_at:"2026-11-12 09:00:00",last_used_at:1794477600}],total:1};
  await app.settle();
  await app.open("profile");
  assert.match(app.root.innerHTML,/profile-grid/);
  assert.match(app.root.innerHTML,/profile-summary/);
  assert.match(app.root.innerHTML,/Passkey 通行密钥/);
  assert.match(app.root.innerHTML,/Work laptop/);
  assert.match(app.root.innerHTML,/data-action="passkey-register"/);
  assert.doesNotMatch(app.root.innerHTML,/max-width:650px/);
});

test("SSO and LDAP accounts can set a local password without an existing password",async()=>{
  const external=createConsole(false);
  external.fixtures["/user/current"].has_local_password=false;
  await external.settle();await external.settle();await external.open("profile");
  assert.match(external.root.innerHTML,/设置备用登录密码/);
  assert.match(external.root.innerHTML,/SSO 或 LDAP 暂时不可用/);
  assert.match(external.root.innerHTML,/name="new_password"/);
  assert.doesNotMatch(external.root.innerHTML,/name="old_password"/);
  assert.match(external.root.innerHTML,/设置本地密码/);

  const local=createConsole(false);
  local.fixtures["/user/current"].has_local_password=true;
  await local.settle();await local.settle();await local.open("profile");
  assert.match(local.root.innerHTML,/修改登录密码/);
  assert.match(local.root.innerHTML,/name="old_password"[^>]*required/);
  assert.doesNotMatch(local.root.innerHTML,/设置备用登录密码/);
});

test("administrator account security actions are inside the user editor",async()=>{
  const app=createConsole(true);
  app.fixtures["/user/list"]={list:[{id:3,username:"owner",nickname:"Owner",status:1,group_id:1}],total:1};
  await app.settle();await app.open("users");
  assert.doesNotMatch(app.root.innerHTML,/data-action="reset-password"/);
  assert.doesNotMatch(app.root.innerHTML,/data-action="reset-user-mfa"/);
  await app.action("edit",3);
  assert.match(app.root.innerHTML,/data-action="reset-password"/);
  assert.match(app.root.innerHTML,/data-action="reset-user-mfa"/);
  await app.action("reset-password",3);
  assert.match(app.root.innerHTML,/重设用户密码/);
  assert.match(app.root.innerHTML,/minlength="8"/);
  const personal=createConsole(false);
  await personal.settle();
  assert.doesNotMatch(personal.root.innerHTML,/重设密码/);
  assert.doesNotMatch(personal.root.innerHTML,/清除双重认证/);
});

test("device and user references display link controls with status markers",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{row_id:1,id:"office",hostname:"Office",alias:"Desk",user_id:7}],total:1};
  await app.settle();await app.settle();
  await app.open("devices");
  assert.match(app.root.innerHTML,/data-action="entity-detail" data-entity-type="device" data-entity-id="office"/);
  assert.match(app.root.innerHTML,/device-status-dot online/);
  assert.match(app.root.innerHTML,/data-entity-label="Desk（Office）"/);
  app.fixtures["/user/list"]={list:[{id:7,username:"alice",nickname:"Alice",email:"alice@example.test"}],total:1};
  await app.open("users");
  assert.match(app.root.innerHTML,/data-action="entity-detail" data-entity-type="user" data-entity-id="7"/);
});

test("device presence keeps unknown status distinct from offline status",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{id:"unknown-id",row_id:12,hostname:"Unknown"}],total:1};
  await app.settle();await app.settle();
  await app.open("devices");
  assert.match(app.root.innerHTML,/device-status-dot unknown/);
});

test("owned-device actions and detail panel open WebClient v2 with the selected identity",async()=>{
  const app=createConsole(false);
  app.fixtures["/my/peer/list"]={list:[{id:"office-01",hostname:"Workstation",alias:"Office desk",user_id:2}],total:1};
  await app.settle();await app.settle();await app.open("mydevices");
  assert.match(app.root.innerHTML,/data-action="device-connect" data-device-id="office-01" data-device-name="Office desk"/);
  const fire=async dataset=>app.events.click({target:{closest:()=>({dataset})}});
  await fire({action:"device-connect",deviceId:"office-01",deviceName:"Office desk"});
  assert.equal(app.opened.length,1);
  assert.equal(app.opened[0].target,"_blank");
  assert.equal(app.opened[0].tab.opener,null);
  const first=new URL(app.opened[0].url,"https://example.test");
  assert.equal(first.pathname,"/webclient2/");
  assert.equal(first.searchParams.get("id"),"office-01");
  assert.equal(first.searchParams.get("name"),"Office desk");
  await fire({action:"entity-detail",entityType:"device",entityId:"office-01",entityLabel:"Office desk"});
  await app.settle();
  assert.match(app.root.innerHTML,/data-action="entity-open-list">查看设备<\/button>/);
  assert.match(app.root.innerHTML,/data-action="device-connect" data-device-id="office-01"/);
  await fire({action:"device-connect",deviceId:"office-01",deviceName:"Office desk"});
  assert.equal(app.opened.length,2);
});

test("log retention page exposes terminal-history categories and a confirmed cleanup flow",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();await app.open("retention");
  assert.match(app.root.innerHTML,/结束连接记录/);
  assert.match(app.root.innerHTML,/设置修改历史/);
  assert.match(app.root.innerHTML,/事件处理失败/);
  assert.match(app.root.innerHTML,/服务器运行日志归档文件/);
  assert.match(app.root.innerHTML,/data-action="log-cleanup-preview"/);
  assert.match(app.root.innerHTML,/data-action="log-cleanup-run" disabled/);
  const original=app.document.getElementById.bind(app.document);
  app.document.getElementById=id=>({
    "cleanup-kind":{value:"connection"},
    "cleanup-mode":{value:"retention"},
    "cleanup-days":{value:"30"}
  }[id]||original(id));
  await app.action("log-cleanup-preview");
  assert.ok(app.calls.some(x=>x.endpoint==="/audit/cleanup/preview"&&x.method==="GET"));
  assert.match(app.root.innerHTML,/预计清理 2 项历史数据/);
  await app.action("log-cleanup-run");
  assert.ok(app.calls.some(x=>x.endpoint==="/audit/cleanup"&&x.method==="POST"));
});
test("ordinary users cannot open the log cleanup administration page",async()=>{
  const app=createConsole(false);
  await app.settle();await app.settle();
  assert.doesNotMatch(app.root.innerHTML,/data-view="retention"/);
});

test("list paging defaults to 100 and uses numbered page navigation",async()=>{
  const app=createConsole(true);
  app.fixtures["/user/list"]={list:[{id:1,username:"admin"}],total:205};
  await app.settle();await app.settle();await app.open("users");
  assert.match(app.root.innerHTML,/第 1 \/ 3 页/);
  assert.match(app.root.innerHTML,/data-page="3"/);
  assert.match(app.root.innerHTML,/value="100" selected/);
  assert.ok(app.calls.some(call=>call.endpoint==="/user/list"&&call.query.includes("page_size=100")));
  await app.events.click({target:{closest:()=>({dataset:{action:"page-index",page:"3",scope:""}})}});
  assert.match(app.root.innerHTML,/第 3 \/ 3 页/);
  assert.ok(app.calls.some(call=>call.endpoint==="/user/list"&&call.query.includes("page=3")&&call.query.includes("page_size=100")));
});

test("server read-only commands execute when selected",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();await app.open("server");
  const before=app.calls.filter(call=>call.endpoint==="/rustdesk/execute").length;
  await app.events.click({target:{closest:()=>({dataset:{action:"server-tab",tab:"id"}})}});
  const after=app.calls.filter(call=>call.endpoint==="/rustdesk/execute").length;
  assert.equal(after,before+1);
  assert.match(app.root.innerHTML,/客户端登录要求设置/);
  assert.doesNotMatch(app.root.innerHTML,/查询客户端登录要求<\/strong>/);
  assert.doesNotMatch(app.root.innerHTML,/启用客户端登录要求<\/strong>/);
});

test("clicking the empty overlay closes the user editor",async()=>{
  const app=createConsole(true);
  app.fixtures["/user/list"]={list:[{id:3,username:"owner",status:1}],total:1};
  await app.settle();await app.open("users");
  await app.action("edit",3);
  assert.match(app.root.innerHTML,/id="modal-overlay"/);
  await app.events.click({target:{id:"modal-overlay",classList:{contains:()=>false},closest:()=>null}});
  assert.doesNotMatch(app.root.innerHTML,/id="modal-overlay"/);
});

test("connection direction uses icons with accessible labels",async()=>{
  const app=createConsole(false);
  await app.settle();await app.settle();await app.open("myconnections");
  assert.match(app.root.innerHTML,/class="connection-direction"/);
  assert.match(app.root.innerHTML,/aria-label="我发起的连接"/);
  assert.doesNotMatch(app.root.innerHTML,/data-column="direction"[^>]*>[^<]*我发起的连接/);
});

test("admin user details do not duplicate editing and security actions",async()=>{
  const app=createConsole(true);
  app.fixtures["/user/list"]={list:[{id:17,username:"helper",status:1,group_id:1}],total:1};
  await app.settle();await app.open("users");
  await app.events.click({target:{closest:()=>({dataset:{action:"entity-detail",entityType:"user",entityId:"17",entityLabel:"helper"}})}});
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/data-action="entity-admin-edit"/);
  assert.doesNotMatch(app.root.innerHTML,/data-action="entity-open-list"/);
  assert.doesNotMatch(app.root.innerHTML,/data-action="reset-password"/);
  assert.doesNotMatch(app.root.innerHTML,/data-action="reset-user-mfa"/);
});


test("desktop connection tables stack names and identifiers without changing the mobile list",async()=>{
  const app=createConsole(false);
  app.fixtures["/my/audit_conn/list"]={list:[{
    id:21,direction:"incoming",from_peer:"123456789",from_device_name:"工作站 A",
    peer_id:"987654321",to_device_name:"家庭电脑",from_account:"operator",
    to_account:"owner",created_at:1794530000
  }],total:1};
  await app.settle();await app.settle();await app.open("myconnections");
  assert.match(app.root.innerHTML,/data-column="from_peer"[^>]*class="[^"]*cell-device-stack/);
  assert.match(app.root.innerHTML,/class="device-reference-name">工作站 A<\/span>/);
  assert.match(app.root.innerHTML,/class="device-reference-id">123456789<\/span>/);
  assert.match(app.root.innerHTML,/class="device-reference-id">987654321<\/span>/);
  assert.doesNotMatch(app.root.innerHTML,/data-column="from_device_name"/);
  assert.match(app.root.innerHTML,/class="mobile-list"/);
  assert.match(app.root.innerHTML,/工作站 A/);
  assert.match(app.root.innerHTML,/data-direction="incoming"/);
});

test("user dialog includes linked group and device details with permission-scoped data",async()=>{
  const app=createConsole(true);
  app.fixtures["/user/list"]={list:[{id:17,username:"helper",nickname:"支持人员",status:1,group_id:1}],total:1};
  app.fixtures["/user/detail/17"]={id:17,username:"helper",nickname:"支持人员",email:"helper@example.test",group_id:1,is_admin:false,status:1};
  app.fixtures["/user/relations/17"]={
    groups:[{id:1,name:"默认组",type:1},{id:3,name:"技术支持",type:2}],
    devices:[{id:"desk101",alias:"工作电脑",hostname:"desk-a",os:"Windows"}],device_total:1
  };
  app.fixtures["/group/detail/3"]={id:3,name:"技术支持",type:2,owner_user_id:17};
  app.fixtures["/group/members"]={user_ids:[17],users:[{id:17,username:"helper",nickname:"支持人员"}]};
  await app.settle();await app.settle();await app.open("users");
  await app.events.click({target:{closest:()=>({dataset:{action:"entity-detail",entityType:"user",entityId:"17",entityLabel:"helper"}})}});
  await app.settle();
  assert.match(app.root.innerHTML,/所属用户组/);
  assert.match(app.root.innerHTML,/名下设备/);
  assert.match(app.root.innerHTML,/技术支持/);
  assert.match(app.root.innerHTML,/工作电脑/);
  assert.match(app.root.innerHTML,/data-entity-type="group" data-entity-id="3"/);
  assert.match(app.root.innerHTML,/data-entity-type="device" data-entity-id="desk101"/);
  await app.events.click({target:{closest:()=>({dataset:{action:"entity-detail",entityType:"group",entityId:"3",entityLabel:"技术支持"}})}});
  assert.match(app.root.innerHTML,/分组成员/);
  assert.match(app.root.innerHTML,/data-action="entity-back">返回/);
  await app.events.click({target:{closest:()=>({dataset:{action:"entity-back"}})}});
  assert.match(app.root.innerHTML,/名下设备/);
});

test("secondary server categories switch the command menu and preserve server selection",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();await app.open("server");
  await app.events.click({target:{closest:()=>({dataset:{action:"server-tab",tab:"id"}})}});
  assert.match(app.root.innerHTML,/server-categories/);
  assert.match(app.root.innerHTML,/data-action="server-category"/);
  await app.events.click({target:{closest:()=>({dataset:{action:"server-category",category:"访问与网络记录"}})}});
  assert.match(app.root.innerHTML,/aria-selected="true"/);
  assert.match(app.root.innerHTML,/data-action="server-select"/);
});


test("registered devices render unique names and identifiers in two lines",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[
    {row_id:1,id:"office",alias:"Office",hostname:"Office",user_id:2},
    {row_id:2,id:"home-remote",alias:"",hostname:"Home PC",user_id:2},
    {row_id:3,id:"unknown-id",alias:"",hostname:"",user_id:2}
  ],total:3};
  await app.settle();await app.settle();await app.open("devices");
  assert.match(app.root.innerHTML,/class="entity-link-name">Office<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(app.root.innerHTML,/class="entity-link-name">Home PC<\/span><span class="entity-link-id">home-remote<\/span>/);
  assert.match(app.root.innerHTML,/class="entity-link-name">未设置设备名称<\/span><span class="entity-link-id">unknown-id<\/span>/);
  assert.doesNotMatch(app.root.innerHTML,/class="entity-link-name">Office<\/span><span class="entity-link-id">Office<\/span>/);
});

test("browser sessions and virtual connection origins have no device link or presence dot",async()=>{
  const app=createConsole(false);
  app.fixtures["/my/user_token/list"]={list:[
    {id:98,device_id:"webadmin",client:"webadmin",device_uuid:"",active:true},
    {id:97,device_id:"windows-Chrome",client:"windows-Chrome",device_uuid:"",active:true}
  ],total:2};
  app.fixtures["/my/audit_conn/list"]={list:[
    {id:26,from_peer:"web",peer_id:"office",to_device_name:"Office",direction:"outgoing"}
  ],total:1};
  await app.settle();await app.settle();await app.open("mysessions");
  assert.doesNotMatch(app.root.innerHTML,/data-entity-id="webadmin"|data-entity-id="windows-Chrome"/);
  assert.match(app.root.innerHTML,/class="device-reference-id">webadmin<\/span>/);
  assert.match(app.root.innerHTML,/class="device-reference-id">windows-Chrome<\/span>/);
  await app.open("myconnections");
  assert.match(app.root.innerHTML,/class="device-reference-name">浏览器客户端<\/span><span class="device-reference-id">web<\/span>/);
  assert.doesNotMatch(app.root.innerHTML,/data-entity-id="web"/);
  assert.match(app.root.innerHTML,/data-entity-id="office"/);
});

test("desktop table headings expose accessible resizing and automatic width controls",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{row_id:4,id:"office",alias:"Office"}],total:1};
  await app.settle();await app.settle();await app.open("devices");
  assert.match(app.root.innerHTML,/class="column-resize-handle" data-column="id" role="separator"/);
  assert.match(app.root.innerHTML,/data-action="table-autofit"/);
  assert.match(app.root.innerHTML,/<col data-column="id" style="width:\d+px">/);
});

test("login sessions link registered RustDesk devices and leave browser clients as text",async()=>{
  const app=createConsole(true);
  app.fixtures["/user_token/list"]={list:[
    {id:51,device_id:"office",device_uuid:"owned-uuid",client:"rustdesk",active:true},
    {id:52,device_id:"webadmin",device_uuid:"",client:"webadmin",active:true},
    {id:53,device_id:"home-remote",device_uuid:"shared-uuid",client:"rustdesk",active:true}
  ],total:3};
  await app.settle();await app.settle();await app.open("sessions");
  assert.match(app.root.innerHTML,/data-entity-type="device" data-entity-id="office"/);
  assert.match(app.root.innerHTML,/data-entity-type="device" data-entity-id="home-remote"/);
  assert.match(app.root.innerHTML,/class="entity-link-name">Office<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(app.root.innerHTML,/device-status-dot online/);
  assert.match(app.root.innerHTML,/device-status-dot offline/);
  assert.doesNotMatch(app.root.innerHTML,/data-entity-type="device" data-entity-id="webadmin"/);
  assert.match(app.root.innerHTML,/class="device-reference-name">浏览器客户端<\/span><span class="device-reference-id">webadmin<\/span>/);
});

test("recent device overview rows keep names and IDs in separate elements",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{id:"office",alias:"Office",hostname:"Office"}],total:1};
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/class="status-row recent-device-row"/);
  assert.match(app.root.innerHTML,/class="recent-device-identity"/);
  assert.match(app.root.innerHTML,/class="entity-link-name">[^<]+<\/span><span class="entity-link-id">office<\/span>/);
  assert.match(app.root.innerHTML,/class="pill muted recent-device-time"/);
});


test("new client build profiles inherit the currently configured RustDesk server",async()=>{
  const app=createConsole(true);
  app.fixtures["/config/server"]={
    id_server:"internal.example.test:21116",
    key:"base64-server-key",
    api_server:"https://management.example.test",
    relay_server:"relay.example.test:21117"
  };
  await app.settle();await app.settle();
  await app.open("clientBuilder");
  assert.match(app.root.innerHTML,/name="host" value="internal.example.test:21116"/);
  assert.match(app.root.innerHTML,/name="public_key" value="base64-server-key"/);
  assert.match(app.root.innerHTML,/name="api_server" value="https:\/\/management.example.test"/);
  assert.match(app.root.innerHTML,/name="relay_server" value="relay.example.test:21117"/);
  assert.match(app.root.innerHTML,/新建配置时自动填写本 API 服务器/);
  assert.ok(app.calls.some(call=>call.endpoint==="/config/server"));
});


test("all WebClient navigation commands open a separate browser tab",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();
  await app.action("webclient");
  assert.equal(app.opened.length,1);
  assert.equal(app.opened[0].url,"/webclient2/");
  assert.equal(app.opened[0].target,"_blank");
  assert.equal(app.opened[0].tab.opener,null);
  assert.deepEqual(app.assigned,[]);
  await app.open("shared");
  assert.match(app.root.innerHTML,/data-action="webclient"/);
  await app.action("webclient");
  assert.equal(app.opened.length,2);
  assert.equal(app.opened[1].target,"_blank");
});

test("editable OAuth provider display name is shown while the GitHub login identifier stays stable",async()=>{
  const app=createConsole(true);
  app.fixtures["/oauth/list"]={list:[{
    id:42,op:"github",display_name:"企业 GitHub 单点登录",oauth_type:"github",
    client_id:"test-client",client_secret:"",auto_group_ids:"",issuer:"",
    scopes:"",auto_register:false,pkce_enable:false,pkce_method:"S256"
  }],total:1};
  await app.settle();await app.settle();await app.open("oauth");
  assert.match(app.root.innerHTML,/企业 GitHub 单点登录/);
  await app.action("edit",42);
  assert.match(app.root.innerHTML,/name="display_name" type="text" value="企业 GitHub 单点登录"/);
  assert.match(app.root.innerHTML,/name="op" type="text" value="github" readonly/);
  assert.match(app.root.innerHTML,/登录按钮名称可以修改/);
});

test("console session survives a new tab and a temporary API restart", async () => {
  const prior={ "rustdesk.console.token": "persistent-login" };
  const app=createConsole(true,{"rustdesk.console.token":""},prior,{failCurrent:true});
  await app.settle();await app.settle();
  assert.equal(app.local.getItem("rustdesk.console.token"),"persistent-login");
  assert.match(app.root.innerHTML,/正在等待 API 服务恢复/);
  app.behavior.failCurrent=false;
  await app.action("retry-session");
  assert.match(app.root.innerHTML,/控制台概览/);
  assert.equal(app.local.getItem("rustdesk.console.token"),"persistent-login");
  assert.equal(app.session.getItem("rustdesk.console.token"),"persistent-login");

  const reopened=createConsole(true,{"rustdesk.console.token":""},prior);
  await reopened.settle();await reopened.settle();
  assert.match(reopened.root.innerHTML,/控制台概览/);
  assert.equal(reopened.local.getItem("rustdesk.console.token"),"persistent-login");
});

test("console clears persisted credentials when the server rejects a session",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},
    {"rustdesk.console.token":"revoked-session"},{rejectCurrent:true});
  await app.settle();await app.settle();
  assert.equal(app.local.getItem("rustdesk.console.token"),null);
  assert.match(app.root.innerHTML,/登录控制台/);
});

test("device ID joins alias and hostname while dates have two lines",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{
    row_id:44,id:"1184774058",alias:"pc408",hostname:"desktop-i4ferju",
    user_id:5,last_online_time:1794624000,os:"Windows"
  }],total:1};
  await app.settle();
  await app.open("devices");
  const html=app.root.innerHTML;
  assert.match(html,/pc408（desktop-i4ferju）/);
  assert.match(html,/class="entity-link-id">1184774058<\/span>/);
  assert.doesNotMatch(html,/data-column="alias"/);
  assert.doesNotMatch(html,/data-column="hostname"/);
  assert.match(html,/date-time-stack/);
  assert.match(html,/date-time-clock/);
  assert.match(html,/data-column="last_online_time"[^>]*width:1(?:3|4|5|6)\dpx/);
});
test("transfer selection starts empty with existing owner shown separately",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{
    row_id:15,id:"1184774058",alias:"pc408",hostname:"desktop-i4ferju",user_id:5
  }],total:1};
  await app.settle();
  await app.open("devices");
  await app.action("transfer-device",15);
  assert.match(app.root.innerHTML,/当前所有者：/);
  assert.match(app.root.innerHTML,/id="transfer-user-id" value=""/);
  assert.doesNotMatch(app.root.innerHTML,/picker-trigger-label">编号 5/);
});

test("WebClient2 return intent survives opening the login page in a separate tab",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/login?return=webclient2"
  });
  await app.settle();await app.settle();
  assert.equal(app.session.getItem("rustdesk.console.return"),"/webclient2/");
  assert.match(app.root.innerHTML,/登录控制台/);
});

test("SSO return intent from the login URL is limited to WebClient2",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/login?return=https%3A%2F%2Fexample.invalid"
  });
  await app.settle();await app.settle();
  assert.equal(app.session.getItem("rustdesk.console.return"),null);
});

test("mobile OIDC callback restores session after losing the original tab's storage",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=MobileSSOState12345",
    oidcResult:{token:"mobile-sso-token"}
  });
  await app.settle();await app.settle();await app.settle();
  assert.ok(app.calls.some(x=>x.endpoint==="/oidc/auth-query"&&x.query==="?code=MobileSSOState12345"));
  assert.equal(app.local.getItem("rustdesk.console.token"),"mobile-sso-token");
  assert.equal(app.session.getItem("rustdesk.console.token"),"mobile-sso-token");
  assert.equal(app.session.getItem("rustdesk.console.oidc"),null);
  assert.deepEqual(app.replaced,["/_admin/#/overview"]);
  assert.match(app.root.innerHTML,/我的设备/);
  assert.doesNotMatch(app.root.innerHTML,/登录控制台/);
});

test("mobile OIDC callback takes precedence over a stale authorization state",async()=>{
  const app=createConsole(false,{
    "rustdesk.console.token":"","rustdesk.console.oidc":"stale-oidc-state"
  },{},{
    initialHash:"#/oidc/complete?code=NewCallbackState1234"
  });
  await app.settle();await app.settle();
  const codes=app.calls.filter(x=>x.endpoint==="/oidc/auth-query").map(x=>x.query);
  assert.deepEqual(codes,["?code=NewCallbackState1234"]);
  assert.equal(app.session.getItem("rustdesk.console.oidc"),null);
  assert.equal(app.local.getItem("rustdesk.console.token"),"mobile-sso-token");
});

test("mobile SSO callback returns to WebClient2 with an approved return target",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=MobileReturnState123&return=webclient2"
  });
  await app.settle();await app.settle();
  assert.deepEqual(app.assigned,["/webclient2/"]);
  assert.equal(app.local.getItem("rustdesk.console.token"),"mobile-sso-token");
  assert.equal(app.session.getItem("rustdesk.console.return"),null);
  assert.deepEqual(app.replaced,["/_admin/#/overview"]);
});

test("invalid and expired mobile SSO callbacks show an error instead of a silent login loop",async()=>{
  const expired=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=ExpiredState12345",
    oidcError:"授权状态已过期"
  });
  await expired.settle();await expired.settle();
  assert.match(expired.root.innerHTML,/单点登录未能完成：授权状态已过期/);
  assert.equal(expired.session.getItem("rustdesk.console.oidc"),null);
  assert.deepEqual(expired.replaced,["/_admin/#/overview"]);

  const invalid=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=bad%24code"
  });
  await invalid.settle();await invalid.settle();
  assert.match(invalid.root.innerHTML,/单点登录回调缺少有效的授权状态/);
  assert.ok(!invalid.calls.some(x=>x.endpoint==="/oidc/auth-query"));
});

test("temporary mobile OAuth query failure leaves a retry action",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=RetryState1234567",oidcNetworkError:true
  });
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/单点登录未能完成：Network unavailable/);
  assert.match(app.root.innerHTML,/data-action="oidc-retry"/);
  assert.equal(app.session.getItem("rustdesk.console.oidc"),"RetryState1234567");
  assert.deepEqual(app.replaced,["/_admin/#/overview"]);
});

test("mobile SSO login updates the original tab when another tab stores its token",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""});
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/登录控制台/);
  app.local.setItem("rustdesk.console.token","shared-mobile-token");
  app.windowEvents.storage({key:"rustdesk.console.token",newValue:"shared-mobile-token"});
  await app.settle();await app.settle();await app.settle();
  assert.equal(app.session.getItem("rustdesk.console.token"),"shared-mobile-token");
  assert.match(app.root.innerHTML,/我的设备/);
  assert.doesNotMatch(app.root.innerHTML,/登录控制台/);
});

test("a resumed mobile login does not retry an outdated OAuth state",async()=>{
  const app=createConsole(false,{
    "rustdesk.console.token":"",
    "rustdesk.console.oidc":"prior-oauth-state-123"
  },{},{
    oidcNetworkError:true
  });
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/单点登录未能完成/);
  app.local.setItem("rustdesk.console.token","shared-mobile-token");
  app.windowEvents.pageshow();
  await app.settle();await app.settle();
  assert.equal(app.session.getItem("rustdesk.console.oidc"),null);
  assert.equal(app.session.getItem("rustdesk.console.token"),"shared-mobile-token");
  assert.match(app.root.innerHTML,/我的设备/);
});

test("logging out from a second tab clears the original tab's session",async()=>{
  const app=createConsole(false);
  await app.settle();await app.settle();
  assert.match(app.root.innerHTML,/我的设备/);
  app.local.removeItem("rustdesk.console.token");
  app.windowEvents.storage({key:"rustdesk.console.token",newValue:null});
  await app.settle();
  assert.equal(app.session.getItem("rustdesk.console.token"),null);
  assert.match(app.root.innerHTML,/登录控制台/);
});

test("shared device rows expose the WebClient connection button",async()=>{
  const app=createConsole(false);
  app.fixtures["/shared_devices/list"]={list:[{
    id:"shared-123456789",alias:"研发设备",hostname:"workstation-01",
    owner_username:"alice",sources:[{type:"group",name:"研发组"}]
  }],total:1};
  await app.settle();await app.settle();
  await app.open("shared");
  assert.match(app.root.innerHTML,/data-action="device-connect" data-device-id="shared-123456789"/);
  assert.match(app.root.innerHTML,/data-device-name="研发设备"/);
  await app.events.click({target:{closest(){return {dataset:{
    action:"device-connect",deviceId:"shared-123456789",deviceName:"研发设备"
  }};}}});
  assert.match(app.opened[0].url,/^\/webclient2\/\?id=shared-123456789/);
  assert.equal(app.opened[0].target,"_blank");
});

test("WebClient device return preserves the intended device without exposing arbitrary destinations",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":"example-session"},{},{
    initialHash:"#/login?return=webclient2&id=464956471&name=meili"
  });
  await app.settle();await app.settle();
  assert.deepEqual(app.assigned,["/webclient2/?id=464956471&name=meili"]);
  assert.equal(app.session.getItem("rustdesk.console.return"),null);
});
test("malformed WebClient return target is ignored during SSO",async()=>{
  const app=createConsole(false,{
    "rustdesk.console.token":"example-session",
    "rustdesk.console.return":"//outside.example/steal"
  });
  await app.settle();await app.settle();
  assert.deepEqual(app.assigned,[]);
  assert.match(app.root.innerHTML,/我的设备/);
});
test("OIDC callback carries approved device metadata between mobile tabs",async()=>{
  const app=createConsole(false,{"rustdesk.console.token":""},{},{
    initialHash:"#/oidc/complete?code=MobileState12345&return=webclient2&id=464956471&name=meili"
  });
  await app.settle();await app.settle();
  assert.deepEqual(app.assigned,["/webclient2/?id=464956471&name=meili"]);
  assert.equal(app.local.getItem("rustdesk.console.token"),"mobile-sso-token");
});
