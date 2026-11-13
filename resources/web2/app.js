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
    el("v2-performance-status").textContent="连接后显示帧率与浏览器视频处理时间。";
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
    syncClipboardSession();
    syncFullscreen();
  }
  function closeSession(key){
    const session=sessions.get(key);
    if(!session)return;
    if(clipboardTransfer?.busy&&clipboardTransfer.source===session.frame.contentWindow)
      cancelClipboardDownload("远程连接已关闭，下载已取消。");
    sessions.delete(key);
    try{session.frame.contentWindow?.__rustdeskClipboardAccess?.disable();}catch{}
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
      else {emptySession.hidden=false;renderSessionTabs();syncClipboardSession();}
    }else renderSessionTabs();
    message(sessions.size?"已断开："+session.name:"当前连接已关闭，可以创建新的连接标签。");
    if(!sessions.size)el("v2-performance-status").textContent="连接远程设备后显示画面性能。";
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
          if(event.key==="Escape"&&(state.pageFullscreen||nativeFullscreen))exitFullscreen();
        },true);
      }catch{}
      try{
        frame.contentWindow?.addEventListener("hashchange",()=>{
          const hash=String(frame.contentWindow.location?.hash||"");
          if(/^#\/?(?:home)?\/?$/.test(hash)&&frame.contentWindow.__rustdeskNativeSessionSeen){
            window.__rustdeskOnSessionClosed(frame.contentWindow,frame.contentWindow.__rustdeskNativeSessionSeen);
          }
        });
      }catch{}
      syncFullscreen();
      syncClipboardSession();
    });
  }
  watchClient(firstClient);
  const stageDropMask=document.createElement("div");
  stageDropMask.className="webclient-stage-drop";
  stageDropMask.textContent="将文件放到这里，上传至远程设备";
  stageDropMask.hidden=true;
  stage.append(stageDropMask);
  const isFileDrag=event=>[...(event.dataTransfer?.types||[])].includes("Files");
  stage.addEventListener("dragover",event=>{
    if(!isFileDrag(event))return;
    event.preventDefault();
    if(event.dataTransfer)event.dataTransfer.dropEffect="copy";
    const session=sessions.get(activeSessionKey);
    stageDropMask.textContent=session?.mode==="desktop"&&clipboardEnabled
      ?"将文件放到这里，通过剪贴板粘贴到远程设备"
      :"将文件放到这里，上传至远程设备";
    stageDropMask.hidden=false;
  });
  stage.addEventListener("dragleave",event=>{
    if(!stage.contains?.(event.relatedTarget))stageDropMask.hidden=true;
  });
  stage.addEventListener("drop",event=>{
    if(!isFileDrag(event))return;
    event.preventDefault();
    stageDropMask.hidden=true;
    const session=sessions.get(activeSessionKey);
    if(session?.mode==="desktop"&&clipboardEnabled){
      try{
        const accepted=session.frame.contentWindow?.__rustdeskClipboardAccess?.offerFiles?.(
          [...event.dataTransfer.files],true);
        if(accepted){
          message("已将文件交给远程剪贴板，等待远程设备确认。");
          return;
        }
      }catch(error){message("通过剪贴板发送文件失败："+error.message);return;}
    }
    if(!session||session.mode!=="file"){
      message("远程文件剪贴板尚未准备好，可切换到文件传输标签上传。");
      return;
    }
    try{
      const receive=session.frame.contentWindow?.__rustdeskReceiveFiles;
      if(typeof receive!=="function"){
        message("远程文件管理器正在初始化，请连接成功后重试。");
        return;
      }
      receive([...event.dataTransfer.files],false);
    }catch(error){message("上传文件失败："+error.message);}
  });
  const clipboardButton=el("v2-clipboard-toggle");
  const clipboardFileButton=el("v2-clipboard-select-files");
  const clipboardFileInput=el("v2-clipboard-input");
  const clipboardPasteButton=el("v2-clipboard-paste-text");
  const clipboardTextInput=el("v2-clipboard-text-input");
  const clipboardTextSendButton=el("v2-clipboard-text-send");
  const clipboardCopyButton=el("v2-clipboard-copy");
  const clipboardDownloadButton=el("v2-download-clipboard");
  const clipboardStatus=el("v2-clipboard-status");
  const performanceStatus=el("v2-performance-status");
  const progressPanel=el("v2-clipboard-progress-panel");
  stage.append(progressPanel);
  const progressTitle=el("v2-clipboard-progress-title");
  const progressFile=el("v2-clipboard-progress-file");
  const progressBar=el("v2-clipboard-progress-bar");
  const progressStats=el("v2-clipboard-progress-stats");
  const progressCancel=el("v2-clipboard-progress-cancel");
  const progressMinimize=el("v2-clipboard-progress-minimize");
  const progressClose=el("v2-clipboard-progress-close");
  let clipboardTransfer=null;
  let clipboardEnabled=true,clipboardReadGranted=false,pendingRemoteText=null;
  let progressDismissed=false,progressTimer=null;
  function clearProgressTimer(){
    if(progressTimer!==null)window.clearTimeout?.(progressTimer);
    progressTimer=null;
  }
  function dismissProgress(){
    clearProgressTimer();
    progressDismissed=true;
    progressPanel.hidden=true;
  }
  function scheduleProgressDismiss(){
    clearProgressTimer();
    progressTimer=window.setTimeout?.(()=>{
      if(!clipboardTransfer?.busy)progressPanel.hidden=true;
      progressTimer=null;
    },5500);
  }
  progressClose.addEventListener("click",dismissProgress);
  const formatBytes=value=>{
    const bytes=Math.max(0,Number(value)||0);
    if(bytes<1024)return bytes+" B";
    const unit=bytes<1024*1024?"KB":"MB",divisor=unit==="KB"?1024:1048576;
    return (bytes/divisor).toFixed(1)+" "+unit;
  };
  function updateDownloadProgress({title,file,received,total,stats,complete=false}){
    if(!progressDismissed)progressPanel.hidden=false;
    if(title)progressTitle.textContent=title;
    if(file)progressFile.textContent=file;
    if(typeof received==="number"&&typeof total==="number"){
      const percent=total<=0?(complete?100:0):Math.min(100,Math.floor(received/total*100));
      progressBar.value=percent;
      progressBar.setAttribute("aria-valuenow",String(percent));
    }
    if(stats)progressStats.textContent=stats;
    if(complete)progressBar.value=100;
  }
  function cancelClipboardDownload(text){
    const task=clipboardTransfer;
    if(!task?.busy)return;
    task.busy=false;
    try{task.source?.__rustdeskClipboardAccess?.cancelRemoteFiles?.();}catch{}
    progressCancel.hidden=true;
    updateDownloadProgress({title:"剪贴板下载已取消",stats:text||"已取消下载，远程桌面可以继续使用。"});
    clipboardDownloadButton.disabled=false;
    scheduleProgressDismiss();
  }
  progressCancel.addEventListener("click",()=>cancelClipboardDownload());
  progressMinimize.addEventListener("click",()=>{
    const collapsed=progressPanel.dataset.collapsed!=="true";
    progressPanel.dataset.collapsed=String(collapsed);
    progressMinimize.textContent=collapsed?"展开":"收起";
    progressMinimize.setAttribute("aria-expanded",String(!collapsed));
  });
  const clipboardAPI=()=>window.navigator?.clipboard;
  function clipboardSession(){
    const session=sessions.get(activeSessionKey);
    return session&&["desktop","camera"].includes(session.mode)?session:null;
  }
  function setClipboardStatus(text){clipboardStatus.textContent=text;}
  window.__rustdeskRemotePerformance=(stats,source)=>{
    const active=sessions.get(activeSessionKey);
    if(!active||active.frame.contentWindow!==source)return;
    const fps=Number(stats?.fps||0),decode=Number(stats?.decode_ms||0),
      render=Number(stats?.render_ms||0),buffer=Number(stats?.buffered_bytes||0);
    const transport=stats?.transport==="WebRTC"?"WebRTC":"WebSocket";
    performanceStatus.textContent="画面性能："+transport+" · "+
      fps.toFixed(1)+" 帧/秒 · 解码 "+decode.toFixed(1)+
      " 毫秒 · 绘制 "+render.toFixed(1)+" 毫秒"+
      (buffer>0?" · 待发送 "+formatBytes(buffer):"");
    performanceStatus.dataset.slow=String(decode+render>30);
    performanceStatus.title=decode+render>30
      ?"浏览器处理视频帧耗时较长，可降低远程分辨率或画质。"
      :"如远程操作仍有延迟，可以比较 WebRTC 和 WebSocket 的连接表现。";
  };
  function currentClipboardBridge(){
    return clipboardSession()?.frame.contentWindow?.__rustdeskClipboardAccess;
  }
  function showManualClipboardInput(){
    clipboardTextInput.hidden=false;
    clipboardTextSendButton.hidden=false;
    clipboardTextInput.focus();
  }
  function sendTextToRemote(text){
    const value=String(text||"");
    if(!value){setClipboardStatus("本机剪贴板没有文本，请先复制文本。");return false;}
    if(new TextEncoder().encode(value).byteLength>1024*1024){
      setClipboardStatus("粘贴的文本过长，最多允许 1 MiB。");
      return false;
    }
    const bridge=currentClipboardBridge();
    if(!clipboardEnabled||typeof bridge?.pasteText!=="function"){
      setClipboardStatus("请先连接远程桌面并开启剪贴板共享。");
      return false;
    }
    if(bridge.pasteText(value)!==true){
      setClipboardStatus("远程桌面尚未允许发送剪贴板文本。");
      return false;
    }
    setClipboardStatus("已将本机文本发送至远程设备，并执行粘贴。");
    clipboardTextInput.value="";
    clipboardTextInput.hidden=true;
    clipboardTextSendButton.hidden=true;
    return true;
  }
  window.__rustdeskReadHostClipboard=async source=>{
    const current=clipboardSession();
    if(!clipboardEnabled||!current||current.frame.contentWindow!==source)
      return {available:false};
    const api=clipboardAPI();
    if(typeof api?.readText!=="function")return {available:false};
    try{
      const text=await api.readText();
      clipboardReadGranted=true;
      return {available:true,text:String(text||"")};
    }catch{
      clipboardReadGranted=false;
      setClipboardStatus("浏览器没有授予剪贴板读取权限。可点击“粘贴本机文本”重试，或使用下方输入框。");
      return {available:false};
    }
  };
  clipboardPasteButton.addEventListener("click",async()=>{
    if(!clipboardEnabled||!clipboardSession()){
      setClipboardStatus("请先连接远程桌面并开启剪贴板共享。");
      return;
    }
    const result=await window.__rustdeskReadHostClipboard(clipboardSession().frame.contentWindow);
    if(result.available&&result.text){
      sendTextToRemote(result.text);
    }else{
      showManualClipboardInput();
      setClipboardStatus(result.available
        ?"本机剪贴板没有文本。可在输入框中粘贴或输入需要发送的内容。"
        :"请在输入框中粘贴本机文本，再点击“发送并粘贴”。");
    }
  });
  clipboardTextSendButton.addEventListener("click",()=>{
    sendTextToRemote(clipboardTextInput.value);
  });
  function syncClipboardSession(initialText){
    const active=clipboardSession();
    for(const [key,session] of sessions){
      try{
        const api=session.frame.contentWindow?.__rustdeskClipboardAccess;
        if(key===activeSessionKey&&active&&clipboardEnabled){
          api?.enable(typeof initialText==="string"?initialText:undefined);
        }else if(clipboardTransfer?.busy&&session.frame.contentWindow===clipboardTransfer.source){
          api?.suspendText?.();
        }else api?.disable();
      }catch{}
    }
    clipboardButton.setAttribute("aria-pressed",String(clipboardEnabled));
    clipboardButton.textContent=clipboardEnabled?"关闭剪贴板共享":"启用剪贴板共享";
    clipboardFileButton.disabled=!active||active.mode!=="desktop"||!clipboardEnabled;
    clipboardPasteButton.disabled=!active||active.mode!=="desktop"||!clipboardEnabled;
    if(!clipboardEnabled)clipboardCopyButton.hidden=true;
  }
  window.__rustdeskClipboardSendFailure=source=>{
    const current=clipboardSession();
    if(current?.frame.contentWindow===source)
      setClipboardStatus("无法将本机剪贴板内容发送至远程设备，请检查连接状态和剪贴板权限。");
  };
  window.__rustdeskFileClipboardPermission=(allowed,source)=>{
    const current=clipboardSession();
    if(!current||current.frame.contentWindow!==source)return;
    if(!allowed)setClipboardStatus(
      "远程设备已禁止文件传输，请在远程 RustDesk 的安全设置中允许文件传输。");
    else if(clipboardEnabled)setClipboardStatus(
      "远程设备允许文件传输，可以在远程资源管理器复制文件后下载。");
  };
  window.__rustdeskClipboardReady=source=>{
    const session=[...sessions.values()].find(item=>item.frame.contentWindow===source);
    if(session)syncClipboardSession();
  };
  clipboardFileButton.addEventListener("click",()=>{
    if(!clipboardEnabled||!clipboardSession())return;
    clipboardFileInput.click();
  });
  clipboardFileInput.addEventListener("change",()=>{
    const current=clipboardSession();
    const files=Array.from(clipboardFileInput.files||[]);
    if(files.length&&current){
      try{
        const supported=current.frame.contentWindow?.__rustdeskClipboardAccess?.offerFiles?.(files,true);
        setClipboardStatus(supported
          ?"已选择 "+files.length+" 个文件，正在发送至远程剪贴板。"
          :"远程文件剪贴板尚未准备好，请连接成功后重试。");
      }catch(error){setClipboardStatus("无法传输文件："+error.message);}
    }
    clipboardFileInput.value="";
  });
  async function enableClipboard(){
    clipboardEnabled=true;
    const api=clipboardAPI();
    let initialText;
    if(typeof api?.readText==="function"){
      try{
        initialText=await api.readText();
        clipboardReadGranted=true;
      }catch{
        clipboardReadGranted=false;
      }
    }
    syncClipboardSession(initialText);
    setClipboardStatus(clipboardReadGranted
      ?"剪贴板共享已开启，支持文本及资源管理器中的文件。"
      :"剪贴板共享已开启。浏览器可能要求授权，请使用 Ctrl+C / Ctrl+V 复制文本或文件。");
  }
  clipboardButton.addEventListener("click",async()=>{
    if(clipboardEnabled){
      cancelClipboardDownload("剪贴板共享已关闭，下载已取消。");
      clipboardEnabled=false;
      clipboardReadGranted=false;
      pendingRemoteText=null;
      syncClipboardSession();
      setClipboardStatus("剪贴板共享已关闭。");
    }else await enableClipboard();
  });
  clipboardCopyButton.addEventListener("click",async()=>{
    if(!pendingRemoteText)return;
    try{
      if(!clipboardAPI()?.writeText)throw Error("浏览器未提供剪贴板写入接口");
      await clipboardAPI().writeText(pendingRemoteText);
      clipboardCopyButton.hidden=true;
      setClipboardStatus("远程文本已复制到本机剪贴板。");
    }catch(error){setClipboardStatus("无法复制远程文本："+error.message);}
  });
  window.__rustdeskReceiveRemoteClipboard=async(text,source)=>{
    const current=clipboardSession();
    if(!clipboardEnabled||!current||current.frame.contentWindow!==source)return;
    pendingRemoteText=String(text||"");
    if(!pendingRemoteText)return;
    try{
      if(!clipboardAPI()?.writeText)throw Error("当前浏览器不支持写入剪贴板");
      await clipboardAPI().writeText(pendingRemoteText);
      clipboardCopyButton.hidden=true;
      setClipboardStatus("远程文本已同步至本机剪贴板。");
    }catch{
      clipboardCopyButton.hidden=false;
      setClipboardStatus("收到远程文本。点击“复制远程文本”后，浏览器将请求剪贴板权限。");
    }
  };
  window.__rustdeskClipboardFileStatus=(kind,detail,source)=>{
    const current=clipboardSession();
    const currentSession=!!current&&current.frame.contentWindow===source;
    const task=clipboardTransfer?.source===source?clipboardTransfer:null;
    if(!currentSession&&!task)return;
    if(kind==="remote-list"){
      if(detail&&currentSession)setClipboardStatus("检测到远程文件剪贴板，可点击右上角下载。");
    }else if(kind==="local-files"){
      if(currentSession)setClipboardStatus("已将 "+detail+" 个本地文件交给远程剪贴板。");
    }else if(kind==="start"&&task){
      task.total=Number(detail?.total||0);
      updateDownloadProgress({title:"正在下载剪贴板文件",file:"正在获取 "+Number(detail?.count||0)+" 个文件",
        received:0,total:task.total,stats:"正在连接远程文件服务。可以继续操作远程桌面。"});
    }else if(kind==="progress"&&task?.busy){
      const received=Number(detail?.overallReceived??detail?.received??0);
      const total=Number(detail?.overallTotal??detail?.total??task.total??0);
      const file=String(detail?.name||"远程文件");
      updateDownloadProgress({title:"正在下载剪贴板文件",file,received,total,
        stats:formatBytes(received)+" / "+formatBytes(total)+" · 文件 "+
          Number(detail?.index||1)+"/"+Number(detail?.count||1)+" · 可继续操作远程桌面"});
      if(currentSession)setClipboardStatus("正在下载 "+file+"："+(total?Math.floor(received/total*100):100)+"%");
    }else if(kind==="saving"&&task?.busy){
      updateDownloadProgress({title:"正在保存剪贴板文件",stats:"文件接收完成，正在准备浏览器下载。"});
    }else if(kind==="downloaded"&&task){
      updateDownloadProgress({title:"剪贴板文件下载完成",file:String(detail?.name||"远程文件"),
        stats:"已保存 "+Number(detail?.count||0)+" 个文件，可以继续远程操作。",complete:true});
      progressCancel.hidden=true;
      if(currentSession)setClipboardStatus("已下载 "+detail.count+" 个剪贴板文件。");
    }else if(kind==="cancelled"&&task){
      updateDownloadProgress({title:"剪贴板文件下载已取消",stats:"已取消下载，其他远程操作不受影响。"});
      progressCancel.hidden=true;
    }else if(kind==="unsupported-files"&&currentSession){
      setClipboardStatus("远程设备未确认文件剪贴板操作，请切换到“文件”标签上传。");
    }
  };
  clipboardDownloadButton.addEventListener("click",async()=>{
    const current=clipboardSession();
    if(!current){message("请先连接远程桌面或摄像会话。");return;}
    if(clipboardTransfer?.busy){
      progressDismissed=false;
      progressPanel.hidden=false;
      progressPanel.dataset.collapsed="false";
      progressMinimize.textContent="收起";
      progressMinimize.setAttribute("aria-expanded","true");
      return;
    }
    if(!clipboardEnabled)await enableClipboard();
    const source=current.frame.contentWindow;
    const remote=source?.__rustdeskClipboardAccess;
    if(typeof remote?.downloadRemoteFiles!=="function"){
      message("远程桌面剪贴板组件尚未初始化。");return;
    }
    const task={source,busy:true,total:0};
    clipboardTransfer=task;
    clearProgressTimer();
    progressDismissed=false;
    clipboardDownloadButton.disabled=true;
    progressCancel.hidden=false;
    progressPanel.dataset.collapsed="false";
    progressMinimize.textContent="收起";
    progressMinimize.setAttribute("aria-expanded","true");
    updateDownloadProgress({title:"正在获取剪贴板文件列表",file:"正在读取远程剪贴板",
      received:0,total:100,stats:"下载任务已开始，远程桌面仍可正常操作。"});
    try{
      const result=await remote.downloadRemoteFiles();
      if(clipboardTransfer===task&&task.busy){
        updateDownloadProgress({title:"剪贴板文件下载完成",file:result.name,
          stats:"已保存 "+result.count+" 个文件。",complete:true});
        progressCancel.hidden=true;
        message("已下载远程剪贴板文件："+result.name);
        scheduleProgressDismiss();
      }
    }catch(error){
      if(clipboardTransfer===task&&task.busy){
        const detail=String(error?.message||error);
        if(detail.includes("远程剪贴板中暂无文件")){
          dismissProgress();
          setClipboardStatus(detail);
        }else{
          updateDownloadProgress({title:"剪贴板文件下载失败",stats:detail});
          scheduleProgressDismiss();
        }
        progressCancel.hidden=true;
        message("下载剪贴板文件失败："+detail);
      }
    }finally{
      if(clipboardTransfer===task){
        task.busy=false;
        clipboardDownloadButton.disabled=false;
        if(!progressPanel.hidden)scheduleProgressDismiss();
        syncClipboardSession();
      }
    }
  });
  document.addEventListener("paste",event=>{
    if(!clipboardEnabled)return;
    const current=clipboardSession();
    if(!current)return;
    const api=current.frame.contentWindow?.__rustdeskClipboardAccess;
    const files=Array.from(event.clipboardData?.files||[]);
    if(files.length){api?.offerFiles?.(files);return;}
    const text=event.clipboardData?.getData("text/plain");
    if(text)api?.sendLocal?.(text);
  },true);
  window.__rustdeskOnSessionClosed=(source,id)=>{
    const found=[...sessions].find(([,session])=>session.frame.contentWindow===source&&session.id===id);
    if(!found)return;
    window.setTimeout?.(()=>{
      if(sessions.get(found[0])===found[1])closeSession(found[0]);
    },0);
  };
  const pageFullscreenButton=el("v2-page-fullscreen"),screenFullscreenButton=el("v2-screen-fullscreen");
  const exitButton=el("v2-fullscreen-exit");
  let nativeFullscreen=false;
  async function lockNativeKeyboard(){
    try{await window.navigator?.keyboard?.lock?.();}
    catch{ /* Browser keyboard lock is optional. */ }
  }
  function unlockNativeKeyboard(){
    try{window.navigator?.keyboard?.unlock?.();}catch{}
  }
  function syncFullscreen(){
    const monitorFullscreen=document.fullscreenElement===stage;
    if(monitorFullscreen)state.pageFullscreen=false;
    else if(nativeFullscreen){
      nativeFullscreen=false;
      unlockNativeKeyboard();
    }
    stage.classList.toggle("webclient-native-fullscreen",nativeFullscreen&&monitorFullscreen);
    document.body.classList.toggle("webclient-page-fullscreen",state.pageFullscreen&&!monitorFullscreen);
    pageFullscreenButton.setAttribute("aria-pressed",String(state.pageFullscreen));
    screenFullscreenButton.setAttribute("aria-pressed",String(monitorFullscreen));
    exitButton.hidden=nativeFullscreen||!(state.pageFullscreen||monitorFullscreen);
    try{
      const frame=clipboardSession()?.frame||sessions.get(activeSessionKey)?.frame;
      frame?.contentWindow?.onFullscreenChanged?.(state.pageFullscreen||monitorFullscreen);
    }catch{}
  }
  window.__rustdeskIsFullscreen=()=>state.pageFullscreen||document.fullscreenElement===stage;
  window.__rustdeskSetFullscreen=(wanted,source)=>{
    const active=sessions.get(activeSessionKey);
    if(!active||active.frame.contentWindow!==source)
      return Promise.reject(Error("该会话当前未显示"));
    if(wanted){
      if(typeof stage.requestFullscreen!=="function")return Promise.reject(Error("浏览器不支持全屏"));
      nativeFullscreen=true;
      return Promise.resolve(stage.requestFullscreen({navigationUI:"hide"}))
        .then(async()=>{syncFullscreen();await lockNativeKeyboard();}).catch(error=>{
          nativeFullscreen=false;
          syncFullscreen();
          throw error;
        });
    }
    return exitFullscreen();
  };
  async function exitFullscreen(){
    if(document.fullscreenElement===stage){
      try{await document.exitFullscreen();}catch(error){message("退出全屏失败："+error.message);}
    }
    state.pageFullscreen=false;
    if(nativeFullscreen)unlockNativeKeyboard();
    nativeFullscreen=false;
    syncFullscreen();
  }
  pageFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement===stage){await exitFullscreen();return;}
    nativeFullscreen=false;
    state.pageFullscreen=!state.pageFullscreen;
    syncFullscreen();
  });
  screenFullscreenButton.addEventListener("click",async()=>{
    if(document.fullscreenElement===stage){await exitFullscreen();return;}
    try{
      if(!stage.requestFullscreen)throw Error("当前浏览器不支持此功能");
      nativeFullscreen=false;
      await stage.requestFullscreen({navigationUI:"hide"});
      state.pageFullscreen=false;
      syncFullscreen();
    }catch(error){message("无法进入显示器全屏："+error.message);}
  });
  exitButton.addEventListener("click",exitFullscreen);
  document.addEventListener("fullscreenchange",syncFullscreen);
  document.addEventListener("keydown",event=>{
    if(event.key==="Escape"&&(state.pageFullscreen||nativeFullscreen)){
      exitFullscreen();
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
    connectToDevice(selectedMode);
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