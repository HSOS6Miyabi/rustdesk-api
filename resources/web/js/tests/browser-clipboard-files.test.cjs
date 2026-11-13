"use strict";
const {test}=require("node:test");
const assert=require("node:assert/strict");
const {File}=require("node:buffer");
const {pathToFileURL}=require("node:url");
const path=require("node:path");

let fileClipboard;
async function lib(){
  fileClipboard ||= await import(pathToFileURL(path.join(__dirname,"../src/browser_clipboard_files.mjs")).href);
  return fileClipboard;
}
function mock(BrowserFileClipboard){
  const sent=[],notes=[],saved=[];
  const clipboard=new BrowserFileClipboard({
    send:msg=>sent.push(msg),
    save:async (blob,name)=>saved.push({blob,name}),
    notify:(event,value)=>notes.push({event,value})
  });
  return {clipboard,sent,notes,saved};
}
async function next(){await new Promise(resolve=>setImmediate(resolve));}
test("file cliprdr browser advertises descriptors and serves content and size requests",async()=>{
  const {BrowserFileClipboard}=await lib();
  const {clipboard,sent}=mock(BrowserFileClipboard);
  clipboard.enable();
  const data=new File([new TextEncoder().encode("contents")],"report.txt",{lastModified:1700000000000});
  clipboard.offerFiles([data]);
  assert.ok(sent.some(item=>item.cliprdr.ready));
  assert.equal(sent.at(-1).cliprdr.format_list.formats[0].format,"FileGroupDescriptorW");
  await clipboard.handle({format_data_request:{requested_format_id:49334}});
  const descriptor=sent.at(-1).cliprdr.format_data_response;
  assert.equal(descriptor.msg_flags,1);
  const files=(await lib()).parseClipboardDescriptors(descriptor.format_data);
  assert.deepEqual(files.map(f=>[f.name,f.size]),[["report.txt",8]]);
  await clipboard.handle({file_contents_request:{stream_id:4,list_index:0,dw_flags:1}});
  assert.equal(new DataView(sent.at(-1).cliprdr.file_contents_response.requested_data.buffer).getBigUint64(0,true),8n);
  await clipboard.handle({file_contents_request:{stream_id:5,list_index:0,dw_flags:2,n_position_low:3,n_position_high:0,cb_requested:4}});
  assert.equal(new TextDecoder().decode(sent.at(-1).cliprdr.file_contents_response.requested_data),"tent");
  clipboard.close();
});
test("browser downloads real remote clipboard file bytes with chunked Windows Cliprdr requests",async()=>{
  const {BrowserFileClipboard}=await lib();
  let source,target;
  const saved=[],received=[];
  source=new BrowserFileClipboard({
    send:msg=>queueMicrotask(()=>target.handle(msg.cliprdr)),
    notify:()=>{}
  });
  target=new BrowserFileClipboard({
    send:msg=>queueMicrotask(()=>source.handle(msg.cliprdr)),
    save:async (blob,name)=>saved.push({blob,name}),
    notify:(event,value)=>received.push([event,value])
  });
  source.enable();target.enable();
  source.offerFiles([new File(["Clipboard bytes"],"notes.txt")]);
  await next();await next();
  const result=await target.download();
  assert.equal(result.name,"notes.txt");
  assert.equal(saved.length,1);
  assert.equal(saved[0].name,"notes.txt");
  assert.equal(await saved[0].blob.text(),"Clipboard bytes");
  assert.ok(received.some(([type])=>type==="downloaded"));
  source.close();target.close();
});
test("multiple clipboard files produce a readable zip with preserved relative names",async()=>{
  const {BrowserFileClipboard}=await lib();
  let source,target;
  const saved=[];
  source=new BrowserFileClipboard({send:msg=>queueMicrotask(()=>target.handle(msg.cliprdr))});
  target=new BrowserFileClipboard({send:msg=>queueMicrotask(()=>source.handle(msg.cliprdr)),
    save:async(blob,name)=>saved.push({blob,name})});
  source.enable();target.enable();
  const files=[new File(["a"],"a.txt"),new File(["b"],"b.txt")];
  source.offerFiles(files);
  await next();await next();
  await target.download();
  assert.equal(saved[0].name,"rustdesk-clipboard-files.zip");
  const bytes=new Uint8Array(await saved[0].blob.arrayBuffer());
  assert.equal(new DataView(bytes.buffer).getUint32(0,true),0x04034b50);
  assert.ok(new TextDecoder().decode(bytes).includes("a.txt"));
  assert.ok(new TextDecoder().decode(bytes).includes("b.txt"));
  source.close();target.close();
});
test("invalid file descriptors and oversized clipboard files cannot be downloaded",async()=>{
  const {BrowserFileClipboard,parseClipboardDescriptors}=await lib();
  assert.throws(()=>parseClipboardDescriptors(new Uint8Array(2)),/Missing clipboard/);
  const bad=new Uint8Array(4+592);
  new DataView(bad.buffer).setUint32(0,1,true);
  bad[4+72]=".".charCodeAt(0);bad[4+74]=".".charCodeAt(0);
  assert.throws(()=>parseClipboardDescriptors(bad),/Invalid clipboard/);
  const {clipboard}=mock(BrowserFileClipboard);
  clipboard.enable();
  assert.throws(()=>clipboard.offerFiles([{name:"bad",size:268435457,slice(){}}]),/memory limit/);
  clipboard.close();
});
test("clipboard file download requires explicit enablement and a remote format list",async()=>{
  const {BrowserFileClipboard}=await lib();
  const {clipboard}=mock(BrowserFileClipboard);
  await assert.rejects(clipboard.download(),/请先启用/);
  clipboard.enable();
  await assert.rejects(clipboard.download(),/远程剪贴板/);
  clipboard.disable();
  await assert.rejects(clipboard.download(),/请先启用/);
  clipboard.close();
});
