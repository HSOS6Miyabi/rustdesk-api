"use strict";
(() => {
  const el = id => document.getElementById(id);
  const store = (key, value) => localStorage.setItem(key, value);
  const read = key => localStorage.getItem(key)||"";
  const state = { token:sessionStorage.getItem("rustdesk.console.token")||"",captchaId:"",ready:false,pageFullscreen:false };
  const config = {
    host: el("v2-host"), key: el("v2-key"), id: el("v2-id")
  };
  config.host.value=read("custom-rendezvous-server")||read("rendezvous-server");
  config.key.value=read("key");
  config.id.value=read("remote-id")||read("id");
  el("v2-webrtc").checked=sessionStorage.getItem("rustdesk.webclient.webrtc")==="Y";
  function message(text){el("v2-message").textContent=text;}
  const stage=document.querySelector(".stage");
  const pageFullscreenButton=el("v2-page-fullscreen"),screenFullscreenButton=el("v2-screen-fullscreen");
  function syncFullscreen(){
    const monitorFullscreen=document.fullscreenElement===stage;
    document.body.classList.toggle("webclient-page-fullscreen",state.pageFullscreen&&!monitorFullscreen);
    pageFullscreenButton.setAttribute("aria-pressed",String(state.pageFullscreen&&!monitorFullscreen));
    screenFullscreenButton.setAttribute("aria-pressed",String(monitorFullscreen));
    pageFullscreenButton.textContent=state.pageFullscreen&&!monitorFullscreen?"退出网页全屏":"网页全屏";
    screenFullscreenButton.textContent=monitorFullscreen?"退出显示器全屏":"显示器全屏";
    if(monitorFullscreen)state.pageFullscreen=false;
  }
  pageFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement){
      try{await document.exitFullscreen();}catch(error){message("退出显示器全屏失败："+error.message);return;}
    }
    state.pageFullscreen=!state.pageFullscreen;
    syncFullscreen();
  });
  screenFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement===stage){
      try{await document.exitFullscreen();}catch(error){message("退出显示器全屏失败："+error.message);}
      return;
    }
    try{
      if(!stage.requestFullscreen)throw Error("当前浏览器不支持此功能");
      await stage.requestFullscreen({navigationUI:"hide"});
      state.pageFullscreen=false;
      syncFullscreen();
    }catch(error){
      message("无法进入显示器全屏："+error.message+"。您仍可使用网页全屏。");
    }
  });
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
  function useDevice(id){
    config.id.value=id;
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
        connect.addEventListener("click",()=>useDevice(item.id));
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
  el("connect-form").addEventListener("submit",e=>{
    e.preventDefault();
    const host=config.host.value.trim(),key=config.key.value.trim(),id=config.id.value.trim();
    if(!host||!id){message("请填写服务器和设备 ID");return;}
    store("custom-rendezvous-server",host);store("rendezvous-server",host);
    store("key",key);store("id",id);store("remote-id",id);
    sessionStorage.setItem("rustdesk.webclient.webrtc",el("v2-webrtc").checked?"Y":"N");

    const nativeHost=document.getElementById("host"),nativeKey=document.getElementById("key"),nativeID=document.querySelector("#app #id");
    if(nativeHost)nativeHost.value=host;
    if(nativeKey)nativeKey.value=key;
    if(nativeID)nativeID.value=id;
    if(typeof window.connect!=="function"){message("远程连接组件尚未加载完毕，请刷新页面。");return;}
    window.connect();
    message("正在连接："+id);
  });
  el("v2-disconnect").addEventListener("click",()=>{
    if(typeof window.cancel==="function")window.cancel();
    message("连接已断开");
  });
  el("v2-refresh").addEventListener("click",loadDevices);
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
  function watchEngine(){
    const check=()=>{
      const native=document.querySelector("#app #connect");
      if(native&&typeof window.connect==="function"){
        state.ready=true;
        el("engine-state").textContent="远程连接组件已就绪";
        el("engine-state").classList.add("ready");
        const placeholder=document.querySelector("#app .placeholder");if(placeholder)placeholder.remove();
        return true;
      }
      return false;
    };
    if(check())return;
    const observer=new MutationObserver(()=>{if(check())observer.disconnect();});
    observer.observe(el("app"),{childList:true,subtree:true});
  }
  watchEngine();
  localStorage.removeItem("access_token");
  loadConnectionDefaults().then(loadDevices);
  if(!state.token){setupLoginOptions();showCaptcha();}
})();