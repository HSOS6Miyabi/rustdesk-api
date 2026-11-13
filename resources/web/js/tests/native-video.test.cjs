"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const path=require("node:path");
const {pathToFileURL}=require("node:url");
const load=()=>import(pathToFileURL(path.join(__dirname,"../src/native_video.mjs")).href);

function createDecoder(){
  const instances=[],paintCallbacks=[],renders=[],acks=[],drops=[];
  class Chunk{
    constructor(config){Object.assign(this,config)}
  }
  class Decoder{
    static async isConfigSupported(config){
      return {supported:["vp09.00.10.08","vp8","avc1.42001E"].includes(config.codec)};
    }
    constructor(callbacks){this.callbacks=callbacks;this.chunks=[];this.decodeQueueSize=0;this.closed=false;instances.push(this)}
    configure(config){this.config=config}
    decode(chunk){this.chunks.push(chunk);this.decodeQueueSize++}
    emit(timestamp){
      this.decodeQueueSize--;
      const frame={timestamp,closed:0,close(){this.closed++}};
      this.callbacks.output(frame);
      return frame;
    }
    close(){this.closed=true}
  }
  const options={
    codec:"vp9",
    config:{codec:"vp09.00.10.08"},
    Decoder,Chunk,
    now:()=>10,
    schedule:callback=>paintCallbacks.push(callback),
    acknowledge:()=>acks.push("ack"),
    requestKeyFrame:()=>{},
    onFailure:error=>{throw error},
    onDropped:()=>drops.push("drop"),
    display:async(frame)=>{renders.push(frame)}
  };
  return {Decoder,Chunk,instances,paintCallbacks,renders,acks,drops,options};
}

test("advertises only codecs supported by the browser decoder",async()=>{
  const {discoverNativeVideoCodecs}=await load();
  const {Decoder,Chunk}=createDecoder();
  const codecs=await discoverNativeVideoCodecs(Decoder,Chunk);
  assert.deepEqual(Object.keys(codecs),["vp9","h264","vp8"]);
  assert.equal(codecs.h264.avc.format,"annexb");
  assert.deepEqual(await discoverNativeVideoCodecs(undefined,undefined).then(x=>Object.keys(x)).catch(()=>[]),[]);
});

test("acknowledges decoded batches before painting and keeps the newest completed image",async()=>{
  const {NativeVideoDecoder}=await load();
  const fixture=createDecoder();
  const decoder=new NativeVideoDecoder(fixture.options);
  const frames=[{key:true,data:Uint8Array.of(1)},{key:false,data:Uint8Array.of(2)}];
  decoder.decode(frames);
  assert.equal(fixture.instances.length,1);
  const first=fixture.instances[0].emit(1);
  assert.equal(fixture.acks.length,0);
  const second=fixture.instances[0].emit(2);
  assert.equal(fixture.acks.length,1);
  assert.equal(first.closed,1,"a superseded decoded frame must be closed");
  assert.equal(second.closed,0);
  assert.equal(fixture.drops.length,1);
  assert.equal(fixture.paintCallbacks.length,1);
  await fixture.paintCallbacks.shift()();
  assert.deepEqual(fixture.renders,[second]);
  assert.equal(second.closed,1);
  decoder.close();
});

test("waits for a key frame, requests a refresh once, and never queues invalid delta frames",async()=>{
  const {NativeVideoDecoder}=await load();
  const fixture=createDecoder();
  let requests=0;
  const decoder=new NativeVideoDecoder({...fixture.options,requestKeyFrame(){requests++}});
  decoder.decode([{key:false,data:Uint8Array.of(5)}]);
  decoder.decode([{key:false,data:Uint8Array.of(6)}]);
  assert.equal(requests,1);
  assert.equal(fixture.acks.length,2);
  assert.equal(fixture.instances.length,0);
  decoder.decode([{key:true,data:Uint8Array.of(7)}]);
  assert.equal(fixture.instances[0].chunks.length,1);
  decoder.close();
  assert.equal(fixture.acks.length,3,"closing must acknowledge in-flight decoding");
});

test("retries software decoding, then reports a persistent failure",async()=>{
  const {NativeVideoDecoder}=await load();
  const fixture=createDecoder();
  const failures=[],requests=[];
  const decoder=new NativeVideoDecoder({...fixture.options,
    onFailure:error=>failures.push(String(error)),
    requestKeyFrame:()=>requests.push(1)
  });
  decoder.decode([{key:true,data:Uint8Array.of(7)}]);
  fixture.instances[0].callbacks.error(new Error("hardware decoder error"));
  assert.equal(fixture.acks.length,1);
  assert.equal(fixture.instances.length,2);
  assert.equal(fixture.instances[1].config.hardwareAcceleration,"prefer-software");
  assert.equal(requests.length,1);
  decoder.decode([{key:false,data:Uint8Array.of(8)}]);
  assert.equal(fixture.acks.length,2);
  decoder.decode([{key:true,data:Uint8Array.of(9)}]);
  fixture.instances[1].callbacks.error(new Error("software decoder error"));
  assert.equal(failures.length,1);
  assert.equal(fixture.acks.length,3);
  decoder.close();
});

test("closing a decoder releases its queued frames and sends pending acknowledgements once",async()=>{
  const {NativeVideoDecoder}=await load();
  const fixture=createDecoder();
  const decoder=new NativeVideoDecoder(fixture.options);
  decoder.decode([{key:true,data:Uint8Array.of(1)}]);
  fixture.instances[0].emit(1);
  decoder.decode([{key:false,data:Uint8Array.of(2)}]);
  decoder.close();
  assert.equal(fixture.acks.length,2);
  assert.equal(fixture.instances[0].closed,true);
  assert.equal(fixture.drops.length,0);
  assert.equal(fixture.paintCallbacks.length,1);
  await fixture.paintCallbacks.shift()();
  assert.equal(fixture.renders.length,0);
});
