"use strict";

const MAX_CLIPBOARD_BYTES=1024*1024;
const utf8=new TextEncoder();

function validText(value){
  if(typeof value!=="string")return false;
  return utf8.encode(value).length<=MAX_CLIPBOARD_BYTES;
}

// Text clipboard synchronization is opt-in and scoped to one remote session.
export class BrowserClipboardSync {
  constructor({send,receive}){
    this.send=send;
    this.receive=receive;
    this.enabled=false;
    this.lastLocal=null;
    this.lastRemote=null;
  }

  setEnabled(enabled,initialText){
    this.enabled=enabled===true;
    if(!this.enabled){
      this.lastLocal=null;
      this.lastRemote=null;
      return false;
    }
    return typeof initialText==="string" ? this.sendLocal(initialText) : true;
  }

  sendLocal(text){
    if(!this.enabled||!validText(text)||!text||text===this.lastLocal||text===this.lastRemote)
      return false;
    if(this.send(text)===false)return false;
    this.lastLocal=text;
    return true;
  }

  fromRemote(text){
    if(!this.enabled||!validText(text)||text===this.lastRemote||text===this.lastLocal)
      return false;
    this.lastRemote=text;
    this.receive(text);
    return true;
  }
}
