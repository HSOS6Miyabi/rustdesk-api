"use strict";

// Acknowledge each decoded batch before the synchronous RGBA conversion.
// The decoder may invoke callbacks asynchronously; only the latest completed
// batch frame is painted and all callbacks must finish before acknowledgement.
export function decodeVp9Frames(frames,decoder,acknowledge,paint,now=()=>Date.now()){
  if(!Array.isArray(frames)||!frames.length||!decoder){
    acknowledge();
    return;
  }
  const started=now();
  const size=frames.reduce((total,frame)=>total+(frame.data?.byteLength||0),0);
  let pending=frames.length;
  for(const frame of frames){
    decoder.processFrame(frame.data.slice(0).buffer,ok=>{
      pending--;
      if(pending!==0)return;
      acknowledge();
      if(ok&&decoder.frameBuffer)paint(decoder.frameBuffer,now()-started,size);
    });
  }
}
