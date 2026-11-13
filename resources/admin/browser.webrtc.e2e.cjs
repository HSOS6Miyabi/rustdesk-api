"use strict";

const test=require("node:test");
const assert=require("node:assert/strict");
const http=require("node:http");
const path=require("node:path");
const esbuild=require("../web/js/node_modules/esbuild");
const {chromium}=require("playwright");

function buildBrowserTransport(){
  const sourceRoot=path.resolve(__dirname,"../web/js/src");
  const result=esbuild.buildSync({
    stdin:{
      contents:[
        'import Transport from "./webrtc_transport";',
        'import {fingerprintFromSDP} from "./webrtc_protocol";',
        'export {Transport,fingerprintFromSDP};'
      ].join("\n"),
      resolveDir:sourceRoot,sourcefile:"webrtc-browser-integration.ts",loader:"ts"
    },
    bundle:true,platform:"browser",format:"iife",globalName:"RustDeskTransportTest",
    write:false,logLevel:"error",target:"es2020"
  });
  return result.outputFiles[0].text;
}

let browser,server,origin,bundle;

test.before(async()=>{
  bundle=buildBrowserTransport();
  server=http.createServer((req,res)=>{
    res.writeHead(200,{"Content-Type":"text/html; charset=utf-8","Cache-Control":"no-store"});
    res.end("<!doctype html><html><head><title>WebRTC Integration</title></head><body>WebRTC data channel test</body></html>");
  });
  await new Promise(resolve=>server.listen(0,"127.0.0.1",resolve));
  origin="http://127.0.0.1:"+server.address().port;
  browser=await chromium.launch({
    headless:true,args:["--no-sandbox","--disable-dev-shm-usage"]
  });
});

test.after(async()=>{
  if(browser)await browser.close();
  if(server)await new Promise(resolve=>server.close(resolve));
});

test("Chromium exchanges RustDesk-framed messages over a real local WebRTC connection",async()=>{
  const page=await browser.newPage();
  try{
    await page.goto(origin,{waitUntil:"load"});
    await page.addScriptTag({content:bundle});
    const result=await page.evaluate(async()=>{
      const {Transport,fingerprintFromSDP}=window.RustDeskTransportTest;
      const first=new RTCPeerConnection({iceServers:[]});
      const second=new RTCPeerConnection({iceServers:[]});
      let sender,receiver;
      const result={receivedLength:0,fragmentCount:0,secondMessage:"",fingerprint:false};
      const withTimeout=(promise,ms,label)=>Promise.race([
        promise,new Promise((_,reject)=>setTimeout(()=>reject(new Error(label+" timed out")),ms))
      ]);
      const pendingLeft=[],pendingRight=[];
      const routeCandidate=(source,dest,buffer)=>{
        source.onicecandidate=e=>{
          if(!e.candidate)return;
          if(dest.remoteDescription){
            dest.addIceCandidate(e.candidate).catch(error=>console.warn("ICE candidate:",error));
          }else buffer.push(e.candidate);
        };
      };
      routeCandidate(first,second,pendingRight);
      routeCandidate(second,first,pendingLeft);
      const channelFirst=first.createDataChannel("bootstrap",{ordered:true});
      const channelSecond=new Promise(resolve=>{
        second.ondatachannel=e=>resolve(e.channel);
      });
      try{
        const offer=await first.createOffer();
        await first.setLocalDescription(offer);
        await second.setRemoteDescription(first.localDescription);
        for(const candidate of pendingRight)await second.addIceCandidate(candidate);
        const answer=await second.createAnswer();
        await second.setLocalDescription(answer);
        await first.setRemoteDescription(second.localDescription);
        for(const candidate of pendingLeft)await first.addIceCandidate(candidate);
        const receivedChannel=await withTimeout(channelSecond,15000,"data channel");
        sender=new Transport(first,channelFirst);
        receiver=new Transport(second,receivedChannel);
        await withTimeout(Promise.all([sender.waitOpen(12000),receiver.waitOpen(12000)]),15000,"channel open");
        result.fingerprint=!!fingerprintFromSDP(first.localDescription.sdp)&&
          !!fingerprintFromSDP(second.localDescription.sdp);
        const payload=new Uint8Array(135000);
        for(let i=0;i<payload.length;i++)payload[i]=i%251;
        const receive=receiver.next(10000);
        sender.sendMessage({clipboard:{content:payload,compress:false}});
        const item=await withTimeout(receive,12000,"first RustDesk message");
        if(!item.clipboard||item.clipboard.content.length!==payload.length)
          throw new Error("RustDesk data-channel framing lost bytes");
        for(let i=0;i<payload.length;i++){
          if(item.clipboard.content[i]!==payload[i])throw new Error("RustDesk message bytes differ");
        }
        result.receivedLength=item.clipboard.content.length;
        const pendingIdle=receiver.next();
        await new Promise(resolve=>setTimeout(resolve,200));
        sender.sendMessage({clipboard:{content:new TextEncoder().encode("Idle session remains active"),compress:false}});
        const message=await withTimeout(pendingIdle,3000,"idle RustDesk message");
        result.secondMessage=new TextDecoder().decode(message.clipboard.content);
      }finally{
        sender?.close();receiver?.close();
        first.close();second.close();
      }
      return result;
    });
    assert.equal(result.receivedLength,135000);
    assert.equal(result.secondMessage,"Idle session remains active");
    assert.equal(result.fingerprint,true);
  }finally{await page.close();}
});

test("Chromium converts native VideoFrames directly to RGBA for the Flutter image",async()=>{
  const page=await browser.newPage();
  try{
    await page.goto(origin,{waitUntil:"load"});
    const result=await page.evaluate(async()=>{
      const canvas=document.createElement("canvas");
      canvas.width=4;canvas.height=4;
      const ctx=canvas.getContext("2d");
      ctx.fillStyle="rgb(32, 64, 128)";
      ctx.fillRect(0,0,4,4);
      const frame=new VideoFrame(canvas,{timestamp:0});
      const bytes=new Uint8Array(4*4*4);
      try{
        await frame.copyTo(bytes,{format:"RGBA",layout:[{offset:0,stride:16}]});
        return {width:frame.displayWidth,height:frame.displayHeight,
          rgba:[...bytes.subarray(0,4)]};
      }finally{frame.close()}
    });
    assert.equal(result.width,4);
    assert.equal(result.height,4);
    assert.deepEqual(result.rgba,[32,64,128,255]);
  }finally{await page.close()}
});
