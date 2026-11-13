"use strict";

const MAX_ENTRIES = 4096;
const MAX_DOWNLOAD_BYTES = 512 * 1024 * 1024;
const BLOCK_BYTES = 64 * 1024;
const decoder = new TextDecoder();
const encoder = new TextEncoder();

function validRelativePath(path) {
  const text = String(path || "").replace(/\\/g, "/");
  const parts = text.split("/");
  if (!text || text.startsWith("/") || parts.some(part =>
    !part || part === "." || part === ".." || /[\x00-\x1f]/.test(part))) {
    throw new Error("Invalid file or directory name");
  }
  return text;
}

function baseName(path) {
  const pieces = String(path || "").replace(/\\/g, "/").split("/").filter(Boolean);
  return pieces[pieces.length - 1] || "download";
}

function little16(num) {
  return Uint8Array.of(num & 255, (num >>> 8) & 255);
}
function little32(num) {
  return Uint8Array.of(num & 255, (num >>> 8) & 255, (num >>> 16) & 255, (num >>> 24) & 255);
}
function binary(...parts) {
  const size = parts.reduce((total, part) => total + part.byteLength, 0);
  const result = new Uint8Array(size);
  let offset = 0;
  for (const part of parts) {
    result.set(part, offset);
    offset += part.byteLength;
  }
  return result;
}

const crcTable = (() => {
  const table = new Uint32Array(256);
  for (let n = 0; n < 256; n++) {
    let value = n;
    for (let i = 0; i < 8; i++) value = (value & 1) ? 0xedb88320 ^ (value >>> 1) : value >>> 1;
    table[n] = value >>> 0;
  }
  return table;
})();
function crc32(chunks) {
  let result = 0xffffffff;
  for (const chunk of chunks) {
    for (const byte of chunk) result = crcTable[(result ^ byte) & 255] ^ (result >>> 8);
  }
  return (result ^ 0xffffffff) >>> 0;
}

export function packageZip(entries) {
  const parts = [], directory = [];
  let offset = 0;
  for (const entry of entries) {
    const name = encoder.encode(validRelativePath(entry.name));
    if (name.length > 65535) throw new Error("File name is too long");
    const chunks = entry.chunks;
    const length = chunks.reduce((total, chunk) => total + chunk.byteLength, 0);
    const crc = crc32(chunks);
    const header = binary(little32(0x04034b50), little16(20), little16(0x0800),
      little16(0), little16(0), little16(0), little32(crc), little32(length),
      little32(length), little16(name.length), little16(0), name);
    parts.push(header, ...chunks);
    directory.push(binary(little32(0x02014b50), little16(20), little16(20),
      little16(0x0800), little16(0), little16(0), little16(0), little32(crc),
      little32(length), little32(length), little16(name.length), little16(0),
      little16(0), little16(0), little16(0), little32(0), little32(offset), name));
    offset += header.length + length;
  }
  const dirSize = directory.reduce((total, part) => total + part.length, 0);
  parts.push(...directory, binary(little32(0x06054b50), little16(0), little16(0),
    little16(entries.length), little16(entries.length), little32(dirSize),
    little32(offset), little16(0)));
  return new Blob(parts, {type:"application/zip"});
}

export class BrowserFileTransfer {
  constructor({send, emit, save, pendingBytes, message, decompress}) {
    this.send = send;
    this.decompress=decompress || (async data=>data);
    this.emit = emit;
    this.save = save || this.downloadBlob.bind(this);
    this.pendingBytes = pendingBytes || (()=>0);
    this.message = message || (()=>{});
    this.handles = new Map();
    this.jobs = new Map();
    this.nextHandle = 1;
    this.closed = false;
  }

  register(files, folder = false) {
    if (this.closed) return 0;
    const chosen = Array.from(files || []).filter(file => file && typeof file.slice === "function");
    if (!chosen.length) return 0;
    if (chosen.length > MAX_ENTRIES) throw new Error("Too many files in one transfer");
    const groups = new Map();
    for (const file of chosen) {
      const relative = folder && file.webkitRelativePath ? validRelativePath(file.webkitRelativePath) :
        validRelativePath(file.name);
      const segments = relative.split("/");
      const root = folder && segments.length > 1 ? segments[0] : relative;
      const label = folder && segments.length > 1 ? root : file.name;
      const key = (folder && segments.length > 1 ? "folder:" : "file:") + label;
      if (!groups.has(key)) groups.set(key, {name:label, folder:folder && segments.length > 1, files:[]});
      groups.get(key).files.push({file, relative:folder && segments.length > 1 ?
        validRelativePath(segments.slice(1).join("/")) : ""});
    }
    for (const selection of groups.values()) {
      if (this.handles.size >= MAX_ENTRIES) this.handles.delete(this.handles.keys().next().value);
      const id = this.nextHandle++;
      this.handles.set(id, selection);
      const total = selection.files.reduce((n, item) => n + item.file.size, 0);
      this.emit("selected_files", {
        handleIndex:String(id),
        file:JSON.stringify({entry_type:selection.folder?0:4,
          name:selection.name,size:total,modified_time:0})
      });
    }
    return groups.size;
  }

  startUpload(request) {
    const id = Number(request.id), handle = Number(request.handle_index);
    const selected = this.handles.get(handle);
    if (!selected || !Number.isSafeInteger(id) || id <= 0 ||
        this.jobs.has(id) || this.closed) throw new Error("File upload is unavailable");
    if (this.jobs.size >= 8) throw new Error("Too many concurrent file transfers");
    const destination = String(request.to || "");
    if (!destination || destination.length > 4096) throw new Error("Invalid destination path");
    const files = selected.files.map(({file,relative})=>({
      file,
      path:selected.folder ? relative : "",
      entry_type:4, size:file.size, modified_time:Math.floor(file.lastModified / 1000) || 0
    }));
    if (files.length > MAX_ENTRIES) throw new Error("Too many uploaded files");
    const job = {id, direction:"upload", destination, files, fileIndex:0,
      sent:0, total:files.reduce((v,f)=>v+f.size,0), waiting:null, timer:null, cancelled:false};
    this.jobs.set(id,job);
    this.handles.delete(handle);
    this.emit("update_folder_files",{info:JSON.stringify({
      id,num_entries:files.length,total_size:job.total
    })});
    this.send({file_action:{receive:{
      id, path:destination,
      files:files.map(x=>({entry_type:4,name:x.path,is_hidden:false,
        size:x.size,modified_time:x.modified_time})),
      file_num:0, total_size:job.total
    }}});
    this.runUpload(job).catch(error=>this.fail(job,error));
  }

  async runUpload(job) {
    for (let index = 0; index < job.files.length; index++) {
      if (job.cancelled || this.closed) return;
      const entry=job.files[index];
      job.fileIndex=index;
      const confirmed=new Promise((resolve,reject)=>{
        job.waiting={resolve,reject,index};
        job.timer=setTimeout(()=>reject(new Error("Remote file confirmation timed out")),30000);
      });
      this.send({file_response:{digest:{id:job.id,file_num:index,last_modified:entry.modified_time,
        file_size:entry.size,is_upload:true}}});
      const reply=await confirmed;
      clearTimeout(job.timer);
      job.timer=null;
      job.waiting=null;
      if (reply.skip) {
        job.sent+=entry.size;
        continue;
      }
      const start=Math.min(entry.size, Math.max(0, Number(reply.offset_blk||0))*BLOCK_BYTES);
      if(entry.size===0){
        this.send({file_response:{block:{id:job.id,file_num:index,
          data:new Uint8Array(),compressed:false,blk_id:0}}});
      }
      for (let offset=start,blk=0;offset<entry.size;offset+=BLOCK_BYTES,blk++) {
        if (job.cancelled || this.closed) return;
        const slice=entry.file.slice(offset,Math.min(offset+BLOCK_BYTES,entry.size));
        const bytes=new Uint8Array(await slice.arrayBuffer());
        while (this.pendingBytes() > 256 * 1024 && !this.closed && !job.cancelled) {
          await new Promise(done=>setTimeout(done,16));
        }
        if (this.closed || job.cancelled) return;
        this.send({file_response:{block:{id:job.id,file_num:index,data:bytes,
          compressed:false,blk_id:blk}}});
        job.sent+=bytes.length;
        if(blk%8===0)this.progress(job,index);
        await new Promise(done=>setTimeout(done,0));
      }
    }
    if (job.cancelled || this.closed) return;
    this.send({file_response:{done:{id:job.id,file_num:job.files.length-1}}});
    this.progress(job,job.files.length-1);
    this.emit("job_done",{id:job.id,file_num:job.files.length-1,speed:0});
    this.jobs.delete(job.id);
    this.message("文件已发送至远程设备");
  }

  confirm(message) {
    const job=this.jobs.get(message?.id);
    if (!job || job.direction!=="upload" || !job.waiting ||
        job.waiting.index !== message.file_num) return;
    job.waiting.resolve(message);
  }

  startDownload(request) {
    const id=Number(request.id);
    if (!Number.isSafeInteger(id) || id<=0 || this.jobs.has(id) || this.closed)
      throw new Error("File download is unavailable");
    const remote=String(request.path || "");
    if(!remote || remote.length>4096)throw new Error("Invalid remote file path");
    this.jobs.set(id,{id,direction:"download",remote,name:baseName(remote),entries:[],
      total:0, received:0, cancelled:false});
    this.send({file_action:{send:{id,path:remote,file_num:Number(request.file_num||0),
      include_hidden:!!request.include_hidden}}});
  }

  async receive(message) {
    const response=message?.file_response;
    if (!response) return false;
    const result=response.dir||response.digest||response.block||response.done||response.error;
    const job=result&&this.jobs.get(result.id);
    if(!job || job.direction!=="download")return false;
    if(response.dir){
      if(response.dir.entries.length>MAX_ENTRIES)throw new Error("Too many files in download");
      job.entries=response.dir.entries.map((entry,index)=>({
        name:entry.name ? validRelativePath(entry.name):job.name,
        size:entry.size,modified_time:entry.modified_time,chunks:[],received:0
      }));
      job.total=job.entries.reduce((value,item)=>value+item.size,0);
      if(job.total>MAX_DOWNLOAD_BYTES)throw new Error("Download exceeds browser file size limit");
      return true;
    }
    if(response.digest){
      const index=result.file_num;
      if(!job.entries[index])throw new Error("Unexpected downloaded file");
      this.send({file_action:{send_confirm:{id:job.id,file_num:index,offset_blk:0}}});
      return true;
    }
    if(response.block){
      const index=result.file_num;
      if(!job.entries[index])throw new Error("Unexpected downloaded block");
      const encoded=new Uint8Array(result.data);
      if(encoded.length>BLOCK_BYTES * 4)throw new Error("Invalid compressed file block");
      const bytes=result.compressed?await this.decompress(encoded):encoded;
      if(!bytes || bytes.length>BLOCK_BYTES * 2 ||
         job.received+bytes.length>MAX_DOWNLOAD_BYTES)throw new Error("Invalid download payload");
      job.entries[index].chunks.push(bytes);
      job.entries[index].received+=bytes.length;
      job.received+=bytes.length;
      this.progress(job,index);
      return true;
    }
    if(response.error){
      throw new Error(result.error||"Remote file transfer failed");
    }
    if(response.done){
      if(job.entries.some(entry=>entry.received!==entry.size)){
        throw new Error("The remote file transfer ended before all bytes arrived");
      }
      const blob=job.entries.length===1 && job.entries[0].name===job.name ?
        new Blob(job.entries[0].chunks,{type:"application/octet-stream"}) :
        packageZip(job.entries);
      await this.save(blob,job.entries.length===1?job.name:job.name+".zip");
      this.jobs.delete(job.id);
      this.emit("job_done",{id:job.id,file_num:result.file_num,speed:0});
      this.message("文件已下载到浏览器");
      return true;
    }
    return false;
  }

  async downloadBlob(blob, name) {
    const href=URL.createObjectURL(blob);
    const anchor=document.createElement("a");
    anchor.href=href;
    anchor.download=name;
    anchor.style.display="none";
    document.body.append(anchor);
    anchor.click();
    anchor.remove();
    setTimeout(()=>URL.revokeObjectURL(href),60000);
  }

  progress(job,fileIndex) {
    this.emit("job_progress",{id:job.id,file_num:fileIndex,
      speed:0,finished_size:job.direction==="upload"?job.sent:job.received});
  }

  cancel(id) {
    const job=this.jobs.get(Number(id));
    if(job){
      job.cancelled=true;
      clearTimeout(job.timer);
      job.waiting?.reject(new Error("File transfer cancelled"));
      this.jobs.delete(Number(id));
    }
  }

  fail(job,error) {
    if(job.cancelled || this.closed)return;
    this.jobs.delete(job.id);
    clearTimeout(job.timer);
    this.emit("job_error",{id:job.id,file_num:job.fileIndex||0,err:String(error?.message||error)});
    this.message("文件传输失败："+String(error?.message||error));
    try{this.send({file_action:{cancel:{id:job.id}}})}catch{}
  }

  async handleMessage(message) {
    if(message?.file_action?.send_confirm){
      this.confirm(message.file_action.send_confirm);
      return true;
    }
    const response=message?.file_response;
    if(!response)return false;
    const id=response.dir?.id??response.digest?.id??response.block?.id??
      response.done?.id??response.error?.id;
    const job=this.jobs.get(id);
    if(!job)return false;
    if(job.direction==="upload"&&response.error){
      this.fail(job,new Error(response.error.error||"Remote upload failed"));
      return true;
    }
    try{return await this.receive(message)}
    catch(error){this.fail(job,error);return true}
  }

  close() {
    this.closed=true;
    for(const id of [...this.jobs.keys()])this.cancel(id);
    this.handles.clear();
  }
}
