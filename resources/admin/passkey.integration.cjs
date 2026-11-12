"use strict";

const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const os=require("node:os");
const path=require("node:path");
const net=require("node:net");
const {spawn,spawnSync}=require("node:child_process");
const {chromium}=require("playwright");

const root=path.resolve(__dirname,"../..");
const binary=path.resolve(process.argv[2]||"/tmp/rustdesk-passkey-api");
const password="Admin#PasskeyTest2026";
let tmp,port,api,browser,proc,logfd;

async function availablePort(){
  const server=net.createServer();
  await new Promise((resolve,reject)=>{server.once("error",reject);server.listen(0,"127.0.0.1",resolve)});
  const port=server.address().port;
  await new Promise(resolve=>server.close(resolve));
  return port;
}
const delay=ms=>new Promise(resolve=>setTimeout(resolve,ms));
async function start(){
  logfd=fs.openSync(path.join(tmp,"api.log"),"a");
  proc=spawn(binary,[],{cwd:tmp,stdio:["ignore",logfd,logfd]});
  for(let n=0;n<140;n++){
    if(proc.exitCode!==null)throw Error("API start failed:\n"+fs.readFileSync(path.join(tmp,"api.log"),"utf8").slice(-4000));
    try{
      const r=await fetch(api+"/api/admin/config/admin",{signal:AbortSignal.timeout(400)});
      if(r.status===200)return;
    }catch{}
    await delay(120);
  }
  throw Error("API startup timed out:\n"+fs.readFileSync(path.join(tmp,"api.log"),"utf8").slice(-4000));
}
async function stop(){
  if(!proc)return;
  const current=proc;
  if(current.exitCode===null)current.kill("SIGTERM");
  await Promise.race([new Promise(resolve=>current.once("exit",resolve)),delay(6500)]);
  if(current.exitCode===null)current.kill("SIGKILL");
  proc=null;
  if(logfd!==undefined){fs.closeSync(logfd);logfd=undefined;}
}
async function call(route,method="GET",body,token=""){
  const response=await fetch(api+"/api/admin"+route,{method,
    headers:{"Accept":"application/json",...(body===undefined?{}:{"Content-Type":"application/json"}),
      ...(token?{"api-token":token}:{})},
    body:body===undefined?undefined:JSON.stringify(body)
  });
  const data=await response.json();
  return {status:response.status,data};
}
async function ok(route,method="GET",body,token=""){
  const r=await call(route,method,body,token);
  assert.equal(r.status,200,route+": "+JSON.stringify(r.data));
  assert.equal(r.data.code,0,route+": "+JSON.stringify(r.data));
  return r.data.data;
}
async function pageFor(token="",width=1440){
  const context=await browser.newContext({viewport:{width,height:900},locale:"zh-CN"});
  const page=await context.newPage();
  const cdp=await context.newCDPSession(page);
  await cdp.send("WebAuthn.enable",{enableUI:false});
  await cdp.send("WebAuthn.addVirtualAuthenticator",{options:{
    protocol:"ctap2",transport:"internal",hasResidentKey:true,
    hasUserVerification:true,isUserVerified:true,automaticPresenceSimulation:true
  }});
  if(token)await page.addInitScript(value=>sessionStorage.setItem("rustdesk.console.token",value),token);
  return {context,page};
}

test.before(async()=>{
  assert.ok(fs.existsSync(binary),"Missing Go API executable "+binary);
  tmp=fs.mkdtempSync(path.join(os.tmpdir(),"rustdesk-passkey-"));
  fs.cpSync(path.join(root,"conf"),path.join(tmp,"conf"),{recursive:true});
  fs.mkdirSync(path.join(tmp,"data"));
  fs.mkdirSync(path.join(tmp,"runtime"));
  fs.symlinkSync(path.join(root,"resources"),path.join(tmp,"resources"),"dir");
  port=await availablePort();api="http://127.0.0.1:"+port;
  const file=path.join(tmp,"conf/config.yaml");
  const config=fs.readFileSync(file,"utf8");
  assert.ok(config.includes('"0.0.0.0:21114"'));
  fs.writeFileSync(file,config.replace('"0.0.0.0:21114"','"127.0.0.1:'+port+'"'));
  await start();await stop();
  const reset=spawnSync(binary,["reset-admin-pwd",password],{cwd:tmp,encoding:"utf8",timeout:25000});
  assert.equal(reset.status,0,"Admin password command: "+reset.stderr);
  await start();
  browser=await chromium.launch({headless:true,args:["--no-sandbox","--disable-dev-shm-usage"]});
});
test.after(async()=>{
  if(browser)await browser.close();
  await stop();
  if(tmp)fs.rmSync(tmp,{recursive:true,force:true});
});

test("Passkey enrollment, authenticated login, replay denial and revocation",async()=>{
  const admin=await ok("/login","POST",{username:"admin",password,platform:"web"});
  assert.ok(admin.token);
  const {context,page}=await pageFor(admin.token);
  try{
    await page.goto(api+"/_admin/#/profile",{waitUntil:"networkidle"});
    await page.getByRole("heading",{name:"个人设置"}).waitFor();
    await page.getByRole("heading",{name:"Passkey 通行密钥"}).waitFor();
    const dimensions=await page.evaluate(()=>{
      const container=document.querySelector(".content").getBoundingClientRect();
      const profile=document.querySelector(".profile-summary").getBoundingClientRect();
      const cards=document.querySelector(".profile-grid").getBoundingClientRect();
      return {available:container.width,header:profile.width,cards:cards.width,
        scrollWidth:document.documentElement.scrollWidth,viewport:innerWidth};
    });
    assert.ok(dimensions.cards>dimensions.available*.8,"Profile card grid is too narrow "+JSON.stringify(dimensions));
    assert.ok(dimensions.header>dimensions.available*.8,"Profile summary is too narrow "+JSON.stringify(dimensions));
    assert.ok(dimensions.scrollWidth<=dimensions.viewport+2,"Profile overflows horizontally");
    await page.locator("#passkey-name").fill("测试设备");
    await page.locator('[data-action="passkey-register"]').click();
    try{
      await page.getByText("Passkey 已添加").waitFor({timeout:16000});
    }catch(err){
      const messages=await page.locator(".toast-wrap").allTextContents();
      throw Error("Passkey registration failed: "+JSON.stringify(messages)+"; "+err.message);
    }
    const list=await ok("/passkey/list","GET",undefined,admin.token);
    assert.equal(list.total,1);
    assert.equal(list.list[0].name,"测试设备");
    assert.doesNotMatch(JSON.stringify(list),/credential_json|public_key/i);
    fs.mkdirSync(path.join(root,"resources/admin/test-screenshots"),{recursive:true});
    await page.screenshot({path:path.join(root,"resources/admin/test-screenshots/passkey-integration.png"),animations:"disabled"});

    await page.locator('[data-action="logout"]').click();
    await page.locator('[data-action="passkey-login"]').waitFor();
    await page.locator('[data-action="passkey-login"]').click();
    await page.getByText("Passkey 登录成功").waitFor({timeout:16000});
    const session=await page.evaluate(()=>sessionStorage.getItem("rustdesk.console.token"));
    assert.ok(session&&session!==admin.token);
    assert.equal((await ok("/user/current","GET",undefined,session)).username,"admin");

    page.on("dialog",dialog=>dialog.accept());
    await page.locator('nav.side-scroll button[data-view="profile"]').click();
    await page.locator('[data-action="passkey-revoke"]').first().click();
    await page.getByText("Passkey 已移除").waitFor({timeout:9000});
    const removed=await ok("/passkey/list","GET",undefined,session);
    assert.equal(removed.total,0);

    const forgery=await call("/passkey/login/finish","POST",{
      session_id:"a".repeat(43),credential:{id:"forged",response:{}}
    });
    assert.notEqual(forgery.data.code,0,"Forged assertion must not create a session");
    const noOrigin=await call("/passkey/login/begin","POST",{});
    assert.notEqual(noOrigin.data.code,0,"HTTP requests without a trusted browser Origin must be denied");
  }finally{await context.close();}
});

test("administrators update a user's profile and securely reset credentials",async()=>{
  const admin=await ok("/login","POST",{username:"admin",password,platform:"web"});
  const username="assisted_"+Date.now().toString(36);
  const initial="First#SecurePassword2026",replacement="Second#SecurePassword2026";
  await ok("/user/create","POST",{username,password:initial,nickname:"旧昵称",
    email:"old@example.test",group_id:1,is_admin:false,status:1,remark:""},admin.token);
  const list=await ok("/user/list?keyword="+encodeURIComponent(username)+"&page=1&page_size=20","GET",undefined,admin.token);
  const account=list.list.find(u=>u.username===username);
  assert.ok(account?.id,"New account missing from the directory");
  await ok("/user/update","POST",{id:account.id,username,nickname:"新昵称",
    email:"new@example.test",group_id:1,is_admin:false,status:1,remark:"用户备注"},admin.token);
  const updated=await ok("/user/detail/"+account.id,"GET",undefined,admin.token);
  assert.equal(updated.nickname,"新昵称");assert.equal(updated.email,"new@example.test");
  assert.equal(updated.remark,"用户备注");
  const member=await ok("/login","POST",{username,password:initial,platform:"web"});
  const forbidden=await call("/user/changePwd","POST",{id:account.id,password:replacement},member.token);
  assert.notEqual(forbidden.data.code,0,"Ordinary member changed account password through administration API");

  await ok("/user/changePwd","POST",{id:account.id,password:replacement},admin.token);
  assert.notEqual((await call("/user/current","GET",undefined,member.token)).data.code,0,
    "Password reset did not terminate previous sessions");
  assert.notEqual((await call("/login","POST",{username,password:initial,platform:"web"})).data.code,0,
    "The old password can still log in");
  const restored=await ok("/login","POST",{username,password:replacement,platform:"web"});
  await ok("/user/mfa/reset","POST",{id:account.id},admin.token);
  assert.notEqual((await call("/user/current","GET",undefined,restored.token)).data.code,0,
    "Second-factor reset did not terminate current sessions");

  const {context,page}=await pageFor(admin.token);
  try{
    await page.goto(api+"/_admin/#/users",{waitUntil:"networkidle"});
    await page.locator("#table-search").fill(username);
    await page.locator('#table-search').press("Enter");
    await page.getByRole("heading",{name:"用户账户"}).waitFor();
    const actions=page.locator('.table-scroll [data-action="reset-password"]');
    await actions.first().waitFor({timeout:12000});
    assert.ok((await page.locator('.table-scroll [data-action="reset-user-mfa"]').count())>0);
    await actions.first().click();
    await page.getByText("重设用户密码").first().waitFor();
    assert.equal(await page.locator('#new-user-password').getAttribute("minlength"),"8");
  }finally{await context.close();}
});
