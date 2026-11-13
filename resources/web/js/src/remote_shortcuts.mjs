"use strict";

// Capture browser-handled shortcut combinations while the remote desktop is
// focused. The connection sends the corresponding key events to the peer.
const modifierNames=new Set(["Control","Shift","Alt","Meta","AltGraph","OS"]);
function remoteKey(event){
  const code=String(event.code||"");
  if(/^Key[A-Z]$/.test(code))return code.slice(3).toLowerCase();
  if(/^Digit[0-9]$/.test(code))return code.slice(5);
  const key=String(event.key||"");
  return key.length===1?key.toLowerCase():key;
}
export function isBrowserEditable(target){
  // Flutter receives physical keyboard events through a hidden editable node.
  // Treat that node as the remote desktop, not as a normal local text field.
  if(target?.closest?.("flt-text-editing-host,flt-glass-pane,flt-semantics-host,flutter-view"))
    return false;
  if(target?.isContentEditable)return true;
  return Boolean(target?.closest?.("input,textarea,select,[contenteditable]"));
}

export function installRemoteShortcutCapture({target,isActive,send,paste}){
  const held=new Map();
  const keydown=event=>{
    if(!isActive()||event.isComposing||isBrowserEditable(event.target)||
       event.getModifierState?.("AltGraph"))return;
    if(!event.ctrlKey&&!event.metaKey&&!event.altKey)return;
    if(modifierNames.has(event.key))return;
    const key=remoteKey(event);
    if(!key)return;
    const id=event.code||key;
    if(held.has(id)){
      event.preventDefault?.();
      event.stopImmediatePropagation?.();
      return;
    }
    held.set(id,{key,copy:!(key==="v"&&(event.ctrlKey||event.metaKey)&&!event.altKey)});
    // Keep the native paste event available to read clipboardData. Browser
    // Clipboard API reads also begin within this user-initiated key event.
    if(key!=="v"||(event.altKey||(!event.ctrlKey&&!event.metaKey)))
      event.preventDefault?.();
    event.stopImmediatePropagation?.();
    if(key==="v"&&(event.ctrlKey||event.metaKey)&&!event.altKey)
      paste(event);
    else send(key,true,event);
  };
  const keyup=event=>{
    const key=event.code||remoteKey(event);
    const item=held.get(key);
    if(!item)return;
    held.delete(key);
    event.preventDefault?.();
    event.stopImmediatePropagation?.();
    if(item.copy)send(item.key,false,event);
  };
  const release=()=>held.clear();
  target.addEventListener("keydown",keydown,true);
  target.addEventListener("keyup",keyup,true);
  target.addEventListener("blur",release);
  return ()=> {
    held.clear();
    target.removeEventListener?.("keydown",keydown,true);
    target.removeEventListener?.("keyup",keyup,true);
    target.removeEventListener?.("blur",release);
  };
}
