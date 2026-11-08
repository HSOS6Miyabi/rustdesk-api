"use strict";

const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const vm=require("node:vm");
const typescript=require("typescript");

const source=fs.readFileSync(path.resolve(__dirname,"../src/websock.ts"),"utf8");

function createFixture(){
  let scheduled=0;
  class FakeWebSocket {
    static CONNECTING=0;
    static OPEN=1;
    static CLOSED=3;
    constructor(uri){
      this.uri=uri;
      this.readyState=FakeWebSocket.CONNECTING;
      this.protocol="";
      this.sent=[];
    }
    send(data){this.sent.push(data);}
    close(){this.readyState=FakeWebSocket.CLOSED;this.onclose?.({code:1000});}
    emit(bytes){this.onmessage?.({data:Uint8Array.from(bytes).buffer});}
    accept(){this.readyState=FakeWebSocket.OPEN;this.onopen?.();}
  }
  const fakeRendezvous={
    RendezvousMessage:{
      encode(data){return{finish(){return Uint8Array.from([Number(data?.code||0)]);}};},
      fromPartial(data){return data;},
      decode(bytes){return {code:bytes[0]};}
    }
  };
  const fakeMessage={
    Message:{
      encode(data){return{finish(){return Uint8Array.from([Number(data?.code||0)]);}};},
      fromPartial(data){return data;},
      decode(bytes){return {code:bytes[0]};}
    }
  };
  const transpiled=typescript.transpileModule(source,{
    compilerOptions:{module:typescript.ModuleKind.CommonJS,target:typescript.ScriptTarget.ES2020,esModuleInterop:true}
  }).outputText;
  const module={exports:{}};
  const ctx={
    module,exports:module.exports,window:{ArrayBuffer},
    WebSocket:FakeWebSocket,Uint8Array,ArrayBuffer,Date,Promise,console,
    setTimeout(fn,delay){scheduled++;return setTimeout(fn,delay);},
    clearTimeout,
    require(request){
      if(request==="./message.js")return fakeMessage;
      if(request==="./rendezvous.js")return fakeRendezvous;
      if(request==="./globals")return {encrypt(data){return data},decrypt(data){return data}};
      if(request==="./webrtc_protocol")return{
        decodeRendezvous(bytes){return fakeRendezvous.RendezvousMessage.decode(bytes)},
        encodeOffer(){return new Uint8Array([44])},
        encodeICE(){return new Uint8Array([45])}
      };
      throw Error("Unexpected dependency: "+request);
    }
  };
  vm.runInNewContext(transpiled,ctx,{filename:"websock.ts"});
  const Connection=module.exports.default;
  const open=async()=>{
    const ws=new Connection("wss://desk.test/ws/id");
    const ready=ws.open(1000);
    ws._websocket.accept();
    await ready;
    return ws;
  };
  return {open,scheduled:()=>scheduled};
}

test("event-driven WebSocket messages do not use short-interval polling",async()=>{
  const fixture=createFixture();
  const ws=await fixture.open();
  const pending=ws.next(1000);
  const initial=fixture.scheduled();
  await new Promise(resolve=>setTimeout(resolve,40));
  assert.equal(fixture.scheduled(),initial,"message reception should not schedule repeated polls");
  ws._websocket.emit([42]);
  const reply=await pending;
  assert.equal(reply.code,42);
  ws.close();
});

test("WebSocket processes queued responses in arrival order",async()=>{
  const fixture=createFixture();
  const ws=await fixture.open();
  ws._websocket.emit([1]);
  ws._websocket.emit([2]);
  assert.equal((await ws.next(100)).code,1);
  assert.equal((await ws.next(100)).code,2);
  const pending1=ws.next(200),pending2=ws.next(200);
  ws._websocket.emit([3]);
  ws._websocket.emit([4]);
  assert.equal((await pending1).code,3);
  assert.equal((await pending2).code,4);
  ws.close();
});

test("WebSocket rejects pending receives when connection closes",async()=>{
  const fixture=createFixture();
  const ws=await fixture.open();
  const pending=ws.next(1000);
  ws.close();
  await assert.rejects(pending,/closed/i);
});

test("WebSocket rejects pending receives after the deadline",async()=>{
  const fixture=createFixture();
  const ws=await fixture.open();
  await assert.rejects(ws.next(10),/timed out/i);
  ws._websocket.emit([9]);
  assert.equal((await ws.next(100)).code,9);
  ws.close();
});

test("WebSocket closes when unsolicited messages exceed the queue limit",async()=>{
  const fixture=createFixture();
  const ws=await fixture.open();
  for(let i=0;i<257;i++)ws._websocket.emit([i%250]);
  assert.equal(ws._websocket.readyState,2);
  await assert.rejects(ws.next(100),/not open/i);
});
