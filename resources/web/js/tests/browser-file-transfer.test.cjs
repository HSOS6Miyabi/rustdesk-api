"use strict";
const {test}=require("node:test");
const assert=require("node:assert/strict");
const {File}=require("node:buffer");
const {pathToFileURL}=require("node:url");
const path=require("node:path");

let library;
async function load(){
  library ||= await import(pathToFileURL(path.join(__dirname,"../src/browser_file_transfer.mjs")).href);
  return library;
}
function harness(BrowserFileTransfer){
  const sent=[],events=[],saved=[],notes=[];
  const transfer=new BrowserFileTransfer({
    send:message=>sent.push(message), emit:(type,value)=>events.push([type,value]),
    save:async (blob,name)=>saved.push({blob,name}),
    pendingBytes:()=>0, message:note=>notes.push(note),
    decompress:async data=>data
  });
  return {transfer,sent,events,saved,notes};
}
async function settle(){
  await new Promise(resolve=>setTimeout(resolve,20));
}
test("selected local files create Flutter file-selection events and stream bytes after remote approval",async()=>{
  const {BrowserFileTransfer}=await load();
  const {transfer,sent,events}=harness(BrowserFileTransfer);
  const file=new File(["Hello from browser"],"report.txt",{type:"text/plain",lastModified:1794500000000});
  assert.equal(transfer.register([file]),1);
  const selected=events.find(([name])=>name==="selected_files")[1];
  assert.equal(JSON.parse(selected.file).name,"report.txt");
  assert.equal(JSON.parse(selected.file).entry_type,4);
  transfer.startUpload({id:41,handle_index:Number(selected.handleIndex),to:"C:\\Users\\Meili\\report.txt"});
  assert.equal(sent[0].file_action.receive.files[0].name,"");
  assert.equal(sent[0].file_action.receive.total_size,file.size);
  assert.equal(sent[1].file_response.digest.file_size,file.size);
  assert.equal(sent[1].file_response.digest.is_upload,true);
  transfer.confirm({id:41,file_num:0,offset_blk:0});
  await settle();
  assert.deepEqual(sent.filter(x=>x.file_response?.block).map(x=>x.file_response.block.data.length),[file.size]);
  assert.equal(sent.at(-1).file_response.done.id,41);
  assert.ok(events.some(([name,value])=>name==="job_progress"&&value.finished_size===file.size));
  assert.ok(events.some(([name])=>name==="job_done"));
  assert.equal(transfer.jobs.size,0);
});

test("folder selections preserve relative names and zero-length files are transferred",async()=>{
  const {BrowserFileTransfer}=await load();
  const {transfer,sent,events}=harness(BrowserFileTransfer);
  const folderFiles=[
    Object.assign(new File(["abc"],"hello.txt"),{webkitRelativePath:"photos/2026/hello.txt"}),
    Object.assign(new File([],"empty.dat"),{webkitRelativePath:"photos/empty.dat"})
  ];
  assert.equal(transfer.register(folderFiles,true),1);
  const ev=events[0][1];
  assert.equal(JSON.parse(ev.file).entry_type,0);
  assert.equal(JSON.parse(ev.file).name,"photos");
  transfer.startUpload({id:51,handle_index:Number(ev.handleIndex),to:"D:\\Saved\\photos"});
  assert.deepEqual(sent[0].file_action.receive.files.map(x=>x.name),["2026/hello.txt","empty.dat"]);
  transfer.confirm({id:51,file_num:0,offset_blk:0});
  await settle();
  assert.ok(sent.some(x=>x.file_response?.digest?.file_num===1));
  transfer.confirm({id:51,file_num:1,offset_blk:0});
  await settle();
  assert.ok(sent.some(x=>x.file_response?.block?.file_num===1&&x.file_response.block.data.length===0));
  assert.ok(events.some(([name])=>name==="job_done"));
});

test("remote file download requests and confirms the official RustDesk protocol",async()=>{
  const {BrowserFileTransfer}=await load();
  const {transfer,sent,saved,events}=harness(BrowserFileTransfer);
  transfer.startDownload({id:17,path:"C:\\Users\\Meili\\notes.txt"});
  assert.deepEqual(sent[0].file_action.send,{id:17,path:"C:\\Users\\Meili\\notes.txt",file_num:0,include_hidden:false});
  assert.equal(await transfer.handleMessage({file_response:{dir:{id:17,path:"C:\\Users\\Meili\\notes.txt",
    entries:[{entry_type:4,name:"",size:5,modified_time:100}]}}}),true);
  assert.equal(await transfer.handleMessage({file_response:{digest:{id:17,file_num:0,file_size:5}}}),true);
  assert.equal(sent.at(-1).file_action.send_confirm.offset_blk,0);
  await transfer.handleMessage({file_response:{block:{id:17,file_num:0,data:new TextEncoder().encode("hello"),compressed:false}}});
  await transfer.handleMessage({file_response:{done:{id:17,file_num:0}}});
  assert.equal(saved.length,1);
  assert.equal(saved[0].name,"notes.txt");
  assert.equal(await saved[0].blob.text(),"hello");
  assert.ok(events.some(([name])=>name==="job_done"));
});

test("remote directory is downloaded as a valid zip archive without paths escaping its root",async()=>{
  const {BrowserFileTransfer}=await load();
  const {transfer,saved}=harness(BrowserFileTransfer);
  transfer.startDownload({id:99,path:"/home/remote/Documents"});
  await transfer.handleMessage({file_response:{dir:{id:99,path:"Documents",entries:[
    {name:"report.txt",size:3,entry_type:4},{name:"folder/a.txt",size:2,entry_type:4}
  ]}}});
  await transfer.handleMessage({file_response:{digest:{id:99,file_num:0}}});
  await transfer.handleMessage({file_response:{block:{id:99,file_num:0,data:new TextEncoder().encode("one")}}});
  await transfer.handleMessage({file_response:{digest:{id:99,file_num:1}}});
  await transfer.handleMessage({file_response:{block:{id:99,file_num:1,data:new TextEncoder().encode("ok")}}});
  await transfer.handleMessage({file_response:{done:{id:99,file_num:1}}});
  assert.equal(saved[0].name,"Documents.zip");
  const bytes=new Uint8Array(await saved[0].blob.arrayBuffer());
  const text=new TextDecoder().decode(bytes);
  assert.equal(new DataView(bytes.buffer).getUint32(0,true),0x04034b50);
  assert.ok(text.includes("report.txt"));
  assert.ok(text.includes("folder/a.txt"));
  assert.equal(new DataView(bytes.buffer).getUint32(bytes.length-22,true),0x06054b50);
});

test("invalid directory names, premature transfer completion and oversized transfers fail safely",async()=>{
  const {BrowserFileTransfer,packageZip}=await load();
  assert.throws(()=>packageZip([{name:"../secret.txt",chunks:[]}]),/Invalid file/);
  const {transfer,saved,events}=harness(BrowserFileTransfer);
  transfer.startDownload({id:73,path:"/home/remote/data.bin"});
  await transfer.handleMessage({file_response:{dir:{id:73,path:"data.bin",entries:[{name:"",size:500,entry_type:4}]}}});
  await transfer.handleMessage({file_response:{done:{id:73,file_num:0}}});
  assert.equal(saved.length,0);
  assert.ok(events.some(([name])=>name==="job_error"));
  assert.equal(transfer.jobs.size,0);
  transfer.close();
  assert.throws(()=>transfer.startDownload({id:74,path:"file.txt"}),/unavailable/);
});

test("remote upload errors close outstanding confirmation tasks without uploading data",async()=>{
  const {BrowserFileTransfer}=await load();
  const {transfer,sent,events}=harness(BrowserFileTransfer);
  transfer.register([new File(["abc"],"name.txt")]);
  transfer.startUpload({id:91,handle_index:1,to:"C:\\name.txt"});
  await transfer.handleMessage({file_response:{error:{id:91,file_num:0,error:"Permission denied"}}});
  await settle();
  assert.ok(events.some(([name,value])=>name==="job_error"&&value.err==="Permission denied"));
  assert.equal(sent.filter(x=>x.file_response?.block).length,0);
});
