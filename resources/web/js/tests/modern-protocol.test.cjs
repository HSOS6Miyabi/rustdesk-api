"use strict";
const {test}=require("node:test");
const assert=require("node:assert/strict");
const {pathToFileURL}=require("node:url");
const path=require("node:path");
let lib;
test("Modern RustDesk session protocol appends camera and terminal login types",async()=>{
  lib=await import(pathToFileURL(path.join(__dirname,"../src/modern_protocol.mjs")).href);
  const original=Uint8Array.from([58,2,10,0]);
  assert.deepEqual([...lib.addSessionMode(original,"camera")],[58,4,10,0,122,0]);
  assert.deepEqual([...lib.addSessionMode(original,"terminal")],[58,5,10,0,130,1,0]);
  assert.deepEqual([...lib.addSessionMode(original,"desktop")],[...original]);
  assert.throws(()=>lib.addSessionMode(new Uint8Array([1,2,3]),"terminal"));
});
test("Terminal action encoding follows the official RustDesk 1.5.3 field assignments",()=>{
  assert.deepEqual([...lib.encodeTerminalMessage({open:{terminal_id:3,rows:24,cols:80}})],
    [250,1,8,10,6,8,3,16,24,24,80]);
  const data=lib.encodeTerminalMessage({data:{terminal_id:7,data:new TextEncoder().encode("ls\n")}});
  assert.equal(data[0],250);
  assert.equal(data[1],1);
  assert.ok(data.includes(108));
});
test("Terminal response decoding preserves messages and input bytes",()=>{
  for(const response of [
    {type:"opened",terminal_id:2,success:true,message:"ready",service_id:"sample"},
    {type:"data",terminal_id:2,data:new TextEncoder().encode("hello\n")},
    {type:"closed",terminal_id:2,exit_code:0},
    {type:"error",terminal_id:2,message:"permission denied"}
  ]){
    const decoded=lib.decodeTerminalMessage(lib.encodeTerminalResponseForTest(response));
    assert.equal(decoded.type,response.type);
    assert.equal(decoded.terminal_id,response.terminal_id);
    for(const key of Object.keys(response).filter(x=>x!=="terminal_id"&&x!=="type")){
      if(key==="data")assert.deepEqual(decoded.data,response.data);
      else assert.equal(decoded[key],response[key]);
    }
  }
  assert.equal(lib.decodeTerminalMessage(new Uint8Array([58,0])),null);
});
test("Invalid protocol payloads are rejected before they can be processed",()=>{
  assert.throws(()=>lib.decodeTerminalMessage(Uint8Array.from([130,2,255])));
  assert.throws(()=>lib.encodeTerminalMessage({open:{terminal_id:-1,rows:30,cols:120}}));
});
