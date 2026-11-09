"use strict";
(() => {
  const root = document.getElementById("app");
  if (!root) return;

  const pages = {
    overview: {name:"概览",icon:"home",section:"工作空间"},
    devices: {name:"设备管理",icon:"monitor",section:"工作空间",endpoint:"peer",search:"id",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["os","系统"],["last_online_time","最后在线"],["note","设备备注"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","操作系统用户"],["os","操作系统"],["group_id","设备组 ID","number"],["note","设备备注","textarea"]]},
    shared: {name:"共享设备",icon:"share",section:"工作空间",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["owner_username","设备所有者"],["sources","共享来源"],["last_online_time","最后在线"]]},
    contacts: {name:"联系人",icon:"users",section:"工作空间"},
    addressbooks: {name:"地址簿",icon:"book",section:"工作空间",endpoint:"address_book",columns:[["id","设备 ID"],["alias","备注"],["hostname","主机名"],["username","登录账户"],["platform","平台"]],fields:[["id","设备 ID","text",true],["alias","备注"],["hostname","主机名"],["username","登录账户"],["password","连接密码","password"],["platform","平台"],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"],["forceAlwaysRelay","强制使用中继","checkbox"]]},
    collections: {name:"地址簿分类",icon:"folder",section:"工作空间",endpoint:"address_book_collection",columns:[["id","编号"],["name","分类名称"],["user_id","所有者 ID"]],fields:[["name","名称","text",true],["user_id","所有者 ID","number",true]]},
    tags: {name:"设备标签",icon:"tag",section:"工作空间",endpoint:"tag",columns:[["id","编号"],["name","标签名称"],["color","颜色"],["collection_id","地址簿 ID"]],fields:[["name","名称","text",true],["color","颜色（ARGB 整数）","number",true],["user_id","所有者 ID","number"],["collection_id","地址簿 ID","number"]]},
    mygroups: {name:"我的群组",icon:"layers",section:"个人管理",endpoint:"my/group",columns:[["id","编号"],["name","群组名称"],["type","分组类型"]],fields:[["name","名称","text",true],["type","类型","group_type",true]]},
    groups: {name:"用户分组",icon:"layers",section:"成员管理",endpoint:"group",admin:true,columns:[["id","编号"],["name","分组名称"],["type","分组类型"],["owner_user_id","创建者用户 ID"]],fields:[["name","名称","text",true],["type","类型","group_type",true]]},
    users: {name:"用户账户",icon:"users",section:"成员管理",endpoint:"user",admin:true,search:"username",columns:[["id","编号"],["username","用户名"],["nickname","昵称"],["email","邮箱"],["group_id","主要分组"],["status","状态"]],fields:[["username","用户名","text",true],["password","初始密码","password",true],["nickname","昵称"],["email","邮箱","email"],["group_id","主要分组 ID","number",true],["is_admin","管理员","checkbox"],["status","账户状态","status"],["remark","备注","textarea"]]},
    deployed: {name:"设备部署记录",icon:"monitor",section:"系统管理",endpoint:"device_deployment",admin:true,readonly:true,columns:[["id","编号"],["device_id","设备 ID"],["owner_user_id","部署账户 ID"],["status","部署状态"],["created_at","部署时间"]]},
    devicegroups: {name:"设备分组",icon:"grid",section:"成员管理",endpoint:"device_group",admin:true,columns:[["id","编号"],["name","名称"]],fields:[["name","名称","text",true]]},
    oauth: {name:"OIDC / SSO",icon:"key",section:"系统管理",endpoint:"oauth",admin:true,columns:[["id","编号"],["op","提供者"],["oauth_type","类型"],["issuer","颁发者"],["auto_group_ids","自动加入分组"]],fields:[["op","提供者标识"],["oauth_type","登录类型","oauth_type",true],["issuer","Issuer URL"],["scopes","Scopes"],["client_id","Client ID","text",true],["client_secret","Client Secret","password",true],["auto_register","允许自动注册","checkbox"],["auto_group_ids","自动加入的分组 ID（逗号分隔）"],["pkce_enable","启用 PKCE","checkbox"],["pkce_method","PKCE 方法"]]},
    trash: {name:"地址簿回收站",icon:"book",section:"个人管理",endpoint:"my/address_book/trash",readonly:true,columns:[["id","设备 ID"],["alias","别名"],["hostname","主机名"],["collection_id","地址簿编号"],["deleted_at","删除时间"]]},
    myRules: {name:"我的地址簿共享规则",icon:"shield",section:"个人管理",endpoint:"my/address_book_collection_rule",columns:[["id","编号"],["collection_id","地址簿 ID"],["type","分享对象"],["to_id","对象 ID"],["rule","权限"]],fields:[["collection_id","地址簿 ID","number",true],["type","对象类型","rule_type",true],["to_id","对象 ID（所有用户填 0）","number",true],["rule","权限级别","rule",true]]},
    rules: {name:"地址簿共享权限",icon:"shield",section:"系统管理",endpoint:"address_book_collection_rule",admin:true,columns:[["id","编号"],["collection_id","地址簿 ID"],["type","对象类型"],["to_id","对象 ID"],["rule","权限级别"]],fields:[["collection_id","地址簿 ID","number",true],["type","对象类型","rule_type",true],["to_id","对象 ID","number",true],["rule","权限级别","rule",true],["user_id","所有者 ID","number"]]},
    server: {name:"服务器管理",icon:"terminal",section:"系统管理",admin:true},
    myconnections: {name:"我的连接记录",icon:"link",section:"我的审计记录",endpoint:"my/audit_conn",readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_device_name","连接方设备"],["from_account","连接方账户"],["from_name","设备登录用户名"],["peer_id","被连接设备 ID"],["to_device_name","被连接设备"],["to_account","被连接方账户"],["ip","来源 IP"],["primary_auth","认证方式"],["two_factor","双重认证"],["conn_audit_ref","连接关联标识"],["note","连接备注"],["created_at","时间"]]},
    mytransfers: {name:"我的文件传输",icon:"file",section:"我的审计记录",endpoint:"my/audit_file",readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_account","连接方账户"],["peer_id","被连接设备 ID"],["to_account","被连接方账户"],["conn_id","连接编号"],["path","文件路径"],["ip","来源 IP"],["created_at","时间"]]},
    myalarms: {name:"我的安全告警",icon:"shield",section:"我的审计记录",endpoint:"my/audit_alarm",readonly:true,columns:[["id","编号"],["typ","告警类别"],["peer_id","设备 ID"],["conn_id","连接编号"],["info","告警详情"],["created_at","时间"]]},
    alarms: {name:"安全告警",icon:"shield",section:"系统管理",endpoint:"audit_alarm",admin:true,readonly:true,columns:[["id","编号"],["typ","告警类别"],["peer_id","设备 ID"],["from_user_id","发起方用户 ID"],["to_user_id","接收方用户 ID"],["conn_id","连接编号"],["source_ip","来源 IP"],["info","告警详情"],["created_at","时间"]]},
    operations: {name:"管理操作记录",icon:"clock",section:"系统管理",endpoint:"audit_operation",admin:true,readonly:true,columns:[["id","编号"],["account","操作账户"],["method","请求方法"],["path","操作内容"],["client_ip","来源 IP"],["success","结果"],["created_at","时间"]]},
    policies: {name:"设备安全策略",icon:"shield",section:"系统管理",endpoint:"device_policy",admin:true,columns:[["id","编号"],["name","策略名称"],["id_whitelist","设备 ID 白名单"],["ip_whitelist","IP 白名单"],["enable_webrtc","启用 WebRTC"],["trust_duration_days","可信设备有效期（天）"],["status","策略状态"]],fields:[["name","策略名称","text",true],["status","策略状态","status"],["id_whitelist","设备 ID 白名单","textarea"],["ip_whitelist","IP 白名单（单个地址或 CIDR）","textarea"],["enable_webrtc","允许 WebRTC 连接","checkbox"],["trust_duration_days","可信设备有效期（天）","trusted_days"]]},
    policyAssignments: {name:"设备策略分配",icon:"grid",section:"系统管理",endpoint:"device_policy/assignment",admin:true,columns:[["id","编号"],["device_id","设备 ID"],["policy_id","安全策略编号"]],fields:[["device_id","设备 ID","text",true],["policy_id","安全策略编号","number",true]]},
    userPolicyAssignments: {name:"用户策略分配",icon:"users",section:"系统管理",endpoint:"device_policy/user_assignment",admin:true,columns:[["id","编号"],["user_id","用户 ID"],["policy_id","策略编号"]],fields:[["user_id","用户 ID","number",true],["policy_id","策略编号","number",true]]},
    groupPolicyAssignments: {name:"设备组策略分配",icon:"grid",section:"系统管理",endpoint:"device_policy/group_assignment",admin:true,columns:[["id","编号"],["group_id","设备组 ID"],["policy_id","策略编号"]],fields:[["group_id","设备组 ID","number",true],["policy_id","策略编号","number",true]]},
    retention: {name:"日志保留设置",icon:"settings",section:"系统管理",admin:true},
    myloginlogs: {name:"我的登录记录",icon:"clock",section:"我的审计记录",endpoint:"my/login_log",readonly:true,columns:[["id","编号"],["client","客户端"],["ip","IP 地址"],["platform","平台"],["created_at","时间"]]},
    loginlogs: {name:"登录记录",icon:"clock",section:"系统管理",endpoint:"login_log",admin:true,readonly:true,columns:[["id","编号"],["user_id","用户 ID"],["client","客户端"],["ip","IP 地址"],["platform","平台"],["created_at","时间"]]},
    connections: {name:"全部连接记录",icon:"link",section:"系统管理",endpoint:"audit_conn",admin:true,readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_device_name","连接方设备"],["from_account","连接方账户"],["from_name","设备登录用户名"],["peer_id","被连接设备 ID"],["to_device_name","被连接设备"],["to_account","被连接方账户"],["ip","来源 IP"],["primary_auth","认证方式"],["two_factor","双重认证"],["conn_audit_ref","连接关联标识"],["note","连接备注"],["created_at","时间"]]},
    transfers: {name:"全部文件传输",icon:"file",section:"系统管理",endpoint:"audit_file",admin:true,readonly:true,columns:[["id","编号"],["direction","方向"],["from_peer","连接方设备 ID"],["from_account","连接方账户"],["peer_id","被连接设备 ID"],["to_account","被连接方账户"],["conn_id","连接编号"],["path","文件路径"],["ip","来源 IP"],["created_at","时间"]]},
    invitations: {name:"成员邮件邀请",icon:"users",section:"成员管理",endpoint:"invite",admin:true,columns:[["id","编号"],["email","收件邮箱"],["group_id","用户组 ID"],["expires_at","有效期截止"],["accepted_at","接受时间"],["revoked_at","撤销时间"]],fields:[["email","成员邮箱","email",true],["group_id","加入的用户组 ID（默认 1）","number"],["expires_in_days","有效天数（默认 7）","number"]]},
    relayNodes: {name:"中继节点管理",icon:"network",section:"系统管理",endpoint:"relay",admin:true,
      columns:[["id","编号"],["name","节点名称"],["address","中继服务器地址"],["subnets","优先使用的网络范围"],["priority","优先级"],["enabled","是否启用"]],
      fields:[["name","节点名称","text",true],["address","中继服务器地址","text",true],
        ["subnets","优先使用的 IP 网络（每行一个 CIDR）","textarea"],["priority","优先级（越小越高）","number"],
        ["enabled","是否启用","checkbox"]]},
    mail: {name:"SMTP 邮件服务",icon:"file",section:"系统管理",admin:true},
    apiTokens: {name:"API 管理令牌",icon:"key",section:"系统管理",admin:true,endpoint:"api_token",columns:[["id","编号"],["name","用途"],["prefix","令牌前缀"],["user_id","所属账户 ID"],["permissions","允许的操作"],["expires_at","过期时间"],["revoked_at","撤销时间"],["last_used_at","最近使用"]],fields:[["name","令牌名称","text",true],["user_id","所属账户 ID（留空为自己）","number"],["permissions","允许的操作（以逗号分隔）","text"],["expires_in_days","有效天数（0 表示长期有效）","number"]]},
    activeSessions: {name:"活动远程连接",icon:"link",section:"系统管理",admin:true,readonly:true,endpoint:"session",columns:[["device_id","设备 ID"],["conn_id","连接编号"],["last_seen_at","最后一次心跳"]]},
    disconnectHistory: {name:"结束连接记录",icon:"clock",section:"系统管理",admin:true,readonly:true,endpoint:"session/disconnect",columns:[["device_id","设备 ID"],["conn_id","连接编号"],["status","处理状态"],["requested_by","操作账户 ID"],["created_at","请求时间"],["delivered_at","通知时间"],["completed_at","结束时间"]]},
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
    if(k==="direction")return v==="outgoing"?"我发起的连接":v==="incoming"?"连接我的设备":v==="both"?"双方均为我的设备":"—";
    if(k==="primary_auth")return (["未报告","点击授权","临时密码","永久密码","切换控制方向"][Number(v)]||"其他方式")+" ("+esc(txt(v))+")";
    if(k==="two_factor")return (["未报告","TOTP 动态验证码","可信设备"][Number(v)]||"其他方式")+" ("+esc(txt(v))+")";
    if(k==="typ")return ({0:"IP 白名单拒绝",1:"密码尝试次数过多",2:"短时间连续登录失败",6:"IPv6 地址段登录尝试过多",7:"系统账户登录暂缓",8:"系统账户登录并发限制",9:"会话权限违规",10:"设备 ID 白名单拒绝"}[Number(v)]||"其他告警")+" ("+esc(txt(v))+")";
    if(k==="success")return v?'<span class="pill good">成功</span>':'<span class="pill warn">失败</span>';
    if(k==="is_admin")return v?"管理员":"普通用户";
    if(k==="enable_webrtc")return v?"启用":"停用";
    if(k==="enabled")return v?"已启用":"已停用";
    if(k==="status"&&typeof v==="string")return esc({
      approved:"已登记",revoked:"已撤销",pending:"等待通知",
      delivered:"已通知客户端",completed:"连接已结束",expired:"已过期"
    }[v]||v);
    if(k==="status")return '<span class="pill '+(Number(v)===1?"good":"warn")+'">'+(Number(v)===1?"启用":"停用")+"</span>";
    if(["created_at","last_online_time","expire","expires_at","revoked_at","last_used_at","last_seen_at","delivered_at","completed_at","deleted_at","accepted_at"].includes(k))return date(v);
    if(typeof v==="boolean")return v?"是":"否";
    return esc(txt(v));
  }
  const state={
    token:sessionStorage.getItem("rustdesk.console.token")||"",user:null,isAdmin:false,
    page:"overview",bookTab:"entries",pageNo:1,pageSize:20,rows:[],total:0,loading:false,error:"",
    query:"",sourceType:"all",sourceId:"0",contacts:{outgoing:[],incoming:[]},
    groups:[],server:{},commands:[],modal:null,mobile:false,metrics:null,retention:null,mail:null,mfa:null,emailVerification:null,
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
    const passwordForm=options.disable_pwd?'<p class="warning">当前服务已关闭密码登录，请选择下方的单点登录服务。</p>':'<form id="login-form"><div class="login-field"><label for="username">用户名</label><input id="username" autocomplete="username" name="username" placeholder="输入用户名" required></div><div class="login-field"><label for="password">密码</label><input type="password" id="password" name="password" autocomplete="current-password" placeholder="输入密码" required></div>'+cap+'<button type="submit" class="btn btn-primary login-submit">登录控制台 '+icon("arrow",17)+'</button></form><button type="button" class="btn" data-action="forgot-password">忘记密码</button>';
    const ops=(options.ops||[]).map(op=>{const n=typeof op==="string"?op:op.op||op.name;return n?'<button type="button" class="btn" data-action="oidc" data-id="'+esc(n)+'">使用 '+esc(n)+' 登录 '+icon("arrow",14)+'</button>':"";}).join("");
    root.innerHTML='<div class="login-shell"><section class="login-art"><div class="brand-row">'+brand()+'</div><div><p class="eyebrow" style="color:#83e7d4">PRIVATE INFRASTRUCTURE</p><h1>远程设备管理，<br><em>一切尽在掌握。</em></h1><p>统一管理远程设备、联系人、共享群组与用户账户。界面可适应电脑、平板和手机。</p></div><div class="login-foot">RustDesk Self-hosted Control Center</div></section><section class="login-right"><div class="login-card"><p class="eyebrow">WELCOME BACK</p><h2>登录控制台</h2><p class="login-intro">'+esc(state.config.title||"请使用 RustDesk API 账户登录")+'</p>'+passwordForm+'<div class="login-options">'+ops+'</div><p class="login-help">首次安装的管理员密码可在服务端启动日志中查看。请通过 HTTPS 连接管理页面。</p></div></section></div>';
    root.innerHTML+=loginRecoveryModal();
  }
  function loginRecoveryModal(){
    const current=state.modal;
    if(!current)return "";
    if(current.kind==="mfa-login"){
      const emailMethod=current.method==="email_check";
      const label=emailMethod?"邮箱验证码":"动态验证码或一次性恢复码";
      const tip=emailMethod?"请输入发送到该账户邮箱的 6 位验证码。":
        "使用身份验证器中的 6 位验证码，或之前保存的恢复码。";
      const body='<form id="mfa-login-form"><div class="field"><label for="mfa-login-code">'+label+'</label>'+
        '<input id="mfa-login-code" name="code" autocomplete="one-time-code" minlength="6" maxlength="20" required autofocus>'+
        '<small>'+tip+'</small></div></form>';
      return modal("双重认证","账户："+current.username,body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-login-form">完成登录</button>');
    }
    if(current.kind==="forgot-password"){
      const body='<form id="forgot-password-form"><div class="field"><label for="recovery-email">账户邮箱</label>'+
        '<input type="email" name="email" id="recovery-email" autocomplete="email" required></div></form>';
      return modal("找回密码","将向已登记的账户邮箱发送密码恢复链接。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" type="submit" form="forgot-password-form">发送恢复邮件</button>');
    }
    if(current.kind==="accept-invitation"){
      const body='<form id="accept-invitation-form"><div class="form-grid">'+
        '<div class="field"><label for="invited-username">用户名</label>'+
        '<input id="invited-username" name="username" autocomplete="username" minlength="3" maxlength="64" required></div>'+
        '<div class="field"><label for="invited-nickname">显示名称</label>'+
        '<input id="invited-nickname" name="nickname" autocomplete="name"></div>'+
        '<div class="field"><label for="invited-password">登录密码</label>'+
        '<input id="invited-password" name="password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required></div>'+
        '<div class="field"><label for="invited-password-confirm">确认密码</label>'+
        '<input id="invited-password-confirm" name="confirm_password" type="password" autocomplete="new-password" minlength="8" maxlength="72" required></div>'+
        '</div></form>';
      return modal("接受团队邀请","创建账号后，即可使用 RustDesk 自托管服务。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="accept-invitation-form">创建账户</button>');
    }
    if(current.kind==="public-reset-password"){
      const body='<form id="public-password-reset-form"><div class="form-grid">'+
        '<div class="field"><label for="recovery-new-password">新密码</label>'+
        '<input type="password" name="password" id="recovery-new-password" minlength="8" maxlength="72" autocomplete="new-password" required></div>'+
        '<div class="field"><label for="recovery-confirm-password">确认新密码</label>'+
        '<input type="password" name="confirm_password" id="recovery-confirm-password" minlength="8" maxlength="72" autocomplete="new-password" required></div>'+
        '</div></form>';
      return modal("设置新密码","使用邮件中的一次性恢复链接。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" type="submit" form="public-password-reset-form">保存新密码</button>');
    }
    return "";
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
  function returnToWebClient(){
    if(sessionStorage.getItem("rustdesk.console.return")!=="/webclient2/")return false;
    sessionStorage.removeItem("rustdesk.console.return");
    window.location.assign("/webclient2/");
    return true;
  }
  async function bootstrap(){
    const recoveryHash=location.hash||"";
    if(recoveryHash.startsWith("#/invite?")){
      const token=new URLSearchParams(recoveryHash.slice(recoveryHash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;
        sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind:"accept-invitation",token};
      }
    }
    if(recoveryHash.startsWith("#/password-reset?")){
      const token=new URLSearchParams(recoveryHash.slice(recoveryHash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;
        sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind:"public-reset-password",token};
      }
    }
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
      }catch{state.token="";sessionStorage.removeItem("rustdesk.console.token");}
    }
    if(!state.token){await getOptions();render();return;}
    if(returnToWebClient())return;
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
      if(result?.type==="tfa_check"||result?.type==="email_check"){
        if(!result.secret||!result.user?.name)throw Error("双重认证请求不完整");
        state.modal={kind:"mfa-login",username:result.user.name,secret:result.secret,
          method:result.tfa_type||result.type};
        render();
        return;
      }
      if(!result||!result.token)throw Error("登录未返回访问令牌");
      state.token=result.token;sessionStorage.setItem("rustdesk.console.token",result.token);
      state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
      if(returnToWebClient())return;
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
    state.token="";state.user=null;state.isAdmin=false;sessionStorage.removeItem("rustdesk.console.token");localStorage.removeItem("access_token");
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
      }else if(k==="profile"){
        const results=await Promise.all([request("/mfa/status"),request("/mfa/email/status")]);
        state.mfa=results[0];
        state.emailVerification=results[1];
      }else if(k==="mail"){
        state.mail=await request("/mail/settings");
      }else if(k==="retention"){
        state.retention=await request("/audit/retention");
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
    const shortcuts=["devices","shared","contacts",state.isAdmin?"groups":"mygroups"];
    const quick=shortcuts.map(k=>'<button data-view="'+k+'" class="quick-card">'+icon(pages[k].icon,21)+'<strong>'+esc(pages[k].name)+" "+icon("arrow",14)+'</strong></button>').join("");
    const recent=(m.recent||[]).map(x=>'<div class="status-row"><span><strong>'+esc(x.alias||x.hostname||x.id)+'</strong><span class="cell-sub">'+esc(x.id)+'</span></span><span class="pill muted">'+date(x.last_online_time)+'</span></div>').join("")||'<div class="empty">暂无设备</div>';
    return pageHeading("控制台概览","一个统一的空间，用于管理远程设备与共享关系。")+
      '<div class="metric-grid">'+metric("设备数量",state.loading?"…":m.devices,"monitor")+metric(state.isAdmin?"用户数量":"地址簿条目",state.loading?"…":m.users,"users")+metric(state.isAdmin?"用户分组":"联系人数量",state.loading?"…":m.groups===undefined?state.contacts.incoming.length:m.groups,"layers")+metric("当前账户",state.user.username,"shield")+'</div>'+
      '<div class="two-cols"><section class="panel"><div class="panel-header"><div><h2 class="panel-title">快速访问</h2><p class="panel-sub">常用管理功能</p></div></div><div class="quick-grid">'+quick+'</div></section><section class="panel"><div class="panel-header"><div><h2 class="panel-title">最近出现的设备</h2><p class="panel-sub">按最后活动时间显示</p></div><button data-view="devices" class="btn btn-sm">查看全部</button></div><div class="tips">'+recent+'</div></section></div>';
  }
  function rowIdentifier(row){return String(row.row_id===undefined?row.id:row.row_id);}
  function rowActions(row){
    const p=pages[activeKey()];
    if(state.page==="trash")return '<button type="button" class="btn btn-sm" data-action="trash-restore" data-id="'+esc(row.row_id)+'">恢复</button>'+
      '<button type="button" class="btn btn-sm btn-danger" data-action="trash-purge" data-id="'+esc(row.row_id)+'">彻底删除</button>';
    if(state.page==="invitations"){
      if(row.accepted_at)return '<span class="pill good">已接受</span>';
      if(row.revoked_at)return '<span class="pill muted">已撤销</span>';
      return '<button class="btn btn-sm btn-danger" data-action="invitation-revoke" data-id="'+esc(row.id)+'">撤销邀请</button>';
    }
    if(state.page==="connections")return '<button type="button" class="btn btn-sm" data-action="audit-note" data-id="'+esc(row.id)+'">编辑备注</button>';
    if(state.page==="apiTokens"){
      return row.revoked_at?'<span class="pill muted">已撤销</span>':
        '<button type="button" class="btn btn-sm btn-danger" data-action="revoke-api-token" data-id="'+esc(row.id)+'">撤销令牌</button>';
    }
    if(state.page==="activeSessions")return '<button type="button" class="btn btn-sm btn-danger" data-action="session-disconnect" data-id="'+esc(row.conn_id)+'" data-device="'+esc(row.device_id)+'">结束连接</button>';
    if(state.page==="deployed")return '<button type="button" class="btn btn-sm" data-action="deployment-toggle" data-id="'+esc(row.id)+'">'+(row.status==="approved"?"撤销部署":"恢复部署")+"</button>";
    if(p.readonly)return "";
    const id=esc(rowIdentifier(row));
    let out="";
    if(state.isAdmin||state.page!=="devices")out+='<button class="btn btn-sm" data-action="edit" data-id="'+id+'">编辑</button>';
    if(state.page==="users"){out+='<button class="btn btn-sm" data-action="reset-password" data-id="'+id+'">重设密码</button>';}
    if(state.page==="groups"||state.page==="mygroups"){out+='<button class="btn btn-sm" data-action="members" data-id="'+id+'">成员</button>';if(row.type===3)out+='<button class="btn btn-sm" data-action="anchors" data-id="'+id+'">公共成员</button>';}
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
    const description=state.page==="relayNodes"?"配置多个 hbbr 节点及各自优先服务的 IP 网络。hbbs 会在启用自动同步后更新中继列表，并保留节点不可用时的替代选择。":isBook?"在一个页面中管理地址簿分类及其中的设备，原有数据将继续保留。":state.page==="mygroups"?"创建和管理您拥有的群组。其他人的群组需要由其所有者或系统管理员管理。":state.page==="trash"?"找回误删的设备条目，也可以永久删除指定条目。删除整个地址簿分类时将永久清除该分类下的记录。":state.page==="apiTokens"?"创建可随时撤销的管理令牌，用于官方客户端命令行的设备部署与分配。":state.page==="policies"?"为自行管理的 RustDesk 客户端制定设备 ID 白名单、IP 白名单、WebRTC 和可信设备有效期设置。":state.page==="policyAssignments"?"选择设备并分配安全策略。支持策略读取接口的客户端可自动应用设置。":state.page.startsWith("my")&&p.readonly?"这里只显示与您本人或您名下设备有关的记录。":"查看、编辑和管理相关数据。";
    const exportKinds={loginlogs:"login_log",connections:"audit_conn",transfers:"audit_file",alarms:"audit_alarm",operations:"audit_operation"};
    const exportBtn=exportKinds[state.page]?'<button type="button" class="btn" data-action="audit-export" data-id="'+exportKinds[state.page]+'">下载 CSV（最多 10000 条）</button>':"";
    const csvActions=isBook?
      '<button type="button" class="btn" data-view="trash">查看回收站</button>'+
      '<button type="button" class="btn" data-action="csv-import">导入 CSV</button>'+
      '<button type="button" class="btn" data-action="csv-export">导出 CSV</button>':"";
    return pageHeading(isBook?"地址簿":p.name,description,add+exportBtn+csvActions)+tabs+'<section class="panel">'+search+table(p,rows)+pagination()+'</section>';
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
  function retentionPage(){
    const data=state.retention||{};
    const fields=[["connection_days","远程连接记录"],["file_days","文件传输记录"],["alarm_days","安全告警"],["login_days","用户登录记录"],["operation_days","管理操作记录"]];
    const inputs=fields.map(([key,label])=>
      '<div class="field"><label for="'+key+'">'+label+'</label><input id="'+key+'" name="'+key+'" type="number" min="0" max="3650" step="1" required value="'+esc(data[key]??0)+'"><small>单位：天。填写 0 表示永久保留。</small></div>'
    ).join("");
    return pageHeading("日志保留设置","按日志类型设置自动清理期限。新安装默认永久保留，最长可设置 3650 天。")+
      '<section class="panel" style="max-width:860px"><div class="panel-header"><div><h2 class="panel-title">自动清理期限</h2><p class="panel-sub">服务启动后以及此后每 24 小时检查一次。您可随时修改期限。</p></div></div><form id="retention-form" class="tips"><div class="form-grid">'+inputs+'</div><button type="submit" class="btn btn-primary" style="margin-top:18px">保存设置</button></form></section>';
  }
  async function exportAudit(kind){
    try{
      const result=await fetch(API+"/audit/export?kind="+encodeURIComponent(kind),{
        method:"GET",headers:{"api-token":state.token,"Accept":"text/csv"},
        cache:"no-store",credentials:"same-origin"
      });
      if(!result.ok)throw Error("导出失败，HTTP "+result.status);
      const blob=await result.blob(),url=URL.createObjectURL(blob);
      try{
        const link=document.createElement("a");
        link.href=url;link.download=kind+".csv";
        document.body.append(link);link.click();link.remove();
      }finally{URL.revokeObjectURL(url);}
      toast("已生成 CSV 文件");
    }catch(err){toast(err.message,true);}
  }
  function mailPage(){
    const m=state.mail||{};
    const security=m.security||"starttls";
    const options=[["starttls","STARTTLS（通常为 587 端口）"],["tls","TLS 加密连接（通常为 465 端口）"],["none","不加密（适用于可信的内部邮件服务）"]]
      .map(([value,label])=>'<option value="'+value+'" '+(value===security?"selected":"")+'>'+label+'</option>').join("");
    const textField=(name,label,value,type="text")=>
      '<div class="field"><label for="mail-'+name+'">'+esc(label)+'</label><input id="mail-'+name+
      '" type="'+type+'" name="'+name+'" value="'+esc(value||"")+'" autocomplete="off"></div>';
    const inputs=
      '<label class="field"><span>启用邮件发送</span><input name="enabled" type="checkbox" '+(m.enabled?"checked":"")+'></label>'+
      textField("host","SMTP 主机",m.host)+
      '<div class="field"><label for="mail-port">端口</label><input id="mail-port" name="port" type="number" min="1" max="65535" value="'+esc(m.port||587)+'" required></div>'+
      '<div class="field"><label for="mail-security">连接方式</label><select id="mail-security" name="security">'+options+'</select></div>'+
      textField("username","SMTP 用户名",m.username)+
      '<div class="field"><label for="mail-password">SMTP 密码</label><input id="mail-password" name="password" type="password" autocomplete="new-password" placeholder="'+(m.password_configured?"留空可保留已配置的密码":"请输入 SMTP 密码（可留空）")+'">'+
      '<small>保存密码前需在服务器设置 RUSTDESK_API_MAIL_ENCRYPTION_KEY。</small></div>'+
      '<label class="field"><span>清除已保存的 SMTP 密码</span><input name="clear_password" type="checkbox"></label>'+
      textField("sender_email","发件邮箱",m.sender_email,"email")+
      textField("sender_name","发件人名称",m.sender_name)+
      textField("base_url","用户访问控制台的公开地址",m.base_url,"url");
    return pageHeading("SMTP 邮件服务","配置邮件发送后，可以向团队成员发送测试邮件并用于账户恢复。")+
      '<section class="panel" style="max-width:960px"><div class="panel-header"><div><h2 class="panel-title">发送邮件设置</h2>'+
      '<p class="panel-sub">SMTP 配置保存在数据库中。密码经过加密保存，不会再次显示原文。</p></div></div>'+
      '<form id="mail-settings-form" class="tips"><div class="form-grid">'+inputs+'</div>'+
      '<button class="btn btn-primary" type="submit" style="margin-top:16px">保存邮件设置</button></form></section>'+
      '<section class="panel" style="max-width:960px;margin-top:16px"><div class="panel-header"><h2 class="panel-title">发送测试邮件</h2></div>'+
      '<form id="mail-test-form" class="tips"><div class="form-grid">'+textField("to","收件邮箱","","email")+
      '</div><button type="submit" class="btn btn-primary" style="margin-top:16px">发送测试邮件</button></form></section>';
  }

  function profile(){
    const m=state.mfa||{};
    const mfaText=m.enabled?
      "已启用身份验证器双重认证，剩余恢复码："+esc(m.remaining_recovery_codes??0)+" 个。":
      "尚未启用双重认证，开启后登录需要输入动态验证码。";
    const mfaActions=m.enabled?
      '<button type="button" class="btn" data-action="mfa-regenerate">重新生成恢复码</button>'+
      '<button type="button" class="btn btn-danger" data-action="mfa-disable">解除双重认证</button>':
      '<button type="button" class="btn btn-primary" data-action="mfa-setup">启用 TOTP 双重认证</button>';
    const email=state.emailVerification||{};
    const emailDescription=email.email?
      "当前邮箱："+esc(email.email)+(email.enabled?"（已启用登录验证）":"（尚未启用邮箱登录验证）"):
      "当前账户没有有效的邮箱地址，请联系管理员设置邮箱。";
    const emailButton=email.enabled?
      '<button type="button" class="btn btn-danger" data-action="email-verification-disable">关闭邮箱登录验证</button>':
      '<button type="button" class="btn btn-primary" data-action="email-verification-setup" '+
      (!email.mail_configured||!email.email?'disabled':'')+'>发送邮箱验证码并启用</button>';
    return pageHeading("个人设置","管理自己的登录密码与可选双重认证。")+
      '<section class="panel" style="max-width:650px"><div class="panel-header"><h2 class="panel-title">'+esc(state.user.username)+'</h2></div><form id="password-form" class="tips" style="display:grid;gap:15px"><div class="field"><label>当前密码</label><input type="password" name="old_password" autocomplete="current-password" required></div><div class="field"><label>新密码</label><input type="password" name="new_password" autocomplete="new-password" minlength="4" required></div><button class="btn btn-primary" type="submit">修改密码</button></form></section>'+
      '<section class="panel" style="max-width:650px;margin-top:16px"><div class="panel-header"><div><h2 class="panel-title">双重认证（TOTP）</h2><p class="panel-sub">'+mfaText+'</p></div></div><div class="tips" style="display:flex;gap:10px;flex-wrap:wrap">'+mfaActions+'</div></section>'+
      '<section class="panel" style="max-width:650px;margin-top:16px"><div class="panel-header"><div><h2 class="panel-title">邮箱登录验证</h2><p class="panel-sub">'+emailDescription+'</p></div></div><div class="tips" style="display:grid;gap:10px"><p>启用后将通过邮件接收登录验证码。已启用 TOTP 的账户优先使用身份验证器。</p>'+
      emailButton+(!email.mail_configured?'<p>管理员尚未启用 SMTP 邮件服务。</p>':'')+'</div></section>';
  }
  function renderPage(){
    if(state.page==="overview")return overview();
    if(state.page==="contacts")return contacts();
    if(state.page==="shared")return shared();
    if(state.page==="server")return server();
    if(state.page==="retention")return retentionPage();
    if(state.page==="mail")return mailPage();
    if(state.page==="profile")return profile();
    return genericList();
  }
  function fieldInput(field,row){
    const [name,label,type="text",required=false]=field;
    const value=row&&row[name]!==undefined?row[name]:type==="status"?1:type==="group_type"?1:type==="rule_type"?1:type==="rule"?1:type==="checkbox"?false:"";
    let input;
    if(type==="checkbox")input='<label class="check"><input type="checkbox" name="'+esc(name)+'" '+(value?"checked":"")+'>启用</label>';
    else if(type==="textarea")input='<textarea name="'+esc(name)+'" rows="3">'+esc(value)+'</textarea>';
    else if(type==="trusted_days"){
      const choices=[[0,"不限期限"],[1,"1 天"],[7,"7 天"],[30,"30 天"],[90,"90 天"]];
      input='<select name="'+esc(name)+'">'+choices.map(([days,text])=>'<option value="'+days+'" '+(Number(value)===days?"selected":"")+'>'+text+'</option>').join("")+'</select>';
    }
    else if(["group_type","status","oauth_type","rule_type","rule"].includes(type)){
      const opts={group_type:[[1,"普通组"],[2,"共享组"],[3,"公共锚点组"]],status:[[1,"已启用"],[2,"已停用"]],oauth_type:[["oidc","OIDC"],["github","GitHub"],["google","Google"],["linuxdo","Linux.do"]],rule_type:[[1,"用户"],[2,"群组"],[3,"所有用户"]],rule:[[1,"只读"],[2,"读写"],[3,"完全控制"]]}[type];
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
      const p=pages[activeKey()],fields=(p.fields||[]).filter(f=>{
        if(state.page==="users"&&m.action==="edit"&&f[0]==="password")return false;
        if(!state.isAdmin&&state.page==="addressbooks"&&(f[0]==="user_id"))return false;
        return true;
      }).map(f=>state.page==="oauth"&&m.action==="edit"&&f[0]==="client_secret"?[f[0],"Client Secret（留空表示保持当前密钥）",f[2],false]:f);
      const body='<form id="edit-form"><input type="hidden" name="id" value="'+esc(m.data.id||"")+'"><input type="hidden" name="row_id" value="'+esc(m.data.row_id||"")+'"><div class="form-grid">'+fields.map(f=>fieldInput(f,m.data)).join("")+'</div></form>';
      const saveButton=state.page==="apiTokens"?
        '<button class="btn btn-primary" type="button" data-action="submit-api-token">创建令牌</button>':
        state.page==="invitations"?
        '<button class="btn btn-primary" type="button" data-action="submit-invitation">发送邀请</button>':
        '<button class="btn btn-primary" type="submit" form="edit-form">保存</button>';
      return modal(m.action==="create"?"新建"+p.name:"编辑"+p.name,"修改完成后保存，变更会立即生效。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+saveButton);
    }
    if(m.kind==="csv"){
      const importing=m.operation==="import";
      const ownerInput=state.isAdmin?'<div class="field"><label for="csv-owner">所属账户 ID</label>'+
        '<input type="number" id="csv-owner" min="1" value="'+esc(state.user?.id||1)+'" required>'+
        '<small>管理员可以指定需要管理的用户账户。</small></div>':"";
      const upload=importing?'<div class="field"><label for="csv-file">选择 UTF-8 CSV 文件</label>'+
        '<input id="csv-file" type="file" accept=".csv,text/csv" required>'+
        '<small>最多 4 MB、5000 条设备。根据设备 ID 和地址簿编号更新同一条目，保留原有连接密码。</small></div>':
        '<p>将导出设备 ID、别名、主机名、系统、标签、备注等字段。连接密码和认证信息不包含在导出文件中。</p>';
      const body='<div class="form-grid">'+ownerInput+upload+'</div>';
      return modal(importing?"导入地址簿 CSV":"导出地址簿 CSV","仅处理指定账户的地址簿条目。",body,
        '<button class="btn" data-action="close-modal">取消</button>'+
        '<button class="btn btn-primary" data-action="csv-submit">'+(importing?"导入 CSV":"下载 CSV")+'</button>');
    }
    if(m.kind==="email-verification"){
      const body='<p>验证码已发送到当前账户邮箱，有效期为 10 分钟。</p>'+
        '<form id="email-verification-form"><div class="field">'+
        '<label for="email-verification-code">6 位邮箱验证码</label>'+
        '<input id="email-verification-code" name="code" inputmode="numeric" pattern="[0-9]{6}" minlength="6" maxlength="6" autocomplete="one-time-code" required></div></form>';
      return modal("启用邮箱登录验证","验证邮箱后，后续密码登录将发送一次性验证码。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="email-verification-form">启用邮箱验证</button>');
    }
    if(m.kind==="mfa-setup"){
      const body='<p>在身份验证器中添加账户，输入下面的密钥或导入配置链接，再填写当前 6 位验证码。</p>'+
        '<div class="field"><label>身份验证器密钥</label><input readonly value="'+esc(m.secret)+'" autocomplete="off"></div>'+
        '<div class="field"><label>身份验证器配置链接</label><textarea readonly rows="3">'+esc(m.uri)+'</textarea></div>'+
        '<form id="mfa-activate-form"><div class="field"><label for="mfa-activate-code">6 位动态验证码</label>'+
        '<input id="mfa-activate-code" name="code" inputmode="numeric" pattern="[0-9]{6}" minlength="6" maxlength="6" autocomplete="one-time-code" required></div></form>';
      return modal("启用双重认证","身份验证器密钥只在绑定时显示，验证成功后生成恢复码。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-activate-form">验证并启用</button>');
    }
    if(m.kind==="mfa-codes"){
      const list=(m.codes||[]).map(code=>'<li><code>'+esc(code)+'</code></li>').join("");
      const body='<p>请保存下面 6 个一次性恢复码。如果身份验证器无法使用，可以在登录时输入其中一个恢复码。</p>'+
        '<ol style="margin-top:12px">'+list+'</ol><p>恢复码关闭后不会再次显示。</p>';
      return modal("保存双重认证恢复码","建议将恢复码保存在安全的位置。",body,
        '<button type="button" class="btn btn-primary" data-action="close-modal">我已保存恢复码</button>');
    }
    if(m.kind==="mfa-action"){
      const actionName=m.action==="disable"?"解除双重认证":"重新生成恢复码";
      const body='<form id="mfa-confirm-form"><div class="field"><label for="mfa-confirm-code">动态验证码或恢复码</label>'+
        '<input id="mfa-confirm-code" name="code" autocomplete="one-time-code" minlength="6" maxlength="20" required></div></form>';
      return modal(actionName,"确认后将应用当前账户的双重认证设置。",body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="submit" class="btn btn-primary" form="mfa-confirm-form">确认</button>');
    }
    if(m.kind==="audit-note"){
      const body='<form id="audit-note-form">'+
        '<div class="field"><label for="audit-note-input">连接备注</label>'+
        '<textarea id="audit-note-input" name="note" rows="7" maxlength="2000" placeholder="填写连接用途、问题描述或处理记录">'+esc(m.data.note||"")+'</textarea>'+
        '<small>最多 2000 字。内容将显示在连接记录中，并可随 CSV 导出。</small></div></form>';
      return modal("编辑连接备注","连接编号："+m.data.id,body,
        '<button type="button" class="btn" data-action="close-modal">取消</button>'+
        '<button type="button" class="btn btn-primary" data-action="audit-note-save">保存备注</button>');
    }
    if(m.kind==="created-api-token"){
      const body='<div class="field"><label for="api-token-secret">请保存以下令牌</label>'+
        '<input id="api-token-secret" readonly value="'+esc(m.token)+'" autocomplete="off">'+
        '<small>令牌原文仅显示一次，关闭页面后无法再次查看。</small></div>';
      return modal("API 管理令牌已创建","可用于官方 RustDesk 客户端的 --assign 和 --deploy 命令。",body,
        '<button class="btn btn-primary" data-action="copy-api-token">复制令牌</button><button class="btn" data-action="close-modal">关闭</button>');
    }
    if(m.kind==="reset-password"){
      const body='<form id="reset-password-form"><input type="hidden" name="id" value="'+esc(m.user.id)+'"><div class="form-grid"><div class="field full"><label for="new-user-password">新密码 <span class="required">*</span></label><input id="new-user-password" name="password" type="password" required minlength="8" maxlength="72" autocomplete="new-password"></div></div></form>';
      return modal("重设用户密码","账户："+m.user.username,body,'<button class="btn" type="button" data-action="close-modal">取消</button><button class="btn btn-primary" type="submit" form="reset-password-form">保存新密码</button>');
    }
    if(m.kind==="members"||m.kind==="anchors"){
      const anchor=m.kind==="anchors";
      const rows=(m.users||[]).map(u=>'<label class="status-row" style="cursor:pointer"><span><strong>'+esc(u.nickname||u.username)+'</strong><span class="cell-sub">@'+esc(u.username)+' · ID '+esc(u.id)+'</span></span><input style="width:18px;height:18px;accent-color:var(--main)" type="checkbox" name="selected" value="'+esc(u.id)+'" '+(m.selected.has(u.id)?"checked":"")+'></label>').join("");
      const personalHint=m.personal?'<div class="field" style="margin-top:18px"><label for="lookup-username">通过用户名添加成员</label><div style="display:flex;gap:9px"><input id="lookup-username" placeholder="输入完整用户名" autocomplete="off"><button class="btn" type="button" data-action="lookup-user">查找并选中</button></div><small>请输入准确的用户名，系统不会公开其他群组的成员列表。</small></div>':"";
      const details='<p class="subtitle">'+(anchor?"只有选中的公共成员及其设备会显示给该组的成员，包括管理员。":"当前成员已经勾选。取消勾选可以移除成员，添加新用户后保存即可一次修改多人。")+'</p>'+personalHint+'<div class="field" style="margin-top:20px"><label for="user-filter">筛选已显示的用户</label><input id="user-filter" placeholder="输入用户名或编号"></div><form id="member-form"><div class="field" style="margin-top:16px"><label>补充用户 ID（逗号分隔，可选）</label><input name="manual_ids" placeholder="例如：2,5,8"></div><div id="member-choices" style="margin-top:12px">'+rows+'</div></form>';
      const footer='<button class="btn" data-action="close-modal">取消</button><button class="btn btn-primary" data-action="member-replace">'+(anchor?"保存公共成员":"保存群组成员")+'</button>';
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
    for(const f of p.fields.filter(f=>{
      if(state.page==="users"&&state.modal.action==="edit"&&f[0]==="password")return false;
      if(!state.isAdmin&&state.page==="addressbooks"&&f[0]==="user_id")return false;
      return true;
    })){
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
      const result=await request(endpoint(activeKey())+"/"+state.modal.action,"POST",data);
      if(p.endpoint==="api_token"&&state.modal.action==="create"){
        if(!result?.token)throw Error("服务器未返回新令牌");
        state.modal={kind:"created-api-token",token:result.token};
        render();
        return;
      }
      state.modal=null;
      toast(p.endpoint==="invite"?"邀请邮件已发送":"保存成功");
      await load();
    }catch(e){toast(e.message,true);}
  }
  async function deleteEntry(id){
    const p=pages[activeKey()],r=state.rows.find(x=>rowIdentifier(x)===id);
    if(!r||!window.confirm("确定删除这条记录吗？"))return;
    try{await request(endpoint(activeKey())+"/delete","POST",{id:r.id,row_id:r.row_id});toast("已删除");await load();}
    catch(e){toast(e.message,true);}
  }
  async function groupMembers(id,anchor){
    const group=state.rows.find(r=>String(r.id)===String(id));
    if(!group)return;
    const personal=state.page==="mygroups";
    try{
      const rootPath=personal?"/my/group":"/group";
      const selected=await request(rootPath+"/"+(anchor?"anchors":"members")+"?group_id="+encodeURIComponent(id));
      const rows=personal?(selected.users||[]):((await request("/user/list?page=1&page_size=500")).list||[]);
      const displayed=new Set(rows.map(x=>Number(x.id)));
      for(const member of selected.users||[]){
        if(!displayed.has(Number(member.id))){rows.push(member);displayed.add(Number(member.id));}
      }
      state.modal={kind:anchor?"anchors":"members",group,personal,selected:new Set((selected.user_ids||[]).map(Number)),users:rows};
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
      const base=m.personal?"/my/group":"/group";
      const url=m.kind==="anchors"?base+"/anchors":base+"/members/batch";
      const payload=m.kind==="anchors"?{group_id:m.group.id,user_ids:ids}:{group_id:m.group.id,user_ids:ids,action};
      await request(url,"POST",payload);
      state.modal=null;toast("群组成员已更新");await load();
    }catch(e){toast(e.message,true);}
  }
  async function transferAddressBookCSV(){
    const operation=state.modal?.operation;
    if(state.modal?.kind!=="csv"||!["import","export"].includes(operation))return;
    const ownerField=document.getElementById("csv-owner");
    const owner=ownerField?Number(ownerField.value):null;
    if(ownerField&&(!Number.isSafeInteger(owner)||owner<=0)){toast("请填写有效的所属账户 ID",true);return;}
    const path=endpoint("addressbooks")+"/"+operation+(ownerField?"?user_id="+encodeURIComponent(owner):"");
    const headers={"api-token":state.token,"Accept":operation==="import"?"application/json":"text/csv"};
    let body;
    if(operation==="import"){
      const file=document.getElementById("csv-file")?.files?.[0];
      if(!file){toast("请选择 CSV 文件",true);return;}
      if(file.size>4*1024*1024){toast("CSV 文件不能超过 4 MB",true);return;}
      body=await file.text();
      headers["Content-Type"]="text/csv; charset=utf-8";
    }
    try{
      const reply=await fetch(API+path,{
        method:operation==="import"?"POST":"GET",headers,body,
        cache:"no-store",credentials:"same-origin"
      });
      if(operation==="import"){
        const res=await reply.json();
        if(!reply.ok||res.code!==0)throw Error(res.error||res.message||"地址簿导入失败");
        const result=res.data||{};
        state.modal=null;
        toast("CSV 导入完成：新增 "+(result.created||0)+" 条，更新 "+(result.updated||0)+" 条");
        await load();
      }else{
        if(!reply.ok)throw Error("地址簿导出失败，HTTP "+reply.status);
        const blob=await reply.blob(),url=URL.createObjectURL(blob);
        try{
          const link=document.createElement("a");
          link.href=url;link.download="rustdesk-addressbook.csv";
          document.body.append(link);link.click();link.remove();
        }finally{URL.revokeObjectURL(url);}
        state.modal=null;render();toast("CSV 文件已生成");
      }
    }catch(err){toast(err.message,true);}
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
    if(a==="close-modal"){
      if(["public-reset-password","accept-invitation"].includes(state.modal?.kind))location.hash="#/overview";
      state.modal=null;render();return;
    }
    if(a==="forgot-password"){state.modal={kind:"forgot-password"};render();return;}
    if(a==="logout"){await logout();return;}
    if(a==="submit-invitation"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="submit-api-token"){
      const form=document.getElementById("edit-form");
      if(form&&form.reportValidity())await saveEditor(form);
      return;
    }
    if(a==="trash-restore"||a==="trash-purge"){
      const row=state.rows.find(item=>String(item.row_id)===String(el.dataset.id));
      if(!row)return;
      const purge=a==="trash-purge";
      if(purge&&!window.confirm("此操作会永久删除设备 "+row.id+" 的地址簿记录，确定继续吗？"))return;
      try{
        await request(endpoint("trash")+"/"+(purge?"purge":"restore"),"POST",{row_id:Number(row.row_id)});
        toast(purge?"地址簿条目已永久删除":"地址簿条目已恢复");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="csv-import"||a==="csv-export"){
      state.modal={kind:"csv",operation:a==="csv-import"?"import":"export"};
      render();
      return;
    }
    if(a==="csv-submit"){await transferAddressBookCSV();return;}
    if(a==="email-verification-setup"){
      try{
        const setup=await request("/mfa/email/setup","POST",{});
        state.modal={kind:"email-verification",secret:setup.secret};
        render();toast("邮箱验证码已发送");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="email-verification-disable"){
      if(!window.confirm("关闭后将使用现有的密码登录方式。确认关闭邮箱验证吗？"))return;
      try{
        await request("/mfa/email/disable","POST",{});
        state.emailVerification=await request("/mfa/email/status");
        render();toast("邮箱登录验证已关闭");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="mfa-setup"){
      try{
        const setup=await request("/mfa/setup","POST",{});
        state.modal={kind:"mfa-setup",...setup};render();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="mfa-disable"||a==="mfa-regenerate"){
      state.modal={kind:"mfa-action",action:a==="mfa-disable"?"disable":"regenerate"};render();
      return;
    }
    if(a==="audit-note"){
      const record=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(record){state.modal={kind:"audit-note",data:record};render();}
      return;
    }
    if(a==="audit-note-save"){
      if(state.modal?.kind!=="audit-note")return;
      const note=document.getElementById("audit-note-input")?.value;
      try{
        await request("/audit_conn/note","POST",{id:Number(state.modal.data.id),note});
        state.modal=null;toast("连接备注已保存");await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="invitation-revoke"){
      const row=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!row||!window.confirm("确定撤销发送到 "+row.email+" 的邀请吗？"))return;
      try{await request("/invite/revoke","POST",{id:Number(row.id)});toast("邀请已撤销");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="revoke-api-token"){
      const item=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!item||!window.confirm("撤销后使用此令牌的自动部署将停止。确定撤销吗？"))return;
      try{await request("/api_token/delete","POST",{id:Number(item.id)});toast("管理令牌已撤销");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="copy-api-token"){
      const field=document.getElementById("api-token-secret");
      if(!field)return;
      field.focus();field.select();
      try{
        if(window.navigator?.clipboard?.writeText){
          await window.navigator.clipboard.writeText(field.value);
          toast("已复制令牌");
        }else toast("已选择令牌文本，请手动复制");
      }catch{toast("已选择令牌文本，请手动复制");}
      return;
    }
    if(a==="session-disconnect"){
      const deviceId=el.dataset.device, connId=Number(el.dataset.id);
      if(!deviceId||!connId)return;
      if(!window.confirm("确定请求结束设备 "+deviceId+" 的连接 "+connId+" 吗？"))return;
      try{
        await request("/session/disconnect","POST",{device_id:deviceId,conn_id:connId});
        toast("结束连接请求已提交，客户端下次发送心跳时会收到通知");
        await load();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(a==="deployment-toggle"){
      const item=state.rows.find(x=>String(x.id)===String(el.dataset.id));
      if(!item)return;
      const status=item.status==="approved"?"revoked":"approved";
      const question=status==="revoked"?"撤销后该设备将无法在严格登记模式下重新登记。确认撤销吗？":"确定恢复该设备的登记许可吗？";
      if(!window.confirm(question))return;
      try{await request("/device_deployment/status","POST",{id:Number(item.id),status});toast("设备部署状态已更新");await load();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(a==="webclient"){window.location.assign("/webclient2/");return;}
    if(a==="audit-export"){await exportAudit(el.dataset.id);return;}
    if(a==="captcha"){await captcha();render();return;}
    if(a==="oidc"){await oidc(el.dataset.id);return;}
    if(a==="create"||a==="edit"){await openEditor(el.dataset.id,a);return;}
    if(a==="delete"){await deleteEntry(el.dataset.id);return;}
    if(a==="reset-password"){
      const user=state.rows.find(r=>String(r.id)===String(el.dataset.id));
      if(user){state.modal={kind:"reset-password",user};render();}
      return;
    }
    if(a==="members"||a==="anchors"){await groupMembers(el.dataset.id,a==="anchors");return;}
    if(a==="lookup-user"){
      const m=state.modal,field=document.getElementById("lookup-username");
      if(!m?.personal||!field)return;
      const username=field.value.trim();
      if(username.length<2){toast("请输入完整的用户名",true);return;}
      try{
        const selected=document.querySelectorAll('#member-choices input[name="selected"]:checked');
        m.selected=new Set([...selected].map(x=>Number(x.value)));
        const found=await request("/my/group/user/lookup?username="+encodeURIComponent(username));
        if(!m.users.some(x=>Number(x.id)===Number(found.id)))m.users.push(found);
        m.selected.add(Number(found.id));
        toast("已选中成员："+found.username);
        render();
      }catch(e){toast(e.message,true);}
      return;
    }
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
    if(!["login-form","edit-form","contact-form","command-form","password-form","reset-password-form","retention-form","mail-settings-form","mail-test-form","forgot-password-form","public-password-reset-form","accept-invitation-form","mfa-login-form","mfa-activate-form","mfa-confirm-form","email-verification-form"].includes(f.id))return;
    event.preventDefault();
    if(f.id==="mfa-login-form"){
      const pending=state.modal;
      if(pending?.kind!=="mfa-login")return;
      try{
        const result=await request("/login","POST",{
          username:pending.username,
          type:pending.method==="email_check"?"email_code":"tfa_code",
          secret:pending.secret,tfaCode:f.elements.code.value.trim(),
          verificationCode:f.elements.code.value.trim(),platform:"web"
        });
        if(!result?.token)throw Error("登录未返回访问令牌");
        state.modal=null;state.token=result.token;
        sessionStorage.setItem("rustdesk.console.token",result.token);
        state.user=result;state.isAdmin=Array.isArray(result.route_names)&&result.route_names.includes("*");
        if(returnToWebClient())return;
        state.page="overview";location.hash="#/overview";render();await load();
        toast("登录成功");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="email-verification-form"){
      if(state.modal?.kind!=="email-verification")return;
      try{
        await request("/mfa/email/enable","POST",{
          secret:state.modal.secret,code:f.elements.code.value.trim()
        });
        state.emailVerification=await request("/mfa/email/status");
        state.modal=null;render();toast("邮箱登录验证已启用");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="mfa-activate-form"){
      try{
        const result=await request("/mfa/enable","POST",{code:f.elements.code.value.trim()});
        if(!Array.isArray(result?.recovery_codes))throw Error("恢复码生成失败");
        state.mfa=await request("/mfa/status");
        state.modal={kind:"mfa-codes",codes:result.recovery_codes};
        render();
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="mfa-confirm-form"){
      const action=state.modal?.action;
      if(!["disable","regenerate"].includes(action))return;
      try{
        const result=await request(action==="disable"?"/mfa/disable":"/mfa/recovery/regenerate",
          "POST",{code:f.elements.code.value.trim()});
        state.mfa=await request("/mfa/status");
        state.modal=action==="regenerate"?{kind:"mfa-codes",codes:result.recovery_codes}:null;
        render();
        if(action==="disable")toast("双重认证已解除");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="accept-invitation-form"){
      const password=f.elements.password.value;
      if(password!==f.elements.confirm_password.value){
        toast("两次输入的密码不一致",true);return;
      }
      try{
        await request("/invite/accept","POST",{
          token:state.modal?.token,
          username:f.elements.username.value.trim(),
          nickname:f.elements.nickname.value.trim(),
          password
        });
        state.modal=null;location.hash="#/overview";render();
        toast("账户已创建，请使用新账户登录");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="forgot-password-form"){
      try{
        await request("/password/forgot","POST",{email:f.elements.email.value.trim()});
        state.modal=null;render();
        toast("如果邮箱已登记，将收到密码恢复邮件");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="public-password-reset-form"){
      const password=f.elements.password.value;
      if(password!==f.elements.confirm_password.value){
        toast("两次输入的新密码不一致",true);return;
      }
      try{
        await request("/password/reset","POST",{token:state.modal?.token,password});
        state.modal=null;
        location.hash="#/overview";
        render();
        toast("密码已更新，请使用新密码登录");
      }catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="mail-settings-form"){
      const values=Object.fromEntries(new FormData(f).entries());
      values.enabled=f.elements.enabled.checked;
      values.clear_password=f.elements.clear_password.checked;
      values.port=Number(values.port);
      try{state.mail=await request("/mail/settings","POST",values);toast("邮件设置已保存");render();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="mail-test-form"){
      try{await request("/mail/test","POST",{to:f.elements.to.value.trim()});toast("测试邮件已发送");}
      catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="retention-form"){
      const values=Object.fromEntries(new FormData(f).entries());
      for(const key of Object.keys(values))values[key]=Number(values[key]);
      try{state.retention=await request("/audit/retention","POST",values);toast("日志保留设置已保存");render();}
      catch(err){toast(err.message,true);}
      return;
    }
    if(f.id==="login-form")await handleLogin(f);
    if(f.id==="edit-form")await saveEditor(f);
    if(f.id==="reset-password-form"){
      const id=state.modal?.user.id, password=f.elements.password.value;
      if(!id||password.length<8)return;
      try{
        await request("/user/changePwd","POST",{id,password});
        state.modal=null;toast("用户密码已更新");await load();
      }catch(e){toast(e.message,true);}
    }
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
  window.addEventListener("hashchange",()=>{
    const hash=location.hash||"";
    for(const [prefix,kind] of [["#/password-reset?","public-reset-password"],["#/invite?","accept-invitation"]]){
      if(!hash.startsWith(prefix))continue;
      const token=new URLSearchParams(hash.slice(hash.indexOf("?")+1)).get("token");
      if(token){
        state.token="";state.user=null;sessionStorage.removeItem("rustdesk.console.token");
        state.modal={kind,token};render();
      }
      return;
    }
    if(!state.user)return;
    const viewName=hash.replace(/^#\/?/,"").split("?")[0];
    if(available(viewName)&&state.page!==viewName)view(viewName);
  });
  bootstrap().catch(e=>{toast(e.message,true);render();});
})();
