"use strict";

const test=require("node:test");
const assert=require("node:assert/strict");
const fs=require("node:fs");
const path=require("node:path");
const Module=require("node:module");
const ts=require("typescript");

const cache=new Map();
function loadTS(filename){
  filename=path.resolve(filename);
  if(cache.has(filename))return cache.get(filename).exports;
  const source=fs.readFileSync(filename,"utf8");
  const js=ts.transpileModule(source,{compilerOptions:{
    module:ts.ModuleKind.CommonJS,target:ts.ScriptTarget.ES2020,esModuleInterop:true
  }}).outputText;
  const loaded=new Module(filename,module);
  loaded.paths=Module._nodeModulePaths(path.dirname(filename));
  cache.set(filename,loaded);
  const originalRequire=loaded.require.bind(loaded);
  loaded.require=(id)=>{
    if(id.startsWith("./")&&id.endsWith(".js")){
      const candidate=path.resolve(path.dirname(filename),id.slice(0,-3)+".ts");
      if(fs.existsSync(candidate))return loadTS(candidate);
    } else if(id.startsWith("./")&&!path.extname(id)){
      const candidate=path.resolve(path.dirname(filename),id+".ts");
      if(fs.existsSync(candidate))return loadTS(candidate);
    }
    return originalRequire(id);
  };
  loaded._compile(js,filename);
  return loaded.exports;
}

const src=path.resolve(__dirname,"../src");
const signaling=loadTS(path.join(src,"webrtc_protocol.ts"));
const rendezvous=loadTS(path.join(src,"rendezvous.ts"));
const messages=loadTS(path.join(src,"message.ts"));
const WebRTCTransport=loadTS(path.join(src,"webrtc_transport.ts")).default;
const protobuf=require("protobufjs/minimal");

test("RustDesk 1.5 WebRTC SDP offer uses the correct protobuf field",()=>{
  const request=rendezvous.PunchHoleRequest.fromPartial({
    id:"desktop-target",nat_type:rendezvous.NatType.SYMMETRIC
  });
  const encoded=signaling.encodeOffer(request,"webrtc://example-base64-sdp");
  const decoded=signaling.decodeRendezvous(encoded);
  assert.equal(decoded.punch_hole_request.id,"desktop-target");
  assert.equal(decoded.punch_hole_request.nat_type,rendezvous.NatType.SYMMETRIC);
  // A modern RustDesk peer reads field 12 directly.
  const reader=protobuf.Reader.create(encoded);
  assert.equal(reader.uint32(),66);
  const nested=protobuf.Reader.create(reader.bytes());
  let offer="";
  while(nested.pos<nested.len){
    const tag=nested.uint32();
    if((tag>>>3)===12)offer=nested.string();
    else nested.skipType(tag&7);
  }
  assert.equal(offer,"webrtc://example-base64-sdp");
});

test("RustDesk ICE candidate payload survives encoding",()=>{
  const original={
    id:"remote-id",socket_addr:new Uint8Array([10,2,1,4]),
    session_key:"sha-256 01:02:03:04",
    candidate:JSON.stringify({
      candidate:"candidate:12345 1 udp 123456 192.0.2.1 4321 typ host",
      sdpMid:"0",sdpMLineIndex:0
    })
  };
  const decoded=signaling.decodeRendezvous(signaling.encodeICE(original)).ice_candidate;
  assert.equal(decoded.id,original.id);
  assert.deepEqual(Array.from(decoded.socket_addr),Array.from(original.socket_addr));
  assert.equal(decoded.session_key,original.session_key);
  assert.equal(decoded.candidate,original.candidate);
  assert.throws(()=>signaling.encodeICE({...original,session_key:""}),/session|limit/i);
  assert.throws(()=>signaling.encodeICE({...original,candidate:"x".repeat(8193)}),/limit/i);
});

test("WebRTC SDP envelope uses the RustDesk base64 format",()=>{
  const offer=signaling.encodeWebRTCEndpoint({
    type:"offer",sdp:"v=0\r\na=fingerprint:sha-256 AA:BB:CC\r\n"
  });
  assert.match(offer,/^webrtc:\/\//);
  const value=JSON.parse(Buffer.from(offer.slice(9),"base64").toString("utf8"));
  assert.equal(value.type,"offer");
  assert.equal(value.ice_policy,"all");
  const answerData=Buffer.from(JSON.stringify({
    type:"answer",sdp:"v=0\r\na=fingerprint:sha-256 DD:EE\r\n"
  })).toString("base64");
  const answer=signaling.decodeWebRTCEndpoint("webrtc://"+answerData);
  assert.equal(answer.type,"answer");
  assert.match(answer.sdp,/DD:EE/);
  assert.throws(()=>signaling.decodeWebRTCEndpoint("webrtc://garbage"),/invalid|unexpected/i);
});

test("WebRTC peer authentication checks the signed DTLS fingerprint",()=>{
  const fingerprint="sha-256 AA:BB:CC:DD";
  const signed=protobuf.Writer.create().uint32(10).string("desktop-target")
    .uint32(26).string(fingerprint).finish();
  assert.equal(signaling.signedPeerFingerprint(signed),fingerprint);
  const remote="v=0\r\na=fingerprint:sha-256 aa:bb:cc:dd\r\n";
  assert.equal(signaling.fingerprintMatches(signed,remote),true);
  assert.equal(signaling.fingerprintMatches(signed,
    "v=0\r\na=fingerprint:sha-256 11:22:33:44\r\n"),false);
  const withoutBinding=protobuf.Writer.create().uint32(10).string("desktop-target").finish();
  assert.equal(signaling.fingerprintMatches(withoutBinding,remote),false);
});

function fakeChannel(){
  const handlers=new Map();
  const channel={
    readyState:"open",bufferedAmount:0,packets:[],
    addEventListener(name,fn){
      if(!handlers.has(name))handlers.set(name,new Set());
      handlers.get(name).add(fn);
    },
    removeEventListener(name,fn){handlers.get(name)?.delete(fn);},
    dispatch(name,data){for(const fn of handlers.get(name)||[])fn(data);},
    send(data){
      const copy=new Uint8Array(data);
      this.packets.push(copy);
      if(this.remote)this.remote.dispatch("message",{data:copy.buffer});
    },
    close(){this.readyState="closed";this.dispatch("close",{});}
  };
  return channel;
}

test("WebRTC transport exchanges RustDesk messages with multi-fragment framing",async()=>{
  const left=fakeChannel(),right=fakeChannel();
  left.remote=right;right.remote=left;
  const transportA=new WebRTCTransport({remoteDescription:{sdp:"v=0"},close(){}},left);
  const transportB=new WebRTCTransport({remoteDescription:{sdp:"v=0"},close(){}},right);
  const content=new Uint8Array(125000);
  for(let i=0;i<content.length;i++)content[i]=i%253;
  const receiving=transportB.next();
  transportA.sendMessage({clipboard:{content,compress:false}});
  const received=await receiving;
  assert.deepEqual(Array.from(received.clipboard.content),Array.from(content));
  assert.ok(left.packets.length>=3);
  assert.ok(left.packets.slice(0,-1).every(x=>x[0]===1));
  assert.equal(left.packets[left.packets.length-1][0],0);
  transportA.close();transportB.close();
});

test("WebRTC transport rejects malformed fragments and closes its connection",async()=>{
  const channel=fakeChannel();
  let closed=false;
  const transport=new WebRTCTransport({remoteDescription:null,close(){closed=true}},channel);
  const waiting=transport.next(200);
  channel.dispatch("message",{data:Uint8Array.from([3,9,4]).buffer});
  await assert.rejects(waiting,/fragment/);
  assert.equal(closed,true);
});
