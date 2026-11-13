"use strict";

// The advertised configurations are verified by the browser before login.
// WebCodecs chooses hardware or software internally; the preference is not a
// guarantee that the device's hardware decoder will be used.
const codecCandidates = {
  vp9: [{codec:"vp09.00.10.08"}],
  h264: [{codec:"avc1.42001E"},{codec:"avc1.640028"}],
  h265: [{codec:"hev1.1.6.L93.B0"}],
  vp8: [{codec:"vp8"}],
  av1: [{codec:"av01.0.08M.08"}]
};

export async function discoverNativeVideoCodecs(Decoder=globalThis.VideoDecoder,
                                                 Chunk=globalThis.EncodedVideoChunk) {
  const supported={};
  if(typeof Decoder?.isConfigSupported!=="function"||typeof Chunk!=="function")
    return supported;
  for(const [name,candidates] of Object.entries(codecCandidates)){
    for(const candidate of candidates){
      try{
        const config={...candidate,hardwareAcceleration:"prefer-hardware",optimizeForLatency:true};
        if((await Decoder.isConfigSupported(config)).supported){
          supported[name]=config;
          break;
        }
      }catch(error){
        // A browser may reject a codec-specific configuration entirely.
      }
    }
  }
  return supported;
}

// A decoder processes every compressed frame, including frames that will not
// be displayed. Only completed VideoFrames may be discarded to control latency.
export class NativeVideoDecoder {
  constructor({codec,config,display,acknowledge,requestKeyFrame,onFailure,
               onDecoded=()=>{},onDropped=()=>{},
               Decoder=globalThis.VideoDecoder,Chunk=globalThis.EncodedVideoChunk,
               now=()=>performance.now(),schedule=callback=>queueMicrotask(callback)}){
    this.codec=codec;
    this.config=config;
    this.display=display;
    this.acknowledge=acknowledge;
    this.requestKeyFrame=requestKeyFrame;
    this.onFailure=onFailure;
    this.onDecoded=onDecoded;
    this.onDropped=onDropped;
    this.Decoder=Decoder;
    this.Chunk=Chunk;
    this.now=now;
    this.schedule=schedule;
    this.decoder=null;
    this.pending=new Map();
    this.batches=new Set();
    this.sequence=0;
    this.waitingForKey=true;
    this.keyRequested=false;
    this.softwareTried=false;
    this.failed=false;
    this.closed=false;
    this.paintScheduled=false;
    this.painting=false;
    this.nextFrame=null;
  }

  startDecoder(){
    const decoder=new this.Decoder({
      output:frame=>this.output(frame),
      error:error=>this.decoderFailed(error)
    });
    try{
      decoder.configure({...this.config,
        hardwareAcceleration:this.softwareTried?"prefer-software":"prefer-hardware"});
    }catch(error){
      try{decoder.close()}catch{}
      throw error;
    }
    this.decoder=decoder;
  }

  decode(frames){
    if(this.closed||this.failed){this.acknowledge();return}
    if(!Array.isArray(frames)||frames.length===0){this.acknowledge();return}
    if(this.waitingForKey){
      const firstKey=frames.findIndex(frame=>frame.key===true);
      if(firstKey<0){
        this.requestKey();
        this.acknowledge();
        return;
      }
      frames=frames.slice(firstKey);
      this.waitingForKey=false;
      this.keyRequested=false;
    }
    const batch={remaining:frames.length,acked:false,timer:null};
    this.batches.add(batch);
    batch.timer=setTimeout(()=>this.decoderFailed(new Error("Video decoder output timed out")),4000);
    try{
      if(!this.decoder)this.startDecoder();
      for(const frame of frames){
        const sequence=++this.sequence;
        const bytes=frame.data;
        this.pending.set(sequence,{batch,started:this.now(),bytes:bytes.byteLength});
        this.decoder.decode(new this.Chunk({
          type:frame.key?"key":"delta",
          timestamp:sequence,
          data:bytes
        }));
      }
    }catch(error){
      this.decoderFailed(error);
    }
  }

  requestKey(){
    if(this.keyRequested)return;
    this.keyRequested=true;
    this.requestKeyFrame();
  }

  output(frame){
    const item=this.pending.get(frame.timestamp);
    if(this.closed||!item){frame.close();return}
    this.pending.delete(frame.timestamp);
    item.batch.remaining--;
    const latency=this.now()-item.started;
    this.onDecoded({codec:this.codec,decodeMs:latency,bytes:item.bytes,
                    queueSize:this.decoder?.decodeQueueSize||0});
    if(item.batch.remaining===0)this.finishBatch(item.batch);
    if(this.nextFrame){
      this.nextFrame.frame.close();
      this.onDropped();
    }
    this.nextFrame={frame,latency,bytes:item.bytes};
    if(!this.paintScheduled&&!this.painting){
      this.paintScheduled=true;
      this.schedule(()=>this.paint());
    }
  }

  async paint(){
    this.paintScheduled=false;
    if(this.painting||this.closed)return;
    const item=this.nextFrame;
    this.nextFrame=null;
    if(!item)return;
    this.painting=true;
    let transferred=false;
    try{
      transferred=(await this.display(item.frame,item.latency,item.bytes))===true;
    }catch(error){
      this.decoderFailed(error);
    }finally{
      // Flutter's onVideoFrame queue owns transferred frames and closes them
      // after GPU import. The RGBA compatibility path keeps local ownership.
      if(!transferred)item.frame.close();
      this.painting=false;
      if(this.nextFrame&&!this.closed&&!this.paintScheduled){
        this.paintScheduled=true;
        this.schedule(()=>this.paint());
      }
    }
  }

  finishBatch(batch){
    if(batch.acked)return;
    batch.acked=true;
    clearTimeout(batch.timer);
    this.batches.delete(batch);
    this.acknowledge();
  }

  releasePending(){
    this.pending.clear();
    for(const batch of [...this.batches])this.finishBatch(batch);
    if(this.nextFrame){this.nextFrame.frame.close();this.nextFrame=null}
  }

  decoderFailed(error){
    if(this.closed||this.failed)return;
    this.releasePending();
    if(this.decoder){
      try{this.decoder.close()}catch{}
      this.decoder=null;
    }
    this.waitingForKey=true;
    this.keyRequested=false;
    if(!this.softwareTried){
      this.softwareTried=true;
      try{
        this.startDecoder();
        this.requestKey();
        return;
      }catch(softwareError){
        error=softwareError;
      }
    }
    this.failed=true;
    this.onFailure(error);
  }

  close(){
    if(this.closed)return;
    this.closed=true;
    this.releasePending();
    if(this.decoder){try{this.decoder.close()}catch{}this.decoder=null}
  }

  get queueSize(){return this.decoder?.decodeQueueSize||0}
}
