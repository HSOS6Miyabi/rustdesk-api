"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const http = require("node:http");
const { chromium } = require("playwright");

let browser, server, baseUrl;
const assetRoot = __dirname;

function answer(res, data) {
  res.writeHead(200, {"Content-Type":"application/json; charset=utf-8","Cache-Control":"no-store"});
  res.end(JSON.stringify({code:0, data}));
}

function serve(req, res) {
  const requestUrl = new URL(req.url, "http://127.0.0.1");
  const pathname = requestUrl.pathname;
  if(pathname.startsWith("/api/admin/")){
    const route = pathname.slice("/api/admin".length);
    const isAdmin = req.headers["api-token"] === "admin-token";
    if(route==="/config/admin")return answer(res, {title:"RustDesk 浏览器测试"});
    if(route==="/user/current")return answer(res, {
      username:isAdmin?"admin":"owner",nickname:isAdmin?"Administrator":"Owner",
      route_names:isAdmin?["*"]:[],token:req.headers["api-token"]
    });
    if(route==="/contact/list")return answer(res, {
      outgoing:[{user_id:11,username:"alice",nickname:"Alice"}],
      incoming:[{user_id:12,username:"bob",nickname:"Bob"}]
    });
    if(route==="/my/group/list")return answer(res, {
      list:[{id:7,name:"我的家庭设备",type:3,owner_user_id:3}],total:1
    });
    if(route==="/group/list")return answer(res, {
      list:[{id:1,name:"系统分组",type:1,owner_user_id:0}],total:1
    });
    if(route==="/my/audit_conn/list")return answer(res, {
      list:[{id:1,from_peer:"123456",from_account:"owner",
        peer_id:"654321",to_account:"alice",direction:"outgoing",created_at:1710000000}],total:1
    });
    if(route==="/my/audit_file/list")return answer(res, {
      list:[{id:2,from_peer:"123456",from_account:"owner",
        peer_id:"654321",to_account:"alice",path:"/files/report.pdf",direction:"outgoing"}],total:1
    });
    if(route==="/my/login_log/list")return answer(res, {list:[],total:0});
    if(route==="/user/list")return answer(res, {
      list:[{id:3,username:"owner",nickname:"Owner",status:1,group_id:1}],total:1
    });
    if(route==="/my/group/members")return answer(res, {
      user_ids:[3],users:[{id:3,username:"owner",nickname:"Owner"}]
    });
    if(route==="/my/group/anchors")return answer(res, {
      user_ids:[11],users:[{id:11,username:"alice",nickname:"Alice"}]
    });
    if(route==="/shared_devices/sources")return answer(res, {sources:[]});
    if(route==="/shared_devices/list")return answer(res, {list:[],total:0});
    if(route==="/login-options")return answer(res, {ops:[],need_captcha:false,disable_pwd:false});
    return answer(res, {list:[],total:0});
  }

  let name = pathname;
  if(name==="/_admin/"||name==="/_admin")name="/_admin/index.html";
  if(!name.startsWith("/_admin/")){
    res.writeHead(404);res.end("Not found");return;
  }
  const filename=name.slice("/_admin/".length);
  if(!["index.html","style.css","app.js","favicon.svg"].includes(filename)){
    res.writeHead(404);res.end("Not found");return;
  }
  const file=path.join(assetRoot,filename);
  const mime=filename.endsWith(".css")?"text/css":
    filename.endsWith(".js")?"application/javascript":
    filename.endsWith(".svg")?"image/svg+xml":"text/html";
  res.writeHead(200, {"Content-Type":mime+"; charset=utf-8","Cache-Control":"no-store"});
  res.end(fs.readFileSync(file));
}

async function pageForViewport(width, height, admin=false) {
  const context = await browser.newContext({
    viewport:{width,height},deviceScaleFactor:1,
    isMobile:width<600,hasTouch:width<600
  });
  const page=await context.newPage();
  await page.addInitScript(token=>{
    sessionStorage.setItem("rustdesk.console.token",token);
  }, admin?"admin-token":"user-token");
  await page.goto(baseUrl+"/_admin/",{waitUntil:"networkidle"});
  await page.locator("h1").first().waitFor({state:"visible"});
  return {context,page};
}

test.before(async()=>{
  server=http.createServer(serve);
  await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
  baseUrl="http://127.0.0.1:"+server.address().port;
  browser=await chromium.launch({headless:true,args:["--no-sandbox","--disable-dev-shm-usage"]});
});

test.after(async()=>{
  if(browser)await browser.close();
  if(server)await new Promise(resolve=>server.close(resolve));
});

test("desktop: ordinary users see owned groups and audit records, not global administration",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    assert.equal(await page.locator('nav.side-scroll button[data-view="groups"]').count(),0);
    assert.equal(await page.locator('nav.side-scroll button[data-view="mygroups"]').count(),1);
    await page.locator('nav.side-scroll button[data-view="mygroups"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的群组");
    assert.match(await page.locator(".content").innerText(),/我的家庭设备/);
    await page.locator('nav.side-scroll button[data-view="myconnections"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的连接记录");
    assert.match(await page.locator(".content").innerText(),/654321/);
    assert.match(await page.locator(".content").innerText(),/alice/);
  }finally{await context.close();}
});

test("desktop: administrators have a separate global group entry",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    assert.equal(await page.locator('nav.side-scroll button[data-view="groups"]').count(),1);
    assert.equal(await page.locator('nav.side-scroll button[data-view="mygroups"]').count(),1);
    await page.locator('nav.side-scroll button[data-view="groups"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="用户分组");
    assert.match(await page.locator(".content").innerText(),/系统分组/);
  }finally{await context.close();}
});

test("mobile: navigation, owned group form and contacts fit a narrow screen",async()=>{
  const {context,page}=await pageForViewport(375,812,false);
  try{
    const menu=page.locator('button[data-action="mobile-open"]');
    assert.equal(await menu.isVisible(),true);
    await menu.click();
    await page.locator("aside.sidebar.open").waitFor();
    await page.waitForTimeout(250);
    let bounds=await page.locator("aside.sidebar").boundingBox();
    assert.ok(bounds && bounds.x>=-2,"mobile navigation must be visible");
    await page.locator('nav.side-scroll button[data-view="mygroups"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的群组");
    await page.locator('button[data-action="create"]').click();
    const drawer=page.locator("section.drawer");
    await drawer.waitFor({state:"visible"});
    bounds=await drawer.boundingBox();
    assert.ok(bounds&&bounds.width<=377,"group editor must fit a 375px viewport");
    assert.match(await drawer.innerText(),/公共锚点组/);
    await page.locator('button[data-action="close-modal"]').last().click();

    await menu.click();
    await page.locator('nav.side-scroll button[data-view="contacts"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="联系人");
    await page.getByText("Alice",{exact:true}).waitFor();
    await page.getByText("Bob",{exact:true}).waitFor();
    assert.match(await page.locator(".content").innerText(),/Alice/);
    assert.match(await page.locator(".content").innerText(),/Bob/);
    const width=await page.evaluate(()=>({
      document:document.documentElement.scrollWidth,
      viewport:window.innerWidth
    }));
    assert.ok(width.document<=width.viewport+2,
      "mobile document has unexpected horizontal overflow: "+JSON.stringify(width));
  }finally{await context.close();}
});

test("tablet: shared device and audit pages are accessible without horizontal overflow",async()=>{
  const {context,page}=await pageForViewport(768,1024,false);
  try{
    await page.locator('button[data-action="mobile-open"]').click();
    await page.locator('nav.side-scroll button[data-view="shared"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="共享设备");
    const width=await page.evaluate(()=>({
      document:document.documentElement.scrollWidth,
      viewport:window.innerWidth
    }));
    assert.ok(width.document<=width.viewport+2,
      "tablet document has unexpected horizontal overflow: "+JSON.stringify(width));
  }finally{await context.close();}
});
