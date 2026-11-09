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
  const root = { innerHTML: "" };
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
    "/access_rule/list": {list:[{id:8,name:"Office support",source_type:"group",source_id:2,target_type:"device_group",target_id:"3",effect:"allow",enabled:true}],total:1},
    "/admin_role/list": {list:[{id:12,name:"Read-only devices",permissions:"devices.view",scope_type:"global",scope_id:0,enabled:true}],total:1},
    "/admin_role_assignment/list": {list:[{id:4,user_id:15,role_id:12}],total:1},
    "/mfa/status": {enabled:false,pending:false,remaining_recovery_codes:0},
    "/mfa/email/status": {enabled:false,email:"owner@example.test",mail_configured:true},
    "/my/login_log/list": { list: [], total: 0 },
    "/user/list": { list: [], total: 0 },
    "/peer/list": { list: [], total: 0 },
    "/my/peer/list": { list: [], total: 0 },
    "/my/address_book/list": { list: [], total: 0 },
    "/contact/list": { outgoing: [], incoming: [] },
    "/shared_devices/list": { list: [], total: 0 },
    "/shared_devices/sources": { sources: [] },
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

test("active session actions appear only in administrator console", async () => {
  const owner = createConsole();
  await owner.settle();
  await owner.settle();
  assert.doesNotMatch(owner.root.innerHTML, /活动远程连接/);
  const admin = createConsole(true);
  await admin.settle();
  await admin.settle();
  assert.match(admin.root.innerHTML, /活动远程连接/);
  assert.match(admin.root.innerHTML, /结束连接记录/);
  await admin.open("activeSessions");
  assert.ok(admin.calls.some(call => call.endpoint === "/session/list"));
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
  assert.ok(!app.calls.some(x=>x.endpoint==="/my/peer/list"));
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
  assert.match(app.root.innerHTML, /联系人账户编号/);
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
  for (const label of ["地址簿编号", "目标用户编号", "只读", "可读写", "完全控制"]) {
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
