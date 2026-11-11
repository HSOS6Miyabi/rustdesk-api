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
    if(route==="/settings/effective")return answer(res,{
      bootstrap_version:0,database_version:1,items:[
        {key:"app.register",name:"开放注册",description:"允许新用户自行注册",
          category:"账户与安全",environment:"RUSTDESK_API_APP_REGISTER",
          value_type:"bool",current_value:"true",source:"环境变量",
          override_enabled:false,bootstrap:false,active:true,requires_restart:true},
        {key:"jwt.key",name:"JWT 签名密钥",description:"签名登录令牌",
          category:"令牌",environment:"RUSTDESK_API_JWT_KEY",
          value_type:"string",sensitive:true,configured:true,current_value:"",
          source:"环境变量",override_enabled:false,bootstrap:true,active:true,requires_restart:true},
        {key:"gorm.max-open-conns",name:"数据库最大连接数",description:"数据库连接池的连接上限",
          category:"数据库",environment:"RUSTDESK_API_GORM_MAX_OPEN_CONNS",
          value_type:"int",current_value:"4",source:"配置文件",
          override_enabled:false,bootstrap:true,active:true,requires_restart:true}
      ]
    });
    if(route==="/settings/overview")return answer(res,{
      summary:{
        push_channels:1,device_policies:2,access_rules:1,relay_nodes:1,
        smtp_settings:{enabled:true,host:"smtp.example.test",port:587,sender_email:"alerts@example.test"},
        notification_settings:{enabled:true,events:"login.failure",recipients:"alerts@example.test"},
        audit_retention:{connection_days:90,login_days:30}
      },outbox:{push:{pending:0,failed:1},notification:{pending:0,failed:0}}
    });
    if(route==="/settings/history")return answer(res,{list:[]});
    if(route==="/settings/outbox")return answer(res,{list:[
      {id:requestUrl.searchParams.get("category")==="notification"?58:47,
        status:"failed",attempts:8,last_error:"temporary delivery preparation failure"}
    ]});
    if(route==="/settings/validate")return answer(res,{valid:true});
    if(route==="/settings/overrides"||route==="/settings/overrides/reset")return answer(res,{saved:true});

    if(route==="/password/forgot"||route==="/password/reset")return answer(res,{});
    if(route==="/user/current")return answer(res, {
      username:isAdmin?"admin":"owner",nickname:isAdmin?"Administrator":"Owner",
      route_names:isAdmin?["*"]:(req.headers["api-token"]==="delegated-token"?["users.view","devices.view"]:[]),token:req.headers["api-token"]
    });
    if(route==="/contact/list")return answer(res, {
      outgoing:[{user_id:11,username:"alice",nickname:"Alice"}],
      incoming:[{user_id:12,username:"bob",nickname:"Bob"}]
    });
    if(route==="/my/group/list")return answer(res, {
      list:[{id:7,name:"我的家庭设备",type:3,owner_user_id:3}],total:1
    });
    if(route==="/group/list")return answer(res, {
      list:[{id:1,name:"系统分组",type:1,owner_user_id:0},
        {id:2,name:"帮助台用户组",type:1,owner_user_id:0},
        {id:3,name:"研究人员组",type:1,owner_user_id:0}],total:3
    });
    if(route==="/device_group/list")return answer(res,{
      list:[{id:3,name:"实验室设备组"}],total:1
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
    if(route==="/admin_role/list")return answer(res,{list:[
      {id:13,name:"Device support viewers",permissions:"devices.view",scope_type:"global",scope_id:0,enabled:true}
    ],total:1});
    if(route==="/admin_role_assignment/list")return answer(res,{list:[
      {id:21,user_id:3,role_id:13}
    ],total:1});
    if(["/admin_role/create","/admin_role/update","/admin_role/delete",
      "/admin_role_assignment/create","/admin_role_assignment/update","/admin_role_assignment/delete"].includes(route)){
      return answer(res,{});
    }
    if(route==="/access_rule/list")return answer(res,{list:[
      {id:18,name:"Helpdesk access",source_type:"group",source_id:2,target_type:"device_group",target_id:"3",effect:"allow",enabled:true}
    ],total:1});
    if(["/access_rule/create","/access_rule/update","/access_rule/delete"].includes(route))return answer(res,{});
    if(route==="/control_role/list")return answer(res,{list:[
      {id:7,name:"Helpdesk role",keyboard:1,clipboard:2,file:1,terminal:1}
    ],total:1});
    if(route==="/control_assignment/list")return answer(res,{list:[
      {id:5,target_type:"device",target_id:"desk-001",role_id:7}
    ],total:1});
    if(["/control_role/create","/control_role/update","/control_role/delete",
      "/control_assignment/create","/control_assignment/update","/control_assignment/delete"].includes(route)){
      return answer(res,{});
    }
    if(route==="/relay/list")return answer(res,{list:[
      {id:10,name:"East site",address:"relay-east.example.test:21117",
       subnets:"203.0.113.0/24",countries:"JP",priority:10,enabled:true}
    ],total:1});
    if(route==="/relay/create"||route==="/relay/update"||route==="/relay/delete")return answer(res,{});
    if(route==="/notification/settings")return answer(res,{
      settings:{enabled:true,recipients:"staff@example.test",events:"login.failure,deployment.changed,audit.file"},
      catalog:["audit.file","deployment.changed","login.failure","session.disconnect_requested","session.ended"]
    });
    if(route==="/notification/list")return answer(res,{
      list:[
        {id:10,event_type:"deployment.changed",recipient:"staff@example.test",
          status:"sent",attempts:1,sent_at:1731000000,last_error:""},
        {id:11,event_type:"audit.file",recipient:"staff@example.test",
          status:"failed",attempts:3,sent_at:0,last_error:"temporary SMTP failure"}
      ],total:2
    });
    if(route==="/notification/retry")return answer(res,{});
    if(route==="/geoip/lookup")return answer(res,{
      locations:{"10.0.0.5":"局域网地址","192.0.2.1":"位置未知","127.0.0.1":"本地回环地址"},
      device_status:{"office":true,"home-remote":false,"654321":true,"123456":false,
        "managed-desk":false,"helpdesk":true,"desk-001":true,"support-123456":true},
      database_ready:false
    });
    if(route==="/push/events")return answer(res,[
      {code:"device.online",name:"设备上线"},
      {code:"device.offline",name:"设备离线"},
      {code:"connection.started",name:"远程连接建立"}
    ]);
    if(route==="/push/channels"){
      if(req.method==="POST")return answer(res,{id:42,name:"新通知",enabled:true});
      return answer(res,{list:[{
        id:5,kind:"messagepusher",name:"家庭通知",enabled:true,
        base_url:"https://push.example.test",api_key_configured:true,
        events:["device.offline"]
      }]});
    }
    if(route==="/push/history")return answer(res,{list:[
      {id:23,title:"[RustDesk · 设备上线] 家庭电脑",event_type:"device.online",status:"sent",attempts:1,created_at:1710000000}
    ],total:1});
    if(route==="/push/channels/test"||route==="/push/history/retry")return answer(res,{status:"queued"});
    if(route==="/push/channels/delete")return answer(res,{});

    if(route==="/client_builder/profiles"){
      if(req.method==="POST")return answer(res,{id:3});
      return answer(res,{list:[
        {id:2,name:"Office Windows",platform:"windows-x64",version:"1.5.1",
          host:"desk.example.test:21116",
          public_key:"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
          api_server:"https://desk.example.test",relay_server:"relay.example.test:21117"}
      ],total:1});
    }
    if(route==="/client_builder/tasks"){
      if(req.method==="POST")return answer(res,{id:7,status:"pending"});
      return answer(res,{list:[
        {id:5,profile_id:2,platform:"windows-x64",version:"1.5.1",
          status:"ready",filename:"rustdesk-licensed-fixture.exe",bytes:26211184},
        {id:6,profile_id:2,platform:"windows-x64",version:"1.5.1",
          status:"failed",error:"connection unavailable"}
      ],total:2});
    }
    if(route==="/client_builder/tasks/retry"||route==="/client_builder/profiles/delete")return answer(res,{});
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
    if(route==="/my/login_log/list")return answer(res, {list:[{id:9,ip:"10.0.0.5",platform:"Windows",created_at:1710000000}],total:1});
    if(route==="/peer/list"||route==="/my/peer/list")return answer(res,{
      list:[{row_id:101,id:"office",hostname:"办公电脑",last_online_ip:"10.0.0.5"},
        {row_id:102,id:"home-remote",hostname:"家庭电脑",last_online_ip:"127.0.0.1"}],
      total:2
    });
    if(route==="/user/mfa/reset")return answer(res,{});
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
    viewport:{width,height},deviceScaleFactor:1,locale:"zh-CN",
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

async function waitForMobileBreadcrumb(page) {
  await page.waitForFunction(()=>{
    const breadcrumb=document.querySelector(".breadcrumbs");
    const title=breadcrumb?.querySelector(".crumb-current");
    if(!breadcrumb?.isConnected||!title?.isConnected)return false;
    const bounds=title.getBoundingClientRect();
    return bounds.width>0&&bounds.height>0&&bounds.height<=22&&
      [...breadcrumb.querySelectorAll("span:not(.crumb-current)")].
        every(element=>getComputedStyle(element).display==="none");
  },null,{timeout:10000});
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
    assert.equal(await page.locator('#retention-form input[name="push_sent_days"]').inputValue(),"30");
    assert.equal(await page.locator('#retention-form input[name="notification_failed_days"]').inputValue(),"90");
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
    await editor.locator('input[name="to_id"]').waitFor({state:"attached"});
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

test("administrator can configure relay node preferences",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="relayNodes"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="中继节点管理");
    await page.getByText("relay-east.example.test:21117",{exact:true}).first().waitFor();
    await page.locator('button[data-action="create"]').click();
    await page.locator('#edit-form input[name="name"]').fill("West site");
    await page.locator('#edit-form input[name="address"]').fill("relay-west.example.test:21117");
    await page.locator('#edit-form textarea[name="subnets"]').fill("198.51.100.0/24");
    await page.locator('#edit-form textarea[name="countries"]').fill("US,CA");
    await page.locator('#edit-form input[name="priority"]').fill("20");
    await page.locator('#edit-form input[name="enabled"]').check();
    await page.locator('button[form="edit-form"]').click();
    await page.locator("#edit-form").waitFor({state:"detached"});
  }finally{await context.close();}
});

test("administrator can save a control role with native permissions",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="controlRoles"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="远程控制角色");
    await page.getByText("Helpdesk role",{exact:true}).first().waitFor();
    await page.locator('button[data-action="create"]').click();
    await page.locator('#edit-form input[name="name"]').fill("Temporary support");
    await page.locator('#edit-form select[name="keyboard"]').selectOption("1");
    await page.locator('#edit-form select[name="clipboard"]').selectOption("2");
    await page.locator('#edit-form select[name="file"]').selectOption("1");
    await page.locator('#edit-form select[name="usb"]').selectOption("1");
    const submittedRequest=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/control_role/create")&&req.method()==="POST");
    await page.locator('button[data-action="submit-control-role"]').click();
    const submitted=(await submittedRequest).postDataJSON();
    assert.equal(submitted.name,"Temporary support");
    assert.equal(submitted.keyboard,1);
    assert.equal(submitted.clipboard,2);
    assert.equal(submitted.file,1);
    assert.equal(submitted.usb,1);
    await page.locator('#edit-form').waitFor({state:"detached"});
    await page.locator('nav.side-scroll button[data-view="controlAssignments"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="远程控制角色分配");
    await page.getByText("desk-001",{exact:true}).first().waitFor();
  }finally{await context.close();}
});

test("full administrator configures management role permissions and scope",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="adminRoles"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="管理员角色");
    await page.getByText("Device support viewers",{exact:true}).first().waitFor();
    await page.locator('button[data-action="create"]').click();
    await page.locator('#edit-form input[name="name"]').fill("Helpdesk read only");
    await page.locator('#edit-form input[name="permissions"][value="users.view"]').check();
    await page.locator('#edit-form input[name="permissions"][value="devices.view"]').check();
    await page.locator('#edit-form select[name="scope_type"]').selectOption("group");
    const scopePicker=page.locator('.entity-picker[data-picker-field="scope_id"]');
    await scopePicker.locator('.picker-query').fill("研究人员组");
    await scopePicker.locator('button[data-action="picker-search"]').click();
    await scopePicker.locator('button[data-action="picker-select"][data-id="3"]').click();
    await page.locator('#edit-form input[name="enabled"]').check();
    const submit=page.waitForRequest(req=>req.url().endsWith("/api/admin/admin_role/create")&&req.method()==="POST");
    await page.locator('button[data-action="submit-admin-role"]').click();
    const body=(await submit).postDataJSON();
    assert.equal(body.name,"Helpdesk read only");
    assert.ok(body.permissions.includes("users.view"));
    assert.ok(body.permissions.includes("devices.view"));
    assert.equal(body.scope_type,"group");
    assert.equal(body.scope_id,3);
    assert.equal(body.enabled,true);
    await page.locator("#edit-form").waitFor({state:"detached"});
    await page.locator('nav.side-scroll button[data-view="adminRoleAssignments"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="管理员角色分配");
    await page.getByText("21",{exact:true}).first().waitFor();
  }finally{await context.close();}
});

test("delegated viewer gets scoped directory navigation without management actions",async()=>{
  const context=await browser.newContext({viewport:{width:1280,height:800}});
  const page=await context.newPage();
  try{
    await page.addInitScript(()=>sessionStorage.setItem("rustdesk.console.token","delegated-token"));
    await page.goto(baseUrl+"/_admin/",{waitUntil:"networkidle"});
    await page.locator('nav.side-scroll button[data-view="users"]').waitFor();
    assert.equal(await page.locator('nav.side-scroll button[data-view="adminRoles"]').count(),0);
    await page.locator('nav.side-scroll button[data-view="users"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="用户账户");
    assert.equal(await page.locator('button[data-action="create"]').count(),0);
    await page.getByText("owner",{exact:true}).first().waitFor();
    assert.equal(await page.locator('button[data-action="delete"]').count(),0);
    const listing=page.waitForRequest(req=>
      req.url().includes("/api/admin/peer/list")&&req.method()==="GET");
    await page.locator('nav.side-scroll button[data-view="devices"]').click();
    await listing;
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备管理");
  }finally{await context.close();}
});

test("administrator can reset a team member's lost second-factor credentials",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="users"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="用户账户");
    await page.getByText("owner",{exact:true}).first().waitFor();
    const reset=page.locator('button[data-action="reset-user-mfa"]').first();
    await reset.waitFor({state:"visible"});
    const saved=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/user/mfa/reset")&&req.method()==="POST");
    await reset.click();
    const body=(await saved).postDataJSON();
    assert.equal(body.id,3);
    await page.getByText("用户双重认证已重置").waitFor();
  }finally{await context.close();}
});

test("administrator can create a device-group connection access rule",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="accessRules"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="连接访问规则");
    await page.getByText("Helpdesk access",{exact:true}).first().waitFor();
    await page.locator('button[data-action="create"]').click();
    const editor=page.locator('#edit-form');
    await editor.locator('input[name="name"]').fill("Shared support group");
    await editor.locator('select[name="source_type"]').selectOption("group");
    const sourcePicker=editor.locator('.entity-picker[data-picker-field="source_id"]');
    await sourcePicker.locator('.picker-query').fill("帮助台用户组");
    await sourcePicker.locator('button[data-action="picker-search"]').click();
    await sourcePicker.locator('button[data-action="picker-select"][data-id="2"]').click();
    await editor.locator('select[name="target_type"]').selectOption("device_group");
    const targetPicker=editor.locator('.entity-picker[data-picker-field="target_id"]');
    await targetPicker.locator('.picker-query').fill("实验室设备组");
    await targetPicker.locator('button[data-action="picker-search"]').click();
    await targetPicker.locator('button[data-action="picker-select"][data-id="3"]').click();
    await editor.locator('select[name="effect"]').selectOption("allow");
    await editor.locator('input[name="enabled"]').check();
    const sending=page.waitForRequest(req=>req.url().endsWith("/api/admin/access_rule/create")&&req.method()==="POST");
    await page.locator('button[data-action="submit-access-rule"]').click();
    const body=(await sending).postDataJSON();
    assert.equal(body.name,"Shared support group");
    assert.equal(body.source_type,"group");
    assert.equal(body.source_id,2);
    assert.equal(body.target_type,"device_group");
    assert.equal(body.target_id,"3");
    assert.equal(body.effect,"allow");
    assert.equal(body.enabled,true);
    await editor.waitFor({state:"detached"});
  }finally{await context.close();}
});


test("administrator can configure Windows installers and inspect email delivery on desktop",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="clientBuilder"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="客户端安装包构建");
    await page.getByText("Office Windows",{exact:true}).first().waitFor();
    assert.match(await page.locator(".content").innerText(),/安装包已准备好/);
    assert.match(await page.locator(".content").innerText(),/处理失败/);
    fs.mkdirSync(path.join(__dirname,"test-screenshots"),{recursive:true});
    await page.screenshot({path:path.join(__dirname,"test-screenshots","client-builder-desktop.png"),fullPage:true});

    await page.locator('input[name="name"]').fill("Support group");
    await page.locator('input[name="host"]').fill("support.example.test:21116");
    await page.locator('input[name="public_key"]').fill("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=");
    await page.locator('input[name="api_server"]').fill("https://support.example.test");
    const profileForm=page.locator("#client-profile-form");
    const invalid=await profileForm.evaluate(form=>[...form.elements].filter(
      field=>!field.checkValidity()).map(field=>field.name));
    assert.deepEqual(invalid,[],"client installer profile inputs must be valid");
    const saved=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/client_builder/profiles")&&req.method()==="POST");
    await profileForm.locator('button[data-action="client-save"]').click();
    const payload=(await saved).postDataJSON();
    assert.equal(payload.platform,"windows-x64");
    assert.equal(payload.host,"support.example.test:21116");
    assert.equal(payload.api_server,"https://support.example.test");
    assert.equal(payload.version,"1.5.1");
    await page.getByText("客户端配置已保存").first().waitFor();
    await page.locator('nav.side-scroll button[data-view="notifications"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="邮件事件通知");
    await page.getByText("temporary SMTP failure").first().waitFor();
    assert.equal(await page.locator('.table-scroll button[data-action="notification-retry"]').count(),1);
    await page.screenshot({path:path.join(__dirname,"test-screenshots","notifications-desktop.png"),fullPage:true});
  }finally{await context.close();}
});

test("client builder and email notifications remain readable on phone and tablet",async()=>{
  const sizes=[
    {width:375,height:812,label:"mobile"},
    {width:768,height:1024,label:"tablet"}
  ];
  for(const {width,height,label} of sizes){
    const {context,page}=await pageForViewport(width,height,true);
    try{
      await page.locator('button[data-action="mobile-open"]').click();
      await page.locator('nav.side-scroll button[data-view="clientBuilder"]').click();
      await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="客户端安装包构建");
      await page.getByText("Office Windows",{exact:true}).first().waitFor();
      if(width<=600)await waitForMobileBreadcrumb(page);
      const dimensions=await page.evaluate(()=>({
        scroll:document.documentElement.scrollWidth,width:window.innerWidth
      }));
      assert.ok(dimensions.scroll<=dimensions.width+2,
        label+" client builder has horizontal page overflow: "+JSON.stringify(dimensions));
      await page.screenshot({
        path:path.join(__dirname,"test-screenshots","client-builder-"+label+".png"),fullPage:true
      });
      await page.locator('button[data-action="mobile-open"]').click();
      await page.locator('nav.side-scroll button[data-view="notifications"]').click();
      await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="邮件事件通知");
      assert.match(await page.locator(".content").innerText(),/发送记录/);
      if(width<=600)await waitForMobileBreadcrumb(page);
      const pageWidth=await page.evaluate(()=>({
        scroll:document.documentElement.scrollWidth,width:window.innerWidth
      }));
      assert.ok(pageWidth.scroll<=pageWidth.width+2,
        label+" notifications have horizontal page overflow: "+JSON.stringify(pageWidth));
      await page.screenshot({
        path:path.join(__dirname,"test-screenshots","notifications-"+label+".png"),fullPage:true
      });
    }finally{await context.close();}
  }
});


test("ordinary users can choose delivery providers, subscribe to events, and send a test",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.locator('nav.side-scroll button[data-view="push"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="消息推送");
    await page.getByText("家庭通知",{exact:true}).first().waitFor();
    assert.match(await page.locator(".content").innerText(),/MessagePusher/);
    assert.match(await page.locator(".content").innerText(),/SMTP 邮件/);
    assert.match(await page.locator(".content").innerText(),/Server酱 Turbo/);

    const form=page.locator("#push-channel-form");
    await form.locator('select[name="kind"]').selectOption("smtp");
    assert.equal(await form.locator('[data-push-kind="smtp"]').isVisible(),true);
    assert.equal(await form.locator('[data-push-kind="messagepusher"]').isVisible(),false);
    await form.locator('select[name="kind"]').selectOption("serverchan");
    assert.equal(await form.locator('[data-push-kind="serverchan"]').isVisible(),true);
    await form.locator('select[name="kind"]').selectOption("messagepusher");
    await form.locator('input[name="name"]').fill("新通知");
    await form.locator('input[name="mp_base_url"]').fill("https://push.example.test");
    await form.locator('input[name="mp_api_key"]').fill("mpk_fixture-key");
    await form.locator('input[name="event"][value="device.online"]').check();
    const saving=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/push/channels")&&req.method()==="POST");
    await form.locator('button[type="submit"]').click();
    const payload=(await saving).postDataJSON();
    assert.equal(payload.kind,"messagepusher");
    assert.equal(payload.api_key,"mpk_fixture-key");
    assert.ok(payload.events.includes("device.online"));
    await page.getByText("推送渠道已保存").first().waitFor();
    const sending=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/push/channels/test")&&req.method()==="POST");
    await page.locator('button[data-action="push-test"]').first().click();
    const request=(await sending).postDataJSON();
    assert.equal(request.id,5);
    await page.getByText("测试通知已加入发送队列").first().waitFor();
    fs.mkdirSync(path.join(__dirname,"test-screenshots"),{recursive:true});
    await page.screenshot({path:path.join(__dirname,"test-screenshots","personal-push-desktop.png"),fullPage:true});
  }finally{await context.close();}
});

test("device and audit IP addresses have GeoIP explanations on desktop and mobile",async()=>{
  const administrator=await pageForViewport(1440,900,true);
  try{
    await administrator.page.locator('nav.side-scroll button[data-view="operations"]').click();
    await administrator.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="管理操作记录");
    await administrator.page.getByText("192.0.2.1",{exact:false}).first().waitFor();
    assert.match(await administrator.page.locator(".content").innerText(),/位置未知/);
  }finally{await administrator.context.close();}
  const owner=await pageForViewport(375,812,false);
  try{
    await owner.page.locator('button[data-action="mobile-open"]').click();
    await owner.page.locator('nav.side-scroll button[data-view="myloginlogs"]').click();
    await owner.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的登录记录");
    await owner.page.waitForFunction(()=>{
      const text=document.querySelector(".content")?.innerText||"";
      return text.includes("10.0.0.5")&&text.includes("局域网地址");
    });
    const dimensions=await owner.page.evaluate(()=>({
      width:window.innerWidth,scroll:document.documentElement.scrollWidth
    }));
    assert.ok(dimensions.scroll<=dimensions.width+2,"GeoIP display overflows mobile screen");
  }finally{await owner.context.close();}
});


test("online and offline dots appear beside devices on desktop and mobile",async()=>{
  const admin=await pageForViewport(1440,900,true);
  try {
    await admin.page.locator('nav.side-scroll button[data-view="devices"]').click();
    await admin.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备管理");
    await admin.page.getByText("office",{exact:true}).first().waitFor();
    const row=admin.page.locator(".table-scroll tbody tr");
    assert.equal(await row.count(),2);
    assert.equal(await row.nth(0).locator(".device-status-dot.online").count(),1);
    assert.equal(await row.nth(1).locator(".device-status-dot.offline").count(),1);
    assert.equal(await admin.page.getByRole("columnheader",{name:"在线状态"}).count(),0);
  } finally { await admin.context.close(); }

  const regular=await pageForViewport(375,812,false);
  try {
    await regular.page.locator('button[data-action="mobile-open"]').click();
    await regular.page.locator('nav.side-scroll button[data-view="mydevices"]').click();
    await regular.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的设备");
    const list=regular.page.locator(".mobile-item");
    await list.first().waitFor();
    assert.equal(await list.nth(0).locator(".device-status-dot.online").count(),1);
    assert.equal(await list.nth(1).locator(".device-status-dot.offline").count(),1);
    const size=await regular.page.evaluate(()=>({
      scroll:document.documentElement.scrollWidth,
      width:window.innerWidth
    }));
    assert.ok(size.scroll<=size.width+2,"presence dots must not overflow the phone");
  } finally {await regular.context.close();}
});

test("settings center: administrator can inspect and save overrides on desktop", async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="settings"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设置中心");
    await page.locator('[data-setting-item="app.register"]').waitFor({state:"visible"});
    assert.match(await page.locator(".content").innerText(),/环境变量/);
    assert.match(await page.locator(".content").innerText(),/smtp.example.test/);
    assert.match(await page.locator(".content").innerText(),/连接日志 90 天/);
    assert.match(await page.locator(".content").innerText(),/已配置（隐藏具体内容）/);
    assert.equal(await page.locator('[data-setting-item="jwt.key"] input').getAttribute("type"),"password");
    assert.ok(!(await page.locator(".content").innerText()).includes("private-jwt-test-value"));
    await page.locator('[id="setting-value-app.register"]').selectOption("false");
    const saved=page.waitForRequest(req=>req.url().endsWith("/api/admin/settings/overrides")&&req.method()==="PATCH");
    await page.locator('button[data-action="setting-save"][data-id="app.register"]').click();
    const payload=(await saved).postDataJSON();
    assert.equal(payload.changes[0].key,"app.register");
    assert.equal(payload.changes[0].value,"false");
    assert.equal(payload.database_version,1);
    await page.getByText(/已保存覆盖设置/).waitFor();
    fs.mkdirSync(path.join(__dirname,"test-screenshots"),{recursive:true});
    await page.screenshot({path:path.join(__dirname,"test-screenshots","settings-desktop.png"),fullPage:true});
  }finally{await context.close();}
});

test("settings center: administrator inspects and retries failed email events",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="settings"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设置中心");
    await page.locator('[data-action="settings-outbox-category"][data-category="notification"]').click();
    const failure=page.locator('[data-action="settings-outbox-retry"][data-category="notification"]');
    await failure.waitFor({state:"visible"});
    const retry=page.waitForRequest(req=>
      req.url().endsWith("/api/admin/settings/outbox/retry")&&req.method()==="POST");
    await failure.click();
    const request=(await retry).postDataJSON();
    assert.equal(request.category,"notification");
    assert.equal(request.id,58);
  }finally{await context.close();}
});

test("settings center: mobile and tablet navigation do not overflow",async()=>{
  for(const [width,height,label] of [[375,812,"mobile"],[768,1024,"tablet"]]){
    const {context,page}=await pageForViewport(width,height,true);
    try{
      await page.locator('button[data-action="mobile-open"]').click();
      await page.locator("aside.sidebar.open").waitFor();
      await page.locator('nav.side-scroll button[data-view="settings"]').click();
      await page.locator('[data-setting-item="app.register"]').waitFor({state:"visible"});
      const dims=await page.evaluate(()=>({document:document.documentElement.scrollWidth,viewport:innerWidth}));
      assert.ok(dims.document<=dims.viewport+2, label+" settings overflow: "+JSON.stringify(dims));
      const card=await page.locator('[data-setting-item="app.register"]').boundingBox();
      assert.ok(card&&card.width<=width+2,label+" settings card exceeds viewport");
      fs.mkdirSync(path.join(__dirname,"test-screenshots"),{recursive:true});
      await page.screenshot({path:path.join(__dirname,"test-screenshots","settings-"+label+".png"),fullPage:true});
    }finally{await context.close();}
  }
});

test("README gallery captures current administration pages at desktop and mobile sizes", async()=>{
  const desktop=await pageForViewport(1440,900,true);
  const folder=path.join(__dirname,"test-screenshots");
  fs.mkdirSync(folder,{recursive:true});
  try{
    for(const [section,title,filename] of [
      ["overview","控制台概览","overview-preview-desktop.png"],
      ["devices","设备管理","devices-preview-desktop.png"],
      ["settings","设置中心","settings-preview-desktop.png"],
    ]){
      if(section!=="overview")await desktop.page.locator('nav.side-scroll button[data-view="'+section+'"]').click();
      await desktop.page.waitForFunction(name=>document.querySelector("h1")?.textContent===name,title);
      await desktop.page.screenshot({path:path.join(folder,filename),fullPage:false,animations:"disabled"});
    }
  }finally{await desktop.context.close()}

  const mobile=await pageForViewport(375,812,true);
  try{
    await mobile.page.screenshot({
      path:path.join(folder,"overview-preview-mobile.png"),fullPage:false,animations:"disabled"
    });
    await mobile.page.locator('button[data-action="mobile-open"]').click();
    await mobile.page.locator('nav.side-scroll button[data-view="devices"]').click();
    await mobile.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备管理");
    await mobile.page.screenshot({
      path:path.join(folder,"devices-preview-mobile.png"),fullPage:false,animations:"disabled"
    });
    await mobile.page.locator('button[data-action="mobile-open"]').click();
    await mobile.page.locator('nav.side-scroll button[data-view="settings"]').click();
    await mobile.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设置中心");
    await mobile.page.locator('[data-setting-item="app.register"]').waitFor({state:"visible"});
    await mobile.page.locator('[data-setting-item="app.register"]').scrollIntoViewIfNeeded();
    const width=await mobile.page.evaluate(()=>({
      page:document.documentElement.scrollWidth,viewport:window.innerWidth
    }));
    assert.ok(width.page<=width.viewport+2,"settings mobile preview overflows viewport");
    await mobile.page.screenshot({
      path:path.join(folder,"settings-preview-mobile.png"),fullPage:false,animations:"disabled"
    });
  }finally{await mobile.context.close()}
});

test("layout: brand, shortcuts and settings are arranged consistently",async()=>{
  for(const [width,height] of [[1440,900],[820,1000],[375,812],[320,700]]){
    const {context,page}=await pageForViewport(width,height,true);
    try{
      if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
      const brand=await page.evaluate(()=>{
        const sidebar=document.querySelector(".sidebar");
        const name=document.querySelector(".side-head .side-name")?.getBoundingClientRect();
        const caption=document.querySelector(".side-head .side-sub")?.getBoundingClientRect();
        return {
          nameBottom:name?.bottom,captionTop:caption?.top,
          captionRight:caption?.right,sidebarRight:sidebar?.getBoundingClientRect().right
        };
      });
      assert.ok(brand.nameBottom<=brand.captionTop+1,"brand lines must not overlap at "+width+"px");
      assert.ok(brand.captionRight<=brand.sidebarRight-7,"brand caption must fit its sidebar at "+width+"px");
      if(width<=800)await page.locator('nav.side-scroll button[data-view="overview"]').click();
      const shortcuts=await page.locator(".quick-card").evaluateAll(cards=>cards.map(card=>{
        const boxes=[".quick-icon",".quick-label",".quick-arrow"].map(sel=>card.querySelector(sel)?.getBoundingClientRect());
        const center=box=>box?.top+box?.height/2;
        return {
          verticalDifference:Math.max(...boxes.map(center))-Math.min(...boxes.map(center)),
          textFits:card.querySelector(".quick-label").scrollWidth<=card.querySelector(".quick-label").clientWidth+1
        };
      }));
      assert.equal(shortcuts.length,4);
      assert.ok(shortcuts.every(x=>x.verticalDifference<=4&&x.textFits),
        "shortcut icon, title and arrow must align at "+width+"px");
      if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
      await page.locator('nav.side-scroll button[data-view="settings"]').click();
      await page.locator('[data-setting-item="app.register"]').waitFor({state:"visible"});
      const layout=await page.evaluate(()=>{
        const section=document.querySelector(".settings-section");
        const body=section.querySelector(".settings-section-body");
        const sectionBox=section.getBoundingClientRect();
        const bodyBox=body.querySelector(".settings-section-intro").getBoundingClientRect();
        const links=[...section.querySelectorAll(".settings-feature-link")];
        return {
          width:document.documentElement.scrollWidth,viewport:innerWidth,
          leftInset:bodyBox.left-sectionBox.left,
          rightInset:sectionBox.right-bodyBox.right,
          links:links.length,
          outside:links.some(link=>{
            const rect=link.getBoundingClientRect();
            return rect.left<sectionBox.left+8||rect.right>sectionBox.right-8;
          }),
          summaries:section.querySelectorAll(".settings-highlight").length,
          notificationCards:document.querySelectorAll(".settings-outbox-card").length
        };
      });
      assert.ok(layout.width<=layout.viewport+2,"page must fit width "+width+"px");
      assert.ok(layout.leftInset>=14&&layout.rightInset>=14,"settings body needs inner spacing");
      assert.equal(layout.links,12);
      assert.equal(layout.outside,false,"settings links must remain inside their panel");
      assert.equal(layout.summaries,4);
      assert.equal(layout.notificationCards,2);
    }finally{await context.close();}
  }
});

test("font audit: verify actual Chinese glyph font",async()=>{
 if(process.env.RUSTDESK_REQUIRE_CJK_FONT!=="1")return;
 const {context,page}=await pageForViewport(1440,900,true);
 try{
   const cdp=await context.newCDPSession(page);
   await cdp.send("DOM.enable");await cdp.send("CSS.enable");
   const {root}=await cdp.send("DOM.getDocument");
   const {nodeId}=await cdp.send("DOM.querySelector",{nodeId:root.nodeId,selector:".page-head h1"});
   const data=await cdp.send("CSS.getPlatformFontsForNode",{nodeId});
   assert.ok(data.fonts.some(font=>font.glyphCount>0&&font.familyName.includes("Noto Sans CJK SC")),
      "Chromium must render Chinese text with Noto Sans CJK SC: "+JSON.stringify(data));
 }finally{await context.close()}
});



if(process.env.RUSTDESK_VISUAL_AUDIT==="1"){
test("visual audit: capture all administration pages", async()=>{
  const folder=path.join(__dirname,"test-screenshots","audit");
  fs.mkdirSync(folder,{recursive:true});
  const results=[];
  for(const [role,width,height,asAdmin] of [["admin-desktop",1440,900,true],["admin-tablet",820,1000,true],["admin-phone",375,812,true],["user-phone",375,812,false]]){
    const {context,page}=await pageForViewport(width,height,asAdmin);
    try{
      const navigation=await page.locator('nav.side-scroll button[data-view]').evaluateAll(list=>
        [...new Set(list.map(el=>el.dataset.view).filter(Boolean))]);
      for(const view of navigation){
        try{
          if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
          await page.locator('nav.side-scroll button[data-view="'+view+'"]').first().click();
          await page.waitForTimeout(100);
          const info=await page.evaluate(()=>{
            const main=document.querySelector(".main");
            const content=document.querySelector(".content");
            const panelHeaders=[...document.querySelectorAll(".panel-header")];
            return {
              title: document.querySelector(".page-head h1")?.textContent??"",
              documentWidth:document.documentElement.scrollWidth,viewport:window.innerWidth,
              contentWidth:Math.round(content?.getBoundingClientRect().width??0),
              viewportOverflow:document.documentElement.scrollWidth>window.innerWidth+2,
              clipped: [...document.querySelectorAll(".main .panel,.main .quick-card,.main .metric,.main .btn,.main .status-row")]
                 .filter(el=>{const st=getComputedStyle(el),rect=el.getBoundingClientRect();return st.display!=="none"&&rect.width>0&&rect.right>window.innerWidth+4&&!el.closest(".table-scroll")})
                 .slice(0,10).map(el=>el.tagName.toLowerCase()+"."+el.className),
              panelCount:document.querySelectorAll(".panel").length,
              panelHeaderPadding:panelHeaders.slice(0,4).map(el=>getComputedStyle(el).padding),
              mainWidth:main?.scrollWidth
            };
          });
          const file=role+"-"+view+".png";
          await page.screenshot({path:path.join(folder,file),fullPage:false,animations:"disabled"});
          if(["settings","overview","notifications","mail","push","profile","clientBuilder"].includes(view)&&role==="admin-desktop")
            await page.screenshot({path:path.join(folder,role+"-"+view+"-full.png"),fullPage:true,animations:"disabled"});
          results.push({role,view,...info,file});
        }catch(error){results.push({role,view,error:String(error).slice(0,220)});}
      }
    }finally{await context.close();}
  }
  fs.writeFileSync(path.join(folder,"layout-report.json"),JSON.stringify(results,null,2));
  const defects=results.filter(item=>item.error||item.viewportOverflow||item.clipped?.length);
  assert.equal(defects.length,0,JSON.stringify(defects.slice(0,8)));
  console.log("Captured "+results.length+" page screenshots across desktop, tablet and phone layouts.");
});


test("quality audit: inspect layout, controls, text and fonts",async()=>{
 const out=path.join(__dirname,"test-screenshots","quality-audit");
 fs.mkdirSync(out,{recursive:true});
 const results=[];
 const screens=[["desktop",1440,900,true],["laptop",1100,800,true],["tablet",768,1000,true],["phone",375,812,true],["small-phone",320,700,true],["member-phone",375,812,false]];
 for(const [role,width,height,admin] of screens){
  const {page,context}=await pageForViewport(width,height,admin);
  try{
   const views=await page.locator('nav.side-scroll button[data-view]').evaluateAll(nodes=>[...new Set(nodes.map(e=>e.dataset.view))]);
   for(const view of views){
    try{
     if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
     await page.locator('nav.side-scroll button[data-view="'+view+'"]').first().click();
     await page.waitForTimeout(60);
     const metrics=await page.evaluate(()=>{
      function rect(node){const r=node.getBoundingClientRect();return {x:r.x,y:r.y,w:r.width,h:r.height,r:r.right,b:r.bottom};}
      function visible(el){const st=getComputedStyle(el),r=el.getBoundingClientRect();return !el.hidden && st.visibility!=="hidden" && st.display!=="none" &&r.width>0&&r.height>0;}
      const formGaps=[];
      const forms=[...document.querySelectorAll('form')].filter(visible);
      for(const f of forms){
       const children=[...f.children].filter(e=>visible(e)&&!["hidden"].includes(e.type));
       for(let i=0;i<children.length-1;i++){
        const a=children[i],b=children[i+1],r=rect(a),s=rect(b);
        const overlap=Math.min(r.r,s.r)-Math.max(r.x,s.x);
        if(overlap>Math.min(r.w,s.w)*.4&&s.y>=r.b-1&&s.y-r.b<8)
         formGaps.push({form:f.id||f.className,from:a.tagName+"."+a.className,to:b.tagName+"."+b.className,gap:Math.round((s.y-r.b)*10)/10});
       }
      }
      const controlGaps=[];
      for(const f of forms){
       const els=[...f.querySelectorAll('button,input:not([type="hidden"]),textarea,select')].filter(visible);
       for(let i=0;i<els.length;i++)for(let j=i+1;j<els.length;j++){
        const a=els[i],b=els[j],r=rect(a),s=rect(b);
        if(!(a.matches('button')||b.matches('button')))continue;
        const overlapY=Math.min(r.b,s.b)-Math.max(r.y,s.y);
        const overlapX=Math.min(r.r,s.r)-Math.max(r.x,s.x);
        if(overlapY>Math.min(r.h,s.h)*.65){
           const gap=Math.max(r.x,s.x)-Math.min(r.r,s.r);
           if(gap>=0&&gap<8)controlGaps.push({parent:f.id,a:a.tagName,b:b.tagName,gap:Math.round(gap)});
        }else if(overlapX>Math.min(r.w,s.w)*.65){
           const gap=Math.max(r.y,s.y)-Math.min(r.b,s.b);
           if(gap>=0&&gap<8)controlGaps.push({parent:f.id,a:a.tagName,b:b.tagName,gap:Math.round(gap)});
        }
       }
      }
      const wrapped=[];
      for(const btn of [...document.querySelectorAll('button')].filter(visible)){
       const first=[...btn.childNodes].filter(n=>n.nodeType===Node.TEXT_NODE&&n.textContent.trim());
       const allRects=[];
       for(const n of first){const range=document.createRange();range.selectNodeContents(n);allRects.push(...[...range.getClientRects()])}
       const ys=[...new Set(allRects.filter(x=>x.width>0).map(r=>Math.round(r.y)))];
       if(ys.length>1)wrapped.push({label:btn.innerText.slice(0,50),lines:ys.length,cls:btn.className});
      }
      const clips=[...document.querySelectorAll('.page-head h1,.panel-title,.btn,.nav-link,.field label,.settings-highlight h3')].filter(visible)
         .filter(e=>e.scrollWidth>e.clientWidth+3).slice(0,20).map(e=>({text:e.textContent?.slice(0,32),cls:e.className,w:e.clientWidth,scroll:e.scrollWidth}));
      return {title:document.querySelector('h1')?.textContent||'',
        pageWidth:document.documentElement.scrollWidth,viewport:innerWidth,
        fonts:getComputedStyle(document.querySelector('h1')||document.body).fontFamily,
        formCount:forms.length,formGaps,controlGaps,wrapped,clips};
     });
     if(view==="overview"||view==="settings"||view==="notifications"||view==="mail"||view==="clientBuilder"||view==="push"||view==="contacts"||view==="server"){
      await page.screenshot({path:path.join(out,role+"-"+view+".png"),fullPage:true,animations:"disabled"});
     }else{
      await page.screenshot({path:path.join(out,role+"-"+view+".png"),fullPage:false,animations:"disabled"});
     }
     results.push({role,view,...metrics});
    }catch(err){results.push({role,view,error:String(err).slice(0,300)});}
   }
  }finally{await context.close()}
 }
 const data=JSON.stringify(results,null,2);
 fs.writeFileSync(path.join(out,"report.json"),data);
 const issues=results.filter(x=>x.error||x.formGaps?.length||x.controlGaps?.length||x.wrapped?.length||x.clips?.length||x.pageWidth>x.viewport+2);
 assert.equal(issues.length,0,"Layout defects: "+JSON.stringify(issues.slice(0,8)));
 console.log("Checked "+results.length+" responsive page screenshots and control spacing.");
});



test("modal audit: inspect all creation and editing drawers",async()=>{
 const out=path.join(__dirname,"test-screenshots","quality-audit","drawers");
 fs.mkdirSync(out,{recursive:true});
 const reports=[];
 for(const [role,width,height] of [["desktop",1440,900],["phone",375,812],["small-phone",320,700]]){
  const {context,page}=await pageForViewport(width,height,true);
  try{
   const nav=await page.locator('nav.side-scroll button[data-view]').evaluateAll(nodes=>[...new Set(nodes.map(x=>x.dataset.view))]);
   for(const view of nav){
    if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
    await page.locator('nav.side-scroll button[data-view="'+view+'"]').first().click();
    const create=page.locator('.content button[data-action="create"]').first();
    if(!await create.count())continue;
    try{
     await create.click();
     const drawer=page.locator('.drawer');
     if(!await drawer.count()){reports.push({role,view,error:"no drawer"});continue;}
     const m=await page.evaluate(()=>{
      const d=document.querySelector('.drawer'),body=d.querySelector('.drawer-body'),foot=d.querySelector('.drawer-footer');
      const rect=e=>e.getBoundingClientRect(),visible=e=>e&&getComputedStyle(e).display!=="none"&&rect(e).width>0;
      const r=rect(d),rBody=rect(body),buttons=[...foot.querySelectorAll('button')].filter(visible);
      const buttonGaps=[];
      for(let i=1;i<buttons.length;i++){
       const a=rect(buttons[i-1]),b=rect(buttons[i]),gap=b.left-a.right;
       if(gap<8&&b.top<a.bottom)buttonGaps.push({gap,one:buttons[i-1].textContent,two:buttons[i].textContent});
      }
      const fields=[...d.querySelectorAll('.form-grid .field')].filter(visible);
      const widths=fields.filter(e=>rect(e).right>r.right-6||rect(e).left<r.left+6).slice(0,4).map(e=>e.textContent.slice(0,30));
      return{width:r.width,viewport:innerWidth,right:r.right,bodyInset:rBody.left-r.left,
       footerButtonGaps:buttonGaps,fieldOverflow:widths,fields:fields.length,
       documentWidth:document.documentElement.scrollWidth,
       scrollHeight:d.scrollHeight,viewportHeight:innerHeight};
     });
     await page.screenshot({path:path.join(out,role+"-"+view+".png"),fullPage:false,animations:"disabled"});
     reports.push({role,view,...m});
     await page.locator('.drawer-head button[data-action="close-modal"]').click();
    }catch(err){reports.push({role,view,error:String(err).slice(0,250)});}
   }
  }finally{await context.close()}
 }
 fs.writeFileSync(path.join(out,"report.json"),JSON.stringify(reports,null,2));
 const issues=reports.filter(x=>x.error||x.fieldOverflow?.length||x.footerButtonGaps?.length||x.documentWidth>x.viewport+2||x.right>x.viewport+3);
 assert.equal(issues.length,0,"Drawer layout defects: "+JSON.stringify(issues.slice(0,8)));
 console.log("Checked "+reports.length+" creation drawer screenshots.");
});


}

if(process.env.RUSTDESK_VISUAL_AUDIT==="1"){
test("responsive data tables keep columns readable across screen sizes",async()=>{
  const folder=path.join(__dirname,"test-screenshots","table-layout");
  fs.mkdirSync(folder,{recursive:true});
  const report=[];
  for(const [role,width,height,isAdmin] of [
    ["desktop",1440,900,true],
    ["laptop",1024,800,true],
    ["tablet",820,1000,true],
    ["small-tablet",768,1024,true],
    ["phone",375,812,true],
    ["small-phone",320,700,true],
    ["member-phone",375,812,false]
  ]){
    const {context,page}=await pageForViewport(width,height,isAdmin);
    try{
      const keys=await page.locator('nav.side-scroll button[data-view]').evaluateAll(links=>
        [...new Set(links.map(link=>link.dataset.view).filter(Boolean))]);
      for(const key of keys){
        if(width<=800)await page.locator('button[data-action="mobile-open"]').click();
        await page.locator('nav.side-scroll button[data-view="'+key+'"]').first().click();
        await page.waitForFunction(()=>!document.querySelector(".loading-table"),null,{timeout:10000});
        const table=page.locator(".data-table");
        if(await table.count()===0)continue;
        const result=await page.evaluate(()=>{
          const table=document.querySelector(".data-table");
          const region=table.closest(".table-scroll");
          const cards=table.parentElement.nextElementSibling?.classList.contains("table-scroll-hint")?
            table.parentElement.nextElementSibling.nextElementSibling:
            table.parentElement.nextElementSibling;
          const visibleTable=getComputedStyle(region).display!=="none";
          const headings=[...table.querySelectorAll("thead th")];
          const cells=[...table.querySelectorAll("tbody tr:first-child td")];
          const getWidth=element=>element.getBoundingClientRect().width;
          const firstColumn=table.querySelector("colgroup col");
          const direction=table.querySelector('tbody td[data-column="direction"] span');
          const device=table.querySelector('tbody td[data-column="id"] span, tbody td[data-column="device_id"] span');
          const oneLine=(span)=>{
            if(!span)return true;
            const s=getComputedStyle(span);
            const lineHeight=parseFloat(s.lineHeight)||20;
            return span.getBoundingClientRect().height<=lineHeight*1.4;
          };
          const narrow=cells.filter(td=>{
            const field=td.getAttribute("data-column");
            const value=td.textContent.trim();
            if(td.classList.contains("cell-multiline")||td.classList.contains("cell-actions"))return false;
            return value.length>=4&&getWidth(td)<100;
          }).map(td=>({field:td.getAttribute("data-column"),value:td.textContent.trim(),width:Math.round(getWidth(td))}));
          const expectedWidth=parseFloat(table.style.getPropertyValue("--table-min-width"))||0;
          const cardsVisible=getComputedStyle(cards).display!=="none";
          const cardItems=[...cards.querySelectorAll(".mobile-item")];
          const mobileField=cardItems[0]?.querySelector("dd");
          return{
            pageWidth:document.documentElement.scrollWidth,
            viewport:innerWidth,
            tableVisible:visibleTable,
            cardsVisible,
            columns:headings.length,
            columnWidth:Math.round(getWidth(firstColumn)),
            minWidth:expectedWidth,
            renderedTableWidth:Math.round(getWidth(table)),
            regionWidth:Math.round(getWidth(region)),
            cardCount:cardItems.length,
            mobileFieldWidth:mobileField?Math.round(getWidth(mobileField)):0,
            narrow,nonwrappingDirection:oneLine(direction),nonwrappingDevice:oneLine(device),
            firstCellText:cells[0]?.textContent.trim()||"",
            hasScrollableRegion:region.scrollWidth>region.clientWidth+1
          };
        });
        assert.ok(result.pageWidth<=result.viewport+2,
          "Page "+key+" overflows at "+role+": "+JSON.stringify(result));
        assert.equal(result.tableVisible,width>900,
          "Unexpected table visibility on "+role+" "+key);
        if(width>900){
          assert.equal(result.narrow.length,0,
            "Narrow table cells on "+role+" "+key+": "+JSON.stringify(result.narrow));
          assert.ok(result.renderedTableWidth>=result.minWidth-2,
            "Table lost its required minimum width on "+role+" "+key);
          assert.ok(result.nonwrappingDirection,
            "Connection direction wrapped on "+role+" "+key);
          assert.ok(result.nonwrappingDevice,
            "Device identifier wrapped on "+role+" "+key);
          if(result.minWidth>result.regionWidth+2)
            assert.ok(result.hasScrollableRegion,"Wide table cannot scroll on "+role+" "+key);
          const cellWidths=await page.locator(".data-table tbody tr:first-child td:not(.cell-actions)").evaluateAll(cells=>
            cells.map(cell=>({label:cell.dataset.column,width:cell.getBoundingClientRect().width})));
          assert.ok(cellWidths.every(item=>item.width>=78),
            "Found compressed columns in "+role+" "+key+": "+JSON.stringify(cellWidths));
        }else{
          assert.ok(result.cardsVisible&&result.cardCount>0,
            "Detail cards are missing on "+role+" "+key);
          assert.ok(result.mobileFieldWidth>=88,
            "Mobile field content is too narrow on "+role+" "+key);
        }
        await page.mouse.move(width-8,height-8);
        const file=role+"-"+key+".png";
        await page.screenshot({
          path:path.join(folder,file),fullPage:width<=900,animations:"disabled"
        });
        if(result.tableVisible&&result.hasScrollableRegion){
          const scroller=page.locator(".table-scroll");
          if(result.renderedTableWidth>result.regionWidth*1.8){
            await scroller.evaluate(element=>{
              element.scrollLeft=(element.scrollWidth-element.clientWidth)/2;
            });
            await page.screenshot({
              path:path.join(folder,role+"-"+key+"-middle.png"),animations:"disabled"
            });
          }
          const end=await scroller.evaluate(element=>{
            element.scrollLeft=element.scrollWidth;
            return element.scrollLeft;
          });
          assert.ok(end>0,"Cannot scroll to the last column of "+key);
          await page.screenshot({
            path:path.join(folder,role+"-"+key+"-end.png"),animations:"disabled"
          });
        }
        report.push({role,view:key,...result,file});
      }
    }finally{await context.close()}
  }
  fs.writeFileSync(path.join(folder,"table-layout-report.json"),JSON.stringify(report,null,2));
  assert.ok(report.length>=100,
    "Expected to inspect every populated table in different viewport sizes");
  console.log("Verified "+report.length+" responsive table screenshots.");
});
}

test("navigation groups remain stable while opening lower items",async()=>{
  const {context,page}=await pageForViewport(1440,850,true);
  try{
    const sidebar=page.locator(".side-scroll");
    const headings=await page.locator(".side-section-toggle").allTextContents();
    assert.ok(headings.includes("设备管理"));
    assert.ok(headings.includes("审计管理"));
    await page.locator('button[data-section="系统管理"]').click();
    assert.equal(await page.locator('button[data-section="系统管理"]').getAttribute("aria-expanded"),"false");
    await page.locator('button[data-section="系统管理"]').click();
    await sidebar.evaluate(node=>node.scrollTop=node.scrollHeight);
    const before=await sidebar.evaluate(node=>node.scrollTop);
    await page.locator('nav button[data-view="settings"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设置中心");
    const after=await sidebar.evaluate(node=>node.scrollTop);
    assert.ok(after>before-65,"navigation scroll must remain at the selected lower section");
  }finally{await context.close();}
});

test("device inventories have separate owner and administration navigation",async()=>{
  const owner=await pageForViewport(1360,850,false);
  const administrator=await pageForViewport(1360,850,true);
  try{
    assert.equal(await owner.page.locator('nav button[data-view="devices"]').count(),0);
    assert.equal(await owner.page.locator('nav button[data-view="mydevices"]').count(),1);
    assert.equal(await administrator.page.locator('nav button[data-view="devices"]').count(),1);
    assert.equal(await administrator.page.locator('nav button[data-view="mydevices"]').count(),1);
    await owner.page.locator('nav button[data-view="mydevices"]').click();
    await owner.page.waitForFunction(()=>document.querySelector("h1")?.textContent==="我的设备");
  }finally{await owner.context.close();await administrator.context.close();}
});

test("administrator group editing sends update endpoint and table headings sort",async()=>{
  const {context,page}=await pageForViewport(1360,850,true);
  try{
    await page.locator('nav button[data-view="groups"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="用户分组");
    const heading=page.locator('button[data-action="sort-table"][data-column="name"]');
    await heading.click();
    assert.equal(await page.locator('th[data-column="name"]').getAttribute("aria-sort"),"ascending");
    await heading.click();
    assert.equal(await page.locator('th[data-column="name"]').getAttribute("aria-sort"),"descending");
    await page.locator('button[data-action="edit"]').first().click();
    const form=page.locator("#edit-form");
    await form.waitFor({state:"visible"});
    await form.locator('[name="name"]').fill("更新后的分组");
    let requestPath="";
    await page.route("**/api/admin/group/update",async route=>{
      requestPath=new URL(route.request().url()).pathname;
      await route.fulfill({status:200,contentType:"application/json",body:JSON.stringify({code:0,data:{}})});
    });
    await page.locator('[type="submit"][form="edit-form"]').click();
    await page.waitForTimeout(300);
    assert.equal(requestPath,"/api/admin/group/update");
  }finally{await context.close();}
});

test("interactive server management presents form explanations",async()=>{
  const {context,page}=await pageForViewport(1360,850,true);
  try{
    await page.locator('nav button[data-view="server"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="服务器管理");
    await page.locator('button[data-id="id-block-add"]').click();
    const form=page.locator("#command-form");
    assert.equal(await form.locator('[name="option"]').count(),1);
    await form.locator('[name="option"]').fill("invalid-ip");
    await form.locator('button[type="submit"]').click();
    assert.match(await page.locator(".toast-wrap").innerText(),/有效的 IPv4 或 IPv6/);
  }finally{await context.close();}
});

test("device and user names open detail panels with management links",async()=>{
  const {context,page}=await pageForViewport(1440,900,true);
  try{
    await page.locator('nav.side-scroll button[data-view="devices"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="设备管理");
    const device=page.locator('.table-scroll [data-action="entity-detail"][data-entity-type="device"]').first();
    await device.click();
    await page.locator('.drawer[aria-label="设备信息"]').waitFor();
    assert.match(await page.locator(".drawer").innerText(),/设备 ID/);
    assert.equal(await page.locator(".drawer .device-status-dot").count()>0,true);
    await page.locator('[data-action="entity-open-list"]').click();
    assert.match(await page.locator("#table-search").inputValue(),/office|managed/);
    await page.locator('nav.side-scroll button[data-view="users"]').click();
    await page.waitForFunction(()=>document.querySelector("h1")?.textContent==="用户账户");
    await page.locator('.table-scroll [data-action="entity-detail"][data-entity-type="user"]').first().click();
    await page.locator('.drawer[aria-label="用户信息"]').waitFor();
    assert.match(await page.locator(".drawer").innerText(),/用户名/);
    assert.equal(await page.locator('[data-action="entity-open-list"]').count(),1);
  }finally{await context.close();}
});

test("direct entity links from WebClient open the authorized detail view",async()=>{
  const {context,page}=await pageForViewport(1440,900,false);
  try{
    await page.goto(baseUrl+"/_admin/#/overview?entity=device&value=office",{waitUntil:"networkidle"});
    await page.locator('.drawer[aria-label="设备信息"]').waitFor();
    assert.match(await page.locator(".drawer").innerText(),/office/);
  }finally{await context.close();}
});
