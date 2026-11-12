"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const vm=require("node:vm");
const ts=require("typescript");

const source=fs.readFileSync(path.join(__dirname,"../src/connection.ts"),"utf8");
const globalsSource=fs.readFileSync(path.join(__dirname,"../src/globals.js"),"utf8");

function harness(){
  const memory=new Map();
  const localStorage={
    getItem(key){return memory.has(key)?memory.get(key):null;},
    setItem(key,value){memory.set(key,String(value));},
    removeItem(key){memory.delete(key);}
  };
  const peers=()=>JSON.parse(localStorage.getItem("peers")||"{}");
  const globals={msgbox(){},draw(){},pushEvent(){},getPeers:peers};
  const js=ts.transpileModule(source,{compilerOptions:{
    module:ts.ModuleKind.CommonJS,
    target:ts.ScriptTarget.ES2020,
    esModuleInterop:true
  }}).outputText;
  const mod={exports:{}};
  const requireStub=name=>name==="./globals"?globals:
    name==="./websock"||name==="./webrtc_transport"?{default:class{}}:{};
  vm.runInNewContext(js,{
    module:mod,exports:mod.exports,require:requireStub,localStorage,
    location:{protocol:"https:"},console,Uint8Array,Date,JSON,Number,
    setTimeout,clearTimeout
  },{filename:"connection.js"});
  return {Connection:mod.exports.default,localStorage,peers};
}

async function restorePassword(Connection,id){
  const connection=new Connection();
  connection.loadVideoDecoder=()=>{throw Error("credential test stop");};
  await assert.rejects(async()=>connection._start(id),/credential test stop/);
  return connection._password?Array.from(connection._password):null;
}

test("Remembered RustDesk credentials survive a new session for the same device and server",async()=>{
  const {Connection,localStorage,peers}=harness();
  localStorage.setItem("custom-rendezvous-server","desk.example.test:21116");
  const connection=new Connection();
  connection._id="peer-a";
  connection._options={remember:true};
  connection._password=Uint8Array.from({length:32},(_,i)=>i+1);
  connection.handlePeerInfo({displays:[{}],username:""});
  const key=connection.credentialStorageKey();
  assert.deepEqual(JSON.parse(localStorage.getItem(key)),Array.from(connection._password));
  assert.equal(peers()["peer-a"].password,undefined);
  const restored=await restorePassword(Connection,"peer-a");
  assert.deepEqual(restored,Array.from(connection._password));
});

test("Remembered passwords are isolated by server and device and can be cleared",async()=>{
  const {Connection,localStorage,peers}=harness();
  localStorage.setItem("custom-rendezvous-server","first.example.test");
  const connection=new Connection();
  connection._id="peer-a";
  connection._options={remember:true};
  connection._password=new Uint8Array(32).fill(45);
  connection.handlePeerInfo({displays:[{}],username:""});
  const credentialKey=connection.credentialStorageKey();
  assert.equal(await restorePassword(Connection,"peer-b"),null);
  localStorage.setItem("custom-rendezvous-server","second.example.test");
  assert.equal(await restorePassword(Connection,"peer-a"),null);
  localStorage.setItem("custom-rendezvous-server","first.example.test");
  connection.setRemember(false);
  assert.equal(localStorage.getItem(credentialKey),null);
  assert.equal(connection._password,undefined);
  assert.equal(peers()["peer-a"].remember,false);
  assert.equal(await restorePassword(Connection,"peer-a"),null);
});

test("Browser password bridge follows the checkbox and does not log credential payloads",()=>{
  assert.match(globalsSource,/case 'login':[\\s\\S]*?setRemember\\(request\\.remember === true/);
  assert.match(globalsSource,/case 'session_login':[\\s\\S]*?setRemember\\(request\\.remember === true/);
  assert.match(globalsSource,/case 'forget_password'/);
  assert.doesNotMatch(globalsSource,/myconsole\\('setByName', name, value\\)/);
});
