"use strict";

// Keys are routed through one capture handler while an authenticated remote
// desktop owns focus. Local settings fields retain their native keyboard input.
const modifierNames=new Set(["Control","Shift","Alt","Meta","OS"]);
const aliases={
  " ":"Space",Spacebar:"Space",Enter:"Return",
  ArrowLeft:"LeftArrow",ArrowRight:"RightArrow",
  ArrowUp:"UpArrow",ArrowDown:"DownArrow",
  Esc:"Escape",Del:"Delete",ControlLeft:"Control",ControlRight:"RControl",
  ShiftLeft:"Shift",ShiftRight:"RShift",AltLeft:"Alt",AltRight:"RAlt",
  MetaLeft:"Meta",MetaRight:"RWin"
};
function remoteKey(event){
  const code=String(event.code||"");
  if(/^Key[A-Z]$/.test(code))return code.slice(3).toLowerCase();
  if(/^Digit[0-9]$/.test(code))return code.slice(5);
  if(/^Numpad[0-9]$/.test(code))return code;
  if(aliases[code])return aliases[code];
  const key=String(event.key||"");
  return aliases[key]||key;
}
export function isBrowserEditable(target){
  // Flutter uses a hidden editable element for the remote keyboard.
  if(target?.closest?.("flt-text-editing-host,flt-glass-pane,flt-semantics-host,flutter-view"))
    return false;
  if(target?.isContentEditable)return true;
  return Boolean(target?.closest?.("input,textarea,select,[contenteditable]"));
}
function isModifier(event){
  return modifierNames.has(event.key)||/^((Control|Shift|Alt|Meta)(Left|Right))$/.test(event.code||"");
}
const snapshot=event=>({
  ctrlKey:!!event.ctrlKey,altKey:!!event.altKey,
  shiftKey:!!event.shiftKey,metaKey:!!event.metaKey
});
export function installRemoteShortcutCapture({target,isActive,send,paste,sendText=()=>{},allowEscape=()=>false}){
  const held=new Map();
  let composing=false,compositionEnd="",compositionAt=0;
  const eligible=event=>isActive()&&!isBrowserEditable(event.target);
  const cancel=event=>{
    event.preventDefault?.();
    event.stopImmediatePropagation?.();
  };
  const keydown=event=>{
    if(!eligible(event))return;
    const key=remoteKey(event);
    if(allowEscape()&&(key==="Escape"||key==="Esc"))return;
    if(event.isComposing||composing||key==="Process"||key==="Dead"||
       event.getModifierState?.("AltGraph")){
      event.stopImmediatePropagation?.();
      return;
    }
    if(!key||key==="Unidentified")return;
    const id=String(event.code||key);
    const isPaste=key.toLowerCase()==="v"&&(event.ctrlKey||event.metaKey)&&!event.altKey;
    cancel(event);
    if(held.has(id)){
      if(event.repeat&&!isPaste)send(key,true,event);
      return;
    }
    held.set(id,{key,forward:!isPaste,flags:snapshot(event)});
    if(isPaste){paste(event);return;}
    // Extended Unicode characters are sent as text, including emoji.
    if(!event.ctrlKey&&!event.altKey&&!event.metaKey&&key.length>1&&!isModifier(event)&&
       !/^(F\d+|Numpad\d|LeftArrow|RightArrow|UpArrow|DownArrow|PageUp|PageDown)$/.test(key)&&
       !/^[A-Za-z]+$/.test(key)){
      held.get(id).forward=false;
      sendText(key);
      return;
    }
    send(key,true,event);
  };
  const keyup=event=>{
    const key=String(event.code||remoteKey(event));
    const item=held.get(key);
    if(!item)return;
    held.delete(key);
    cancel(event);
    if(item.forward)send(item.key,false,event);
  };
  const release=()=>{
    for(const item of held.values()){
      if(item.forward)send(item.key,false,{ctrlKey:false,altKey:false,shiftKey:false,metaKey:false});
    }
    held.clear();
    composing=false;
  };
  const beforeinput=event=>{
    if(!eligible(event))return;
    const inputType=String(event.inputType||"");
    if(!["insertText","insertFromComposition","insertCompositionText","insertReplacementText"].includes(inputType))return;
    const value=String(event.data||"");
    if(!value||inputType==="insertCompositionText")return;
    if(composing)return;
    if(value===compositionEnd&&Date.now()-compositionAt<200){
      cancel(event);
      return;
    }
    cancel(event);
    sendText(value);
  };
  const compositionstart=event=>{
    if(!eligible(event))return;
    composing=true;
    event.stopImmediatePropagation?.();
  };
  const compositionend=event=>{
    if(!eligible(event))return;
    composing=false;
    const value=String(event.data||"");
    if(value){
      compositionEnd=value;
      compositionAt=Date.now();
      sendText(value);
    }
    event.stopImmediatePropagation?.();
  };
  const handlers={keydown,keyup,beforeinput,compositionstart,compositionend,blur:release};
  for(const [name,handler] of Object.entries(handlers)){
    target.addEventListener(name,handler,name!=="blur");
  }
  return ()=>{
    release();
    for(const [name,handler] of Object.entries(handlers))
      target.removeEventListener?.(name,handler,name!=="blur");
  };
}
