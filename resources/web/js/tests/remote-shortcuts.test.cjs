"use strict";
const test=require("node:test");
const assert=require("node:assert/strict");
const path=require("node:path");
const {pathToFileURL}=require("node:url");
const load=()=>import(pathToFileURL(path.join(__dirname,"../src/remote_shortcuts.mjs")).href);

function capture(active=true){
  const listeners={};
  const sends=[],pastes=[];
  const target={addEventListener(type,fn){listeners[type]=fn;},removeEventListener(){}};
  return {listeners,sends,pastes,target,install:async()=>{
    const {installRemoteShortcutCapture}=await load();
    return installRemoteShortcutCapture({
      target,isActive:()=>active,
      send:(name,down,event)=>sends.push([name,down,!!event.ctrlKey,!!event.shiftKey]),
      paste:event=>pastes.push(event.code)
    });
  }};
}
function key(key,code,modifiers={}){
  return {key,code,ctrlKey:false,metaKey:false,altKey:false,shiftKey:false,
    ...modifiers,preventDefault(){this.prevented=true;},
    stopImmediatePropagation(){this.stopped=true;}};
}
test("Ctrl+C and Ctrl+Shift+Z are captured and forwarded with matching key releases",async()=>{
  const c=capture();await c.install();
  const copy=key("c","KeyC",{ctrlKey:true});
  c.listeners.keydown(copy);
  c.listeners.keyup(copy);
  assert.equal(copy.prevented,true);
  assert.equal(copy.stopped,true);
  const redo=key("Z","KeyZ",{ctrlKey:true,shiftKey:true});
  c.listeners.keydown(redo);c.listeners.keyup(redo);
  assert.deepEqual(c.sends,[["c",true,true,false],["c",false,true,false],
    ["z",true,true,true],["z",false,true,true]]);
});
test("Ctrl+V lets the paste event occur while intercepting Flutter's keyboard handler",async()=>{
  const c=capture();await c.install();
  const v=key("v","KeyV",{ctrlKey:true});
  c.listeners.keydown(v);c.listeners.keyup(v);
  assert.equal(v.stopped,true);
  assert.equal(v.prevented,true); // keyup is cancelled, keydown remains available to paste
  assert.deepEqual(c.pastes,["KeyV"]);
  assert.deepEqual(c.sends,[]);
});
test("shortcuts in editable fields or outside the remote desktop remain local",async()=>{
  const c=capture(false);await c.install();
  c.listeners.keydown(key("c","KeyC",{ctrlKey:true}));
  assert.deepEqual(c.sends,[]);
  const active=capture();await active.install();
  active.listeners.keydown(key("c","KeyC",{ctrlKey:true,
    target:{closest:selector=>selector.includes("input")?{}:null}}));
  assert.deepEqual(active.sends,[]);
});


test("shortcuts targeting Flutter's hidden text editor reach the remote connection",async()=>{
  const c=capture();await c.install();
  const hiddenFlutterInput={
    closest(selector){
      if(selector.includes("flt-text-editing-host"))return {tagName:"FLT-TEXT-EDITING-HOST"};
      if(selector.includes("input"))return {tagName:"INPUT"};
      return null;
    },isContentEditable:false
  };
  const e=key("c","KeyC",{ctrlKey:true,target:hiddenFlutterInput});
  c.listeners.keydown(e);c.listeners.keyup(e);
  assert.deepEqual(c.sends,[["c",true,true,false],["c",false,true,false]]);
  assert.equal(e.stopped,true);
});


test("plain characters, spaces, arrows, function keys and modifiers reach the remote session",async()=>{
  const c=capture();await c.install();
  for(const value of [
    key("a","KeyA"),key(" ","Space"),key("ArrowLeft","ArrowLeft"),
    key("Backspace","Backspace"),key("F5","F5"),
    key("Shift","ShiftLeft",{shiftKey:true})]){
    c.listeners.keydown(value);
    c.listeners.keyup(value);
    assert.equal(value.prevented,true);
  }
  assert.deepEqual(c.sends.map(x=>[x[0],x[1]]),[
    ["a",true],["a",false],["Space",true],["Space",false],
    ["LeftArrow",true],["LeftArrow",false],
    ["Backspace",true],["Backspace",false],
    ["F5",true],["F5",false],["Shift",true],["Shift",false]
  ]);
});

test("keyboard repeat and blur release keys rather than leaving modifiers held",async()=>{
  const c=capture();await c.install();
  c.listeners.keydown(key("Control","ControlLeft",{ctrlKey:true}));
  c.listeners.keydown(key("a","KeyA",{ctrlKey:true}));
  c.listeners.keydown(key("a","KeyA",{ctrlKey:true,repeat:true}));
  c.listeners.blur();
  assert.deepEqual(c.sends.map(x=>[x[0],x[1]]),[
    ["Control",true],["a",true],["a",true],["Control",false],["a",false]
  ]);
});

test("composition commits Unicode text once and keeps local inputs editable",async()=>{
  const listeners={},text=[];
  const target={addEventListener(name,fn){listeners[name]=fn;},removeEventListener(){}};
  const {installRemoteShortcutCapture}=await load();
  installRemoteShortcutCapture({target,isActive:()=>true,send:()=>{},
    paste:()=>{},sendText:value=>text.push(value)});
  const flutter={closest:selector=>selector.includes("flt-text-editing-host")?{}:null};
  listeners.compositionstart({target:flutter,stopImmediatePropagation(){}});
  listeners.beforeinput({target:flutter,inputType:"insertCompositionText",data:"测",preventDefault(){}});
  listeners.compositionend({target:flutter,data:"测试",stopImmediatePropagation(){}});
  listeners.beforeinput({target:flutter,inputType:"insertFromComposition",data:"测试",
    preventDefault(){this.prevented=true;},stopImmediatePropagation(){}});
  assert.deepEqual(text,["测试"]);
  listeners.beforeinput({target:{closest:selector=>selector.includes("input")?{}:null},
    inputType:"insertText",data:"local"});
  assert.deepEqual(text,["测试"]);
});

test("inactive or locally editable focus preserves native keyboard input",async()=>{
  const c=capture();await c.install();
  const local=key("a","KeyA",{target:{closest:selector=>selector.includes("input")?{}:null}});
  c.listeners.keydown(local);
  c.listeners.keyup(local);
  assert.equal(local.prevented,undefined);
  assert.deepEqual(c.sends,[]);
});

test("direct non-ASCII keyboard characters use text input instead of physical key codes",async()=>{
  const listeners={},typed=[],keys=[];
  const {installRemoteShortcutCapture}=await load();
  const target={addEventListener(type,callback){listeners[type]=callback;},removeEventListener(){}};
  installRemoteShortcutCapture({target,isActive:()=>true,
    send:(...args)=>keys.push(args),
    sendText:value=>typed.push(value),paste(){}});
  const event=key("中","Unidentified",{});
  listeners.keydown(event);
  listeners.keyup(event);
  assert.deepEqual(typed,["中"]);
  assert.deepEqual(keys,[]);
});
