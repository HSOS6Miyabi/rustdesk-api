"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const {pathToFileURL}=require("node:url");
const path=require("node:path");
const load=()=>import(pathToFileURL(path.join(__dirname,"../src/video_pacing.mjs")).href);
test("video acknowledgement precedes synchronous display work",async()=>{
  const {decodeVp9Frames}=await load(),order=[],pending=[];
  let current=0;
  const decoder={frameBuffer:{width:1920},processFrame(bytes,cb){
    assert.equal(bytes.byteLength,2);
    pending.push(cb);
  }};
  const frames=[{data:Uint8Array.of(1,2)},{data:Uint8Array.of(3,4)}];
  decodeVp9Frames(frames,decoder,
    ()=>order.push("ack"),
    (frame,decodeMs,encodedBytes)=>{
      order.push("paint");
      assert.equal(frame.width,1920);
      assert.equal(encodedBytes,4);
      assert.equal(decodeMs,30);
    },()=>[100,130][current++]);
  pending[0](true);
  assert.deepEqual(order,[]);
  pending[1](true);
  assert.deepEqual(order,["ack","paint"]);
});
test("no decoder still acknowledges an empty video batch",async()=>{
  const {decodeVp9Frames}=await load();
  let ack=0;
  decodeVp9Frames([],null,()=>ack++,()=>{throw Error("must not paint")});
  assert.equal(ack,1);
});
test("failed final decode does not paint but acknowledges the batch",async()=>{
  const {decodeVp9Frames}=await load();
  const callbacks=[],order=[];
  decodeVp9Frames([{data:Uint8Array.of(2)}],{
    frameBuffer:{},
    processFrame(_bytes,callback){callbacks.push(callback)}
  },()=>order.push("ack"),()=>order.push("paint"));
  callbacks[0](false);
  assert.deepEqual(order,["ack"]);
});
