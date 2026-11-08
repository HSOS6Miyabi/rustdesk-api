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

function createConsole(isAdmin = false) {
  const calls = [];
  const events = {};
  const root = { innerHTML: "" };
  const browserLocation = { hash: "#/overview", assign() {} };
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
    "/my/audit_file/list": {
      list: [{ id: 18, from_peer: "123456", peer_id: "654321", from_account: "demo-user",
        to_account: "another-user", path: "/home/file", direction: "outgoing" }], total: 1
    },
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
  const session = storage({ "rustdesk.console.token": "example-session" });
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
    root, calls, events, fixtures, settle,
    async open(view) {
      const control = { dataset: { view } };
      await events.click({ target: { closest: () => control } });
      await settle();
      await settle();
    },
    async action(name) {
      const control = { dataset: { action: name } };
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
