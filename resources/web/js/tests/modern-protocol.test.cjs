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

test("Administrator terminal login includes temporary OS credentials on the wire",()=>{
  const login=Uint8Array.from([58,2,10,0]);
  const encoded=lib.addSessionMode(login,"terminal",{username:"admin",password:"temporary"});
  assert.ok(encoded.includes(98)); // LoginRequest field 12: OSLogin
  assert.ok(encoded.includes(130)); // LoginRequest field 16: Terminal
  const asText=Array.from(encoded).map(x=>String.fromCharCode(x)).join("");
  assert.ok(asText.includes("admin"));
  assert.ok(asText.includes("temporary"));
});

test("browser desktop login advertises a VP9 decoder while preserving login options",()=>{
  const login=Uint8Array.from([58,6,10,0,50,2,8,3]);
  const encoded=lib.advertiseVP9Decoder(login);
  assert.deepEqual([...encoded],[58,12,10,0,50,8,8,3,82,4,8,1,32,1]);
  assert.deepEqual([...lib.advertiseVP9Decoder(Uint8Array.from([58,2,10,0]))],
    [58,10,10,0,50,6,82,4,8,1,32,1]);
  assert.deepEqual([...lib.advertiseVP9Decoder(encoded)],[...encoded]);
  assert.deepEqual([...lib.advertiseVP9Decoder(Uint8Array.from([250,1,0]))],[250,1,0]);
});

test("standalone peer metadata is extracted from the modern RustDesk message",()=>{
  assert.deepEqual([...lib.extractStandalonePeerInfo(Uint8Array.from([202,1,3,10,1,65]))],[10,1,65]);
  assert.equal(lib.extractStandalonePeerInfo(Uint8Array.from([58,2,10,0])),null);
  assert.throws(()=>lib.extractStandalonePeerInfo(Uint8Array.from([202,1,5,10])));
});
