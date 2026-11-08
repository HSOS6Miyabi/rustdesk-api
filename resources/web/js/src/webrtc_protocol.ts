import _m0 from "protobufjs/minimal";
import * as rendezvous from "./rendezvous.js";

// RustDesk 1.5 uses new protobuf fields. Preserve the older browser engine's
// generated API types while decoding the additional signaling payloads.
export interface IceCandidate {
  id: string;
  socket_addr: Uint8Array;
  session_key: string;
  candidate: string;
}

export type ExtendedRendezvous = rendezvous.RendezvousMessage & {
  ice_candidate?: IceCandidate;
  punch_hole_response?: rendezvous.PunchHoleResponse & { webrtc_sdp_answer?: string };
};

function stringField(bytes: Uint8Array, target: number): string {
  const reader = new _m0.Reader(bytes);
  while (reader.pos < reader.len) {
    const tag = reader.uint32(), field = tag >>> 3;
    if (field === target && (tag & 7) === 2) return reader.string();
    reader.skipType(tag & 7);
  }
  return "";
}

export function signedPeerFingerprint(idPk: Uint8Array): string {
  return stringField(idPk, 3);
}

export function fingerprintFromSDP(sdp: string): string {
  const match = /(?:^|\r?\n)a=fingerprint:([^\r\n]+)/i.exec(sdp);
  return match ? match[1].trim() : "";
}

export function fingerprintMatches(idPk: Uint8Array, remoteSDP: string): boolean {
  const signed = signedPeerFingerprint(idPk).toLowerCase().replace(/\s+/g, " ").trim();
  const remote = fingerprintFromSDP(remoteSDP).toLowerCase().replace(/\s+/g, " ").trim();
  return !!signed && !!remote && signed === remote;
}

export function encodeOffer(
  request: rendezvous.PunchHoleRequest,
  offer: string
): Uint8Array {
  if (!offer.startsWith("webrtc://") || offer.length > 32 * 1024) {
    throw new Error("WebRTC SDP offer is invalid");
  }
  const inner = _m0.Writer.create();
  rendezvous.PunchHoleRequest.encode(request, inner);
  inner.uint32(98).string(offer); // PunchHoleRequest.webrtc_sdp_offer = 12
  const outer = _m0.Writer.create();
  outer.uint32(66).bytes(inner.finish()); // RendezvousMessage.punch_hole_request = 8
  return outer.finish();
}

export function encodeICE(value: IceCandidate): Uint8Array {
  if (!value.session_key || value.session_key.length > 256 ||
      !value.candidate || value.candidate.length > 8 * 1024 ||
      value.id.length > 128 || value.socket_addr.length > 64) {
    throw new Error("WebRTC ICE candidate exceeds protocol limits");
  }
  const inner = _m0.Writer.create();
  if (value.id) inner.uint32(10).string(value.id);
  if (value.socket_addr.length) inner.uint32(18).bytes(value.socket_addr);
  inner.uint32(26).string(value.session_key);
  inner.uint32(34).string(value.candidate);
  const outer = _m0.Writer.create();
  outer.uint32(234).bytes(inner.finish()); // RendezvousMessage.ice_candidate = 29
  return outer.finish();
}

function decodeICE(data: Uint8Array): IceCandidate {
  const result: IceCandidate = {
    id: "", socket_addr: new Uint8Array(), session_key: "", candidate: ""
  };
  const reader = new _m0.Reader(data);
  while (reader.pos < reader.len) {
    const tag = reader.uint32();
    switch (tag >>> 3) {
      case 1: result.id = reader.string(); break;
      case 2: result.socket_addr = reader.bytes(); break;
      case 3: result.session_key = reader.string(); break;
      case 4: result.candidate = reader.string(); break;
      default: reader.skipType(tag & 7); break;
    }
  }
  return result;
}

export function decodeRendezvous(data: Uint8Array): ExtendedRendezvous {
  const result = rendezvous.RendezvousMessage.decode(data) as ExtendedRendezvous;
  const reader = new _m0.Reader(data);
  while (reader.pos < reader.len) {
    const tag = reader.uint32();
    if ((tag >>> 3) === 11 && (tag & 7) === 2) {
      const body = reader.bytes();
      if (result.punch_hole_response) {
        result.punch_hole_response.webrtc_sdp_answer = stringField(body, 12);
      }
    } else if ((tag >>> 3) === 29 && (tag & 7) === 2) {
      result.ice_candidate = decodeICE(reader.bytes());
    } else {
      reader.skipType(tag & 7);
    }
  }
  return result;
}

export function encodeWebRTCEndpoint(description: RTCSessionDescriptionInit): string {
  if (!description.sdp || !description.type) throw new Error("WebRTC description is missing");
  const payload = JSON.stringify({
    type: description.type, sdp: description.sdp, ice_policy: "all"
  });
  const bytes = new TextEncoder().encode(payload);
  let source = "";
  for (const byte of bytes) source += String.fromCharCode(byte);
  return "webrtc://" + btoa(source);
}

export function decodeWebRTCEndpoint(value: string): RTCSessionDescriptionInit {
  if (!value.startsWith("webrtc://") || value.length > 32 * 1024) {
    throw new Error("Invalid WebRTC SDP response");
  }
  const raw = atob(value.substring("webrtc://".length));
  const bytes = Uint8Array.from(raw, char => char.charCodeAt(0));
  const parsed = JSON.parse(new TextDecoder().decode(bytes));
  if (parsed.type !== "answer" || typeof parsed.sdp !== "string" ||
      !parsed.sdp.startsWith("v=0")) throw new Error("Invalid WebRTC SDP answer");
  return {type:"answer",sdp:parsed.sdp};
}
