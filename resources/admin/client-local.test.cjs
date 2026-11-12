"use strict";

const test = require("node:test");
const assert = require("node:assert/strict");
const builder = require("./client-local.js");

const settings = {
  host:"desk.example.test:21116",
  public_key:Buffer.alloc(32,7).toString("base64"),
  api_server:"https://desk.example.test",
  relay_server:"relay.example.test:21117"
};

function file(name,content) {
  return new File([content],name,{type:"application/octet-stream"});
}

async function zipEntries(blob) {
  const bytes=Buffer.from(await blob.arrayBuffer());
  const entries=new Map();
  let pos=0;
  while(pos<bytes.length && bytes.readUInt32LE(pos)===0x04034b50) {
    const method=bytes.readUInt16LE(pos+8);
    assert.equal(method,0);
    const size=bytes.readUInt32LE(pos+18);
    const nameLength=bytes.readUInt16LE(pos+26);
    const extraLength=bytes.readUInt16LE(pos+28);
    const name=bytes.subarray(pos+30,pos+30+nameLength).toString("utf8");
    const content=bytes.subarray(pos+30+nameLength+extraLength,
      pos+30+nameLength+extraLength+size);
    assert.equal(await builder.crc32(new Blob([content])),bytes.readUInt32LE(pos+14));
    entries.set(name,Buffer.from(content));
    pos+=30+nameLength+extraLength+size;
  }
  assert.ok(entries.size>0);
  assert.equal(bytes.readUInt32LE(pos),0x02014b50);
  assert.equal(bytes.readUInt32LE(bytes.length-22),0x06054b50);
  return entries;
}

test("local Windows installer keeps its exact executable bytes and embeds server configuration",async()=>{
  const original=Buffer.concat([Buffer.from("MZ"),Buffer.alloc(256,23)]);
  const input=file("rustdesk-example.exe",original);
  for(const platform of ["windows-x64","windows-arm64"]) {
    const result=await builder.build(input,platform,settings);
    assert.match(result.name,/^rustdesk-licensed-[A-Za-z0-9_-]+\.exe$/);
    assert.deepEqual(Buffer.from(await result.blob.arrayBuffer()),original);
    const name=result.name.replace(/^rustdesk-licensed-/,"").replace(/\.exe$/,"");
    const originalConfig=Buffer.from(name.split("").reverse().join(""),"base64url").toString("utf8");
    assert.deepEqual(JSON.parse(originalConfig),{
      host:settings.host,key:settings.public_key,
      api:settings.api_server,relay:settings.relay_server
    });
  }
});

test("local Linux and macOS ZIP archives contain the unchanged upload and configuration script",async()=>{
  const cases=[
    {platform:"linux-x64",file:"rustdesk-1.3.9-x86_64.deb",prefix:"!<arch>\n",script:"install-and-configure.sh"},
    {platform:"linux-arm64",file:"rustdesk-1.5.3-aarch64.deb",prefix:"!<arch>\n",script:"install-and-configure.sh"},
    {platform:"macos-x64",file:"rustdesk-1.4.5-x86_64.dmg",prefix:"test",script:"configure-rustdesk.command"},
    {platform:"macos-arm64",file:"rustdesk-1.5.3-aarch64.dmg",prefix:"test",script:"configure-rustdesk.command"}
  ];
  for(const tc of cases){
    const original=Buffer.concat([Buffer.from(tc.prefix),Buffer.alloc(256,42)]);
    const result=await builder.build(file(tc.file,original),tc.platform,settings);
    assert.ok(result.archive);
    const entries=await zipEntries(result.blob);
    assert.deepEqual(entries.get(tc.file),original);
    const script=entries.get(tc.script)?.toString("utf8");
    assert.ok(script,tc.platform+" script missing");
    for(const option of ["custom-rendezvous-server","key","api-server","relay-server"]){
      assert.match(script,new RegExp("--option "+option));
      assert.match(script,new RegExp("ACTUAL=.*--option "+option));
    }
    assert.match(entries.get("README.txt").toString("utf8"),/没有计算 SHA-256/);
  }
});

test("local package generation rejects invalid inputs and does not access the network",async()=>{
  const valid=file("rustdesk.exe",Buffer.concat([Buffer.from("MZ"),Buffer.alloc(256)]));
  const cases=[
    [valid,"android",settings],
    [file("rustdesk.deb",Buffer.alloc(256)),"linux-x64",settings],
    [file("rustdesk.exe",Buffer.alloc(256)),"windows-x64",settings],
    [file("rustdesk.exe",Buffer.concat([Buffer.from("MZ"),Buffer.alloc(256)])),"linux-x64",settings],
    [valid,"windows-x64",{...settings,host:"desk.example.test\nunsafe"}],
    [valid,"windows-x64",{...settings,public_key:"bad"}],
    [valid,"windows-x64",{...settings,api_server:"https://user:password@example.test"}],
    [valid,"windows-x64",{...settings,relay_server:"relay.example.test;rm -rf"}],
  ];
  for(const [upload,platform,config] of cases)await assert.rejects(builder.build(upload,platform,config));
  assert.equal(await builder.crc32(new Blob(["123456789"])),0xcbf43926);
  const short=file("rustdesk.exe",Buffer.from("MZ"));
  await assert.rejects(builder.build(short,"windows-x64",settings));
});
