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
    if(route==="/login"){
      let payload="";
      req.on("data",chunk=>payload+=chunk.toString());
      req.on("end",()=>{
        const input=JSON.parse(payload||"{}");
        if(input.type==="tfa_code"||input.type==="email_code"){
          return answer(res,{username:"admin",nickname:"Admin",route_names:["*"],token:"admin-token"});
        }

        if(input.username==="email-user"){
          return answer(res,{type:"email_check",tfa_type:"email_check",secret:"email-verification-challenge",user:{name:input.username,email:"email-user@example.test"}});
        }
        return answer(res,{type:"tfa_check",tfa_type:"tfa_check",secret:"temporary-challenge",user:{name:input.username}});
      });
      return;
    }
    if(route==="/mfa/status")return answer(res,{enabled:false,pending:false,remaining_recovery_codes:0});
    if(route==="/mfa/email/status")return answer(res,{enabled:false,email:"owner@example.test",mail_configured:true});
    if(route==="/mfa/email/setup")return answer(res,{secret:"mail-enrollment-challenge",expires_in_seconds:600});
    if(route==="/mfa/email/enable"||route==="/mfa/email/disable")return answer(res,{});

    if(route==="/mfa/setup")return answer(res,{
      secret:"ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",uri:"otpauth://totp/RustDesk%20OSS%20API?secret=ABCDEFGHIJKLMNOPQRSTUVWXYZ234567",expires_at:1910000000
    });
    if(route==="/mfa/enable")return answer(res,{recovery_codes:[
      "A1111111111111111111","B2222222222222222222","C3333333333333333333",
      "D4444444444444444444","E5555555555555555555","F6666666666666666666"
    ]});
    if(route==="/mfa/disable"||route==="/mfa/recovery/regenerate")return answer(res,{recovery_codes:[]});
    if(route==="/config/admin")return answer(res, {title:"RustDesk 浏览器测试"});
    if(route==="/password/forgot"||route==="/password/reset")return answer(res,{});
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
    if(route==="/audit_conn/list")return answer(res,{list:[
      {id:21,peer_id:"managed-desk",from_peer:"helpdesk",note:"现场支持",created_at:1710000000}
    ],total:1});
    if(route==="/audit_conn/note")return answer(res,{});
    if(route==="/my/audit_conn/list")return answer(res, {
      list:[{id:1,from_peer:"123456",from_account:"owner",
        peer_id:"654321",to_account:"alice",direction:"outgoing",created_at:1710000000}],total:1
    });
    if(route==="/my/audit_file/list")return answer(res, {
      list:[{id:2,from_peer:"123456",from_account:"owner",
        peer_id:"654321",to_account:"alice",path:"/files/report.pdf",direction:"outgoing"}],total:1
    });
    if(route==="/invite/list")return answer(res,{list:[
      {id:17,email:"new.member@example.test",group_id:1,expires_at:1910000000,accepted_at:0,revoked_at:0}
    ],total:1});
    if(route==="/invite/create"||route==="/invite/revoke"||route==="/invite/accept")return answer(res,{});
    if(route==="/mail/settings")return answer(res,{enabled:true,host:"smtp.example.test",port:587,security:"starttls",sender_email:"support@example.test",sender_name:"Support",password_configured:true});
    if(route==="/mail/test")return answer(res,{sent:true});
    if(route==="/api_token/list")return answer(res,{list:[],total:0});
    if(route==="/api_token/create")return answer(res,{id:8,prefix:"rdapi_example",token:"rdapi_example-secret-value"});
    if(route==="/api_token/delete")return answer(res,{});
    if(route==="/session/list")return answer(res, {list:[
      {id:4,device_id:"support-123456",conn_id:41,last_seen_at:1710000000}
    ],total:1});
    if(route==="/session/disconnect")return answer(res, {status:"pending"});
    if(route==="/session/disconnect/list")return answer(res, {list:[],total:0});
    if(route==="/my/address_book/import"||route==="/address_book/import")return answer(res,{created:1,updated:0});
    if(route==="/my/address_book/export"||route==="/address_book/export"){
      res.writeHead(200,{"Content-Type":"text/csv; charset=utf-8",
        "Content-Disposition":"attachment; filename=rustdesk-addressbook.csv"});
      res.end("id,alias\nbackup-001,测试设备\n");
      return;
    }
    if(route==="/my/address_book/trash/list")return answer(res,{list:[
      {row_id:13,id:"deleted-desk-001",alias:"Former desk",collection_id:0,deleted_at:1710000000}
    ],total:1});
    if(route==="/my/address_book/trash/restore"||route==="/my/address_book/trash/purge")return answer(res,{});
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
    if(route==="/device_policy/list")return answer(res,{list:[
      {id:21,name:"办公设备",id_whitelist:"desk-*",ip_whitelist:"192.0.2.0/24",enable_webrtc:false,trust_duration_days:30}
    ],total:1});
    if(route==="/device_policy/assignment/list")return answer(res,{list:[
      {id:7,device_id:"desk-001",policy_id:21}
    ],total:1});
    if(route==="/my/audit_alarm/list")return answer(res,{list:[
      {id:5,typ:10,peer_id:"desk-001",conn_id:12,info:'{"id":"visitor"}'}
    ],total:1});
    if(route==="/audit_alarm/list")return answer(res,{list:[
      {id:5,typ:10,peer_id:"desk-001",to_user_id:3,conn_id:12}
    ],total:1});
    if(route==="/audit_operation/list")return answer(res,{list:[
      {id:6,account:"admin",method:"POST",path:"/api/admin/group/create",client_ip:"192.0.2.1",success:true}
    ],total:1});
    if(route==="/audit/retention")return answer(res,{
      connection_days:90,file_days:30,alarm_days:180,login_days:90,operation_days:180
    });
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
    await page.locator(".table-scroll tbody tr").first().waitFor({state:"visible"});
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

test("administrator: security policies, alarms and operation logs render correctly",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="policies"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备安全策略");
    await page.getByText("办公设备",{exact:true}).first().waitFor();
    assert.match(await page.locator(".content").innerText(),/192.0.2.0\/24/);
    await page.locator('nav.side-scroll button[data-view="policyAssignments"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备策略分配");
    await page.getByText("desk-001",{exact:true}).first().waitFor();
    await page.locator('nav.side-scroll button[data-view="alarms"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="安全告警");
    await page.getByText("设备 ID 白名单拒绝 (10)").first().waitFor();
    await page.locator('nav.side-scroll button[data-view="operations"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="管理操作记录");
    await page.getByText("/api/admin/group/create",{exact:true}).first().waitFor();
  }finally{await context.close();}
});

test("mobile: device policy editor and retention form remain usable",async()=>{
  const {context,page}=await pageForViewport(375,812,true);
  try{
    const openMenu=async()=>{
      await page.locator('button[data-action="mobile-open"]').click();
      await page.locator("aside.sidebar.open").waitFor();
    };
    await openMenu();
    await page.locator('nav.side-scroll button[data-view="policies"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备安全策略");
    await page.locator('button[data-action="create"]').click();
    const drawer=page.locator("section.drawer");
    await drawer.waitFor({state:"visible"});
    const editorBounds=await drawer.boundingBox();
    assert.ok(editorBounds&&editorBounds.width<=377,"device policy editor exceeds phone width");
    assert.match(await drawer.innerText(),/设备 ID 白名单/);
    assert.match(await drawer.innerText(),/可信设备有效期/);
    await page.locator('button[data-action="close-modal"]').last().click();
    await openMenu();
    await page.locator('nav.side-scroll button[data-view="retention"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="日志保留设置");
    await page.waitForFunction(()=>document.querySelector('#retention-form input[name="connection_days"]')?.value==="90");
    assert.equal(await page.locator('#retention-form input[name="connection_days"]').inputValue(),"90");
    const dims=await page.evaluate(()=>({scroll:document.documentElement.scrollWidth,width:innerWidth}));
    assert.ok(dims.scroll<=dims.width+2,"mobile retention form has horizontal overflow: "+JSON.stringify(dims));
  }finally{await context.close();}
});

test("ordinary users see their security alarms but cannot access global security management",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    for(const view of ["alarms","operations","policies","policyAssignments","retention"]){
      assert.equal(await page.locator('nav.side-scroll button[data-view="'+view+'"]').count(),0);
    }
    await page.locator('nav.side-scroll button[data-view="myalarms"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的安全告警");
    await page.getByText("设备 ID 白名单拒绝 (10)").first().waitFor();
  }finally{await context.close();}
});

test("administrator can request disconnection of a reported remote session",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="activeSessions"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="活动远程连接");
    await page.getByText("support-123456",{exact:true}).first().waitFor();
    await page.locator('button[data-action="session-disconnect"]').first().click();
    await page.getByText(/结束连接请求已提交/).first().waitFor();
    await page.locator('nav.side-scroll button[data-view="disconnectHistory"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="结束连接记录");
  }finally{await context.close();}
});

test("administrator can create an API token and view it once",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="apiTokens"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="API 管理令牌");
    await page.locator('button[data-action="create"]').click();
    const editor=page.locator('#edit-form');
    await editor.waitFor();
    await editor.locator('input[name="name"]').fill("My deployment token");
    await editor.locator('input[name="permissions"]').fill("devices.deploy");
    await page.locator('button[data-action="submit-api-token"]').click();
    await page.waitForTimeout(300);
    const tokenField=page.locator('#api-token-secret');
    assert.ok(await tokenField.count()>0,
      "API token must be visible after creation. Page: "+(await page.locator(".content").innerText()).slice(-1200));
    await tokenField.waitFor({state:"visible"});
    assert.equal(await page.locator('#api-token-secret').inputValue(),"rdapi_example-secret-value");
    await page.locator('button[data-action="close-modal"]').last().click();
    assert.equal(await page.locator('#api-token-secret').count(),0);
  }finally{await context.close();}
});

test("ordinary users can configure address books shared with everyone",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.locator('nav.side-scroll button[data-view="myRules"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的地址簿共享规则");
    await page.locator('button[data-action="create"]').click();
    const editor=page.locator('#edit-form');
    await editor.waitFor({state:"visible"});
    await editor.locator('select[name="type"]').selectOption("3");
    await editor.locator('input[name="to_id"]').fill("0");
    assert.equal(await editor.locator('select[name="type"]').inputValue(),"3");
    assert.equal(await editor.locator('input[name="to_id"]').inputValue(),"0");
    assert.match(await page.locator("section.drawer").innerText(),/所有用户/);
  }finally{await context.close();}
});

test("personal address book supports CSV upload and download",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.locator('nav.side-scroll button[data-view="addressbooks"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="地址簿");
    await page.locator('button[data-action="csv-import"]').click();
    await page.locator('#csv-file').setInputFiles({
      name:"devices.csv",mimeType:"text/csv",
      buffer:Buffer.from("id,alias\ndesk-123,测试设备\n")
    });
    await page.locator('button[data-action="csv-submit"]').click();
    await page.getByText(/CSV 导入完成：新增 1 条/).waitFor();
    await page.locator('button[data-action="csv-export"]').click();
    const downloaded=page.waitForEvent("download");
    await page.locator('button[data-action="csv-submit"]').click();
    const item=await downloaded;
    assert.equal(item.suggestedFilename(),"rustdesk-addressbook.csv");
  }finally{await context.close();}
});

test("personal address book trash can restore and permanently remove records",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="trash"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="地址簿回收站");
    await page.getByText("deleted-desk-001",{exact:true}).first().waitFor();
    await page.locator('button[data-action="trash-restore"]').first().click();
    await page.getByText(/地址簿条目已恢复/).first().waitFor();
    await page.locator('button[data-action="trash-purge"]').first().click();
    await page.getByText(/地址簿条目已永久删除/).first().waitFor();
  }finally{await context.close();}
});

test("administrator can edit SMTP settings and send a test email",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="mail"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="SMTP 邮件服务");
    await page.locator('#mail-host').waitFor({state:"visible"});
    assert.equal(await page.locator('#mail-host').inputValue(),"smtp.example.test");
    await page.locator('#mail-test-form input[name="to"]').fill("receiver@example.test");
    await page.locator('#mail-test-form button[type="submit"]').click();
    await page.getByText("测试邮件已发送").first().waitFor();
    await page.locator('nav.side-scroll button[data-view="policies"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备安全策略");
  }finally{await context.close();}
});

test("password recovery and one-time reset forms work on the login screen",async()=>{
  const context=await browser.newContext({viewport:{width:1280,height:800}});
  const page=await context.newPage();
  try{
    await page.goto(baseUrl+"/_admin/",{waitUntil:"networkidle"});
    await page.locator('button[data-action="forgot-password"]').waitFor({state:"visible"});
    await page.locator('button[data-action="forgot-password"]').click();
    await page.locator('#recovery-email').fill("someone@example.test");
    await page.locator('button[form="forgot-password-form"]').click();
    await page.getByText("如果邮箱已登记，将收到密码恢复邮件").waitFor();
    await page.goto(baseUrl+"/_admin/#/password-reset?token=example-reset-token",{waitUntil:"networkidle"});
    await page.locator('#recovery-new-password').waitFor({state:"visible"});
    await page.locator('#recovery-new-password').fill("NewPassword123");
    await page.locator('#recovery-confirm-password').fill("NewPassword123");
    await page.locator('button[form="public-password-reset-form"]').click();
    await page.getByText("密码已更新，请使用新密码登录").waitFor();
  }finally{await context.close();}
});

test("administrator creates and revokes team invitation",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="invitations"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="成员邮件邀请");
    await page.getByText("new.member@example.test",{exact:true}).first().waitFor();
    await page.locator('button[data-action="create"]').click();
    await page.locator('#edit-form input[name="email"]').fill("invitee@example.test");
    await page.locator('button[data-action="submit-invitation"]').click();
    await page.getByText("邀请邮件已发送").first().waitFor();
    await page.locator('button[data-action="invitation-revoke"]').first().click();
    await page.getByText("邀请已撤销").first().waitFor();
  }finally{await context.close();}
});

test("invited member can register from a team email link",async()=>{
  const context=await browser.newContext({viewport:{width:1280,height:800}});
  const page=await context.newPage();
  try{
    await page.goto(baseUrl+"/_admin/#/invite?token=example-invite-token",{waitUntil:"networkidle"});
    await page.locator('#invited-username').waitFor({state:"visible"});
    await page.locator('#invited-username').fill("new.member");
    await page.locator('#invited-nickname').fill("New member");
    await page.locator('#invited-password').fill("NewPassword123");
    await page.locator('#invited-password-confirm').fill("NewPassword123");
    await page.locator('button[form="accept-invitation-form"]').click();
    await page.getByText("账户已创建，请使用新账户登录").waitFor();
  }finally{await context.close();}
});

test("administrator can save a connection audit note",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="connections"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="全部连接记录");
    await page.getByText("managed-desk",{exact:true}).first().waitFor();
    await page.locator('button[data-action="audit-note"]').first().click();
    await page.locator('#audit-note-input').waitFor({state:"visible"});
    assert.equal(await page.locator('#audit-note-input').inputValue(),"现场支持");
    await page.locator('#audit-note-input').fill("远程维护完成");
    await page.locator('button[data-action="audit-note-save"]').click();
    await page.getByText("连接备注已保存").waitFor();
  }finally{await context.close();}
});

test("personal settings can enroll an authenticator and show recovery codes",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.locator('nav.side-scroll button[data-view="profile"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="个人设置");
    await page.locator('button[data-action="mfa-setup"]').click();
    await page.locator('#mfa-activate-code').waitFor({state:"visible"});
    assert.match(await page.locator("#modal-overlay").innerText(),/身份验证器密钥/);
    await page.locator('#mfa-activate-code').fill("123456");
    await page.locator('button[form="mfa-activate-form"]').click();
    await page.getByText("保存双重认证恢复码").waitFor();
    assert.match(await page.locator("#modal-overlay").innerText(),/A1111111111111111111/);
    await page.locator('button[data-action="close-modal"]').last().click();
    assert.equal(await page.locator("#modal-overlay").count(),0);
  }finally{await context.close();}
});

test("web login accepts authenticator challenge before opening the console",async()=>{
  const context=await browser.newContext({viewport:{width:1280,height:800}});
  const page=await context.newPage();
  try{
    await page.goto(baseUrl+"/_admin/",{waitUntil:"networkidle"});
    await page.locator('#username').fill("admin");
    await page.locator('#password').fill("Password123");
    await page.locator('#login-form button[type="submit"]').click();
    await page.locator('#mfa-login-code').waitFor({state:"visible"});
    await page.locator('#mfa-login-code').fill("123456");
    await page.locator('button[form="mfa-login-form"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="控制台概览");
    assert.equal(await page.evaluate(()=>sessionStorage.getItem("rustdesk.console.token")),"admin-token");
  }finally{await context.close();}
});

test("personal settings can opt in to email login verification",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.locator('nav.side-scroll button[data-view="profile"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="个人设置");
    await page.getByText("邮箱登录验证",{exact:true}).waitFor();
    await page.locator('button[data-action="email-verification-setup"]').click();
    await page.locator('#email-verification-code').waitFor({state:"visible"});
    await page.locator('#email-verification-code').fill("123456");
    await page.locator('button[form="email-verification-form"]').click();
    await page.getByText("邮箱登录验证已启用").waitFor();
  }finally{await context.close();}
});

test("web login can complete an email verification challenge",async()=>{
  const context=await browser.newContext({viewport:{width:1280,height:800}});
  const page=await context.newPage();
  try{
    await page.goto(baseUrl+"/_admin/",{waitUntil:"networkidle"});
    await page.locator('#username').fill("email-user");
    await page.locator('#password').fill("Password123");
    await page.locator('#login-form button[type="submit"]').click();
    await page.locator('#mfa-login-code').waitFor({state:"visible"});
    await page.getByText("邮箱验证码",{exact:true}).waitFor();
    await page.locator('#mfa-login-code').fill("123456");
    await page.locator('button[form="mfa-login-form"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="控制台概览");
    assert.equal(await page.evaluate(()=>sessionStorage.getItem("rustdesk.console.token")),"admin-token");
  }finally{await context.close();}
});
