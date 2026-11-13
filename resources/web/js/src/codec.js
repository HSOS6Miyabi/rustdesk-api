// example: https://github.com/rgov/js-theora-decoder/blob/main/index.html
// https://github.com/brion/ogv.js/releases, yarn add has no simd
// dev: copy decoder files from node/ogv/dist/* to project dir
// dist: .... to dist
/*
  OGVDemuxerOggW: 'ogv-demuxer-ogg-wasm.js',
  OGVDemuxerWebMW: 'ogv-demuxer-webm-wasm.js',
  OGVDecoderAudioOpusW: 'ogv-decoder-audio-opus-wasm.js',
  OGVDecoderAudioVorbisW: 'ogv-decoder-audio-vorbis-wasm.js',
  OGVDecoderVideoTheoraW: 'ogv-decoder-video-theora-wasm.js',
  OGVDecoderVideoVP8W: 'ogv-decoder-video-vp8-wasm.js',
  OGVDecoderVideoVP8MTW: 'ogv-decoder-video-vp8-mt-wasm.js',
  OGVDecoderVideoVP9W: 'ogv-decoder-video-vp9-wasm.js',
  OGVDecoderVideoVP9SIMDW: 'ogv-decoder-video-vp9-simd-wasm.js',
  OGVDecoderVideoVP9MTW: 'ogv-decoder-video-vp9-mt-wasm.js',
  OGVDecoderVideoVP9SIMDMTW: 'ogv-decoder-video-vp9-simd-mt-wasm.js',
  OGVDecoderVideoAV1W: 'ogv-decoder-video-av1-wasm.js',
  OGVDecoderVideoAV1SIMDW: 'ogv-decoder-video-av1-simd-wasm.js',
  OGVDecoderVideoAV1MTW: 'ogv-decoder-video-av1-mt-wasm.js',
  OGVDecoderVideoAV1SIMDMTW: 'ogv-decoder-video-av1-simd-mt-wasm.js',
*/
import { simd } from "wasm-feature-detect";

export async function loadVp9(callback) {
  // RustDesk's browser decoder uses single-threaded VP9 workers. Modern
  // browsers require cross-origin isolation for multi-threaded WebAssembly.
  const supportsSIMD=await simd().catch(()=>false);
  const candidates=supportsSIMD?["OGVDecoderVideoVP9SIMDW","OGVDecoderVideoVP9W"]:["OGVDecoderVideoVP9W"];
  let lastError;
  for(const className of candidates){
    for(const worker of [true,false]){
      try{
        const decoder=await new Promise((resolve,reject)=>{
          if(!window.OGVLoader?.loadClass){
            reject(new Error("VP9 decoder library was not loaded"));
            return;
          }
          window.OGVLoader.loadClass(className,(Decoder)=>{
            if(typeof Decoder!=="function"){
              reject(new Error("VP9 decoder module is unavailable"));
              return;
            }
            Promise.resolve(Decoder({videoFormat:{}})).then(instance=>{
              if(!instance||typeof instance.init!=="function"){
                reject(new Error("VP9 decoder initialization is unavailable"));
                return;
              }
              instance.init(()=>resolve(instance));
            }).catch(reject);
          },{worker,threading:false});
        });
        callback(decoder);
        return;
      }catch(error){
        lastError=error;
      }
    }
  }
  throw lastError||new Error("VP9 decoder could not be initialized");
}
