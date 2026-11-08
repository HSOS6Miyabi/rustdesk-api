"use strict";
(() => {
  const root = document.getElementById("app");
  if (!root) return;

  const pages = {
    overview: {name:"概览",icon:"home",section:"工作空间"},
    devices: {name:"设备管理",icon:"monitor",section:"工作空间",endpoint:"peer",search:"id",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["os","系统"],["last_online_time","最后在线"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","操作系统用户"],["os","操作系统"],["group_id","设备组 ID","number"]]},
    shared: {name:"共享设备",icon:"share",section:"工作空间",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["owner_username","设备所有者"],["sources","共享来源"],["last_online_time","最后在线"]]},
    contacts: {name:"联系人",icon:"users",section:"工作空间"},
    addressbooks: {name:"地址簿",icon:"book",section:"工作空间",endpoint:"address_book",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["username","登录账户"],["platform","平台"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","登录账户"],["password","连接密码","password"],["platform","平台"],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"],["forceAlwaysRelay","强制使用中继","checkbox"]]},
    collections: {name:"地址簿分类",icon:"folder",section:"工作空间",endpoint:"address_book_collection",columns:[["id","编号"],["name","分类名称"],["user_id","所有者 ID"]],fields:[["name","名称","text",true],["user_id","所有者 ID","number",true]]},
    tags: {name:"设备标签",icon:"tag",section:"工作空间",endpoint:"tag",columns:[["id","编号"],["name","标签名称"],["color","颜色"],["collection_id","地址簿 ID"]],fields:[["name","名称","text",true],["color","颜色（ARGB 整数）","number",true],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"]]},
    groups: {name:"用户分组",icon:"layers",section:"成员管理",endpoint:"group",admin:true,columns:[["id","编号"],["name","分组名称"],["type","分组类型"]],fields:[["name","名称","text",true],["type","类型","group_type",true]]},
    users: {name:"用户账户",icon:"users",section:"成员管理",endpoint:"user",admin:true,search:"username",columns:[["id","编号"],["username","用户名"],["nickname","昵称"],["email","邮箱"],["group_id","主要分组"],["status","状态"]],fields:[["username","用户名","text",true],["nickname","昵称"],["email","邮箱","email"],["group_id","主要分组 ID","number",true],["is_admin","管理员","checkbox"],["status","账户状态","status"],["remark","备注","textarea"]]},
    devicegroups: {name:"设备分组",icon:"grid",section:"成员管理",endpoint:"device_group",admin:true,columns:[["id","编号"],["name","名称"]],fields:[["name","名称","text",true]]},
    oauth: {name:"OIDC / SSO",icon:"key",section:"系统管理",endpoint:"oauth",admin:true,columns:[["id","编号"],["op","提供者"],["oauth_type","类型"],["issuer","颁发者"],["auto_group_ids","自动加入分组"]],fields:[["op","提供者标识"],["oauth_type","登录类型","oauth_type",true],["issuer","Issuer URL"],["scopes","Scopes"],["client_id","Client ID","text",true],["client_secret","Client Secret","password",true],["auto_register","允许自动注册","checkbox"],["auto_group_ids","自动加入的分组 ID（逗号分隔）"],["pkce_enable","启用 PKCE","checkbox"],["pkce_method","PKCE 方法"]]},
    rules: {name:"地址簿共享权限",icon:"shield",section:"系统管理",endpoint:"address_book_collection_rule",admin:true,columns:[["id","编号"],["collection_id","地址簿 ID"],["type","对象类型"],["to_id","对象 ID"],["rule","权限级别"]],fields:[["collection_id","地址簿 ID","number",true],["type","对象类型","rule_type",true],["to_id","对象 ID","number",true],["rule","权限级别","rule",true],["user_id","所有者 ID","number"]]},
    server: {name:"服务器管理",icon:"terminal",section:"系统管理",admin:true},
    loginlogs: {name:"登录记录",icon:"clock",section:"系统管理",endpoint:"login_log",admin:true,readonly:true,columns:[["id","编号"],["user_id","用户 ID"],["client","客户端"],["ip","IP 地址"],["platform","平台"],["created_at","时间"]]},
    connections: {name:"连接记录",icon:"link",section:"系统管理",endpoint:"audit_conn",admin:true,readonly:true,columns:[["id","编号"],["from_peer","来源设备"],["peer_id","目标设备"],["ip","IP 地址"],["created_at","时间"]]},
    transfers: {name:"文件传输",icon:"file",section:"系统管理",endpoint:"audit_file",admin:true,readonly:true,columns:[["id","编号"],["from_peer","来源设备"],["peer_id","目标设备"],["path","文件路径"],["created_at","时间"]]},
    sessions: {name:"登录会话",icon:"lock",section:"系统管理",endpoint:"user_token",admin:true,readonly:true,columns:[["id","编号"],["user_id","用户 ID"],["device_uuid","设备 UUID"],["created_at","时间"]]},
    profile: {name:"个人设置",icon:"settings",section:"账户"}
  };
  const drawings = {
    home:'<path d="M3 10 12 3l9 7v11h-6v-7H9v7H3z"/>',
    monitor:'<rect x="3" y="4" width="18" height="14" rx="2"/><path d="M8 22h8m-4-4v4"/>',
    users:'<circle cx="9" cy="7" r="4"/><path d="M2 21v-2a5 5 0 0 1 5-5h4a5 5 0 0 1 5 5v2m0-16a4 4 0 0 1 0 8m6 8v-2a5 5 0 0 0-3-4.5"/>',
    share:'<circle cx="18" cy="5" r="3"/><circle cx="6" cy="12" r="3"/><circle cx="18" cy="19" r="3"/><path d="m8.7 13.4 6.6 4.2M15.3 6.5l-6.6 4.1"/>',
    book:'<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>',
    folder:'<path d="M3 7a2 2 0 0 1 2-2h5l2 2h7a2 2 0 0 1 2 2v10a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/>',
    tag:'<path d="M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L2 12V2h10l8.6 8.6a2 2 0 0 1 0 2.8z"/><circle cx="7" cy="7" r="1"/>',
    layers:'<path d="m12 2 10 5-10 5L2 7zm-10 10 10 5 10-5M2 17l10 5 10-5"/>',
    grid:'<rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/><rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>',
    shield:'<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><path d="m9 12 2 2 4-4"/>',
    key:'<circle cx="7.5" cy="15.5" r="5.5"/><path d="m21 2-9.6 9.6M16 4l4 4m-2 2 3 3"/>',
    clock:'<circle cx="12" cy="12" r="9"/><path d="M12 6v6l4 2"/>',
    link:'<path d="M10 13a5 5 0 0 0 7.1 0l3-3A5 5 0 0 0 13 2.9l-1.7 1.7M14 11a5 5 0 0 0-7.1 0l-3 3A5 5 0 0 0 11 21.1l1.7-1.7"/>',
    file:'<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><path d="M14 2v6h6M8 13h8m-8 4h8"/>',
    lock:'<rect x="4" y="11" width="16" height="11" rx="2"/><path d="M8 11V7a4 4 0 0 1 8 0v4"/>',
    terminal:'<rect x="2" y="3" width="20" height="18" rx="2"/><path d="m6 8 4 4-4 4m7 4h5"/>',
    settings:'<circle cx="12" cy="12" r="3"/><path d="M20 12a8 8 0 0 0-.2-1.9l2-1.5-2-3.4-2.4 1a9 9 0 0 0-3.4-2V2h-4v2.2a9 9 0 0 0-3.4 2l-2.4-1-2 3.4 2 1.5a8 8 0 0 0 0 3.8l-2 1.5 2 3.4 2.4-1a9 9 0 0 0 3.4 2V22h4v-2.2a9 9 0 0 0 3.4-2l2.4 1 2-3.4-2-1.5A8 8 0 0 0 20 12z"/>',
    search:'<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>',
    plus:'<path d="M12 5v14M5 12h14"/>',
    refresh:'<path d="M20 7v5h-5M4 17v-5h5"/><path d="M5 9a8 8 0 0 1 13-3l2 3M4 15l2 3a8 8 0 0 0 13-3"/>',
    menu:'<path d="M4 6h16M4 12h16M4 18h16"/>',
    close:'<path d="M5 5l14 14M19 5 5 19"/>',
    sun:'<circle cx="12" cy="12" r="4"/><path d="M12 2v2m0 16v2M2 12h2m16 0h2M5 5l1.5 1.5M18 18l1.5 1.5M5 19l1.5-1.5M18 6l1.5-1.5"/>',
    moon:'<path d="M20 15.4A8 8 0 0 1 8.6 4a8.5 8.5 0 1 0 11.4 11.4z"/>',
    logout:'<path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5m5 5H9"/>',
    arrow:'<path d="M5 12h14m-7-7 7 7-7 7"/>',
    info:'<circle cx="12" cy="12" r="10"/><path d="M12 11v6m0-10h.01"/>'
  };
  function icon(name,size=20) { return '<svg width="'+size+'" height="'+size+'" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">'+(drawings[name]||drawings.info)+'</svg>'; }
  function esc(v) {return String(v===undefined||v===null?"":v).replace(/[&<>"']/g,c=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[c]));}
  function txt(v) {return v===null||v===undefined||v===""?"—":Array.isArray(v)?v.join(", "):typeof v==="object"?JSON.stringify(v):String(v);}
  function date(v) {if(!v)return "—";if(typeof v==="number"||/^[0-9]{10,13}$/.test(String(v))){const n=Number(v);return new Date(n<1e12?n*1000:n).toLocaleString("zh-CN",{hour12:false});}return esc(txt(v));}
  function colValue(v,k){
    if(k==="sources"&&Array.isArray(v))return v.map(x=>'<span class="pill '+(x.type==="contact"?"good":"")+'" style="margin:2px">'+esc(x.type==="contact"?"联系人："+x.name:"群组："+x.name)+'</span>').join("");
    if(k==="type"&&(v===1||v===2||v===3))return ["","普通组","共享组","公共锚点组"][v];
    if(k==="is_admin")return v?"管理员":"普通用户";
    if(k==="status")return '<span class="pill '+(Number(v)===1?"good":"warn")+'">'+(Number(v)===1?"启用":"停用")+"</span>";
    if(k==="created_at"||k==="last_online_time"||k==="expire")return date(v);
    if(typeof v==="boolean")return v?"是":"否";
    return esc(txt(v));
  }
  const state={
    token:sessionStorage.getItem("rustdesk.console.token")||"",user:null,isAdmin:false,
    page:"overview",bookTab:"entries",pageNo:1,pageSize:20,rows:[],total:0,loading:false,error:"",
    query:"",sourceType:"all",sourceId:"0",contacts:{outgoing:[],incoming:[]},
    groups:[],server:{},commands:[],modal:null,mobile:false,metrics:null,
    options:{ops:[],need_captcha:false},captcha:null,config:{},
    theme:localStorage.getItem("rustdesk.console.theme")==="dark"?"dark":"light"
  };
  document.documentElement.dataset.theme=state.theme;
  const API="/api/admin";
  let seq=0,debounce=0;
  async function request(route,method="GET",body) {
    const ctl=new AbortController(), timer=setTimeout(()=>ctl.abort(),18000);
    const headers={"Accept":"application/json","Accept-Language":"zh-CN"};
    if(state.token)headers["api-token"]=state.token;
    if(body!==undefined)headers["Content-Type"]="application/json";
    try{
      const res=await fetch(API+route,{method,headers,body:body===undefined?undefined:JSON.stringify(body),signal:ctl.signal,credentials:"same-origin"});
      let data={};try{data=await res.json();}catch{throw Error("服务器返回的数据无法读取");}
      if(!res.ok)throw Error(data.error||data.message||"HTTP "+res.status);
      if(typeof data.code==="number"&&data.code!==0)throw Error(data.message||"操作失败");
      return data.data===undefined?data:data.data;
    }catch(e){if(e.name==="AbortError")throw Error("请求超时");throw e;}finally{clearTimeout(timer);}
  }
  function toast(message,error=false){
    let wrap=document.querySelector(".toast-wrap");
    if(!wrap){wrap=document.createElement("div");wrap.className="toast-wrap";document.body.append(wrap);}
    const node=document.createElement("div");node.className="toast"+(error?" error":"");node.role="status";node.textContent=String(message);
    wrap.append(node);setTimeout(()=>node.remove(),4300);
  }
  function available(k){return pages[k]&&(!pages[k].admin||state.isAdmin);}
  function activeKey(){return state.page==="addressbooks"&&state.bookTab==="collections"?"collections":state.page;}
  function endpoint(k) {const p=pages[k];return !state.isAdmin&&["peer","address_book","address_book_collection","tag","login_log"].includes(p.endpoint)?"/my/"+p.endpoint:"/"+p.endpoint;}
  function pageHeading(title,subtitle,actions=""){
    return '<div class="page-head"><div><p class="eyebrow">RUSTDESK / CONTROL CENTER</p><h1>'+esc(title)+'</h1><p class="subtitle">'+esc(subtitle)+'</p></div><div class="head-actions">'+actions+'</div></div>';
  }
  function button(label,action,primary=false){return '<button type="button" class="btn '+(primary?"btn-primary":"")+'" data-action="'+esc(action)+'">'+esc(label)+'</button>';}
  function brand(){return '<span class="brand-symbol"><span></span><span></span></span><span><span class="side-name">RustDesk</span><span class="side-sub">CONTROL CENTER</span></span>';}
  function login(){
    const options=state.options||{};
    const cap=options.need_captcha?'<div class="login-field"><label for="captcha">验证码</label><div style="display:flex;gap:10px"><input name="captcha" id="captcha" required placeholder="输入验证码"><button class="btn" type="button" data-action="captcha">刷新</button></div><input type="hidden" name="captcha_id" value="'+esc(state.captcha&&state.captcha.id||"")+'">'+(state.captcha&&state.captcha.b64?'<img alt="验证码" style="max-height:72px;max-width:220px;margin-top:10px" src="'+esc(state.captcha.b64.startsWith("data:")?state.captcha.b64:"data:image/png;base64,"+state.captcha.b64)+'">':"")+"</div>":"";
    const passwordForm=options.disable_pwd?'<p class="warning">当前服务已关闭密码登录，请选择下方的单点登录服务。</p>':'<form id="login-form"><div class="login-field"><label for="username">用户名</label><input id="username" autocomplete="username" name="username" placeholder="输入用户名" required></div><div class="login-field"><label for="password">密码</label><input type="password" id="password" name="password" autocomplete="current-password" placeholder="输入密码" required></div>'+cap+'<button type="submit" class="btn btn-primary login-submit">登录控制台 '+icon("arrow",17)+'</button></form>';
    const ops=(options.ops||[]).map(op=>{const n=typeof op==="string"?op:op.op||op.name;return n?'<button type="button" class="btn" data-action="oidc" data-id="'+esc(n)+'">使用 '+esc(n)+' 登录 '+icon("arrow",14)+'</button>':"";}).join("");
    root.innerHTML='<div class="login-shell"><section class="login-art"><div class="brand-row">'+brand()+'</div><div><p class="eyebrow" style="color:#83e7d4">PRIVATE INFRASTRUCTURE</p><h1>远程设备管理，<br><em>一切尽在掌握。</em></h1><p>统一管理远程设备、联系人、共享群组与用户账户。界面可适应电脑、平板和手机。</p></div><div class="login-foot">RustDesk Self-hosted Control Center</div></section><section class="login-right"><div class="login-card"><p class="eyebrow">WELCOME BACK</p><h2>登录控制台</h2><p class="login-intro">'+esc(state.config.title||"请使用 RustDesk API 账户登录")+'</p>'+passwordForm+'<div class="login-options">'+ops+'</div><p class="login-help">首次安装的管理员密码可在服务端启动日志中查看。请通过 HTTPS 连接管理页面。</p></div></section></div>';
  }
  function shell(){
    const username=state.user&&state.user.nickname||state.user&&state.user.username||"用户";
    let section="",nav="";
    for(const [key,p] of Object.entries(pages)){
      if(!available(key)||key==="collections")continue;
      if(p.section!==section){section=p.section;nav+='<div class="side-section">'+esc(section)+'</div>';}
      nav+='<button class="nav-link '+(key===state.page?"active":"")+'" type="button" data-view="'+esc(key)+'" '+(key===state.page?'aria-current="page"':'')+'>'+icon(p.icon,18)+esc(p.name)+'</button>';
    }
    nav+='<div class="side-section">客户端</div><button class="nav-link" data-action="webclient">'+icon("monitor",18)+'WebClient v2</button>';
    return '<div class="workspace"><div class="sidebar-backdrop '+(state.mobile?"show":"")+'" data-action="mobile-close"></div><aside class="sidebar '+(state.mobile?"open":"")+'"><div class="side-head">'+brand()+'</div><nav class="side-scroll" aria-label="导航">'+nav+'</nav><div class="side-bottom"><div class="side-avatar"><span class="avatar">'+esc(username[0]||"R").toUpperCase()+'</span><div class="side-account"><strong>'+esc(username)+'</strong><small>'+(state.isAdmin?"系统管理员":"普通用户")+'</small></div></div></div></aside><div class="main"><header class="topbar"><div class="breadcrumbs"><button class="icon-btn mobile-only" data-action="mobile-open" aria-label="打开导航">'+icon("menu")+'</button><span>工作空间</span><span>/</span><span class="crumb-current">'+esc(pages[state.page].name)+'</span></div><div class="top-actions"><button class="icon-btn" data-action="refresh" title="刷新">'+icon("refresh")+'</button><button class="icon-btn" data-action="theme" title="切换主题">'+icon(state.theme==="dark"?"sun":"moon")+'</button><button class="top-user" data-view="profile"><span class="mini-avatar">'+esc(username[0]||"R").toUpperCase()+'</span>'+esc(username)+'</button><button class="icon-btn" data-action="logout" title="退出登录">'+icon("logout")+'</button></div></header><main class="content">'+renderPage()+'</main></div></div>'+renderModal();
  }
  function render(){
    if(!state.token||!state.user){login();return;}
    root.innerHTML=shell();
  }
  function view(k){
    if(!available(k))return;
    state.page=k;if(k==="addressbooks")state.bookTab="entries";state.pageNo=1;state.query="";state.sourceType="all";state.sourceId="0";state.mobile=false;state.modal=null;state.error="";
    location.hash="#/"+k;render();load();
  }
  async function bootstrap(){
    try{state.config=await request("/config/admin");}catch{state.config={};}
    if(!state.token){
      const pending=sessionStorage.getItem("rustdesk.console.oidc");
      if(pending){try{const data=await request("/oidc/auth-query?code="+encodeURIComponent(pending));if(data&&data.token){state.token=data.token;sessionStorage.setItem("rustdesk.console.token",data.token);sessionStorage.removeItem("rustdesk.console.oidc");}}catch{}}
    }
    if(state.token){
      try{
        state.user=await request("/user/current");
        const routes=Array.isArray(state.user.route_names)?state.user.route_names:[];
        state.isAdmin=routes.includes("*");
        if(!state.isAdmin){try{await request("/group/list?page=1&page_size=1");state.isAdmin=true;}catch{}}
      }catch{state.token="";sessionStorage.removeItem("rustdesk.console.token");}
    }
    if(!state.token){await getOptions();render();return;}
    const hash=location.hash.replace(/^#\/?/,"").split("?")[0];
    state.page=available(hash)?hash:"overview";render();await load();
  }
  async function getOptions(){
    try{state.options=await request("/login-options");}catch{state.options={ops:[]};}
    if(state.options.need_captcha)await captcha();
  }
  async function captcha(){try{state.captcha=(await request("/captcha")).captcha;}catch{state.captcha=null;}}
  async function handleLogin(form){
    const data=Object.fromEntries(new FormData(form).entries());data.platform="web";
    const submit=form.querySelector('button[type="submit"]');submit.disabled=true;
    try{
      const result=await request("/login","POST",data);
      if(!result||!result.token)throw Error("登录未返回访问令牌");
      state.token=result.token;sessionStorage.setItem("rustdesk.console.token",result.token);
      state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
      state.page="overview";location.hash="#/overview";render();await load();toast("登录成功");
    }catch(e){toast(e.message,true);await getOptions();render();}
    finally{submit.disabled=false;}
  }
  async function oidc(op){
    try{
      const result=await request("/oidc/auth","POST",{op,deviceInfo:{os:"web"},id:"webadmin",uuid:""});
      if(!result.url||!result.code)throw Error("无法启动单点登录");
      sessionStorage.setItem("rustdesk.console.oidc",result.code);
      window.location.assign(result.url);
    }catch(e){toast(e.message,true);}
  }
  async function logout(){
    try{await request("/logout","POST",{});}catch{}
    state.token="";state.user=null;state.isAdmin=false;sessionStorage.removeItem("rustdesk.console.token");
    await getOptions();render();
  }
  async function load(){
    const n=++seq,k=state.page,p=pages[activeKey()];
    state.loading=true;state.error="";render();
    try{
      if(k==="overview"){
        const reqs=[request(endpoint("devices")+"/list?page=1&page_size=5")];
        reqs.push(state.isAdmin?request("/user/list?page=1&page_size=1"):request("/my/address_book/list?page=1&page_size=1"));
        reqs.push(state.isAdmin?request("/group/list?page=1&page_size=1"):request("/contact/list"));
        const out=await Promise.allSettled(reqs);
        const val=i=>out[i].status==="fulfilled"?out[i].value:{};
        state.metrics={devices:val(0).total,users:val(1).total,groups:val(2).total,recent:val(0).list||[]};
      }else if(k==="contacts"){
        state.contacts=await request("/contact/list");
      }else if(k==="shared"){
        const q=new URLSearchParams({page:state.pageNo,page_size:state.pageSize,search:state.query,source_type:state.sourceType,source_id:state.sourceId});
        const result=await request("/shared_devices/list?"+q.toString());
        state.rows=result.list||[];state.total=result.total||0;
        if(!state.groups.length){try{state.groups=(await request("/shared_devices/sources")).sources||[];}catch{state.groups=[];}}
      }else if(k==="server"){
        const out=await Promise.allSettled([request("/config/server"),request("/rustdesk/cmdList")]);
        state.server=out[0].status==="fulfilled"?out[0].value:{};
        state.commands=out[1].status==="fulfilled"?out[1].value.list||[]:[];
      }else if(p.endpoint){
        const q=new URLSearchParams({page:state.pageNo,page_size:state.pageSize});
        if(p.search&&state.query)q.set(p.search,state.query);
        const result=await request(endpoint(activeKey())+"/list?"+q.toString());
        state.rows=result.list||[];state.total=result.total||0;
      }
    }catch(e){state.error=e.message;state.rows=[];state.total=0;}
    if(n!==seq)return;state.loading=false;render();
  }
  function metric(name,value,ic){
    return '<div class="metric"><span class="metric-icon">'+icon(ic)+'</span><div class="metric-value">'+esc(value===undefined?"—":value)+'</div><div class="metric-label">'+esc(name)+'</div></div>';
  }
  function overview(){
    const m=state.metrics||{};
    const shortcuts=["devices","shared","contacts",state.isAdmin?"groups":"addressbooks"];
    const quick=shortcuts.map(k=>'<button data-view="'+k+'" class="quick-card">'+icon(pages[k].icon,21)+'<strong>'+esc(pages[k].name)+" "+icon("arrow",14)+'</strong></button>').join("");
    const recent=(m.recent||[]).map(x=>'<div class="status-row"><span><strong>'+esc(x.alias||x.hostname||x.id)+'</strong><span class="cell-sub">'+esc(x.id)+'</span></span><span class="pill muted">'+date(x.last_online_time)+'</span></div>').join("")||'<div class="empty">暂无设备</div>';
    return pageHeading("控制台概览","一个统一的空间，用于管理远程设备与共享关系。")+
      '<div class="metric-grid">'+metric("设备数量",state.loading?"…":m.devices,"monitor")+metric(state.isAdmin?"用户数量":"地址簿条目",state.loading?"…":m.users,"users")+metric(state.isAdmin?"用户分组":"联系人数量",state.loading?"…":m.groups===undefined?state.contacts.incoming.length:m.groups,"layers")+metric("当前账户",state.user.username,"shield")+'</div>'+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">快速访问</h2><p class="panel-sub">常用管理功能</p></div></div><div class="quick-grid">'+quick+'</div></section><section class="panel"><div class="panel-header"><div><h2 class="panel-title">最近出现的设备</h2><p class="panel-sub">按最后活动时间显示</p></div><button data-view="devices" class="btn btn-sm">查看全部</button></div><div class="tips">'+recent+'</div></section></div>';
  }
  function rowIdentifier(row){return String(row.row_id===undefined?row.id:row.row_id);}
  function rowActions(row){
    const p=pages[activeKey()];
    if(p.readonly)return "";
    const id=esc(rowIdentifier(row));
    let out="";
    if(state.isAdmin||state.page!=="devices")out+='<button class="btn btn-sm" data-action="edit" data-id="'+id+'">编辑</button>';
    if(state.page==="groups"){out+='<button class="btn btn-sm" data-action="members" data-id="'+id+'">成员</button>';if(row.type===3)out+='<button class="btn btn-sm" data-action="anchors" data-id="'+id+'">公共成员</button>';}
    if(state.isAdmin||state.page!=="devices")out+='<button class="btn btn-sm btn-danger" data-action="delete" data-id="'+id+'">删除</button>';
    return out;
  }
  function table(p,rows){
    if(state.loading)return '<div class="loading-table"><span class="spinner"></span>正在读取数据…</div>';
    if(state.error)return '<div class="empty">'+icon("info",32)+'<h3>读取数据失败</h3><p>'+esc(state.error)+'</p></div>';
    if(!rows.length)return '<div class="empty">'+icon("info",32)+'<h3>暂无记录</h3><p>请调整筛选条件或创建新的记录。</p></div>';
    const cols=p.columns;
    const th=cols.map(c=>'<th scope="col">'+esc(c[1])+'</th>').join("");
    const trs=rows.map(row=>'<tr>'+cols.map((c,i)=>'<td><span class="'+(i===0?"cell-title":"")+'">'+colValue(row[c[0]],c[0])+'</span></td>').join("")+'<td class="cell-actions">'+rowActions(row)+'</td></tr>').join("");
    const mobile=rows.map(row=>'<article class="mobile-item"><div class="mobile-title">'+colValue(row[cols[0][0]],cols[0][0])+'</div><dl>'+cols.slice(1).map(c=>'<dt>'+esc(c[1])+'</dt><dd>'+colValue(row[c[0]],c[0])+'</dd>').join("")+'</dl><div class="mobile-actions">'+rowActions(row)+'</div></article>').join("");
    return '<div class="table-scroll"><table class="data-table"><thead><tr>'+th+'<th>操作</th></tr></thead><tbody>'+trs+'</tbody></table></div><div class="mobile-list">'+mobile+'</div>';
  }
  function pagination(){
    const pagesCount=Math.max(1,Math.ceil(state.total/state.pageSize));
    return '<div class="pager"><span>第 '+state.pageNo+' / '+pagesCount+' 页 · 共 '+state.total+' 项</span><div class="pager-buttons"><button data-action="prev" '+(state.pageNo===1?"disabled":"")+' aria-label="上一页">‹</button><button data-action="next" '+(state.pageNo>=pagesCount?"disabled":"")+' aria-label="下一页">›</button></div></div>';
  }
  function genericList(){
    const p=pages[activeKey()];
    const add=p.readonly||state.page==="devices"&&!state.isAdmin?"":'<button class="btn btn-primary" data-action="create">'+icon("plus",16)+'新增'+esc(p.name)+'</button>';
    const search='<div class="table-toolbar"><div class="search">'+icon("search",16)+'<input id="table-search" aria-label="搜索" placeholder="'+(p.search?"按关键词搜索…":"筛选本页内容…")+'" value="'+esc(state.query)+'"></div><span class="toolbar-meta">共 '+state.total+' 项</span></div>';
    const rows=p.search?state.rows:state.rows.filter(row=>!state.query||JSON.stringify(row).toLowerCase().includes(state.query.toLowerCase()));
    const isBook = state.page==="addressbooks";
    const tabs = isBook?'<div style="display:flex;gap:8px;margin-bottom:18px;flex-wrap:wrap"><button class="btn '+(state.bookTab==="entries"?"btn-primary":"")+'" data-action="book-entries">设备条目</button><button class="btn '+(state.bookTab==="collections"?"btn-primary":"")+'" data-action="book-collections">地址簿分类</button></div>':"";
    return pageHeading(isBook?"地址簿":p.name,isBook?"在一个页面中管理地址簿分类及其包含的设备，原有数据自动保留。":"查看、编辑和管理相关数据。",add)+tabs+'<section class="panel">'+search+table(p,rows)+pagination()+'</section>';
  }
  function contacts(){
    const v=state.contacts||{outgoing:[],incoming:[]};
    function accountRows(rows,canDelete){
      if(state.loading)return '<div class="loading-table"><span class="spinner"></span>加载中…</div>';
      if(!rows.length)return '<div class="empty"><h3>暂无联系人</h3><p>'+(canDelete?"您还没有主动向其他用户公开设备。":"目前没有用户向您公开设备。")+'</p></div>';
      return rows.map(x=>'<div class="status-row"><span><strong>'+esc(x.nickname||x.username)+'</strong><span class="cell-sub">@'+esc(x.username)+' · 用户 ID '+esc(x.user_id)+'</span></span>'+(canDelete?'<button class="btn btn-sm btn-danger" data-action="contact-delete" data-id="'+esc(x.user_id)+'">撤销共享</button>':'<button class="btn btn-sm" data-action="contact-devices" data-id="'+esc(x.user_id)+'">查看设备</button>')+'</div>').join("");
    }
    return pageHeading("联系人","添加联系人即表示向对方开放您名下的所有设备，对方无须确认。")+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">我向谁共享设备</h2><p class="panel-sub">只有您主动添加的人才能通过联系人关系查看您的设备</p></div></div><div class="tips">'+accountRows(v.outgoing||[],true)+'</div><form id="contact-form" class="table-toolbar" style="gap:10px"><div class="search">'+icon("search",16)+'<input required name="username" aria-label="用户名" placeholder="输入对方的用户名"></div><button type="submit" class="btn btn-primary">'+icon("plus",16)+'添加联系人</button></form></section><section class="panel"><div class="panel-header"><div><h2 class="panel-title">向我共享设备的人</h2><p class="panel-sub">这些用户已主动向您开放他们的设备</p></div></div><div class="tips">'+accountRows(v.incoming||[],false)+'</div></section></div>';
  }
  function shared(){
    const groups=(state.groups||[]);
    const options='<option value="all">全部共享设备</option><option value="contact">联系人共享</option><option value="group">群组共享</option>';
    const ids=groups.filter(x=>state.sourceType==="all"||x.type===state.sourceType).map(x=>'<option value="'+esc(x.type+":"+x.id)+'" '+(x.type===state.sourceType&&String(x.id)===String(state.sourceId)?"selected":"")+'>'+esc(x.type==="contact"?"联系人 · "+x.name:"群组 · "+x.name)+'</option>').join("");
    const search='<div class="table-toolbar"><div class="search">'+icon("search",16)+'<input id="table-search" placeholder="搜索设备 ID、备注或主机名" value="'+esc(state.query)+'"></div><select id="shared-source-type" class="btn" aria-label="共享类型">'+options.replace('value="'+state.sourceType+'"','value="'+state.sourceType+'" selected')+'</select><select id="shared-source-id" class="btn" aria-label="共享来源"><option value="0">所有来源</option>'+ids+'</select></div>';
    return pageHeading("共享设备","汇总联系人、共享组和公共锚点组公开给您的设备。同一设备只显示一次，并标明所有共享来源。",'<button class="btn btn-primary" data-action="webclient">'+icon("monitor",16)+'打开 WebClient v2</button>')+
      '<section class="panel">'+search+table({...pages.shared},state.rows)+pagination()+'</section>';
  }
  function server(){
    const d=state.server||{};
    const labels=[["ID 服务器",d.id_server],["中继服务器",d.relay_server],["API 服务",d.api_server]];
    const details=labels.map(x=>'<div class="status-row"><span>'+esc(x[0])+'</span><strong>'+esc(txt(x[1]))+'</strong></div>').join("");
    const opts=state.commands.map((cmd,i)=>'<option value="'+i+'">'+esc((cmd.target==="21115"?"ID 服务器":"中继服务器")+" / "+(cmd.explain||cmd.cmd))+'</option>').join("");
    return pageHeading("服务器管理","查看网络配置并向自托管 RustDesk 服务发送管理指令。")+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><h2 class="panel-title">连接参数</h2></div><div class="tips">'+details+'</div></section><section class="panel"><div class="panel-header"><h2 class="panel-title">管理指令</h2></div><form id="command-form" class="tips"><div class="field"><label for="cmd">选择指令</label><select id="cmd" name="command" required>'+opts+'</select></div><div class="field" style="margin-top:16px"><label for="cmd-arg">参数（可选）</label><input id="cmd-arg" name="option" placeholder="按指令说明填写"></div><button type="submit" class="btn btn-primary" style="margin-top:15px">执行指令</button><pre id="command-output" style="white-space:pre-wrap;overflow-wrap:anywhere"></pre></form></section></div>';
  }
  function profile(){
    return pageHeading("个人设置","查看当前账户并修改登录密码。")+
      '<section class="panel" style="max-width:650px"><div class="panel-header"><h2 class="panel-title">'+esc(state.user.username)+'</h2></div><form id="password-form" class="tips" style="display:grid;gap:15px"><div class="field"><label>当前密码</label><input type="password" name="old_password" autocomplete="current-password" required></div><div class="field"><label>新密码</label><input type="password" name="new_password" autocomplete="new-password" minlength="4" required></div><button class="btn btn-primary" type="submit">修改密码</button></form></section>';
  }
  function renderPage(){
    if(state.page==="overview")return overview();
    if(state.page==="contacts")return contacts();
    if(state.page==="shared")return shared();
    if(state.page==="server")return server();
    if(state.page==="profile")return profile();
    return genericList();
  }
  function fieldInput(field,row){
    const [name,label,type="text",required=false]=field;
    const value=row&&row[name]!==undefined?row[name]:type==="status"?1:type==="group_type"?1:type==="rule_type"?1:type==="rule"?1:type==="checkbox"?false:"";
    let input;
    if(type==="checkbox")input='<label class="check"><input type="checkbox" name="'+esc(name)+'" '+(value?"checked":"")+'>启用</label>';
    else if(type==="textarea")input='<textarea name="'+esc(name)+'" rows="3">'+esc(value)+'</textarea>';
    else if(["group_type","status","oauth_type","rule_type","rule"].includes(type)){
      const opts={group_type:[[1,"普通组"],[2,"共享组"],[3,"公共锚点组"]],status:[[1,"已启用"],[2,"已停用"]],oauth_type:[["oidc","OIDC"],["github","GitHub"],["google","Google"],["linuxdo","Linux.do"]],rule_type:[[1,"用户"],[2,"群组"]],rule:[[1,"只读"],[2,"读写"],[3,"完全控制"]]}[type];
      input='<select name="'+esc(name)+'">'+opts.map(o=>'<option value="'+esc(o[0])+'" '+(String(o[0])===String(value)?"selected":"")+'>'+esc(o[1])+'</option>').join("")+'</select>';
    }else input='<input name="'+esc(name)+'" type="'+esc(type)+'" value="'+esc(value)+'" '+(required?'required':"")+' autocomplete="off">';
    return '<div class="field '+(type==="textarea"?"full":"")+'"><label>'+esc(label)+(required?' <span class="required">*</span>':"")+'</label>'+input+(name==="auto_group_ids"?'<small>例如：2,3。登录后将自动加入这些组，并保留原有主要分组。</small>':"")+'</div>';
  }
  function modal(title,desc,body,footer){
    return '<div class="overlay" id="modal-overlay"><section class="drawer" role="dialog" aria-modal="true" aria-label="'+esc(title)+'"><div class="drawer-head"><div><h2>'+esc(title)+'</h2><p>'+esc(desc||"")+'</p></div><button type="button" class="icon-btn" aria-label="关闭" data-action="close-modal">'+icon("close")+'</button></div><div class="drawer-body">'+body+'</div><div class="drawer-footer">'+footer+'</div></section></div>';
  }
  function renderModal(){
    const m=state.modal;if(!m)return "";
    if(m.kind==="edit"){
      const p=pages[activeKey()],fields=p.fields||[];
      const body='<form id="edit-form"><input type="hidden" name="id" value="'+esc(m.data.id||"")+'"><input type="hidden" name="row_id" value="'+esc(m.data.row_id||"")+'"><div class="form-grid">'+fields.map(f=>fieldInput(f,m.data)).join("")+'</div></form>';
      return modal(m.action==="create"?"新建"+p.name:"编辑"+p.name,"修改完成后保存，变更会立即生效。",body,'<button class="btn" data-action="close-modal">取消</button><button class="btn btn-primary" type="submit" form="edit-form">保存</button>');
    }
    if(m.kind==="members"||m.kind==="anchors"){
      const anchor=m.kind==="anchors";
      const rows=(m.users||[]).map(u=>'<label class="status-row" style="cursor:pointer"><span><strong>'+esc(u.nickname||u.username)+'</strong><span class="cell-sub">@'+esc(u.username)+' · ID '+esc(u.id)+'</span></span><input style="width:18px;height:18px;accent-color:var(--main)" type="checkbox" name="selected" value="'+esc(u.id)+'" '+(m.selected.has(u.id)?"checked":"")+'></label>').join("");
      const details='<p class="subtitle">'+(anchor?"只有选中的公共成员及其设备会显示给该组的普通成员。":"当前组成员已经勾选。添加成员会保留现有成员；移除成员将删除勾选成员的群组关系；替换成员会只保留勾选成员。")+'</p><div class="field" style="margin-top:20px"><label for="user-filter">查找用户</label><input id="user-filter" placeholder="筛选用户名"></div><form id="member-form"><div class="field" style="margin-top:16px"><label>用户 ID（可用逗号分隔，适用于大量成员）</label><input name="manual_ids" placeholder="例如：2,5,8"></div><div id="member-choices" style="margin-top:12px">'+rows+'</div></form>';
      const footer='<button class="btn" data-action="close-modal">取消</button>'+(anchor?'<button class="btn btn-primary" data-action="member-replace">保存公共成员</button>':'<button class="btn" data-action="member-add">添加成员</button><button class="btn btn-danger" data-action="member-remove">移除成员</button><button class="btn btn-primary" data-action="member-replace">替换成员</button>');
      return modal(anchor?"设置公共成员":"批量管理成员",m.group.name,details,footer);
    }
    return "";
  }
  async function openEditor(rowId,action){
    const p=pages[activeKey()];let row={};
    if(action==="edit"){
      row=state.rows.find(r=>rowIdentifier(r)===rowId)||{};
      if(!Object.keys(row).length){toast("未找到记录",true);return;}
    }
    state.modal={kind:"edit",action,data:row};render();
  }
  async function saveEditor(form){
    const p=pages[activeKey()],data={};
    const oldData=state.modal.data;
    for(const f of p.fields){
      const [name,,type="text",required=false]=f;
      const el=form.elements.namedItem(name);
      if(!el)continue;
      if(type==="checkbox")data[name]=el.checked;
      else if(type==="number"||["status","group_type","rule_type","rule"].includes(type)){
        if(el.value.trim()!==""||required)data[name]=Number(el.value||0);
      }else data[name]=el.value.trim();
    }
    if(state.modal.action==="edit"){
      if(oldData.id!==undefined)data.id=oldData.id;
      if(oldData.row_id!==undefined)data.row_id=oldData.row_id;
    }
    try{
      await request(endpoint(activeKey())+"/"+state.modal.action,"POST",data);
      state.modal=null;toast("保存成功");await load();
    }catch(e){toast(e.message,true);}
  }
  async function deleteEntry(id){
    const p=pages[activeKey()],r=state.rows.find(x=>rowIdentifier(x)===id);
    if(!r||!window.confirm("确定删除这条记录吗？"))return;
    try{await request(endpoint(activeKey())+"/delete","POST",{id:r.id,row_id:r.row_id});toast("已删除");await load();}
    catch(e){toast(e.message,true);}
  }
  async function groupMembers(id,anchor){
    const group=state.rows.find(r=>String(r.id)===String(id));if(!group)return;
    try{
      const [selected,listing]=await Promise.all([
        request("/group/"+(anchor?"anchors":"members")+"?group_id="+encodeURIComponent(id)),
        request("/user/list?page=1&page_size=500")
      ]);
      const rows=listing.list||[];
      const displayed=new Set(rows.map(x=>Number(x.id)));
      for(const memberId of selected.user_ids||[]){
        if(!displayed.has(Number(memberId)))rows.push({id:memberId,username:"用户 "+memberId,nickname:""});
      }
      state.modal={kind:anchor?"anchors":"members",group,selected:new Set(selected.user_ids||[]),users:rows};
      render();
    }catch(e){toast(e.message,true);}
  }
  function parseIds(value){
    if(!value.trim())return [];
    const list=value.split(",").map(x=>x.trim()).filter(Boolean);
    if(list.some(x=>!/^[1-9][0-9]*$/.test(x)))throw Error("用户 ID 需要用逗号分隔的正整数");
    return list.map(Number);
  }
  async function submitMembers(action){
    const m=state.modal,form=document.getElementById("member-form");if(!m||!form)return;
    let ids;try{
      ids=[...new Set([...Array.from(form.querySelectorAll('input[name="selected"]:checked')).map(x=>Number(x.value)),...parseIds(form.elements.manual_ids.value)])];
    }catch(e){toast(e.message,true);return;}
    if(action==="remove"&&!window.confirm("确定从群组移除这 "+ids.length+" 名选中成员吗？"))return;
    if(action==="replace"&&!window.confirm("确定将这个群组的成员设为选中的 "+ids.length+" 人吗？"))return;
    try{
      const url=m.kind==="anchors"?"/group/anchors":"/group/members/batch";
      const payload=m.kind==="anchors"?{group_id:m.group.id,user_ids:ids}:{group_id:m.group.id,user_ids:ids,action};
      await request(url,"POST",payload);
      state.modal=null;toast("群组成员已更新");await load();
    }catch(e){toast(e.message,true);}
  }
  async function sendCommand(form){
    const idx=Number(form.elements.command.value),cmd=state.commands[idx];
    if(!cmd)return;
    try{
      const response=await request("/rustdesk/sendCmd","POST",{target:cmd.target,cmd:cmd.cmd,option:form.elements.option.value});
      const output=document.getElementById("command-output");if(output)output.textContent=typeof response==="string"?response:JSON.stringify(response,null,2);
    }catch(e){toast(e.message,true);}
  }

  document.addEventListener("click",async event=>{
    const el=event.target.closest("[data-action],[data-view]");
    if(!el)return;
    if(el.dataset.view){view(el.dataset.view);return;}
    const a=el.dataset.action;
    if(a==="refresh"){await load();return;}
    if(a==="book-entries"||a==="book-collections"){state.bookTab=a==="book-entries"?"entries":"collections";state.pageNo=1;state.query="";await load();return;}
    if(a==="theme"){state.theme=state.theme==="dark"?"light":"dark";document.documentElement.dataset.theme=state.theme;localStorage.setItem("rustdesk.console.theme",state.theme);render();return;}
    if(a==="mobile-open"||a==="mobile-close"){state.mobile=a==="mobile-open";render();return;}
    if(a==="close-modal"){state.modal=null;render();return;}
    if(a==="logout"){await logout();return;}
    if(a==="webclient"){window.location.assign("/webclient2/");return;}
    if(a==="captcha"){await captcha();render();return;}
    if(a==="oidc"){await oidc(el.dataset.id);return;}
    if(a==="create"||a==="edit"){await openEditor(el.dataset.id,a);return;}
    if(a==="delete"){await deleteEntry(el.dataset.id);return;}
    if(a==="members"||a==="anchors"){await groupMembers(el.dataset.id,a==="anchors");return;}
    if(a.startsWith("member-")){await submitMembers(a.replace("member-",""));return;}
    if(a==="contact-delete"){
      if(!window.confirm("撤销对该联系人的设备共享吗？"))return;
      try{await request("/contact/delete","POST",{user_id:Number(el.dataset.id)});toast("已撤销共享");await load();}catch(e){toast(e.message,true);}return;
    }
    if(a==="contact-devices"){state.page="shared";state.sourceType="contact";state.sourceId=el.dataset.id;location.hash="#/shared";render();await load();return;}
    if(a==="prev"&&state.pageNo>1){state.pageNo--;await load();return;}
    if(a==="next"&&state.pageNo*state.pageSize<state.total){state.pageNo++;await load();return;}
  });
  document.addEventListener("submit",async event=>{
    const f=event.target;
    if(!["login-form","edit-form","contact-form","command-form","password-form"].includes(f.id))return;
    event.preventDefault();
    if(f.id==="login-form")await handleLogin(f);
    if(f.id==="edit-form")await saveEditor(f);
    if(f.id==="contact-form"){
      try{await request("/contact/add","POST",{username:f.elements.username.value.trim()});toast("已向联系人开放您的设备");await load();}catch(e){toast(e.message,true);}
    }
    if(f.id==="command-form")await sendCommand(f);
    if(f.id==="password-form"){
      try{
        await request("/user/changeCurPwd","POST",Object.fromEntries(new FormData(f).entries()));
        toast("密码已修改，请重新登录");await logout();
      }catch(e){toast(e.message,true);}
    }
  });
  document.addEventListener("input",event=>{
    const el=event.target;
    if(el.id==="table-search"){
      state.query=el.value;clearTimeout(debounce);
      const p=pages[activeKey()];
      if(state.page==="shared"||p.search){
        debounce=setTimeout(()=>{state.pageNo=1;load();},350);
      }else{
        const pos=el.selectionStart;const value=el.value;render();
        const restored=document.getElementById("table-search");if(restored){restored.focus();restored.value=value;restored.setSelectionRange(pos,pos);}
      }
    }
    if(el.id==="user-filter"){
      const q=el.value.toLowerCase();
      document.querySelectorAll("#member-choices .status-row").forEach(row=>{row.style.display=row.textContent.toLowerCase().includes(q)?"":"none";});
    }
  });
  document.addEventListener("change",async event=>{
    const el=event.target;
    if(el.id==="shared-source-type"){state.sourceType=el.value;state.sourceId="0";state.pageNo=1;await load();}
    if(el.id==="shared-source-id"){if(el.value==="0"){state.sourceId="0";}else{const parts=el.value.split(":");state.sourceType=parts[0];state.sourceId=parts[1];}state.pageNo=1;await load();}
  });
  window.addEventListener("hashchange",()=>{if(!state.user)return;const viewName=location.hash.replace(/^#\/?/,"").split("?")[0];if(available(viewName)&&state.page!==viewName)view(viewName);});
  bootstrap().catch(e=>{toast(e.message,true);render();});
})();
