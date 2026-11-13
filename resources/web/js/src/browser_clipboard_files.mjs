"use strict";

import {packageZip} from "./browser_file_transfer.mjs";

const FORMAT_DESCRIPTOR_ID=49334;
const FORMAT_CONTENT_ID=49267;
const DESCRIPTOR_LENGTH=592;
const MAX_FILES=256;
const MAX_TOTAL_BYTES=256*1024*1024;
const MAX_PART_BYTES=256*1024;
const utf16=new TextDecoder("utf-16le");
const encoder=new TextEncoder();

function fileName(value) {
  const path=String(value||"").replace(/\\/g,"/");
  if(!path||path.length>512||path.startsWith("/")||/^[a-z]:/i.test(path)||
     path.split("/").some(part=>!part||part==="."||part===".."||/[\x00-\x1f<>:"|?*]/.test(part))) {
    throw new Error("Invalid clipboard file name");
  }
  return path;
}
function number(value) {
  const n=Number(value);
  if(!Number.isSafeInteger(n)||n<0)return null;
  return n;
}
function fileListId(bytes){
  let hash=2166136261>>>0;
  for(const byte of bytes)hash=Math.imul(hash^byte,16777619)>>>0;
  return hash|0;
}
function makeDescriptors(files) {
  if(!files.length||files.length>MAX_FILES)throw new Error("Unsupported clipboard file count");
  const result=new Uint8Array(4+DESCRIPTOR_LENGTH*files.length);
  const view=new DataView(result.buffer);
  view.setUint32(0,files.length,true);
  files.forEach((file,i)=>{
    const offset=4+i*DESCRIPTOR_LENGTH, name=fileName(file.relativePath||file.name);
    if(name.length>259)throw new Error("Clipboard file name is too long");
    const size=number(file.size);
    if(size===null||size>MAX_TOTAL_BYTES)throw new Error("Invalid clipboard file size");
    view.setUint32(offset,0x44,true);
    view.setUint32(offset+36,0x80,true);
    view.setUint32(offset+64,Math.floor(size/0x100000000),true);
    view.setUint32(offset+68,size>>>0,true);
    for(let j=0;j<name.length;j++)view.setUint16(offset+72+j*2,name.charCodeAt(j),true);
  });
  return result;
}
export function parseClipboardDescriptors(bytes) {
  if(!(bytes instanceof Uint8Array)||bytes.byteLength<4)throw new Error("Missing clipboard file descriptors");
  const view=new DataView(bytes.buffer,bytes.byteOffset,bytes.byteLength);
  const count=view.getUint32(0,true);
  const expected=4+count*DESCRIPTOR_LENGTH;
  if(!count||count>MAX_FILES||bytes.byteLength<expected||bytes.byteLength>expected+65536)
    throw new Error("Invalid clipboard file descriptor count");
  const list=[],found=new Set();
  let total=0;
  for(let i=0;i<count;i++){
    const start=4+i*DESCRIPTOR_LENGTH;
    const raw=utf16.decode(bytes.subarray(start+72,start+592));
    const name=fileName(raw.split("\0",1)[0]);
    if(found.has(name.toLowerCase()))throw new Error("Duplicate clipboard file name");
    found.add(name.toLowerCase());
    const attrs=view.getUint32(start+36,true);
    const directory=!!(attrs&0x10);
    const size=view.getUint32(start+64,true)*0x100000000+view.getUint32(start+68,true);
    if(!Number.isSafeInteger(size)||size>MAX_TOTAL_BYTES)throw new Error("Clipboard file is too large");
    total+=directory?0:size;
    if(total>MAX_TOTAL_BYTES)throw new Error("Clipboard files exceed browser memory limit");
    list.push({name,size:directory?0:size,directory,index:i});
  }
  return list;
}
function downloadBlob(blob,name) {
  const url=URL.createObjectURL(blob);
  const link=document.createElement("a");
  link.href=url;link.download=name;link.style.display="none";
  document.body.append(link);link.click();link.remove();
  setTimeout(()=>URL.revokeObjectURL(url),60000);
}
function waitForReply(pending,timeout,label){
  return new Promise((resolve,reject)=>{
    const timer=setTimeout(()=>{
      pending.delete("pending");
      reject(new Error(label+" timed out"));
    },timeout);
    pending.set("pending",{resolve(value){clearTimeout(timer);pending.delete("pending");resolve(value);},
      reject(error){clearTimeout(timer);pending.delete("pending");reject(error);}});
  });
}
export class BrowserFileClipboard {
  constructor({send,save,notify}){
    this.send=send;
    this.save=save||downloadBlob;
    this.notify=notify||(()=>{});
    this.active=false;
    this.closed=false;
    this.remoteDescriptorFormat=null;
    this.localFiles=[];
    this.descriptors=null;
    this.pendingFormat=new Map();
    this.pendingData=new Map();
    this.nextStream=1;
    this.generation=0;
    this.downloading=false;
    this.downloadSerial=0;
    this.remoteListId=0;
    this.pendingPaste=null;
    this.remoteFormatWaiters=new Set();
  }
  emit(cliprdr){
    if(this.closed||!this.active)return false;
    return this.send({cliprdr})!==false;
  }
  enable(){
    if(this.closed)return;
    if(this.active)return;
    this.active=true;
    this.onConnected();
  }
  onConnected(){
    if(!this.active||this.closed)return;
    this.emit({ready:{}});
    if(this.localFiles.length)this.announce();
  }
  disable(){
    this.active=false;
    this.resolveRemoteFormatWaiters(false);
    this.localFiles=[];
    this.descriptors=null;
    this.remoteDescriptorFormat=null;
    this.remoteListId=0;
    this.generation++;
    if(this.pendingPaste){
      clearTimeout(this.pendingPaste.timer);
      this.pendingPaste=null;
    }
    this.cancelPending(new Error("Clipboard sharing was disabled"));
  }
  resolveRemoteFormatWaiters(found){
    for(const notify of this.remoteFormatWaiters)notify(found);
    this.remoteFormatWaiters.clear();
  }
  async waitForRemoteFiles(timeout=1500){
    if(this.remoteDescriptorFormat!==null)return true;
    if(!this.active||this.closed)return false;
    return new Promise(resolve=>{
      let timer;
      const done=found=>{
        clearTimeout(timer);
        this.remoteFormatWaiters.delete(done);
        resolve(Boolean(found));
      };
      this.remoteFormatWaiters.add(done);
      timer=setTimeout(()=>done(false),timeout);
      // An asynchronous Ready triggers a fresh format list on peers that
      // support the Windows file clipboard protocol.
      if(!this.emit({ready:{}}))done(false);
    });
  }
  hasRemoteFiles(){return this.active&&!this.closed&&this.remoteDescriptorFormat!==null}
  cancelPending(error){
    this.downloadSerial++;
    const waiting=this.pendingFormat.get("pending");
    if(waiting)waiting.reject(error);
    for(const value of this.pendingData.values())value.reject(error);
    this.pendingData.clear();
    this.downloading=false;
  }
  announce(){
    if(!this.localFiles.length)return;
    this.emit({format_list:{formats:[
      {id:FORMAT_DESCRIPTOR_ID,format:"FileGroupDescriptorW"},
      {id:FORMAT_CONTENT_ID,format:"FileContents"}
    ]}});
  }
  offerFiles(files,onAccepted){
    if(!this.active||this.closed)return false;
    const entries=Array.from(files||[]).filter(file=>file && typeof file.slice==="function");
    if(!entries.length)return false;
    if(entries.length>MAX_FILES)throw new Error("Too many files copied");
    let total=0;
    for(const file of entries){
      total+=Number(file.size||0);
      if(total>MAX_TOTAL_BYTES)throw new Error("Copied files exceed browser memory limit");
    }
    this.localFiles=entries;
    this.descriptors=makeDescriptors(entries);
    if(this.pendingPaste)clearTimeout(this.pendingPaste.timer);
    if(typeof onAccepted==="function"){
      const pending={callback:onAccepted,timer:null};
      pending.timer=setTimeout(()=>{
        if(this.pendingPaste!==pending)return;
        this.pendingPaste=null;
        this.notify("unsupported-files",entries.length);
      },4000);
      this.pendingPaste=pending;
    }else this.pendingPaste=null;
    this.announce();
    this.notify("local-files",entries.length);
    return true;
  }
  async handle(message){
    if(!this.active||this.closed||!message)return false;
    if(message.ready){
      if(this.localFiles.length)this.announce();
      return true;
    }
    if(message.format_list){
      const formats=message.format_list.formats||[];
      this.remoteDescriptorFormat=formats.find(item=>String(item.format||"").toLowerCase()==="filegroupdescriptorw")?.id??null;
      this.generation++;
      this.emit({format_list_response:{msg_flags:1}});
      const present=this.remoteDescriptorFormat!==null;
      this.resolveRemoteFormatWaiters(present);
      this.notify("remote-list",present);
      return true;
    }
    if(message.format_list_response){
      const pending=this.pendingPaste;
      if(pending){
        clearTimeout(pending.timer);
        this.pendingPaste=null;
        if(message.format_list_response.msg_flags===1)pending.callback();
        else this.notify("unsupported-files",this.localFiles.length);
      }
      return true;
    }
    if(message.format_data_request){
      const id=message.format_data_request.requested_format_id;
      const bytes=id===FORMAT_DESCRIPTOR_ID?this.descriptors:null;
      this.emit({format_data_response:{msg_flags:bytes?1:2,format_data:bytes||new Uint8Array()}});
      return true;
    }
    if(message.format_data_response){
      this.pendingFormat.get("pending")?.resolve(message.format_data_response);
      return true;
    }
    if(message.file_contents_request){
      await this.serveFileContents(message.file_contents_request);
      return true;
    }
    if(message.file_contents_response){
      const reply=message.file_contents_response;
      this.pendingData.get(reply.stream_id)?.resolve(reply);
      return true;
    }
    return false;
  }
  async serveFileContents(request){
    const id=Number(request.stream_id||0),index=Number(request.list_index||0);
    const file=this.localFiles[index],offset=Number(request.n_position_low>>>0)+
      Number(request.n_position_high>>>0)*0x100000000;
    // FileContents requests without an explicit lock use the current list.
    let data=new Uint8Array(),success=Boolean(file);
    try{
      if(!file)throw new Error("Unknown clipboard file index");
      if(request.dw_flags===1){
        data=new Uint8Array(8);
        new DataView(data.buffer).setBigUint64(0,BigInt(file.size),true);
      }else if(request.dw_flags===2){
        if(!Number.isSafeInteger(offset)||offset<0||offset>file.size)throw new Error("Invalid clipboard file offset");
        const size=Math.min(MAX_PART_BYTES,Math.max(0,Number(request.cb_requested)||0));
        if(!size)throw new Error("Invalid clipboard file range");
        data=new Uint8Array(await file.slice(offset,offset+size).arrayBuffer());
      }else throw new Error("Unknown clipboard file request");
    }catch{success=false}
    this.emit({file_contents_response:{msg_flags:success?1:2,stream_id:id,requested_data:data}});
  }
  async requestContents(index,offset,length){
    const stream=this.nextStream++;
    if(this.nextStream>0x7ffffffe)this.nextStream=1;
    const reply=new Promise((resolve,reject)=>{
      const timer=setTimeout(()=>{
        this.pendingData.delete(stream);
        reject(new Error("Remote clipboard file request timed out"));
      },25000);
      this.pendingData.set(stream,{resolve(value){clearTimeout(timer);resolve(value);},
        reject(error){clearTimeout(timer);reject(error);}});
    });
    const request={stream_id:stream,list_index:index,dw_flags:2,
      n_position_low:offset>>>0,n_position_high:Math.floor(offset/0x100000000),
      cb_requested:length,have_clip_data_id:false};
    this.emit({file_contents_request:request});
    try{
      const result=await reply;
      if(result.msg_flags!==1)throw new Error("The remote device refused the file request");
      const bytes=new Uint8Array(result.requested_data||[]);
      if(!bytes.length||bytes.length>length)throw new Error("Invalid remote clipboard file block");
      return bytes;
    }finally{this.pendingData.delete(stream)}
  }
  cancelDownload(){
    if(!this.downloading)return false;
    this.generation++;
    this.cancelPending(new Error("剪贴板文件下载已取消"));
    this.notify("cancelled",{});
    return true;
  }
  async download(){
    if(!this.active||this.closed)throw new Error("请先启用剪贴板共享");
    if(this.downloading)throw new Error("已有剪贴板文件下载任务");
    if(this.remoteDescriptorFormat===null && !(await this.waitForRemoteFiles()))
      throw new Error("远程剪贴板中暂无文件。请先在远程资源管理器选中文件并按 Ctrl+C。");
    this.downloading=true;
    const downloadSerial=++this.downloadSerial;
    const generation=this.generation;
    try{
      const wait=waitForReply(this.pendingFormat,20000,"Remote clipboard file list");
      this.emit({format_data_request:{requested_format_id:this.remoteDescriptorFormat}});
      const reply=await wait;
      if(reply.msg_flags!==1)throw new Error("无法读取远程剪贴板文件列表");
      const descriptors=new Uint8Array(reply.format_data||[]);
      const files=parseClipboardDescriptors(descriptors);
      this.remoteListId=fileListId(descriptors);
      const totalBytes=files.reduce((sum,file)=>sum+(file.directory?0:file.size),0);
      let completeBytes=0;
      const entries=[];
      this.notify("start",{count:files.length,total:totalBytes});
      for(const [position,file] of files.entries()){
        if(generation!==this.generation||!this.active)throw new Error("Remote clipboard changed during download");
        if(file.directory){
          entries.push({name:file.name,directory:true,chunks:[]});
          continue;
        }
        const chunks=[];
        for(let start=0;start<file.size;){
          if(generation!==this.generation||!this.active)throw new Error("Remote clipboard changed during download");
          const wanted=Math.min(file.size-start,MAX_PART_BYTES);
          const block=await this.requestContents(file.index,start,wanted);
          if(generation!==this.generation||!this.active)throw new Error("剪贴板文件下载已取消");
          chunks.push(block);
          start+=block.byteLength;
          this.notify("progress",{name:file.name,received:start,total:file.size,
            overallReceived:completeBytes+start,overallTotal:totalBytes,index:position+1,count:files.length});
        }
        completeBytes+=file.size;
        entries.push({name:file.name,chunks});
        this.notify("progress",{name:file.name,received:file.size,total:file.size,
          overallReceived:completeBytes,overallTotal:totalBytes,index:position+1,count:files.length});
      }
      if(generation!==this.generation||!this.active)throw new Error("剪贴板文件下载已取消");
      if(!entries.length)throw new Error("远程剪贴板中没有可下载的文件");
      this.notify("saving",{count:entries.length,total:totalBytes});
      const simple=entries.length===1&&!entries[0].directory&&!entries[0].name.includes("/");
      const blob=simple?new Blob(entries[0].chunks,{type:"application/octet-stream"}):packageZip(entries);
      const name=simple?entries[0].name:"rustdesk-clipboard-files.zip";
      await this.save(blob,name);
      if(generation!==this.generation||!this.active)throw new Error("剪贴板文件下载已取消");
      this.notify("downloaded",{count:entries.length,name,total:totalBytes});
      return {name,count:entries.length};
    }finally{
      if(downloadSerial===this.downloadSerial)this.downloading=false;
    }
  }
  close(){
    this.closed=true;
    this.active=false;
    if(this.pendingPaste)clearTimeout(this.pendingPaste.timer);
    this.pendingPaste=null;
    this.resolveRemoteFormatWaiters(false);
    this.cancelPending(new Error("Clipboard session closed"));
    this.localFiles=[];
  }
}
