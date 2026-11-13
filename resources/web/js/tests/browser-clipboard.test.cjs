"use strict";
const {test}=require("node:test");
const assert=require("node:assert/strict");
const {pathToFileURL}=require("node:url");
const path=require("node:path");

let klass;
async function load(){
  klass ||= (await import(pathToFileURL(path.join(__dirname,"../src/browser_clipboard.mjs")).href)).BrowserClipboardSync;
  return klass;
}

test("clipboard sync is disabled until explicit authorization",async()=>{
  const C=await load(),sent=[],received=[];
  const bridge=new C({send:text=>sent.push(text),receive:text=>received.push(text)});
  assert.equal(bridge.sendLocal("hello"),false);
  assert.equal(bridge.fromRemote("peer"),false);
  bridge.setEnabled(true,"local");
  assert.deepEqual(sent,["local"]);
  assert.equal(bridge.fromRemote("remote"),true);
  assert.deepEqual(received,["remote"]);
  assert.equal(bridge.sendLocal("remote"),false);
  assert.equal(bridge.fromRemote("local"),false);
  assert.equal(bridge.sendLocal("changed"),true);
  bridge.setEnabled(false);
  assert.equal(bridge.fromRemote("remote text"),false);
  assert.equal(bridge.sendLocal("local text"),false);
});

test("clipboard rejects oversized content and prevents redundant remote transfers",async()=>{
  const C=await load(),sent=[],received=[];
  const bridge=new C({send:text=>sent.push(text),receive:text=>received.push(text)});
  bridge.setEnabled(true);
  assert.equal(bridge.sendLocal("x".repeat(1024*1024+1)),false);
  assert.equal(bridge.fromRemote("x".repeat(1024*1024+1)),false);
  assert.equal(bridge.sendLocal("pasted"),true);
  assert.equal(bridge.sendLocal("pasted"),false);
  assert.equal(bridge.fromRemote("remote"),true);
  assert.equal(bridge.fromRemote("remote"),false);
  assert.deepEqual(sent,["pasted"]);
  assert.deepEqual(received,["remote"]);
});

test("clipboard sending can be refused when the remote session disallows it",async()=>{
  const C=await load(),sent=[];
  let allowed=false;
  const bridge=new C({send:text=>{if(!allowed)return false;sent.push(text);},receive:()=>{}});
  bridge.setEnabled(true);
  assert.equal(bridge.sendLocal("unavailable"),false);
  allowed=true;
  assert.equal(bridge.sendLocal("unavailable"),true);
  assert.deepEqual(sent,["unavailable"]);
});
