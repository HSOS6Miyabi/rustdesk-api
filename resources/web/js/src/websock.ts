import * as message from "./message.js";
import * as rendezvous from "./rendezvous.js";
import * as globals from "./globals";
import {decodeRendezvous,encodeOffer,encodeICE,IceCandidate} from "./webrtc_protocol";

type Keys = "message" | "open" | "close" | "error";
type Incoming = rendezvous.RendezvousMessage | message.Message;
type Waiter = {resolve:(value:Incoming)=>void;reject:(error:Error)=>void;timeout?:ReturnType<typeof setTimeout>};

export default class Websock {
  _websocket: WebSocket;
  _eventHandlers: { [key in Keys]: Function };
  _buf: Incoming[];
  _waiting: Waiter[];
  _status: any;
  _latency: number;
  _secretKey: [Uint8Array, number, number] | undefined;
  _uri: string;
  _isRendezvous: boolean;

  constructor(uri: string, isRendezvous: boolean = true) {
    this._eventHandlers = {
      message: (_: any) => {},
      open: () => {},
      close: () => {},
      error: () => {},
    };
    this._uri = uri;
    this._status = "";
    this._buf = [];
    this._waiting = [];
    this._websocket = new WebSocket(uri);
    this._websocket.onmessage = this._recv_message.bind(this);
    this._websocket.binaryType = "arraybuffer";
    this._latency = new Date().getTime();
    this._isRendezvous = isRendezvous;
  }

  latency(): number {
    return this._latency;
  }

  setSecretKey(key: Uint8Array) {
    this._secretKey = [key, 0, 0];
  }

  sendMessage(json: message.DeepPartial<message.Message>) {
    let data = message.Message.encode(
      message.Message.fromPartial(json)
    ).finish();
    let k = this._secretKey;
    if (k) {
      k[1] += 1;
      data = globals.encrypt(data, k[1], k[0]);
    }
    this._websocket.send(data);
  }

  sendRendezvous(data: rendezvous.DeepPartial<rendezvous.RendezvousMessage>) {
    this._websocket.send(
      rendezvous.RendezvousMessage.encode(
        rendezvous.RendezvousMessage.fromPartial(data)
      ).finish()
    );
  }

  sendWebRTCOffer(request:rendezvous.PunchHoleRequest,offer:string) {
    this._websocket.send(encodeOffer(request,offer));
  }

  sendICE(candidate:IceCandidate) {
    this._websocket.send(encodeICE(candidate));
  }

  parseMessage(data: Uint8Array) {
    return message.Message.decode(data);
  }

  parseRendezvous(data: Uint8Array) {
    return decodeRendezvous(data);
  }

  // Event Handlers
  off(evt: Keys) {
    this._eventHandlers[evt] = () => {};
  }

  on(evt: Keys, handler: Function) {
    this._eventHandlers[evt] = handler;
  }

  async open(timeout: number = 12000): Promise<Websock> {
    return new Promise((resolve, reject) => {
      const timeoutHandle=setTimeout(() => {
        if (this._status != "open") {
          this.close();
          reject(new Error("WebSocket connection timed out"));
        }
      }, timeout);
      this._websocket.onopen = () => {
        clearTimeout(timeoutHandle);
        this._latency = new Date().getTime() - this._latency;
        this._status = "open";
        console.debug(">> WebSock.onopen");
        if (this._websocket?.protocol) {
          console.info(
            "Server choose sub-protocol: " + this._websocket.protocol
          );
        }

        this._eventHandlers.open();
        console.info("WebSock.onopen");
        resolve(this);
      };
      this._websocket.onclose = (e) => {
        clearTimeout(timeoutHandle);
        this.rejectWaiters(new Error("WebSocket connection was closed"));
        if (this._status == "open") {
          // e.code 1000 means that the connection was closed normally.
          //
        }
        this._status = e;
        console.error("WebSock.onclose: ");
        console.error(e);
        this._eventHandlers.close(e);
        reject("Reset by the peer");
      };
      this._websocket.onerror = (e: any) => {
        clearTimeout(timeoutHandle);
        this.rejectWaiters(new Error("WebSocket connection failed"));
        if (!this._status) {
          reject("Failed to connect to " + (this._isRendezvous ? "rendezvous" : "relay") + " server");
          return;
        }
        this._status = e;
        console.error("WebSock.onerror: ")
        console.error(e);
        this._eventHandlers.error(e);
      };
    });
  }

  next(timeout=0): Promise<Incoming> {
    if (this._buf.length) return Promise.resolve(this._buf.shift()!);
    if (this._status!=="open") {
      return Promise.reject(new Error("WebSocket connection is not open"));
    }
    return new Promise((resolve,reject)=>{
      const waiter:Waiter={
        resolve,reject,
        timeout:timeout>0?setTimeout(()=>{
          this._waiting=this._waiting.filter(item=>item!==waiter);
          reject(new Error("WebSocket message timed out"));
        },timeout):undefined
      };
      this._waiting.push(waiter);
    });
  }

  private rejectWaiters(error:Error){
    for(const item of this._waiting.splice(0)){
      clearTimeout(item.timeout);
      item.reject(error);
    }
    this._buf=[];
  }

  close() {
    this.rejectWaiters(new Error("WebSocket connection closed"));
    this._status = "";
    if (this._websocket) {
      if (
        this._websocket.readyState === WebSocket.OPEN ||
        this._websocket.readyState === WebSocket.CONNECTING
      ) {
        console.info("Closing WebSocket connection");
        this._websocket.close();
      }

      this._websocket.onmessage = () => {};
    }
  }

  _recv_message(e: any) {
    if (e.data instanceof window.ArrayBuffer) {
      let bytes = new Uint8Array(e.data);
      const k = this._secretKey;
      if (k) {
        k[2] += 1;
        bytes = globals.decrypt(bytes, k[2], k[0]);
      }
      try{
        const decoded=this._isRendezvous?this.parseRendezvous(bytes):this.parseMessage(bytes);
        const waiter=this._waiting.shift();
        if(waiter){
          clearTimeout(waiter.timeout);
          waiter.resolve(decoded);
        }else{
          if(this._buf.length>=256){
            this.close();
            return;
          }
          this._buf.push(decoded);
        }
      }catch(error){
        this.close();
        return;
      }
    }
    this._eventHandlers.message(e.data);
  }
}
