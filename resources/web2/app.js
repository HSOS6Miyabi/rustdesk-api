"use strict";
(() => {
  const el = id => document.getElementById(id);
  const store = (key, value) => localStorage.setItem(key, value);
  const read = key => localStorage.getItem(key)||"";
  const linkedDevice=new URLSearchParams(window.location.search||"");
  const initialDeviceId=String(linkedDevice.get("id")||"").trim();
  const state = { token:sessionStorage.getItem("rustdesk.console.token")||"",captchaId:"",ready:false,defaultsLoaded:false,
    pageFullscreen:false,autoConnect:Boolean(initialDeviceId),activeName:String(linkedDevice.get("name")||"").slice(0,160) };
  const config = {
    host: el("v2-host"), key: el("v2-key"), id: el("v2-id")
  };
  config.host.value=read("custom-rendezvous-server")||read("rendezvous-server");
  config.key.value=read("key");
  config.id.value=initialDeviceId||read("remote-id")||read("id");
  function updateDeviceIdentity(){
    el("v2-native-client").title=state.activeName||"RustDesk 远程桌面";
  }
  config.id.addEventListener("input",()=>{state.activeName="";state.autoConnect=false;updateDeviceIdentity();});
  updateDeviceIdentity();
  el("v2-webrtc").checked=sessionStorage.getItem("rustdesk.webclient.webrtc")==="Y";
  function message(text){el("v2-message").textContent=text;}
  const stage=document.querySelector(".stage");
  const firstClient=el("v2-native-client");
  const sessionTabs=el("v2-session-tabs");
  const sessionPanels=el("v2-session-panels");
  const emptySession=el("v2-empty-session");
  const sessions=new Map();
  const modes={desktop:{route:"connect",label:"桌面"},file:{route:"file-transfer",label:"文件"},camera:{route:"view-camera",label:"摄像"},terminal:{route:"terminal",label:"终端"},adminTerminal:{route:"terminal-admin",label:"管理员终端"}};
  const modeButtons=Object.fromEntries(Object.keys(modes).map(mode=>[mode,el("v2-mode-"+mode)]));
  let selectedMode="desktop";
  function renderModeTabs(){
    for(const [mode,button] of Object.entries(modeButtons)){
      button.setAttribute("aria-selected",String(mode===selectedMode));
    }
  }
  let firstClientInUse=false;
  let activeSessionKey="";
  function showSession(key){
    const chosen=sessions.get(key);
    if(!chosen)return;
    activeSessionKey=key;
    config.id.value=chosen.id;
    config.host.value=chosen.host;
    state.activeName=chosen.name;
    selectedMode=chosen.mode;
    renderModeTabs();
    for(const [sessionKey,session] of sessions){
      session.frame.classList.toggle("session-inactive",sessionKey!==key);
    }
    emptySession.hidden=true;
    renderSessionTabs();
  }
  function closeSession(key){
    const session=sessions.get(key);
    if(!session)return;
    sessions.delete(key);
    if(session.frame===firstClient){
      firstClient.src="about:blank";
      firstClient.classList.add("session-inactive");
      firstClientInUse=false;
    }else{
      session.frame.remove();
    }
    if(activeSessionKey===key){
      activeSessionKey="";
      const next=sessions.keys().next().value;
      if(next)showSession(next);
      else {emptySession.hidden=false;renderSessionTabs();}
    }else renderSessionTabs();
    message(sessions.size?"已断开："+session.name:"当前连接已关闭，可以创建新的连接标签。");
  }
  function renderSessionTabs(){
    sessionTabs.replaceChildren();
    sessionTabs.hidden=false;
    if(sessions.size===0){
      const newTab=document.createElement("button");
      newTab.type="button";
      newTab.className="session-tab-new";
      newTab.textContent="＋ 新建连接";
      newTab.title="创建远程连接标签";
      newTab.setAttribute("role","tab");
      newTab.setAttribute("aria-selected","true");
      newTab.addEventListener("click",()=>{
        if(config.host.value.trim()&&config.id.value.trim())connectToDevice(selectedMode);
        else {config.id.focus();message("请选择或输入设备 ID 后发起连接。");}
      });
      sessionTabs.append(newTab);
      return;
    }
    for(const [key,session] of sessions){
      const slot=document.createElement("span");
      slot.className="session-tab-slot";
      const tab=document.createElement("button");
      tab.type="button";
      tab.className="session-tab";
      tab.textContent=session.name+" · "+modes[session.mode].label;
      tab.title=session.name+" · "+session.id+" · "+modes[session.mode].label;
      tab.setAttribute("role","tab");
      tab.setAttribute("aria-selected",String(key===activeSessionKey));
      tab.addEventListener("click",()=>showSession(key));
      const close=document.createElement("button");
      close.type="button";
      close.className="session-tab-close";
      close.textContent="×";
      close.title="断开此远程连接";
      close.setAttribute("aria-label","断开 "+session.name);
      close.addEventListener("click",()=>closeSession(key));
      slot.append(tab,close);
      sessionTabs.append(slot);
    }
  }
  function watchClient(frame){
    frame.addEventListener("load",()=>{
      state.ready=true;
      el("engine-state").textContent="RustDesk 操作界面已加载";
      el("engine-state").classList.add("ready");
      try{
        frame.contentWindow.addEventListener("keydown",event=>{
          if(event.key==="Escape"&&state.pageFullscreen)exitFullscreen();
        },true);
      }catch{}
    });
  }
  watchClient(firstClient);
  const pageFullscreenButton=el("v2-page-fullscreen"),screenFullscreenButton=el("v2-screen-fullscreen");
  const exitButton=el("v2-fullscreen-exit");
  function syncFullscreen(){
    const monitorFullscreen=document.fullscreenElement===stage;
    if(monitorFullscreen)state.pageFullscreen=false;
    document.body.classList.toggle("webclient-page-fullscreen",state.pageFullscreen&&!monitorFullscreen);
    pageFullscreenButton.setAttribute("aria-pressed",String(state.pageFullscreen));
    screenFullscreenButton.setAttribute("aria-pressed",String(monitorFullscreen));
    exitButton.hidden=!(state.pageFullscreen||monitorFullscreen);
  }
  async function exitFullscreen(){
    if(document.fullscreenElement===stage){
      try{await document.exitFullscreen();}catch(error){message("退出全屏失败："+error.message);}
    }
    state.pageFullscreen=false;
    syncFullscreen();
  }
  pageFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement===stage){await exitFullscreen();return;}
    state.pageFullscreen=!state.pageFullscreen;
    syncFullscreen();
  });
  screenFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement===stage){await exitFullscreen();return;}
    try{
      if(!stage.requestFullscreen)throw Error("当前浏览器不支持此功能");
      await stage.requestFullscreen({navigationUI:"hide"});
      state.pageFullscreen=false;
      syncFullscreen();
    }catch(error){message("无法进入显示器全屏："+error.message);}
  });
  exitButton.addEventListener("click",exitFullscreen);
  document.addEventListener("fullscreenchange",syncFullscreen);
  document.addEventListener("keydown",event=>{
    if(event.key==="Escape"&&state.pageFullscreen){
      state.pageFullscreen=false;syncFullscreen();
    }
  });
  syncFullscreen();
  function sanitize(s){return String(s===undefined||s===null?"":s).replace(/[&<>"']/g,ch=>({"&":"&amp;","<":"&lt;",">":"&gt;",'"':"&quot;","'":"&#39;"}[ch]));}
  async function request(path, method="GET", body){
    const headers={"Accept":"application/json"};
    if(state.token)headers["api-token"]=state.token;
    if(body!==undefined)headers["Content-Type"]="application/json";
    const res=await fetch("/api/admin"+path,{method,headers,body:body===undefined?undefined:JSON.stringify(body),credentials:"same-origin"});
    let data=await res.json();
    if(!res.ok||typeof data.code==="number"&&data.code!==0)throw Error(data.message||data.error||"请求失败");
    return data.data===undefined?data:data.data;
  }
  function useDevice(id,name=""){
    config.id.value=id;
    state.activeName=String(name||"");
    state.autoConnect=false;
    updateDeviceIdentity();
    config.id.focus();
    message("已选择设备 "+id+"，请点击发起连接。");
    window.scrollTo({top:0,behavior:"smooth"});
  }
  async function loadConnectionDefaults(){
    if(!state.token)return;
    try{
      const server=await request("/config/server");
      if(!config.host.value.trim()&&server.id_server){
        config.host.value=String(server.id_server);
        store("custom-rendezvous-server",config.host.value);
      }
      if(!config.key.value.trim()&&server.key){
        config.key.value=String(server.key);
        store("key",config.key.value);
      }
    }catch(err){
      message("无法自动读取服务器信息，请手动填写连接设置。");
    }
  }
  async function loadDevices(){
    const info=el("shared-info"),list=el("shared-items"),login=el("shared-login");
    info.textContent="正在读取共享设备…";list.replaceChildren();
    if(!state.token){
      login.hidden=false; info.textContent="登录后可查看联系人和群组公开的设备。";return;
    }
    try {
      const result=await request("/shared_devices/list?page=1&page_size=100");
      login.hidden=true;
      info.textContent=result.total?"共 "+result.total+" 台共享设备":"暂无共享设备";
      const devices=result.list||[];
      let statuses={};
      if(devices.length){
        try{
          const details=await request("/geoip/lookup","POST",{
            device_ids:[...new Set(devices.map(item=>String(item.id||"")).filter(Boolean))].slice(0,256)
          });
          statuses=details.device_status||{};
        }catch{
          statuses={};
        }
      }
      for(const item of devices){
        const card=document.createElement("div");card.className="device";
        const left=document.createElement("span");left.className="device-description";
        const title=document.createElement("a");
        title.href="/_admin/#/overview?entity=device&value="+encodeURIComponent(item.id||"");
        title.title="查看设备信息";
        title.className="device-details-link";
        const dot=document.createElement("span");
        const online=statuses[item.id];
        dot.className="device-status-dot "+(online===true?"online":online===false?"offline":"unknown");
        dot.setAttribute("role","img");
        dot.setAttribute("aria-label",online===true?"在线":online===false?"离线":"状态暂不可用");
        dot.title=online===true?"在线":online===false?"离线":"状态暂不可用";
        title.append(dot,document.createTextNode(item.alias||item.hostname||item.id||"未命名设备"));
        const sub=document.createElement("small");
        const sources=(item.sources||[]).map(x=>(x.type==="contact"?"联系人":"群组")+"："+x.name).join(" · ");
        sub.append(document.createTextNode((item.id||"")+" · "));
        if(item.owner_username){
          const owner=document.createElement("a");
          owner.href="/_admin/#/overview?entity=user&value="+encodeURIComponent(item.owner_username);
          owner.textContent=item.owner_username;
          owner.title="查看所属用户信息";
          sub.append(owner);
        }
        if(sources)sub.append(document.createTextNode(" · "+sources));
        left.append(title,sub);
        const connect=document.createElement("button");connect.type="button";connect.className="device-connect";
        connect.textContent="选中连接";connect.setAttribute("aria-label","选择设备 "+String(item.id||""));
        connect.addEventListener("click",()=>useDevice(item.id,item.alias||item.hostname||""));
        card.append(left,connect);
        list.append(card);
      }
    }catch(e){state.token="";sessionStorage.removeItem("rustdesk.console.token");login.hidden=false;info.textContent="读取共享设备失败："+e.message;}
  }
  async function setupLoginOptions(){
    try {
      const options=await request("/login-options");
      const sso=el("sso-login-link");
      sso.hidden=!Array.isArray(options.ops)||options.ops.length===0;
      if(options.disable_pwd){
        el("shared-login-form").hidden=true;
        if(sso.hidden)el("shared-info").textContent="当前没有可用的登录方式，请联系管理员。";
      }
    }catch(err){message("无法获取登录方式："+err.message);}
  }
  async function showCaptcha(){
    try{
      const options=await request("/login-options");
      if(!options.need_captcha)return;
      const data=(await request("/captcha")).captcha;
      state.captchaId=data.id;
      el("v2-captcha").hidden=false;
      el("v2-captcha-image").src=data.b64.startsWith("data:")?data.b64:"data:image/png;base64,"+data.b64;
      el("v2-captcha").querySelector('input[name="captcha_id"]').value=data.id;
    }catch{}
  }
  function connectToDevice(mode=selectedMode){
    const host=config.host.value.trim(),key=config.key.value.trim(),id=config.id.value.trim();
    if(!modes[mode])return;
    if(!host||!id){message("请填写服务器和设备 ID");return;}
    store("custom-rendezvous-server",host);store("rendezvous-server",host);
    store("key",key);store("id",id);store("remote-id",id);
    sessionStorage.setItem("rustdesk.webclient.webrtc",el("v2-webrtc").checked?"Y":"N");

    updateDeviceIdentity();
    const sessionKey=host+"|"+id+"|"+mode;
    if(sessions.has(sessionKey)){
      showSession(sessionKey);
      message("已切换至："+id);
      return;
    }
    let frame;
    if(!firstClientInUse){
      frame=firstClient;
      firstClientInUse=true;
    }else{
      frame=document.createElement("iframe");
      frame.className="native-client session-inactive";
      frame.setAttribute("allow","fullscreen; clipboard-read; clipboard-write");
      frame.setAttribute("allowfullscreen","");
      watchClient(frame);
      sessionPanels.append(frame);
    }
    const session={id,host,name:state.activeName||id,mode,frame};
    sessions.set(sessionKey,session);
    frame.title=session.name+" · "+modes[mode].label+" · RustDesk";
    frame.name="rustdesk-peer:"+id;
    frame.src="/webclient/#/"+modes[mode].route+"/"+encodeURIComponent(id);
    showSession(sessionKey);
    message("正在通过 RustDesk 连接："+id);
  }
  function attemptAutoConnect(){
    if(!state.autoConnect||!state.defaultsLoaded||!config.host.value.trim())return;
    state.autoConnect=false;
    connectToDevice();
  }
  for(const [mode,button] of Object.entries(modeButtons)){
    button.addEventListener("click",()=>{
      selectedMode=mode;
      renderModeTabs();
      if(activeSessionKey||(config.host.value.trim()&&config.id.value.trim()))connectToDevice(mode);
    });
  }
  renderModeTabs();
  renderSessionTabs();
  el("connect-form").addEventListener("submit",e=>{e.preventDefault();state.autoConnect=false;connectToDevice();});
  el("v2-disconnect").addEventListener("click",()=>{
    if(activeSessionKey)closeSession(activeSessionKey);
    else message("当前没有活动的远程连接。");
  });
  el("v2-clear-password").addEventListener("click",()=>{
    const id=config.id.value.trim(),host=config.host.value.trim();
    if(!id||!host){message("请先填写服务器和设备 ID。");return;}
    const storageKey="rustdesk.webclient.credentials.v1:"+
      JSON.stringify([host.toLowerCase(),id]);
    const existed=localStorage.getItem(storageKey)!==null;
    localStorage.removeItem(storageKey);
    try{
      const peers=JSON.parse(localStorage.getItem("peers")||"{}");
      if(peers[id]&&typeof peers[id]==="object"){
        delete peers[id].password;
        peers[id].remember=false;
        localStorage.setItem("peers",JSON.stringify(peers));
      }
    }catch{}
    for(const session of sessions.values()){
      if(session.id!==id||session.host.toLowerCase()!==host.toLowerCase())continue;
      try{session.frame.contentWindow?.setByName?.("forget_password","");}catch{}
    }
    message(existed?"已清除该设备保存的密码。":"当前设备没有已保存的密码。");
  });
  el("v2-refresh").addEventListener("click",loadDevices);
  let lookupSequence=0;
  async function searchDevices(){
    const term=el("v2-device-query").value.trim(),target=el("v2-device-results");
    const sequence=++lookupSequence;
    if(term.length<2){target.textContent="请至少输入两个字符以搜索设备。";return;}
    if(!state.token){target.textContent="登录后可搜索名下设备及共享设备。";return;}
    target.textContent="正在搜索设备…";
    try{
      const [owned,shared]=await Promise.all([
        request("/my/peer/list?page=1&page_size=30&keyword="+encodeURIComponent(term)),
        request("/shared_devices/list?page=1&page_size=30&search="+encodeURIComponent(term))
      ]);
      if(sequence!==lookupSequence)return;
      const devices=new Map();
      for(const peer of [...(owned.list||[]),...(shared.list||[])]){
        if(peer.id&&!devices.has(String(peer.id)))devices.set(String(peer.id),peer);
      }
      const found=[...devices.values()].slice(0,30);
      let statuses={};
      if(found.length){
        try{
          const data=await request("/geoip/lookup","POST",{device_ids:found.map(device=>String(device.id))});
          statuses=data.device_status||{};
        }catch{}
      }
      if(sequence!==lookupSequence)return;
      target.replaceChildren();
      if(!found.length){target.textContent="没有找到匹配的设备，请尝试其他信息。";return;}
      for(const peer of found){
        const row=document.createElement("div");row.className="webclient-search-result";
        const detail=document.createElement("a");
        detail.href="/_admin/#/overview?entity=device&value="+encodeURIComponent(peer.id);
        detail.className="webclient-search-detail";
        detail.title="查看设备信息";
        const indicator=document.createElement("span");
        const status=statuses[peer.id];
        indicator.className="device-status-dot "+(status===true?"online":status===false?"offline":"unknown");
        indicator.setAttribute("role","img");
        indicator.setAttribute("aria-label",status===true?"在线":status===false?"离线":"状态暂不可用");
        detail.append(indicator,document.createTextNode(peer.alias||peer.hostname||peer.id));
        const meta=document.createElement("small");
        meta.textContent=[peer.id,peer.hostname,peer.os,peer.owner_username].filter(Boolean).join(" · ");
        const label=document.createElement("span");label.className="webclient-search-description";label.append(detail,meta);
        const choose=document.createElement("button");
        choose.type="button";choose.className="secondary";choose.textContent="选择此设备";
        choose.addEventListener("click",()=>useDevice(peer.id,peer.alias||peer.hostname||""));
        row.append(label,choose);target.append(row);
      }
    }catch(error){
      if(sequence===lookupSequence)target.textContent="设备搜索失败："+error.message;
    }
  }
  el("v2-search-device").addEventListener("click",searchDevices);
  el("v2-device-query").addEventListener("keydown",event=>{
    if(event.key==="Enter"){event.preventDefault();searchDevices();}
  });
  el("sso-login-link").addEventListener("click",()=>{
    sessionStorage.setItem("rustdesk.console.return","/webclient2/");
  });
  el("shared-login-form").addEventListener("submit",async e=>{
    e.preventDefault();
    const form=e.currentTarget;
    const submit=form.querySelector("button[type=submit]");submit.disabled=true;
    try{
      const body=Object.fromEntries(new FormData(form).entries());
      body.platform="web";
      const result=await request("/login","POST",body);
      if(!result.token)throw Error("登录未返回令牌");
      state.token=result.token;
      sessionStorage.setItem("rustdesk.console.token",result.token);

      await loadConnectionDefaults();
      await loadDevices();
    }catch(err){el("shared-info").textContent="登录失败："+err.message;await showCaptcha();}
    finally{submit.disabled=false;}
  });
  localStorage.removeItem("access_token");
  loadConnectionDefaults().then(()=>{
    state.defaultsLoaded=true;
    attemptAutoConnect();
    return loadDevices();
  });
  if(initialDeviceId&&state.token){
    request("/my/directory/device?id="+encodeURIComponent(initialDeviceId)).then(info=>{
      if(config.id.value.trim()===initialDeviceId&&info?.id){
        state.activeName=String(info.alias||info.hostname||state.activeName||"").slice(0,160);
        updateDeviceIdentity();
      }
    }).catch(()=>{});
  }
  if(!state.token){setupLoginOptions();showCaptcha();}
})();