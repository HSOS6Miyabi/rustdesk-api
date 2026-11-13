import Websock from "./websock";
import WebRTCTransport from "./webrtc_transport";
import {ExtendedRendezvous,encodeWebRTCEndpoint,decodeWebRTCEndpoint,
  fingerprintFromSDP,fingerprintMatches,IceCandidate} from "./webrtc_protocol";
import * as message from "./message.js";
import * as rendezvous from "./rendezvous.js";
import { loadVp9 } from "./codec";
import * as sha256 from "fast-sha256";
import * as globals from "./globals";
import { decompress, mapKey, sleep } from "./common";

const PORT = 21116;
// only the first is used to init `HOST`
const HOSTS = [
  "rs-sg.rustdesk.com",
  "rs-cn.rustdesk.com",
  "rs-us.rustdesk.com",
];
let HOST = localStorage.getItem("rendezvous-server") || HOSTS[0];
//根据协议设置为ws或wss
const SCHEMA=location.protocol=="https:"?"wss://":"ws://";

type MsgboxCallback = (type: string, title: string, text: string) => void;
type DrawCallback = (data: Uint8Array) => void;
//const cursorCanvas = document.createElement("canvas");

export default class Connection {
  _ws: Websock | WebRTCTransport | undefined;
  _id: string;
  _hash: message.Hash | undefined;
  _msgbox: MsgboxCallback;
  _draw: DrawCallback;
  _peerInfo: message.PeerInfo | undefined;
  _firstFrame: Boolean | undefined;
  _videoDecoder: any;
  _password: Uint8Array | undefined;
  _options: any;
  _videoTestSpeed: number[];
  _sessionMode: string;
  _terminalAdmin: boolean;
  _osUsername: string;
  _osPassword: string;
  //_cursors: { [name: number]: any };

  constructor() {
    this._msgbox = globals.msgbox;
    this._draw = globals.draw;
    this._id = "";
    this._videoTestSpeed = [0, 0];
    this._sessionMode = "desktop";
    this._terminalAdmin = false;
    this._osUsername = "";
    this._osPassword = "";
    //this._cursors = {};
  }

  setSessionMode(mode: string, administrator: boolean = false) {
    if (!["desktop", "file", "camera", "terminal"].includes(mode)) throw new Error("Unknown session mode");
    this._sessionMode = mode;
    this._terminalAdmin = administrator;
  }

  sendTerminalAction(action: any) {
    if (this._sessionMode !== "terminal") throw new Error("Terminal session required");
    this._ws?.sendMessage({__terminal_action:action} as any);
  }

  readRemoteDir(path: string, includeHidden: boolean) {
    if (this._sessionMode !== "file") throw new Error("File-transfer session required");
    this._ws?.sendMessage({file_action:{read_dir:{path,include_hidden:includeHidden}}});
  }

  cancelFileJob(id: number) {
    if (this._sessionMode !== "file") throw new Error("File-transfer session required");
    this._ws?.sendMessage({file_action:{cancel:{id}}});
  }

  credentialStorageKey(id: string = this._id): string {
    const server = String(localStorage.getItem("custom-rendezvous-server") ||
      localStorage.getItem("rendezvous-server") || "").trim().toLowerCase();
    return "rustdesk.webclient.credentials.v1:" + JSON.stringify([server, id]);
  }

  clearRememberedPassword(): void {
    if (this._id) localStorage.removeItem(this.credentialStorageKey());
    this._password = undefined;
    if (this._options?.password !== undefined) {
      delete this._options.password;
      if (this._id) {
        const peers = globals.getPeers();
        peers[this._id] = this._options;
        localStorage.setItem("peers", JSON.stringify(peers));
      }
    }
  }

  async start(id: string) {
    try {
      await this._start(id);
    } catch (e: any) {
      this.msgbox(
        "error",
        "Connection Error",
        e.type == "close" ? "Reset by the peer" : String(e)
      );
    }
  }

  async _start(id: string) {
    if (!this._options) {
      this._options = globals.getPeers()[id] || {};
    }
    // Keep remembered credentials specific to the selected server and device.
    if (this._options.password !== undefined) {
      delete this._options.password;
      const peers = globals.getPeers();
      peers[id] = this._options;
      localStorage.setItem("peers", JSON.stringify(peers));
    }
    if (!this._password && this.getRemember()) {
      const stored = localStorage.getItem(this.credentialStorageKey(id));
      if (stored) {
        try {
          const bytes = JSON.parse(stored);
          if (!Array.isArray(bytes) || bytes.length !== 32 ||
              !bytes.every((byte: unknown) => Number.isInteger(byte) && byte >= 0 && byte <= 255)) {
            throw Error("Invalid credential");
          }
          this._password = Uint8Array.from(bytes);
        } catch {
          localStorage.removeItem(this.credentialStorageKey(id));
        }
      }
    }
    if(this._sessionMode==="desktop"||this._sessionMode==="camera")await this.loadVideoDecoder();
    this._id=id;
    if(sessionStorage.getItem("rustdesk.webclient.webrtc")==="Y" &&
       typeof RTCPeerConnection!=="undefined"){
      try{
        if(await this.connectWebRTC(id))return;
      }catch(error){
        console.warn("WebRTC direct connection failed, switching to WebSocket relay:",error);
      }
    }
    const uri = getDefaultUri();
    const ws = new Websock(uri, true);
    this._ws = ws;
    this._id = id;
    console.log(
      new Date() + ": Conntecting to rendezvoous server: " + uri + ", for " + id
    );
    await ws.open();
    console.log(new Date() + ": Connected to rendezvoous server");
    const conn_type = this._sessionMode==="file"
      ? rendezvous.ConnType.FILE_TRANSFER
      : rendezvous.ConnType.DEFAULT_CONN;
    const nat_type = rendezvous.NatType.SYMMETRIC;
    const punch_hole_request = rendezvous.PunchHoleRequest.fromPartial({
      id,
      licence_key: localStorage.getItem("key") || undefined,
      conn_type,
      nat_type,
      token: sessionStorage.getItem("rustdesk.console.token") || undefined,
    });
    ws.sendRendezvous({ punch_hole_request });
    const msg = (await ws.next(12000)) as rendezvous.RendezvousMessage;
    ws.close();
    console.log(new Date() + ": Got relay response", msg);
    const phr = msg.punch_hole_response;
    const rr = msg.relay_response;
    if (phr) {
      if (phr?.other_failure) {
        this.msgbox("error", "Error", phr?.other_failure);
        return;
      }
      if (phr.failure != rendezvous.PunchHoleResponse_Failure.UNRECOGNIZED) {
        switch (phr?.failure) {
          case rendezvous.PunchHoleResponse_Failure.ID_NOT_EXIST:
            this.msgbox("error", "Error", "ID does not exist");
            break;
          case rendezvous.PunchHoleResponse_Failure.OFFLINE:
            this.msgbox("error", "Error", "Remote desktop is offline");
            break;
          case rendezvous.PunchHoleResponse_Failure.LICENSE_MISMATCH:
            this.msgbox("error", "Error", "Key mismatch");
            break;
          case rendezvous.PunchHoleResponse_Failure.LICENSE_OVERUSE:
            this.msgbox("error", "Error", "Key overuse");
            break;
        }
      }
    } else if (rr) {
      if (!rr.version) {
        this.msgbox("error", "Error", "Remote version is low, not support web");
        return;
      }
      await this.connectRelay(rr);
    }
  }

  // Prefer an authenticated WebRTC data channel, with the established relay
  // connection procedure retained as the fallback for unavailable peers.
  async connectWebRTC(id:string):Promise<boolean>{
    const key=localStorage.getItem("key")||"";
    if(!key.trim())return false;
    const ws=new Websock(getDefaultUri(),true);
    const pc=new RTCPeerConnection({
      iceTransportPolicy:"all",
      iceServers:[
        {urls:"stun:stun.cloudflare.com:3478"},
        {urls:"stun:stun.l.google.com:19302"},
        {urls:"stun:stun.antisip.com:3478"}
      ]
    });
    const channel=pc.createDataChannel("bootstrap",{ordered:true});
    const transport=new WebRTCTransport(pc,channel);
    let connected=false,offerSent=false;
    let sessionKey="",peerKey:Uint8Array|undefined;
    const pendingLocal:IceCandidate[]=[];
    const pendingRemote:IceCandidate[]=[];
    const sendCandidate=(candidate:IceCandidate)=>{
      try{
        ws.sendICE(candidate);
        // The controlled device's first UDP signaling packet may arrive
        // after the initial candidate; send it again once.
        setTimeout(()=>{
          if(!connected && ws._status==="open"){
            try{ws.sendICE(candidate)}catch{}
          }
        },400);
      }catch(error){console.debug("ICE candidate send:",error);}
    };
    pc.onicecandidate=event=>{
      if(!event.candidate||!sessionKey)return;
      const value:IceCandidate={
        id, socket_addr:new Uint8Array(),
        session_key:sessionKey,
        candidate:JSON.stringify(event.candidate.toJSON())
      };
      if(value.candidate.length>8*1024)return;
      if(!offerSent)pendingLocal.push(value);
      else sendCandidate(value);
    };
    const applyRemote=async(ice:IceCandidate)=>{
      if(ice.session_key!==sessionKey||!ice.candidate||ice.candidate.length>8*1024)return;
      if(!pc.remoteDescription){
        if(pendingRemote.length<64)pendingRemote.push(ice);
        return;
      }
      try{await pc.addIceCandidate(JSON.parse(ice.candidate))}
      catch(error){console.debug("ICE candidate rejected:",error);}
    };
    try{
      await ws.open();
      const offer=await pc.createOffer();
      await pc.setLocalDescription(offer);
      const local=pc.localDescription;
      if(!local?.sdp)throw new Error("WebRTC local SDP missing");
      sessionKey=fingerprintFromSDP(local.sdp);
      if(!sessionKey)throw new Error("WebRTC local DTLS fingerprint missing");
      const request=rendezvous.PunchHoleRequest.fromPartial({
        id,licence_key:key,
        conn_type:rendezvous.ConnType.DEFAULT_CONN,
        nat_type:rendezvous.NatType.SYMMETRIC,
        token:sessionStorage.getItem("rustdesk.console.token")||""
      });
      ws.sendWebRTCOffer(request,encodeWebRTCEndpoint(local));
      offerSent=true;
      for(const candidate of pendingLocal.splice(0))sendCandidate(candidate);
      const end=Date.now()+15000;
      let answerReceived=false;
      while(Date.now()<end && !connected){
        if(answerReceived&&channel.readyState==="open"){connected=true;break}
        let message:ExtendedRendezvous;
        try{message=await ws.next(500) as ExtendedRendezvous}
        catch(error){
          if(/timed out|timeout/i.test(String(error)))continue;
          throw error;
        }
        if(message.ice_candidate){await applyRemote(message.ice_candidate)}
        const response=message.punch_hole_response;
        if(response?.other_failure)throw new Error(response.other_failure);
        if(response?.webrtc_sdp_answer&&!answerReceived){
          const answer=decodeWebRTCEndpoint(response.webrtc_sdp_answer);
          await pc.setRemoteDescription(answer);
          answerReceived=true;
          peerKey=response.pk;
          for(const candidate of pendingRemote.splice(0))await applyRemote(candidate);
        }
        // A legacy peer may not answer WebRTC. Keep the relay response and
        // open a new ordinary relay request when this attempt expires.
      }
      if(!answerReceived||channel.readyState!=="open"||!peerKey?.length)return false;
      this._ws=transport;
      if(!(await this.secure(peerKey)))throw new Error("WebRTC peer identity validation failed");
      ws.close();
      globals.pushEvent("connection_ready",{secure:true,direct:true});
      await this.msgLoop();
      return true;
    }finally{
      ws.close();
      pc.onicecandidate=null;
      transport.close();
      if(this._ws===transport)this._ws=undefined;
    }
  }

  async connectRelay(rr: rendezvous.RelayResponse) {
    const pk = rr.pk;
    let uri = rr.relay_server;
    if (uri) {
      uri = getrUriFromRs(uri, true, 2);
    } else {
      uri = getDefaultUri(true);
    }
    const uuid = rr.uuid;
    console.log(new Date() + ": Connecting to relay server: " + uri);
    const ws = new Websock(uri, false);
    await ws.open();
    console.log(new Date() + ": Connected to relay server");
    this._ws = ws;
    const request_relay = rendezvous.RequestRelay.fromPartial({
      licence_key: localStorage.getItem("key") || undefined,
      uuid,
    });
    ws.sendRendezvous({ request_relay });
    const secure = (await this.secure(pk)) || false;
    globals.pushEvent("connection_ready", { secure, direct: false });
    await this.msgLoop();
  }

  async secure(pk: Uint8Array | undefined) {
    const isRTC=this._ws instanceof WebRTCTransport;
    if (pk) {
      const RS_PK = "OeVuKk5nlHiXp+APNn0Y3pC1Iwpwn44JGqrQCsWqmBw=";
      try {
        pk = await globals.verify(pk, localStorage.getItem("key") || RS_PK);
        if (pk) {
          const idpk = message.IdPk.decode(pk);
          if (idpk.id == this._id) {
            pk = idpk.pk;
          }
        }
        if (pk?.length != 32) {
          pk = undefined;
        }
      } catch (e) {
        console.error(e);
        pk = undefined;
      }
      if (!pk)
        console.error(
          "Handshake failed: invalid public key from rendezvous server"
        );
    }
    if(isRTC&&!pk)throw new Error("WebRTC server identity could not be verified");
    if (!pk) {
      // send an empty message out in case server is setting up secure and waiting for first message
      const public_key = message.PublicKey.fromPartial({});
      this._ws?.sendMessage({ public_key });
      return;
    }
    const msg = (await this._ws?.next(12000)) as message.Message;
    let signedId: any = msg?.signed_id;
    if(isRTC&&!signedId)throw new Error("WebRTC peer did not provide a signed identity");
    if (!signedId) {
      console.error("Handshake failed: invalid message type");
      const public_key = message.PublicKey.fromPartial({});
      this._ws?.sendMessage({ public_key });
      return;
    }
    try {
      signedId = await globals.verify(signedId.id, Uint8Array.from(pk!));
    } catch (e) {
      console.error(e);
      // fall back to non-secure connection in case pk mismatch
      console.error("pk mismatch, fall back to non-secure");
      const public_key = message.PublicKey.fromPartial({});
      this._ws?.sendMessage({ public_key });
      return;
    }
    if(isRTC&&!fingerprintMatches(signedId,(this._ws as WebRTCTransport).remoteSDP())){
      throw new Error("WebRTC DTLS certificate does not match the signed peer identity");
    }
    const idpk = message.IdPk.decode(signedId);
    const id = idpk.id;
    const theirPk = idpk.pk;
    if(isRTC&&id!==this._id)throw new Error("WebRTC peer ID mismatch");
    if (id != this._id!) {
      console.error("Handshake failed: sign failure");
      const public_key = message.PublicKey.fromPartial({});
      this._ws?.sendMessage({ public_key });
      return;
    }
    if(isRTC&&theirPk.length!==32)throw new Error("Invalid WebRTC peer public key");
    if (theirPk.length != 32) {
      console.error(
        "Handshake failed: invalid public box key length from peer"
      );
      const public_key = message.PublicKey.fromPartial({});
      this._ws?.sendMessage({ public_key });
      return;
    }
    const [mySk, asymmetric_value] = globals.genBoxKeyPair();
    const secret_key = globals.genSecretKey();
    const symmetric_value = globals.seal(secret_key, theirPk, mySk);
    const public_key = message.PublicKey.fromPartial({
      asymmetric_value,
      symmetric_value,
    });
    this._ws?.sendMessage({ public_key });
    this._ws?.setSecretKey(secret_key);
    console.log("secured");
    return true;
  }

  async msgLoop() {
    while (true) {
      const msg = (await this._ws?.next()) as message.Message;
      // console.log("msg", msg);
      if (msg?.hash) {
        this._hash = msg?.hash;
        const tmp = this.getOption('tmppwd')
        if(!this._password && tmp){
          this._password = Uint8Array.from(JSON.parse("[" + tmp + "]"));
          this.setOption('tmppwd', '')
        }
        if(this._sessionMode==="terminal"&&this._terminalAdmin){
          this.msgbox("terminal-admin-login-password","","");
          continue;
        }
        if (!this._password)
          this.msgbox("input-password", "Password Required", "");
        this.login();
      } else if (msg?.test_delay) {
        const test_delay = msg?.test_delay;
        console.log(test_delay);
        if (!test_delay.from_client) {
          this._ws?.sendMessage({ test_delay });
        }
      } else if (msg?.login_response) {
        const r = msg?.login_response;
        if (r.error) {
          if (r.error == "Wrong Password") {
            this.clearRememberedPassword();
            this.msgbox(
              "re-input-password",
              r.error,
              "Do you want to enter again?"
            );
          } else {
            this.msgbox("error", "Login Error", r.error);
          }
        } else if (r.peer_info) {
          this.handlePeerInfo(r.peer_info);
        }
      } else if ((msg as any)?.peer_info) {
        this.handlePeerInfo((msg as any).peer_info);
      } else if (msg?.video_frame) {
        this.handleVideoFrame(msg?.video_frame!);
      } else if (msg?.clipboard) {
        const cb = msg?.clipboard;
        if (cb.compress) {
          const c = await decompress(cb.content);
          if (!c) continue;
          cb.content = c;
        }
        try {
          globals.copyToClipboard(new TextDecoder().decode(cb.content));
        } catch (e) {
          console.error(e);
        }
        // globals.pushEvent("clipboard", cb);
      } else if (msg?.cursor_data) {
        const cd = msg?.cursor_data;
        const c = await decompress(cd.colors);
        if (!c) continue;
        cd.colors = c;
        globals.pushEvent("cursor_data", cd);
        /*
        let ctx = cursorCanvas.getContext("2d");
        cursorCanvas.width = cd.width;
        cursorCanvas.height = cd.height;
        let imgData = new ImageData(
          new Uint8ClampedArray(c),
          cd.width,
          cd.height
        );
        ctx?.clearRect(0, 0, cd.width, cd.height);
        ctx?.putImageData(imgData, 0, 0);
        let url = cursorCanvas.toDataURL();
        const img = document.createElement("img");
        img.src = url;
        this._cursors[cd.id] = img;
        //cursorCanvas.width /= 2.;
        //cursorCanvas.height /= 2.;
        //ctx?.drawImage(img, cursorCanvas.width, cursorCanvas.height);
        url = cursorCanvas.toDataURL();
        document.body.style.cursor =
          "url(" + url + ")" + cd.hotx + " " + cd.hoty + ", default";
        console.log(document.body.style.cursor);
        */
      } else if (msg?.cursor_id) {
        globals.pushEvent("cursor_id", { id: msg?.cursor_id });
      } else if (msg?.cursor_position) {
        globals.pushEvent("cursor_position", msg?.cursor_position);
      } else if (msg?.misc) {
        if (!this.handleMisc(msg?.misc)) break;
      } else if((msg as any)?.terminal_response){
        const response=(msg as any).terminal_response;
        if(response.type==="data"&&response.data instanceof Uint8Array){
          let bytes=response.data;
          if(response.compressed){
            const uncompressed=await decompress(bytes);
            if(!uncompressed)continue;
            bytes=uncompressed;
          }
          let encoded="";
          for(let i=0;i<bytes.length;i+=8192)encoded+=String.fromCharCode(...bytes.subarray(i,i+8192));
          response.data=btoa(encoded);
        }
        globals.pushEvent("terminal_response",response);
      } else if(msg?.file_response){
        const response=msg.file_response;
        if(response.dir)globals.pushEvent("file_dir",{value:JSON.stringify(response.dir),is_local:false});
        if(response.error)globals.pushEvent("job_error",{id:response.error.id,err:response.error.error});
        if(response.done)globals.pushEvent("job_done",{id:response.done.id,file_num:response.done.file_num});
      } else if (msg?.audio_frame) {
        globals.playAudio(msg?.audio_frame.data);
      }
    }
  }

  msgbox(type_: string, title: string, text: string) {
    this._msgbox?.(type_, title, text);
  }

  draw(frame: any) {
    this._draw?.(frame);
    globals.draw(frame);
  }

  close() {
    this._ws?.close();
    this._videoDecoder?.close();
  }

  refresh() {
    const misc = message.Misc.fromPartial({ refresh_video: true });
    this._ws?.sendMessage({ misc });
  }

  setMsgbox(callback: MsgboxCallback) {
    this._msgbox = callback;
  }

  setDraw(callback: DrawCallback) {
    this._draw = callback;
  }

  login(password: string | undefined = undefined, osUsername: string = "", osPassword: string = "") {
    if (this._sessionMode==="terminal") {
      this._osUsername=osUsername;
      this._osPassword=osPassword;
    }
    if (password) {
      const salt = this._hash?.salt;
      let p = hash([password, salt!]);
      this._password = p;
      const challenge = this._hash?.challenge;
      p = hash([p, challenge!]);
      this.msgbox("connecting", "Connecting...", "Logging in...");
      this._sendLoginMessage(p);
    } else {
      let p = this._password;
      if (p) {
        const challenge = this._hash?.challenge;
        p = hash([p, challenge!]);
      }
      this._sendLoginMessage(p);
    }
  }

  async reconnect() {
    this.close();
    await this.start(this._id);
  }

  _sendLoginMessage(password: Uint8Array | undefined = undefined) {
    const login_request = message.LoginRequest.fromPartial({
      username: this._id!,
      my_id: "web", // to-do
      my_name: "web", // to-do
      password,
      option: this.getOptionMessage(),
      video_ack_required: true,
    });
    if(this._sessionMode==="file"){
      login_request.file_transfer=message.FileTransfer.fromPartial({dir:"",show_hidden:false});
    }
    const outgoing:any={login_request};
    if(this._sessionMode==="camera"||this._sessionMode==="terminal")outgoing.__login_mode=this._sessionMode;
    if(this._sessionMode==="terminal")outgoing.__os_login={username:this._osUsername,password:this._osPassword};
    this._ws?.sendMessage(outgoing);
  }

  getOptionMessage(): message.OptionMessage | undefined {
    let n = 0;
    const msg = message.OptionMessage.fromPartial({});
    const q = this.getImageQualityEnum(this.getImageQuality(), true);
    const yes = message.OptionMessage_BoolOption.Yes;
    if (q != undefined) {
      msg.image_quality = q;
      n += 1;
    }
    if (this._options["show-remote-cursor"]) {
      msg.show_remote_cursor = yes;
      n += 1;
    }
    if (this._options["lock-after-session-end"]) {
      msg.lock_after_session_end = yes;
      n += 1;
    }
    if (this._options["privacy-mode"]) {
      msg.privacy_mode = yes;
      n += 1;
    }
    if (this._options["disable-audio"]) {
      msg.disable_audio = yes;
      n += 1;
    }
    if (this._options["disable-clipboard"]) {
      msg.disable_clipboard = yes;
      n += 1;
    }
    return n > 0 ? msg : undefined;
  }

  sendVideoReceived() {
    const misc = message.Misc.fromPartial({ video_received: true });
    this._ws?.sendMessage({ misc });
  }

  handleVideoFrame(vf: message.VideoFrame) {
    if (!this._firstFrame) {
      this.msgbox("", "", "");
      this._firstFrame = true;
    }
    if (vf.vp9s) {
      const dec = this._videoDecoder;
      var tm = new Date().getTime();
      var i = 0;
      const n = vf.vp9s?.frames.length;
      vf.vp9s.frames.forEach((f) => {
        dec.processFrame(f.data.slice(0).buffer, (ok: any) => {
          i++;
          if (i == n) this.sendVideoReceived();
          if (ok && dec.frameBuffer && n == i) {
            this.draw(dec.frameBuffer);
            const now = new Date().getTime();
            var elapsed = now - tm;
            this._videoTestSpeed[1] += elapsed;
            this._videoTestSpeed[0] += 1;
            if (this._videoTestSpeed[0] >= 30) {
              console.log(
                "video decoder: " +
                  parseInt(
                    "" + this._videoTestSpeed[1] / this._videoTestSpeed[0]
                  )
              );
              this._videoTestSpeed = [0, 0];
            }
          }
        });
      });
    }
  }

  handlePeerInfo(pi: message.PeerInfo) {
    this._peerInfo = pi;
    if (pi.displays.length == 0 && (this._sessionMode==="desktop"||this._sessionMode==="camera")) {
      this.msgbox("error", "Remote Error", "No Display");
      return;
    }
    this.msgbox("success", "Successful", "Connected, waiting for image...");
    globals.pushEvent("peer_info", pi);
    if(this._sessionMode==="file")this.readRemoteDir("",false);
    const p = this.shouldAutoLogin();
    if (p) this.inputOsPassword(p);
    const username = this.getOption("info")?.username;
    if (username && !pi.username) pi.username = username;
    this.setOption("info", pi);
    if (this.getRemember() && this._password?.length === 32) {
      localStorage.setItem(this.credentialStorageKey(), JSON.stringify(Array.from(this._password)));
    } else {
      this.clearRememberedPassword();
    }
  }

  shouldAutoLogin(): string {
    const l = this.getOption("lock-after-session-end");
    const a = !!this.getOption("auto-login");
    const p = this.getOption("os-password");
    if (p && l && a) {
      return p;
    }
    return "";
  }

  handleMisc(misc: message.Misc) {
    if (misc.audio_format) {
      globals.initAudio(
        misc.audio_format.channels,
        misc.audio_format.sample_rate
      );
    } else if (misc.chat_message) {
      globals.pushEvent("chat", { text: misc.chat_message.text });
    } else if (misc.permission_info) {
      const p = misc.permission_info;
      console.info("Change permission " + p.permission + " -> " + p.enabled);
      let name;
      switch (p.permission) {
        case message.PermissionInfo_Permission.Keyboard:
          name = "keyboard";
          break;
        case message.PermissionInfo_Permission.Clipboard:
          name = "clipboard";
          break;
        case message.PermissionInfo_Permission.Audio:
          name = "audio";
          break;
        default:
          return;
      }
      globals.pushEvent("permission", { [name]: p.enabled });
    } else if (misc.switch_display) {
      this.loadVideoDecoder().catch(error=>this.msgbox("error","Video Decoder",String(error)));
      globals.pushEvent("switch_display", misc.switch_display);
    } else if (misc.close_reason) {
      this.msgbox("error", "Connection Error", misc.close_reason);
      this.close();
      return false;
    }
    return true;
  }

  getRemember(): Boolean {
    return this._options["remember"] || false;
  }

  setRemember(v: Boolean) {
    this.setOption("remember", Boolean(v));
    if (!v) this.clearRememberedPassword();
  }

  getOption(name: string): any {
    return this._options[name];
  }

  setOption(name: string, value: any) {
    if (value == undefined) {
      delete this._options[name];
    } else {
      this._options[name] = value;
    }
    this._options["tm"] = new Date().getTime();
    const peers = globals.getPeers();
    peers[this._id] = this._options;
    localStorage.setItem("peers", JSON.stringify(peers));
  }

  inputKey(
    name: string,
    down: boolean,
    press: boolean,
    alt: Boolean,
    ctrl: Boolean,
    shift: Boolean,
    command: Boolean
  ) {
    const key_event = mapKey(name, globals.isDesktop());
    if (!key_event) return;
    if (alt && (name == "VK_MENU" || name == "RAlt")) {
      alt = false;
    }
    if (ctrl && (name == "VK_CONTROL" || name == "RControl")) {
      ctrl = false;
    }
    if (shift && (name == "VK_SHIFT" || name == "RShift")) {
      shift = false;
    }
    if (command && (name == "Meta" || name == "RWin")) {
      command = false;
    }
    key_event.down = down;
    key_event.press = press;
    key_event.modifiers = this.getMod(alt, ctrl, shift, command);
    this._ws?.sendMessage({ key_event });
  }

  ctrlAltDel() {
    const key_event = message.KeyEvent.fromPartial({ down: true });
    if (this._peerInfo?.platform == "Windows") {
      key_event.control_key = message.ControlKey.CtrlAltDel;
    } else {
      key_event.control_key = message.ControlKey.Delete;
      key_event.modifiers = this.getMod(true, true, false, false);
    }
    this._ws?.sendMessage({ key_event });
  }

  inputString(seq: string) {
    const key_event = message.KeyEvent.fromPartial({ seq });
    this._ws?.sendMessage({ key_event });
  }

  switchDisplay(display: number) {
    const switch_display = message.SwitchDisplay.fromPartial({ display });
    const misc = message.Misc.fromPartial({ switch_display });
    this._ws?.sendMessage({ misc });
  }

  async inputOsPassword(seq: string) {
    this.inputMouse();
    await sleep(50);
    this.inputMouse(0, 3, 3);
    await sleep(50);
    this.inputMouse(1 | (1 << 3));
    this.inputMouse(2 | (1 << 3));
    await sleep(1200);
    const key_event = message.KeyEvent.fromPartial({ press: true, seq });
    this._ws?.sendMessage({ key_event });
  }

  lockScreen() {
    const key_event = message.KeyEvent.fromPartial({
      down: true,
      control_key: message.ControlKey.LockScreen,
    });
    this._ws?.sendMessage({ key_event });
  }

  getMod(alt: Boolean, ctrl: Boolean, shift: Boolean, command: Boolean) {
    const mod: message.ControlKey[] = [];
    if (alt) mod.push(message.ControlKey.Alt);
    if (ctrl) mod.push(message.ControlKey.Control);
    if (shift) mod.push(message.ControlKey.Shift);
    if (command) mod.push(message.ControlKey.Meta);
    return mod;
  }

  inputMouse(
    mask: number = 0,
    x: number = 0,
    y: number = 0,
    alt: Boolean = false,
    ctrl: Boolean = false,
    shift: Boolean = false,
    command: Boolean = false
  ) {
    const mouse_event = message.MouseEvent.fromPartial({
      mask,
      x,
      y,
      modifiers: this.getMod(alt, ctrl, shift, command),
    });
    this._ws?.sendMessage({ mouse_event });
  }

  toggleOption(name: string) {
    const v = !this._options[name];
    const option = message.OptionMessage.fromPartial({});
    const v2 = v
      ? message.OptionMessage_BoolOption.Yes
      : message.OptionMessage_BoolOption.No;
    switch (name) {
      case "show-remote-cursor":
        option.show_remote_cursor = v2;
        break;
      case "disable-audio":
        option.disable_audio = v2;
        break;
      case "disable-clipboard":
        option.disable_clipboard = v2;
        break;
      case "lock-after-session-end":
        option.lock_after_session_end = v2;
        break;
      case "privacy-mode":
        option.privacy_mode = v2;
        break;
      case "block-input":
        option.block_input = message.OptionMessage_BoolOption.Yes;
        break;
      case "unblock-input":
        option.block_input = message.OptionMessage_BoolOption.No;
        break;
      default:
        return;
    }
    if (name.indexOf("block-input") < 0) this.setOption(name, v);
    const misc = message.Misc.fromPartial({ option });
    this._ws?.sendMessage({ misc });
  }

  getImageQuality() {
    return this.getOption("image-quality");
  }

  getImageQualityEnum(
    value: string,
    ignoreDefault: Boolean
  ): message.ImageQuality | undefined {
    switch (value) {
      case "low":
        return message.ImageQuality.Low;
      case "best":
        return message.ImageQuality.Best;
      case "balanced":
        return ignoreDefault ? undefined : message.ImageQuality.Balanced;
      default:
        return undefined;
    }
  }

  setImageQuality(value: string) {
    this.setOption("image-quality", value);
    const image_quality = this.getImageQualityEnum(value, false);
    if (image_quality == undefined) return;
    const option = message.OptionMessage.fromPartial({ image_quality });
    const misc = message.Misc.fromPartial({ option });
    this._ws?.sendMessage({ misc });
  }

  async loadVideoDecoder():Promise<void> {
    this._videoDecoder?.close();
    this._videoDecoder=undefined;
    await new Promise<void>((resolve,reject)=>{
      let completed=false;
      const timeout=setTimeout(()=>{
        if(completed)return;
        completed=true;
        reject(new Error("VP9 video decoder initialization timed out"));
      },12000);
      const ready=(decoder:any)=>{
        if(completed){decoder.close?.();return}
        completed=true;
        clearTimeout(timeout);
        this._videoDecoder=decoder;
        resolve();
      };
      loadVp9(ready).catch((error:any)=>{
        if(completed)return;
        completed=true;
        clearTimeout(timeout);
        reject(error);
      });
    });
  }
}

function testDelay() {
  var nearest = "";
  HOSTS.forEach((host) => {
    const now = new Date().getTime();
    new Websock(getrUriFromRs(host), true).open().then(() => {
      console.log("latency of " + host + ": " + (new Date().getTime() - now));
      if (!nearest) {
        HOST = host;
        localStorage.setItem("rendezvous-server", host);
      }
    });
  });
}


function getDefaultUri(isRelay: Boolean = false): string {
  const host = localStorage.getItem("custom-rendezvous-server");
  return getrUriFromRs(host || HOST, isRelay);
}
/*
function isHttps() {
  return window.location.protocol === "https:"
}

function domain(uri: string) {
    return uri.indexOf(":") > 0 ? uri.split(":")[0] : uri
}*/

function getrUriFromRs(
  uri: string,
  isRelay: Boolean = false,
  roffset: number = 0
): string {
  const configuredWsHost = String((window as any).ws_host || "").trim().replace(/\\/g, "/").replace(/\/+$/, "");
  if (location.protocol === "https:" && configuredWsHost) {
    if (!configuredWsHost.startsWith("wss://")) {
      throw new Error("The configured WebSocket endpoint must use wss:// on HTTPS pages");
    }
    return configuredWsHost + (isRelay ? "/ws/relay" : "/ws/id");
  }
  if (uri.indexOf(":") > 0) {
    const tmp = uri.split(":");
    const port = parseInt(tmp[1]);
    uri = tmp[0] + ":" + (port + (isRelay ? roffset || 3 : 2));
  } else {
    uri += ":" + (PORT + (isRelay ? 3 : 2));
  }
  return SCHEMA + uri;
}

function hash(datas: (string | Uint8Array)[]): Uint8Array {
  const hasher = new sha256.Hash();
  datas.forEach((data) => {
    if (typeof data == "string") {
      data = new TextEncoder().encode(data);
    }
    return hasher.update(data);
  });
  return hasher.digest();
}
