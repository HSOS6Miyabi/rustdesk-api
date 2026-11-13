import Connection from "./connection";
import _sodium from "libsodium-wrappers";
import {loadVp9} from "./codec";
import {checkIfRetry, version} from "./gen_js_from_hbb";
import {initZstd, decompress, translate} from "./common";
import {BrowserFileTransfer} from "./browser_file_transfer.mjs";
import {BrowserClipboardSync} from "./browser_clipboard.mjs";
import {BrowserFileClipboard} from "./browser_clipboard_files.mjs";
import {installRemoteShortcutCapture,isBrowserEditable} from "./remote_shortcuts.mjs";
import PCMPlayer from "pcm-player";
import {getServerConf} from "./ljw";

window.myconsole = (...args) => {
    console.log(args);
}
window.curConn = undefined;
window.isMobile = () => {
    return /(android|bb\d+|meego).+mobile|avantgo|bada\/|blackberry|blazer|compal|elaine|fennec|hiptop|iemobile|ip(hone|od)|ipad|iris|kindle|Android|Silk|lge |maemo|midp|mmp|netfront|opera m(ob|in)i|palm( os)?|phone|p(ixi|re)\/|plucker|pocket|psp|series(4|6)0|symbian|treo|up\.(browser|link)|vodafone|wap|windows (ce|phone)|xda|xiino/i.test(navigator.userAgent)
        || /1207|6310|6590|3gso|4thp|50[1-6]i|770s|802s|a wa|abac|ac(er|oo|s\-)|ai(ko|rn)|al(av|ca|co)|amoi|an(ex|ny|yw)|aptu|ar(ch|go)|as(te|us)|attw|au(di|\-m|r |s )|avan|be(ck|ll|nq)|bi(lb|rd)|bl(ac|az)|br(e|v)w|bumb|bw\-(n|u)|c55\/|capi|ccwa|cdm\-|cell|chtm|cldc|cmd\-|co(mp|nd)|craw|da(it|ll|ng)|dbte|dc\-s|devi|dica|dmob|do(c|p)o|ds(12|\-d)|el(49|ai)|em(l2|ul)|er(ic|k0)|esl8|ez([4-7]0|os|wa|ze)|fetc|fly(\-|_)|g1 u|g560|gene|gf\-5|g\-mo|go(\.w|od)|gr(ad|un)|haie|hcit|hd\-(m|p|t)|hei\-|hi(pt|ta)|hp( i|ip)|hs\-c|ht(c(\-| |_|a|g|p|s|t)|tp)|hu(aw|tc)|i\-(20|go|ma)|i230|iac( |\-|\/)|ibro|idea|ig01|ikom|im1k|inno|ipaq|iris|ja(t|v)a|jbro|jemu|jigs|kddi|keji|kgt( |\/)|klon|kpt |kwc\-|kyo(c|k)|le(no|xi)|lg( g|\/(k|l|u)|50|54|\-[a-w])|libw|lynx|m1\-w|m3ga|m50\/|ma(te|ui|xo)|mc(01|21|ca)|m\-cr|me(rc|ri)|mi(o8|oa|ts)|mmef|mo(01|02|bi|de|do|t(\-| |o|v)|zz)|mt(50|p1|v )|mwbp|mywa|n10[0-2]|n20[2-3]|n30(0|2)|n50(0|2|5)|n7(0(0|1)|10)|ne((c|m)\-|on|tf|wf|wg|wt)|nok(6|i)|nzph|o2im|op(ti|wv)|oran|owg1|p800|pan(a|d|t)|pdxg|pg(13|\-([1-8]|c))|phil|pire|pl(ay|uc)|pn\-2|po(ck|rt|se)|prox|psio|pt\-g|qa\-a|qc(07|12|21|32|60|\-[2-7]|i\-)|qtek|r380|r600|raks|rim9|ro(ve|zo)|s55\/|sa(ge|ma|mm|ms|ny|va)|sc(01|h\-|oo|p\-)|sdk\/|se(c(\-|0|1)|47|mc|nd|ri)|sgh\-|shar|sie(\-|m)|sk\-0|sl(45|id)|sm(al|ar|b3|it|t5)|so(ft|ny)|sp(01|h\-|v\-|v )|sy(01|mb)|t2(18|50)|t6(00|10|18)|ta(gt|lk)|tcl\-|tdg\-|tel(i|m)|tim\-|t\-mo|to(pl|sh)|ts(70|m\-|m3|m5)|tx\-9|up(\.b|g1|si)|utst|v400|v750|veri|vi(rg|te)|vk(40|5[0-3]|\-v)|vm40|voda|vulc|vx(52|53|60|61|70|80|81|83|85|98)|w3c(\-| )|webc|whit|wi(g |nc|nw)|wmlb|wonu|x700|yas\-|your|zeto|zte\-/i.test(navigator.userAgent.substr(0, 4));
}

export function isDesktop() {
    return !isMobile();
}

export function msgbox(type, title, text) {
    if (!type || (type == 'error' && !text)) return;
    var hasRetry = checkIfRetry(type, title, text) ? 'true' : '';
    if (typeof window.onGlobalEvent === "function") window.onGlobalEvent(JSON.stringify({name: 'msgbox', type, title, text, hasRetry}));
}

function jsonfyForDart(payload) {
    var tmp = {};
    for (const [key, value] of Object.entries(payload)) {
        if (!key) continue;
        if (value instanceof Uint8Array) {
            tmp[key] = '[' + value.toString() + ']';
        } else if (key === 'persistent_sessions' && Array.isArray(value)) {
            tmp[key] = value;
        } else if (value !== null && typeof value === 'object') {
            tmp[key] = JSON.stringify(value);
        } else {
            tmp[key] = value == null ? '' : String(value);
        }
    }
    return tmp;
}

export function pushEvent(name, payload) {
    payload = jsonfyForDart(payload);
    payload.name = name;
    if (typeof window.onGlobalEvent === "function") window.onGlobalEvent(JSON.stringify(payload));
}

let yuvWorker;
let yuvCanvas;
let gl;
let pixels;
let flipPixels;
let oldSize;
if (YUVCanvas.WebGLFrameSink.isAvailable()) {
    var canvas = document.createElement('canvas');
    yuvCanvas = YUVCanvas.attach(canvas, {webGL: true});
    gl = canvas.getContext("webgl");
} else {
    yuvWorker = new Worker("./yuv.js");
}
let testSpeed = [0, 0];

export function draw(frame) {
    if (typeof window.onRgba !== "function") return;
    if (yuvWorker) {
        // frame's (y/u/v).bytes already detached, can not transferrable any more.
        yuvWorker.postMessage(frame);
    } else {
        var tm0 = new Date().getTime();
        yuvCanvas.drawFrame(frame);
        var width = canvas.width;
        var height = canvas.height;
        var size = width * height * 4;
        if (size != oldSize) {
            pixels = new Uint8Array(size);
            flipPixels = new Uint8Array(size);
            oldSize = size;
        }
        gl.readPixels(0, 0, width, height, gl.RGBA, gl.UNSIGNED_BYTE, pixels);
        const row = width * 4;
        const end = (height - 1) * row;
        for (let i = 0; i < size; i += row) {
            flipPixels.set(pixels.subarray(i, i + row), end - i);
        }
        window.onRgba(0, flipPixels);
        testSpeed[1] += new Date().getTime() - tm0;
        testSpeed[0] += 1;
        if (testSpeed[0] > 30) {
            console.log('gl: ' + parseInt('' + testSpeed[1] / testSpeed[0]));
            testSpeed = [0, 0];
        }
    }
    /*
    var testCanvas = document.getElementById("test-yuv-decoder-canvas");
    if (testCanvas && currentFrame) {
      var ctx = testCanvas.getContext("2d");
      testCanvas.width = frame.format.displayWidth;
      testCanvas.height = frame.format.displayHeight;
      var img = ctx.createImageData(testCanvas.width, testCanvas.height);
      img.data.set(currentFrame);
      ctx.putImageData(img, 0, 0);
    }
    */
}

export function sendOffCanvas(c) {
    let canvas = c.transferControlToOffscreen();
    yuvWorker.postMessage({canvas}, [canvas]);
}

export function setConn(conn) {
    window.curConn = conn;
}

export function getConn() {
    return window.curConn;
}

const clipboardSync=new BrowserClipboardSync({
    send:text=>getConn()?.sendClipboardText(text)===true,
    receive:text=>{
        try{
            if(window.parent!==window &&
               typeof window.parent.__rustdeskReceiveRemoteClipboard==="function"){
                window.parent.__rustdeskReceiveRemoteClipboard(text,window);
                return;
            }
        }catch{}
        if(navigator.clipboard?.writeText){
            navigator.clipboard.writeText(text).catch(()=>
                console.info("Remote clipboard requires browser clipboard-write permission"));
        }
    }
});

export function receiveRemoteClipboard(text){
    clipboardSync.fromRemote(text);
}
let currentFileClipboard;
window.__rustdeskClipboardAccess={
    enable(initialText){
        getConn()?.setBrowserClipboardEnabled?.(true);
        const result=clipboardSync.setEnabled(true,initialText);
        currentFileClipboard?.enable();
        return result;
    },
    disable(){
        clipboardSync.setEnabled(false);
        currentFileClipboard?.disable();
        getConn()?.setBrowserClipboardEnabled?.(false);
    },
    suspendText(){clipboardSync.setEnabled(false);},
    sendLocal(text){return clipboardSync.sendLocal(text);},
    pasteText(text){
        if(!clipboardSync.enabled||getConn()?._sessionMode!=="desktop"||
           !getConn()?._peerInfo)return false;
        const sent=clipboardSync.sendLocal(text);
        if(!sent&&text!==clipboardSync.lastLocal&&text!==clipboardSync.lastRemote)return false;
        setTimeout(pasteIntoRemoteExplorer,90);
        return true;
    },
    async downloadRemoteFiles(){
        if(!currentFileClipboard)throw Error("尚未建立远程桌面连接");
        return currentFileClipboard.download();
    },
    cancelRemoteFiles(){return currentFileClipboard?.cancelDownload()??false;},
    offerFiles(files,paste=false){
        return currentFileClipboard?.offerFiles(files,paste?pasteIntoRemoteExplorer:undefined)??false;
    },
    hasRemoteFiles(){return currentFileClipboard?.hasRemoteFiles()??false;}
};
function announceClipboardReady(){
    try{
        if(window.parent!==window &&
           typeof window.parent.__rustdeskClipboardReady==="function")
            window.parent.__rustdeskClipboardReady(window);
    }catch{}
}
let pendingRemotePaste=false;
function pasteIntoRemoteExplorer(){
    const conn=getConn();
    if(conn?._sessionMode!=="desktop")return;
    conn.inputKey("v",true,false,false,true,false,false);
    conn.inputKey("v",false,false,false,true,false,false);
}
const clipboardPasteTimeout=900;
function acceptClipboardFiles(files){
    if(!files.length||!currentFileClipboard)return false;
    const wantsPaste=pendingRemotePaste;
    pendingRemotePaste=false;
    try{
        return currentFileClipboard.offerFiles(
            files,wantsPaste?pasteIntoRemoteExplorer:undefined);
    }catch(error){
        msgbox("error","文件剪贴板",String(error?.message||error));
        return false;
    }
}
function acceptClipboardText(text){
    if(!pendingRemotePaste)return;
    pendingRemotePaste=false;
    if(text){
        const sent=clipboardSync.sendLocal(text);
        if(!sent&&text!==clipboardSync.lastLocal&&text!==clipboardSync.lastRemote){
            try{window.parent?.__rustdeskClipboardSendFailure?.(window);}catch{}
            return;
        }
    }
    // Wait for the clipboard message to reach the peer before Ctrl+V.
    setTimeout(pasteIntoRemoteExplorer,80);
}
function requestBrowserClipboard(){
    if(!clipboardSync.enabled)return;
    const api=navigator.clipboard;
    if(!api)return;
    try{
        if(typeof api.read==="function"){
            Promise.resolve(api.read()).then(async items=>{
                if(!pendingRemotePaste)return;
                const files=[];
                let text="";
                for(const item of items||[]){
                    const types=Array.from(item.types||[]);
                    if(types.includes("text/plain")){
                        const blob=await item.getType("text/plain");
                        text=await blob.text();
                    }
                    for(const type of types){
                        if(!type.startsWith("image/"))continue;
                        const blob=await item.getType(type);
                        const extension=type==="image/png"?"png":type==="image/jpeg"?"jpg":"bin";
                        files.push(new File([blob],"clipboard-image."+extension,{type}));
                    }
                }
                if(!pendingRemotePaste)return;
                if(files.length)acceptClipboardFiles(files);
                else acceptClipboardText(text);
            }).catch(()=>{});
        }else if(typeof api.readText==="function"){
            Promise.resolve(api.readText()).then(text=>acceptClipboardText(text)).catch(()=>{});
        }
    }catch{}
}
installRemoteShortcutCapture({
    target:window,
    isActive:()=>getConn()?._sessionMode==="desktop"&&
        Boolean(getConn()?._peerInfo)&&document.visibilityState!=="hidden"&&
        !window.frameElement?.classList?.contains("session-inactive"),
    send:(name,down,event)=>getConn()?.inputKey(name,down,false,
        event.altKey,event.ctrlKey,event.shiftKey,event.metaKey),
    sendText:text=>{
        if(text&&getConn()?._peerInfo)getConn()?.inputString(text);
    },
    allowEscape:()=>{
        try{return window.parent!==window&&window.parent.__rustdeskIsFullscreen?.()===true;}
        catch{return false;}
    },
    paste:()=>{
        if(!clipboardSync.enabled){
            pasteIntoRemoteExplorer();
            return;
        }
        pendingRemotePaste=true;
        try{
            const readFromParent=window.parent!==window&&window.parent.__rustdeskReadHostClipboard;
            if(typeof readFromParent==="function"){
                Promise.resolve(readFromParent(window)).then(result=>{
                    if(!pendingRemotePaste)return;
                    if(result?.available&&result.text){
                        acceptClipboardText(result.text);
                        return;
                    }
                    requestBrowserClipboard();
                }).catch(()=>requestBrowserClipboard());
            }else requestBrowserClipboard();
        }catch{requestBrowserClipboard();}
        setTimeout(()=>{
            if(!pendingRemotePaste)return;
            pendingRemotePaste=false;
            pasteIntoRemoteExplorer();
        },clipboardPasteTimeout);
    }
});
window.addEventListener("paste",event=>{
    if(!clipboardSync.enabled)return;
    const target=event.target;
    if(isBrowserEditable(target))return;
    const data=event.clipboardData;
    const files=Array.from(data?.files||[]);
    if(!files.length){
        for(const item of Array.from(data?.items||[])){
            if(item.kind==="file"&&item.getAsFile){
                const file=item.getAsFile();
                if(file)files.push(file);
            }
        }
    }
    if(files.length&&currentFileClipboard){
        if(acceptClipboardFiles(files)){
            event.preventDefault?.();
            event.stopImmediatePropagation?.();
        }
        return;
    }
    if(Array.from(data?.types||[]).includes("Files")&&!files.length){
        pendingRemotePaste=false;
        msgbox("error","File Clipboard","浏览器无法读取资源管理器中的文件，请改用文件传输标签上传。");
        return;
    }
    const text=data?.getData("text/plain");
    if(pendingRemotePaste){
        event.preventDefault?.();
        event.stopImmediatePropagation?.();
        acceptClipboardText(text);
    }else if(text)clipboardSync.sendLocal(text);
},true);
document.addEventListener("fullscreenchange",()=>{
    if(typeof window.onFullscreenChanged==="function" && window.parent===window)
        window.onFullscreenChanged(Boolean(document.fullscreenElement));
});

export async function startConn(id) {
    setByName('remote_id', id);
    await curConn.start(id);
}

export function close() {
    getConn()?.close();
    setConn(undefined);
}

let currentBrowserTransfer;

export function reportRemoteFileClipboardPermission(allowed){
    try{
        if(window.parent!==window&&typeof window.parent.__rustdeskFileClipboardPermission==="function")
            window.parent.__rustdeskFileClipboardPermission(allowed,window);
    }catch{}
}
function reportFileClipboardStatus(kind,detail){
    try{
        if(window.parent!==window&&typeof window.parent.__rustdeskClipboardFileStatus==="function"){
            window.parent.__rustdeskClipboardFileStatus(kind,detail,window);
        }
    }catch{}
}

function notifySessionClosed(){
    const id=window.__rustdeskNativeSessionSeen;
    if(!id)return;
    window.__rustdeskNativeSessionSeen=undefined;
    try{
        if(window.parent!==window&&typeof window.parent.__rustdeskOnSessionClosed==="function"){
            window.parent.__rustdeskOnSessionClosed(window,id);
        }
    }catch{}
}

function fileTransferError(request, error) {
    const id=Number(request?.id||0);
    const message=String(error?.message||error);
    if(id>0)pushEvent("job_error",{id,err:message,file_num:0});
    msgbox("error","File Transfer",message);
}

function withFileTransfer(request, operation) {
    try{
        const conn=getConn();
        if(!conn || conn._sessionMode!=="file" || !currentBrowserTransfer){
            throw Error("请先连接文件传输会话");
        }
        operation(conn,currentBrowserTransfer);
    }catch(error){fileTransferError(request,error);}
}

export function newConn() {
    window.curConn?.close();
    const conn = new Connection();
    setConn(conn);
    currentBrowserTransfer=new BrowserFileTransfer({
        send:payload=>conn.sendFileMessage(payload),
        emit:pushEvent,
        decompress,
        pendingBytes:()=>conn.fileTransportBuffer(),
        message:text=>console.info("RustDesk File Transfer:",text)
    });
    conn.setFileBridge(currentBrowserTransfer);
    currentFileClipboard=new BrowserFileClipboard({
        send:payload=>conn.sendCliprdr(payload.cliprdr),
        notify:reportFileClipboardStatus
    });
    conn.setFileClipboard(currentFileClipboard);
    announceClipboardReady();
    return conn;
}

function openLocalFilePicker(folder=false) {
    const input=document.createElement("input");
    input.type="file";
    input.multiple=true;
    if(folder){
        input.setAttribute("webkitdirectory","");
        input.webkitdirectory=true;
    }
    input.style.cssText="position:fixed;left:-9999px;opacity:0;pointer-events:none";
    document.body.append(input);
    const clean=()=>input.remove();
    input.addEventListener("change",()=>{
        withFileTransfer({},(_conn,transfer)=>transfer.register(input.files,folder));
        clean();
    },{once:true});
    input.addEventListener("cancel",clean,{once:true});
    input.click();
}

async function collectDroppedFiles(transfer) {
    const files=[],items=[...(transfer?.items||[])];
    let directory=false;
    async function walk(entry,prefix="",depth=0) {
        if(depth>20 || files.length>=4096)return;
        if(entry.isFile){
            const file=await new Promise((resolve,reject)=>entry.file(resolve,reject));
            const relative=prefix+file.name;
            files.push({name:file.name,size:file.size,lastModified:file.lastModified,
                webkitRelativePath:relative,slice:file.slice.bind(file)});
            return;
        }
        if(!entry.isDirectory)return;
        directory=true;
        const reader=entry.createReader();
        for(;;){
            const batch=await new Promise((resolve,reject)=>reader.readEntries(resolve,reject));
            if(!batch.length)break;
            for(const item of batch)await walk(item,prefix+entry.name+"/",depth+1);
        }
    }
    for(const item of items){
        const entry=item.kind==="file"&&item.webkitGetAsEntry?.();
        if(entry)await walk(entry);
    }
    if(!files.length){
        for(const file of [...(transfer?.files||[])])files.push(file);
        directory=files.some(file=>file.webkitRelativePath?.includes("/"));
    }
    return {files,directory};
}

// A same-origin embedding page may pass File objects to the connected file session.
window.__rustdeskReceiveFiles=(files,folder=false)=>{
    withFileTransfer({},(conn,transfer)=>{
        if(!conn._peerInfo)throw Error("文件传输连接尚未完成");
        transfer.register(files,folder);
    });
};
function installFileDrop(){
    const overlay=document.createElement("div");
    overlay.className="rustdesk-transfer-dropzone";
    overlay.hidden=true;
    overlay.textContent="松开鼠标，上传文件至远程设备";
    document.body.append(overlay);
    let depth=0;
    const fileDrag=event=>[...(event.dataTransfer?.types||[])].includes("Files");
    const fileDropAllowed=()=>getConn()?._sessionMode==="file"||
        getConn()?._sessionMode==="desktop"&&clipboardSync.enabled&&Boolean(currentFileClipboard);
    window.addEventListener("dragenter",event=>{
        if(!fileDrag(event)||!fileDropAllowed())return;
        event.preventDefault();
        overlay.textContent=getConn()?._sessionMode==="desktop"
          ?"松开鼠标，通过剪贴板粘贴到远程设备"
          :"松开鼠标，上传文件至远程设备";
        depth++; overlay.hidden=false;
    },true);
    window.addEventListener("dragover",event=>{
        if(!fileDrag(event)||!fileDropAllowed())return;
        event.preventDefault();
        if(event.dataTransfer)event.dataTransfer.dropEffect="copy";
    },true);
    window.addEventListener("dragleave",event=>{
        if(!fileDrag(event))return;
        depth=Math.max(0,depth-1);
        if(depth===0)overlay.hidden=true;
    },true);
    window.addEventListener("drop",async event=>{
        if(!fileDrag(event)||!fileDropAllowed())return;
        event.preventDefault();
        event.stopPropagation();
        overlay.hidden=true;depth=0;
        try{
            const selected=await collectDroppedFiles(event.dataTransfer);
            if(getConn()?._sessionMode==="desktop"){
                if(!currentFileClipboard?.offerFiles(selected.files,pasteIntoRemoteExplorer))
                    throw Error("远程文件剪贴板尚未准备好");
            }else window.__rustdeskReceiveFiles(selected.files,selected.directory);
        }catch(error){
            if(getConn()?._sessionMode==="desktop")
                msgbox("error","文件剪贴板",String(error?.message||error));
            else fileTransferError({},error);
        }
    },true);
}
if(typeof window.addEventListener==="function"&&typeof document.body?.append==="function"){
    installFileDrop();
}

let sodium;

export async function verify(signed, pk) {
    if (!sodium) {
        await _sodium.ready;
        sodium = _sodium;
    }
    if (typeof pk == 'string') {
        pk = decodeBase64(pk);
    }
    return sodium.crypto_sign_open(signed, pk);
}

export function decodeBase64(pk) {
    return sodium.from_base64(pk, sodium.base64_variants.ORIGINAL);
}

export function genBoxKeyPair() {
    const pair = sodium.crypto_box_keypair();
    const sk = pair.privateKey;
    const pk = pair.publicKey;
    return [sk, pk];
}

export function genSecretKey() {
    return sodium.crypto_secretbox_keygen();
}

export function seal(unsigned, theirPk, ourSk) {
    const nonce = Uint8Array.from(Array(24).fill(0));
    return sodium.crypto_box_easy(unsigned, nonce, theirPk, ourSk);
}

function makeOnce(value) {
    var byteArray = Array(24).fill(0);

    for (var index = 0; index < byteArray.length && value > 0; index++) {
        var byte = value & 0xff;
        byteArray[index] = byte;
        value = (value - byte) / 256;
    }

    return Uint8Array.from(byteArray);
};

export function encrypt(unsigned, nonce, key) {
    return sodium.crypto_secretbox_easy(unsigned, makeOnce(nonce), key);
}

export function decrypt(signed, nonce, key) {
    return sodium.crypto_secretbox_open_easy(signed, makeOnce(nonce), key);
}

window.setByName = (name, value) => {
    switch (name) {
        case 'session_add_sync': {
            const options = JSON.parse(value || '{}');
            if (!options.id || typeof options.id !== 'string') return 'Invalid remote device';
            window.__rustdeskSessionOptions = options;
            return '';
        }
        case 'session_start': {
            const request = JSON.parse(value || '{}');
            const id = String(request.id || '').trim();
            if (!id) return '';
            const conn = newConn();
            const mode = window.__rustdeskSessionOptions || {};
            conn.setSessionMode(mode.isFileTransfer ? "file" : mode.isViewCamera ? "camera" : mode.isTerminal ? "terminal" : "desktop",
                !!window.__rustdeskTerminalAdmin && !!mode.isTerminal);
            window.__rustdeskTerminalAdmin = false;
            window.__rustdeskNativeSessionSeen = id;
            window.dispatchEvent(new CustomEvent('rustdesk:native-session-open', {detail:{id}}));
            conn.start(id).catch(error=>msgbox("error","Connection Error",String(error)));
            return '';
        }
        case 'session_login': {
            const request = JSON.parse(value || '{}');
            const conn = getConn();
            if (!conn) return '';
            conn.setRemember(request.remember === true || request.remember === 'true');
            conn.login(String(request.password || ''),
                String(request.os_username||''),String(request.os_password||''));
            return '';
        }
        case 'session_close':
            close();
            notifySessionClosed();
            return '';
        case 'forget_password':
            getConn()?.setRemember(false);
            return '';
        case 'envvar': {
            const setting=JSON.parse(value||'{}');
            if(setting.name==='IS_TERMINAL_ADMIN')window.__rustdeskTerminalAdmin=setting.value==='Y';
            return '';
        }
        case 'read_remote_dir': {
            const request=JSON.parse(value||'{}');
            getConn()?.readRemoteDir(String(request.path||''),!!request.include_hidden);
            return '';
        }
        case 'cancel_job':
            currentBrowserTransfer?.cancel(Number(value||0));
            getConn()?.cancelFileJob(Number(value||0));
            return '';
        case 'select_files':
            withFileTransfer({},()=>openLocalFilePicker(value===true||value==="true"||value==="Y"));
            return '';
        case 'send_local_files': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(_conn,transfer)=>transfer.startUpload(request));
            return '';
        }
        case 'send_files': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(_conn,transfer)=>transfer.startDownload(request));
            return '';
        }
        case 'create_dir': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{create:{
                id:Number(request.id||0),path:String(request.path||"")}}}));
            return '';
        }
        case 'remove_file': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{remove_file:{
                id:Number(request.id||0),path:String(request.path||""),
                file_num:Number(request.file_num||0)}}}));
            return '';
        }
        case 'remove_dir': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{remove_dir:{
                id:Number(request.id||0),path:String(request.path||""),recursive:!!request.recursive}}}));
            return '';
        }
        case 'read_dir_to_remove_recursive': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{all_files:{
                id:Number(request.id||0),path:String(request.path||""),include_hidden:!!request.show_hidden}}}));
            return '';
        }
        case 'remove_all_empty_dirs': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{remove_dir:{
                id:Number(request.id||0),path:String(request.path||""),recursive:true}}}));
            return '';
        }
        case 'rename_file': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{rename:{
                id:Number(request.id||0),path:String(request.path||""),
                new_name:String(request.new_name||"")}}}));
            return '';
        }
        case 'confirm_override_file': {
            const request=JSON.parse(value||'{}');
            withFileTransfer(request,(conn)=>conn.sendFileMessage({file_action:{send_confirm:{
                id:Number(request.id||0),file_num:Number(request.file_num||0),
                ...(request.need_override?{offset_blk:0}:{skip:true})}}}));
            return '';
        }
        case 'open_terminal': {
            const request=JSON.parse(value||'{}');
            getConn()?.sendTerminalAction({open:{terminal_id:Number(request.terminal_id||0),
                rows:Number(request.rows||24),cols:Number(request.cols||80)}});
            return '';
        }
        case 'send_terminal_input': {
            const request=JSON.parse(value||'{}');
            getConn()?.sendTerminalAction({data:{terminal_id:Number(request.terminal_id||0),
                data:String(request.data||'')}});
            return '';
        }
        case 'resize_terminal': {
            const request=JSON.parse(value||'{}');
            getConn()?.sendTerminalAction({resize:{terminal_id:Number(request.terminal_id||0),
                rows:Number(request.rows||24),cols:Number(request.cols||80)}});
            return '';
        }
        case 'close_terminal': {
            const request=JSON.parse(value||'{}');
            getConn()?.sendTerminalAction({close:{terminal_id:Number(request.terminal_id||0)}});
            return '';
        }
        case 'option:toggle':
            getConn()?.toggleOption(value);
            return '';
        case 'option:session': {
            const sessionOption = typeof value === 'string' && value.startsWith('{') ? JSON.parse(value) : null;
            if (sessionOption?.name) localStorage.setItem('rustdesk.session.' + sessionOption.name, sessionOption.value);
            return '';
        }
        case 'fullscreen': {
            const wanted=value==="Y";
            try{
                const host=window.parent!==window && window.parent?.__rustdeskSetFullscreen;
                const operation=typeof host==="function"
                    ? host(wanted,window)
                    : wanted
                      ? document.documentElement.requestFullscreen?.({navigationUI:"hide"})
                      : document.fullscreenElement
                        ? document.exitFullscreen?.() : undefined;
                Promise.resolve(operation).catch(error=>{
                    window.onFullscreenChanged?.(!wanted);
                    msgbox("error","Fullscreen",String(error?.message||error));
                });
            }catch(error){
                window.onFullscreenChanged?.(!wanted);
                msgbox("error","Fullscreen",String(error?.message||error));
            }
            return '';
        }
        case 'option:flutter:peer':
        case 'option:flutter:local': {
            const option = JSON.parse(value || '{}');
            if (option.name) localStorage.setItem('rustdesk.flutter.' + option.name, option.value);
            return '';
        }
        case 'remote_id':
            localStorage.setItem('remote-id', value);
            break;
        case 'connect':
            newConn();
            startConn(value);
            break;
        case 'login': {
            const request = JSON.parse(value || '{}');
            const conn = getConn();
            if (!conn) return '';
            conn.setRemember(request.remember === true || request.remember === 'true');
            conn.login(String(request.password || ''),
                String(request.os_username || ''),String(request.os_password || ''));
            return '';
        }
        case 'close':
            close();
            notifySessionClosed();
            break;
        case 'refresh':
            curConn.refresh();
            break;
        case 'reconnect':
            curConn.reconnect();
            break;
        case 'toggle_option':
            curConn.toggleOption(value);
            break;
        case 'image_quality':
            curConn.setImageQuality(value);
            break;
        case 'lock_screen':
            curConn.lockScreen();
            break;
        case 'ctrl_alt_del':
            curConn.ctrlAltDel();
            break;
        case 'switch_display':
            curConn.switchDisplay(value);
            break;
        case 'remove':
            const peers = getPeers();
            delete peers[value];
            localStorage.setItem('peers', JSON.stringify(peers));
            break;
        case 'input_key':
            value = JSON.parse(value);
            curConn.inputKey(value.name, value.down == 'true', value.press == 'true', value.alt == 'true', value.ctrl == 'true', value.shift == 'true', value.command == 'true');
            break;
        case 'input_string':
            curConn.inputString(value);
            break;
        case 'send_mouse':
            let mask = 0;
            value = JSON.parse(value);
            switch (value.type) {
                case 'down':
                    mask = 1;
                    break;
                case 'up':
                    mask = 2;
                    break;
                case 'wheel':
                    mask = 3;
                    break;
            }
            switch (value.buttons) {
                case 'left':
                    mask |= 1 << 3;
                    break;
                case 'right':
                    mask |= 2 << 3;
                    break;
                case 'wheel':
                    mask |= 4 << 3;
            }
            curConn.inputMouse(mask, parseInt(value.x || '0'), parseInt(value.y || '0'), value.alt == 'true', value.ctrl == 'true', value.shift == 'true', value.command == 'true');
            break;
        case 'option':
            value = JSON.parse(value);
            localStorage.setItem(value.name, value.value);
            if (value.name === 'access_token' && value.value) {
                getServerConf(value.value);
            }
            break;
        case 'peer_option':
            value = JSON.parse(value);
            curConn.setOption(value.name, value.value);
            break;
        case 'input_os_password':
            curConn.inputOsPassword(value);
            break;
        default:
            break;
    }
}

window.getByName = (name, arg) => {
    let v = _getByName(name, arg);
    if (typeof v == 'string' || v instanceof String) return v;
    if (v == undefined || v == null) return '';
    return JSON.stringify(v);
}

function getPeersForDart() {
    const peers = [];
    for (const [id, value] of Object.entries(getPeers())) {
        if (!id) continue;
        const tm = value['tm'];
        const info = value['info'];
        if (!tm || !info) continue;
        peers.push([tm, id, info]);
    }
    return peers.sort().reverse().map(x => x.slice(1));
}

function _getByName(name, arg) {
    switch (name) {
        case 'peers':
            return getPeersForDart();
        case 'remote_id':
            return localStorage.getItem('remote-id');
        case 'remember':
            return getConn()?.getRemember() || false;
        case 'toggle_option':
        case 'option:toggle':
            return getConn()?.getOption(arg) || false;
        case 'option:session': {
            const stored=localStorage.getItem('rustdesk.session.' + arg) || getConn()?.getOption(arg) || '';
            if(arg==='view_style')
                return ['adaptive','original','custom'].includes(stored)?stored:'adaptive';
            // Flutter requires a recognized ScrollStyle while processing every video frame.
            return arg === 'scroll_style' && !['scrollauto','scrollbar','scrolledge'].includes(stored)
                ? 'scrollauto' : stored;
        }
        case 'fullscreen': {
            try{
                const host=window.parent!==window && window.parent?.__rustdeskIsFullscreen;
                if(typeof host==="function")return host()?"Y":"N";
            }catch{}
            return document.fullscreenElement?"Y":"N";
        }
        case 'option:flutter:peer':
        case 'option:flutter:local':
            return localStorage.getItem('rustdesk.flutter.' + arg) || '';
        case 'option:local':
            return localStorage.getItem(arg) || '';
        case 'login_method':
        case 'conn_session_id':
            return '';
        case 'api_server':
            return localStorage.getItem('api-server') || '';
        case 'platform':
            return getConn()?._peerInfo?.platform || '';
        case 'envvar':
            return '';
        case 'option':
            const v = localStorage.getItem(arg);
            if (arg === 'access_token' && v) {
                getServerConf(v);
            }
            return v;
        case 'image_quality':
            return getConn()?.getImageQuality() || 'balanced';
        case 'translate':
            arg = JSON.parse(arg);
            return translate(arg.locale, arg.text);
        case 'peer_option':
            return curConn.getOption(arg);
        case 'test_if_valid_server':
            break;
        case 'version':
            return version;
    }
    return '';
}

let opusWorker = new Worker("./libopus.js");
let pcmPlayer;

export function initAudio(channels, sampleRate) {
    pcmPlayer = newAudioPlayer(channels, sampleRate);
    opusWorker.postMessage({channels, sampleRate});
}

export function playAudio(packet) {
    opusWorker.postMessage(packet, [packet.buffer]);
}

window.reloadShortcuts = () => {
    // Browser keyboard input is handled by Flutter's active remote session.
};

window.init = async () => {
    if (yuvWorker) {
        yuvWorker.onmessage = (e) => {
            if (typeof window.onRgba === "function") window.onRgba(0, e.data);
        }
    }
    opusWorker.onmessage = (e) => {
        pcmPlayer.feed(e.data);
    }
    const warming=loadVp9(decoder=>decoder.close?.());
    if(warming?.catch)warming.catch(error=>console.warn("VP9 preloading failed",error));
    await initZstd();
    console.log('init done');
    if (typeof window.onInitFinished === "function") {
        const notify = window.onInitFinished;
        window.onInitFinished = undefined;
        notify();
    }
}

export function getPeers() {
    try {
        return JSON.parse(localStorage.getItem('peers')) || {};
    } catch (e) {
        return {};
    }
}

function newAudioPlayer(channels, sampleRate) {
    return new PCMPlayer({
        channels,
        sampleRate,
        flushingTime: 2000
    });
}

export function copyToClipboard(text) {
    if (window.clipboardData && window.clipboardData.setData) {
        // Internet Explorer-specific code path to prevent textarea being shown while dialog is visible.
        return window.clipboardData.setData("Text", text);

    } else if (document.queryCommandSupported && document.queryCommandSupported("copy")) {
        var textarea = document.createElement("textarea");
        textarea.textContent = text;
        textarea.style.position = "fixed";  // Prevent scrolling to bottom of page in Microsoft Edge.
        document.body.appendChild(textarea);
        textarea.select();
        try {
            return document.execCommand("copy");  // Security exception may be thrown by some browsers.
        } catch (ex) {
            console.warn("Copy to clipboard failed.", ex);
            // return prompt("Copy to clipboard: Ctrl+C, Enter", text);
        } finally {
            document.body.removeChild(textarea);
        }
    }
}