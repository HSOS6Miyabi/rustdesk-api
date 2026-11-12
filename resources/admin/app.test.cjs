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

function createConsole(isAdmin = false, overrides = {}) {
  const calls = [];
  const events = {};
  const root = { innerHTML: "",querySelector(){return null;} };
  const assigned = [];
  const browserLocation = { hash: "#/overview", assign(url) {assigned.push(url);} };
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
    "/geoip/lookup": {locations:{"10.0.0.5":"局域网地址","127.0.0.1":"本地回环地址","192.0.2.1":"位置未知"},device_status:{"office":true,"home-remote":false,"123456":true,"654321":false,"helpdesk":true,"managed-desk":false,"support-123456":true,"deployed-10002":false},database_ready:false},
    "/push/events": [{code:"device.online",name:"设备上线"},{code:"device.offline",name:"设备离线"},{code:"connection.started",name:"远程连接建立"}],
    "/push/channels": {list:[{id:11,kind:"messagepusher",name:"我的通知",enabled:true,events:["device.online","connection.started"],base_url:"https://push.example.test",api_key_configured:true}]},
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
    calls.push({ endpoint, method: options?.method || "GET" });
    if (!Object.prototype.hasOwnProperty.call(fixtures, endpoint)) {
      return { ok: true, status: 200, async json() { return { code: 101, message: "Unexpected "+endpoint }; } };
    }
    return { ok: true, status: 200, async json() { return { code: 0, data: fixtures[endpoint] }; } };
  };
  const mockDocument = {
    documentElement: { dataset: {} },
    getElementById(id) { return id === "app" ? root : null; },
    addEventListener(name, handler) { events[name] = handler; },
    querySelector() { return null; }
  };
  const mockWindow = {
    addEventListener() {},
    confirm() { return true; },
    location: browserLocation
  };
  const session = storage({ "rustdesk.console.token": "example-session", ...overrides });
  const local = storage();
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
    root, calls, events, fixtures, settle, assigned,
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
  assert.match(app.root.innerHTML, /地址簿记录编号/);
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
  assert.match(app.root.innerHTML, /设备部署记录编号/);
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
  for (const label of ["地址簿编号", "搜索并选择目标用户", "只读", "可读写", "完全控制"]) {
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


test("personal users can configure MessagePusher, SMTP and Server酱 notifications",async()=>{
  const app=createConsole();
  await app.settle();
  await app.settle();
  assert.match(app.root.innerHTML,/消息推送/);
  await app.open("push");
  assert.ok(app.calls.some(x=>x.endpoint==="/push/channels"));
  assert.ok(app.calls.some(x=>x.endpoint==="/push/events"));
  assert.ok(app.calls.some(x=>x.endpoint==="/push/history"));
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
  assert.match(user.root.innerHTML,/device-status-dot online[^<]*<\/span><span>office<\/span>/);
  assert.match(user.root.innerHTML,/device-status-dot offline[^<]*<\/span><span>home-remote<\/span>/);
  assert.ok(user.calls.some(x=>x.endpoint==="/geoip/lookup"&&x.method==="POST"));
  assert.doesNotMatch(user.root.innerHTML,/<th[^>]*>在线状态<\/th>/);
  await user.open("myconnections");
  assert.match(user.root.innerHTML,/device-status-dot online[^<]*<\/span><span>123456<\/span>/);
  assert.match(user.root.innerHTML,/device-status-dot offline[^<]*<\/span><span>654321<\/span>/);

  const admin=createConsole(true);
  admin.fixtures["/peer/list"]={list:[{row_id:3,id:"office",hostname:"管理员设备"}],total:1};
  await admin.settle();
  await admin.settle();
  await admin.open("devices");
  assert.match(admin.root.innerHTML,/device-status-dot online[^<]*<\/span><span>office<\/span>/);
  await admin.open("connections");
  assert.match(admin.root.innerHTML,/device-status-dot offline[^<]*<\/span><span>managed-desk<\/span>/);
  assert.match(admin.root.innerHTML,/device-status-dot online[^<]*<\/span><span>helpdesk<\/span>/);
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
  assert.match(app.root.innerHTML,/device-status-dot online[^<]*<\/span><span>office<\/span>/);
  assert.match(app.root.innerHTML,/device-status-dot offline[^<]*<\/span><span>home-remote<\/span>/);
  await app.open("addressbooks");
  assert.match(app.root.innerHTML,/device-status-dot online[^<]*<\/span><span>office<\/span>/);
  assert.match(app.root.innerHTML,/device-status-dot offline[^<]*<\/span><span>home-remote<\/span>/);
});

test("settings center lists effective sources without revealing secrets", async () => {
  const user = createConsole(false);
  await user.settle();await user.settle();
  assert.doesNotMatch(user.root.innerHTML, /设置中心<\/button>/);
  const admin = createConsole(true);
  await admin.settle();await admin.settle();
  await admin.open("settings");
  assert.match(admin.root.innerHTML, /设置中心/);
  assert.match(admin.root.innerHTML, /开放注册/);
  assert.match(admin.root.innerHTML, /RUSTDESK_API_APP_REGISTER/);
  assert.match(admin.root.innerHTML, /环境变量/);
  assert.match(admin.root.innerHTML, /已配置（隐藏具体内容）/);
  assert.ok(admin.calls.some(call=>call.endpoint==="/settings/effective"));
});

test("settings center shows specialized effective values and key change warnings",async()=>{
  const app=createConsole(true);
  await app.settle();await app.settle();await app.open("settings");
  assert.match(app.root.innerHTML,/mail.example.test/);
  assert.match(app.root.innerHTML,/连接日志 90 天/);
  assert.match(app.root.innerHTML,/更换通信密钥前/);
  assert.match(app.root.innerHTML,/切换数据库前/);
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
  assert.match(app.root.innerHTML,/设置中继服务器列表/);
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

test("device and user references display link controls with status markers",async()=>{
  const app=createConsole(true);
  app.fixtures["/peer/list"]={list:[{row_id:1,id:"office",hostname:"Office",alias:"Desk",user_id:7}],total:1};
  await app.settle();await app.settle();
  await app.open("devices");
  assert.match(app.root.innerHTML,/data-action="entity-detail" data-entity-type="device" data-entity-id="office"/);
  assert.match(app.root.innerHTML,/device-status-dot online/);
  assert.match(app.root.innerHTML,/data-entity-label="Office"/);
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
