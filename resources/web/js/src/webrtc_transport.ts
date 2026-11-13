import * as message from "./message.js";
import {addSessionMode,advertiseVideoDecoders,extractStandalonePeerInfo,decodeTerminalMessage,encodeTerminalMessage} from "./modern_protocol.mjs";

const MAX_FRAGMENT_PAYLOAD = 60000;
const MAX_LOGICAL_MESSAGE = 16 * 1024 * 1024;
const MAX_BUFFERED_BYTES = 4 * 1024 * 1024;

type WaitingReader = {
  resolve: (value: message.Message) => void;
  reject: (reason: Error) => void;
  timeout: ReturnType<typeof setTimeout> | undefined;
};

// RustDesk's WebRTC stream uses one header byte before each ordered, reliable
// data-channel fragment: 1 means more fragments; 0 completes the message.
export default class WebRTCTransport {
  private pc: RTCPeerConnection;
  private channel: RTCDataChannel;
  private frames: message.Message[] = [];
  private waiting: WaitingReader[] = [];
  private fragments: Uint8Array[] = [];
  private fragmentBytes = 0;
  private closed = false;

  constructor(pc: RTCPeerConnection, channel: RTCDataChannel) {
    this.pc = pc;
    this.channel = channel;
    channel.binaryType = "arraybuffer";
    channel.addEventListener("message", evt => this.receive(evt));
    channel.addEventListener("close", () => this.fail(new Error("WebRTC data channel closed")));
    channel.addEventListener("error", () => this.fail(new Error("WebRTC data channel error")));
  }

  public remoteSDP(): string {
    return this.pc.remoteDescription?.sdp || "";
  }

  public waitOpen(timeoutMs=12000): Promise<void> {
    if (this.closed) return Promise.reject(new Error("WebRTC channel already closed"));
    if (this.channel.readyState === "open") return Promise.resolve();
    return new Promise((resolve,reject)=>{
      let settled=false;
      const finish=(error?:Error)=>{
        if(settled)return;
        settled=true;
        clearTimeout(timer);
        this.channel.removeEventListener("open",opened);
        this.channel.removeEventListener("close",ended);
        this.channel.removeEventListener("error",ended);
        error?reject(error):resolve();
      };
      const opened=()=>finish();
      const ended=()=>finish(new Error("WebRTC data channel closed before connection"));
      const timer=setTimeout(()=>finish(new Error("WebRTC connection timed out")),timeoutMs);
      this.channel.addEventListener("open",opened);
      this.channel.addEventListener("close",ended);
      this.channel.addEventListener("error",ended);
      if(this.channel.readyState==="open")finish();
    });
  }

  public sendMessage(data: message.DeepPartial<message.Message>) {
    if(this.closed || this.channel.readyState!=="open"){
      throw new Error("WebRTC data channel is not open");
    }
    const extension=data as any;
    let bytes=extension.__terminal_action
      ? encodeTerminalMessage(extension.__terminal_action)
      : message.Message.encode(message.Message.fromPartial(data)).finish();
    if(extension.__login_mode||extension.__os_login)
      bytes=addSessionMode(bytes,extension.__login_mode,extension.__os_login);
    if((extension.login_request && !extension.login_request.file_transfer &&
        !["terminal","adminTerminal"].includes(extension.__login_mode)) ||
       (extension.__video_codecs && extension.misc?.option))
      bytes=advertiseVideoDecoders(bytes,extension.__video_codecs);
    if(bytes.length > MAX_LOGICAL_MESSAGE) throw new Error("WebRTC message exceeds size limit");
    // A failed send must close the transport to avoid leaving the recipient's
    // fragment accumulator in a partially sent state.
    try{
      for(let offset=0;offset<bytes.length||offset===0;offset+=MAX_FRAGMENT_PAYLOAD){
        const end=Math.min(offset+MAX_FRAGMENT_PAYLOAD,bytes.length);
        if(this.channel.bufferedAmount>MAX_BUFFERED_BYTES){
          throw new Error("WebRTC data channel is congested");
        }
        const packet=new Uint8Array(end-offset+1);
        packet[0]=end<bytes.length?1:0;
        packet.set(bytes.subarray(offset,end),1);
        this.channel.send(packet);
      }
    }catch(err){
      this.close();
      throw err;
    }
  }

  public setSecretKey(_key:Uint8Array) {
    // Native RustDesk WebRTC uses the authenticated DTLS channel directly.
    // RustDesk's signed-id handshake still runs, but this transport does not
    // apply a second secretbox cipher to the data channel.
  }

  public next(timeoutMs=0):Promise<message.Message>{
    if(this.frames.length)return Promise.resolve(this.frames.shift()!);
    if(this.closed)return Promise.reject(new Error("WebRTC channel closed"));
    return new Promise((resolve,reject)=>{
      const waiter:WaitingReader={
        resolve,reject,
        timeout:timeoutMs>0?setTimeout(()=>{
          this.waiting=this.waiting.filter(x=>x!==waiter);
          reject(new Error("WebRTC message timed out"));
        },timeoutMs):undefined
      };
      this.waiting.push(waiter);
    });
  }

  private receive(evt:MessageEvent) {
    if(this.closed)return;
    if(!(evt.data instanceof ArrayBuffer)){this.fail(new Error("Unexpected WebRTC data type"));return}
    const bytes=new Uint8Array(evt.data);
    if(bytes.length===0 || bytes[0]>1){this.fail(new Error("Invalid WebRTC fragment header"));return}
    const payload=bytes.subarray(1);
    if(this.fragmentBytes+payload.length > MAX_LOGICAL_MESSAGE){
      this.fail(new Error("WebRTC message exceeds receive limit"));return;
    }
    this.fragments.push(payload);
    this.fragmentBytes+=payload.length;
    if(bytes[0]!==0)return;
    const merged=new Uint8Array(this.fragmentBytes);
    let position=0;
    for(const item of this.fragments){merged.set(item,position);position+=item.length}
    this.fragments=[];this.fragmentBytes=0;
    try{
      const decoded=message.Message.decode(merged);
      const terminal=decodeTerminalMessage(merged);
      if(terminal)(decoded as any).terminal_response=terminal;
      const info=extractStandalonePeerInfo(merged);
      if(info)(decoded as any).peer_info=message.PeerInfo.decode(info);
      const waiter=this.waiting.shift();
      if(waiter){clearTimeout(waiter.timeout);waiter.resolve(decoded)}
      else this.frames.push(decoded);
      if(this.frames.length>256)throw new Error("WebRTC message queue exceeded limit");
    }catch(err){
      this.fail(err instanceof Error?err:new Error(String(err)));
    }
  }

  private fail(reason:Error){
    if(this.closed)return;
    const pending=this.waiting.splice(0);
    for(const waiter of pending){clearTimeout(waiter.timeout);waiter.reject(reason)}
    this.close();
  }

  public close(){
    if(this.closed)return;
    this.closed=true;
    this.fragments=[];this.frames=[];
    try{this.channel.close()}catch{}
    try{this.pc.close()}catch{}
    for(const waiter of this.waiting.splice(0)){
      clearTimeout(waiter.timeout);
      waiter.reject(new Error("WebRTC data channel closed"));
    }
  }
}
