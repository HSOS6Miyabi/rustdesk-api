"use strict";

/* RustDesk 1.5.x protobuf fields used by browser sessions. */
const utf8 = new TextEncoder();
const text = new TextDecoder();

function concat(...parts) {
  const length = parts.reduce((sum, part) => sum + part.length, 0);
  const result = new Uint8Array(length);
  let position = 0;
  for (const part of parts) {
    result.set(part, position);
    position += part.length;
  }
  return result;
}

function varint(input) {
  let value = Number(input);
  if (!Number.isSafeInteger(value) || value < 0) {
    throw new RangeError("Invalid protocol integer");
  }
  const result = [];
  do {
    const byte = value % 128;
    value = Math.floor(value / 128);
    result.push(byte | (value ? 128 : 0));
  } while (value);
  return Uint8Array.from(result);
}

function readNumber(buffer, offset) {
  let value = 0, factor = 1, position = offset;
  for (let i = 0; i < 10; i++) {
    if (position >= buffer.length) throw new Error("Truncated protobuf varint");
    const byte = buffer[position++];
    value += (byte & 127) * factor;
    if ((byte & 128) === 0) return {value, position};
    factor *= 128;
  }
  throw new Error("Overlong protobuf varint");
}

function fields(buffer) {
  if (!(buffer instanceof Uint8Array) || buffer.length > 16 * 1024 * 1024) {
    throw new Error("Invalid protocol payload");
  }
  const result = [];
  let position = 0;
  while (position < buffer.length) {
    const tag = readNumber(buffer, position);
    position = tag.position;
    const field = Math.floor(tag.value / 8);
    const wire = tag.value & 7;
    if (field === 0) throw new Error("Invalid protobuf field");
    let value;
    if (wire === 2) {
      const length = readNumber(buffer, position);
      position = length.position;
      const end = position + length.value;
      if (end > buffer.length || !Number.isSafeInteger(end)) throw new Error("Invalid protobuf length");
      value = buffer.subarray(position, end);
      position = end;
    } else if (wire === 0) {
      const integer = readNumber(buffer, position);
      value = integer.value;
      position = integer.position;
    } else if (wire === 1 || wire === 5) {
      const end = position + (wire === 1 ? 8 : 4);
      if (end > buffer.length) throw new Error("Truncated protobuf value");
      value = buffer.subarray(position, end);
      position = end;
    } else {
      throw new Error("Unsupported protobuf field");
    }
    result.push({field, wire, value});
  }
  return result;
}

function uint(field, value) {
  return concat(varint(field * 8), varint(value));
}

function payload(field, value) {
  const bytes = typeof value === "string" ? utf8.encode(value) : value;
  if (!(bytes instanceof Uint8Array)) throw new TypeError("Expected byte payload");
  return concat(varint(field * 8 + 2), varint(bytes.length), bytes);
}

function readText(value) {
  return value instanceof Uint8Array ? text.decode(value) : "";
}

function reencodeField(field) {
  if(field.wire===2)return payload(field.field,field.value);
  if(field.wire===0)return uint(field.field,field.value);
  throw new Error("Unsupported protocol message field");
}

// Modern RustDesk peers receive the list of codecs supported by this browser.
export function advertiseVP9Decoder(encoded) {
  const messageFields=fields(encoded);
  if(!messageFields.some(field=>field.field===7&&field.wire===2))return encoded;
  const vp9=payload(10,concat(uint(1,1),uint(4,1)));
  return concat(...messageFields.map(field=>{
    if(field.field!==7||field.wire!==2)return reencodeField(field);
    const login=fields(field.value);
    let foundOption=false;
    const updated=login.map(item=>{
      if(item.field!==6||item.wire!==2)return reencodeField(item);
      foundOption=true;
      const prior=fields(item.value).filter(part=>part.field!==10);
      return payload(6,concat(...prior.map(reencodeField),vp9));
    });
    if(!foundOption)updated.push(payload(6,vp9));
    return payload(7,concat(...updated));
  }));
}

// The native protocol may send PeerInfo outside LoginResponse.
export function extractStandalonePeerInfo(encoded) {
  const info=fields(encoded).find(item=>item.field===25&&item.wire===2);
  return info?info.value:null;
}

export function addSessionMode(encoded, mode, osLogin) {
  if (mode !== "camera" && mode !== "terminal" && mode !== "adminTerminal" && !osLogin) return encoded;
  const additions = [];
  if (osLogin) {
    const user=String(osLogin.username||""), password=String(osLogin.password||"");
    additions.push(payload(12, concat(payload(1,user),payload(2,password))));
  }
  if(mode==="camera")additions.push(payload(15, new Uint8Array()));
  if(mode==="terminal"||mode==="adminTerminal")additions.push(payload(16, new Uint8Array()));
  const extra=concat(...additions);
  const output = [];
  let updated = false;
  for (const item of fields(encoded)) {
    if (item.field === 7 && item.wire === 2) {
      output.push(payload(7, concat(item.value, extra)));
      updated = true;
    } else if (item.wire === 2) {
      output.push(payload(item.field, item.value));
    } else if (item.wire === 0) {
      output.push(uint(item.field, item.value));
    } else {
      throw new Error("Unexpected binary protocol field");
    }
  }
  if (!updated) throw new Error("Missing RustDesk login request");
  return concat(...output);
}

function terminalPart(action) {
  const options = ["open", "data", "resize", "close"];
  const chosen = options.filter(key => action[key] != null);
  if (chosen.length !== 1) throw new Error("Exactly one terminal operation is required");
  const type = chosen[0];
  const data = action[type];
  const id = Number(data.terminal_id);
  if (!Number.isInteger(id) || id < 0) throw new Error("Invalid terminal identifier");
  const body = [uint(1, id)];
  if (type === "open" || type === "resize") {
    body.push(uint(2, Number(data.rows)), uint(3, Number(data.cols)));
  } else if (type === "data") {
    body.push(payload(2, data.data instanceof Uint8Array ? data.data : utf8.encode(String(data.data || ""))));
  }
  const field = options.indexOf(type) + 1;
  return payload(31, payload(field, concat(...body)));
}

export function encodeTerminalMessage(action) {
  return terminalPart(action);
}

export function decodeTerminalMessage(encoded) {
  const top = fields(encoded).find(item => item.field === 32 && item.wire === 2);
  if (!top) return null;
  const types = new Map([[1,"opened"],[2,"data"],[3,"closed"],[4,"error"]]);
  const section = fields(top.value).find(item => types.has(item.field) && item.wire === 2);
  if (!section) return null;
  const type = types.get(section.field);
  const detail = {type, terminal_id:0};
  for (const item of fields(section.value)) {
    if (item.field === 1 && item.wire === 0) detail.terminal_id = item.value;
    if (type === "opened") {
      if (item.field === 2) detail.success = !!item.value;
      if (item.field === 3) detail.message = readText(item.value);
      if (item.field === 4) detail.pid = item.value;
      if (item.field === 5) detail.service_id = readText(item.value);
      if (item.field === 6 && item.wire === 0) (detail.persistent_sessions ||= []).push(item.value);
      if (item.field === 6 && item.wire === 2) {
        for (const val of fieldsPackedIntegers(item.value)) (detail.persistent_sessions ||= []).push(val);
      }
      if (item.field === 7) detail.replay_terminal_output = !!item.value;
    } else if (type === "data") {
      if (item.field === 2 && item.wire === 2) detail.data = item.value;
      if (item.field === 3) detail.compressed = !!item.value;
    } else if (type === "closed") {
      if (item.field === 2) detail.exit_code = item.value;
    } else if (type === "error") {
      if (item.field === 2) detail.message = readText(item.value);
    }
  }
  return detail;
}

function fieldsPackedIntegers(data) {
  const values = [];
  let position = 0;
  while (position < data.length) {
    const next = readNumber(data, position);
    values.push(next.value);
    position = next.position;
  }
  return values;
}

export function encodeTerminalResponseForTest(response) {
  const kind = {opened:1,data:2,closed:3,error:4}[response.type];
  if (!kind) throw new Error("Invalid terminal response");
  const content = [uint(1, response.terminal_id || 0)];
  if (kind === 1) {
    content.push(uint(2,response.success ? 1 : 0), payload(3,response.message || ""));
    if (response.service_id) content.push(payload(5,response.service_id));
  }
  if (kind === 2) content.push(payload(2,response.data));
  if (kind === 3) content.push(uint(2,response.exit_code || 0));
  if (kind === 4) content.push(payload(2,response.message || ""));
  return payload(32,payload(kind,concat(...content)));
}
