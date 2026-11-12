(function(){
  "use strict";
  const decode=source=>{
    const normalized=String(source).replace(/-/g,"+").replace(/_/g,"/");
    const value=atob(normalized+"=".repeat((4-normalized.length%4)%4));
    return Uint8Array.from(value,char=>char.charCodeAt(0));
  };
  const encode=value=>{
    const bytes=new Uint8Array(value);
    let binary="";
    for(const byte of bytes)binary+=String.fromCharCode(byte);
    return btoa(binary).replace(/\+/g,"-").replace(/\//g,"_").replace(/=+$/,"");
  };
  function supported(){
    return !!(window.PublicKeyCredential&&navigator.credentials&&
      (window.isSecureContext||["localhost","127.0.0.1"].includes(window.location.hostname)));
  }
  function registrationOptions(data){
    const json=data?.publicKey||data;
    if(typeof PublicKeyCredential.parseCreationOptionsFromJSON==="function"){
      return PublicKeyCredential.parseCreationOptionsFromJSON(json);
    }
    return {...json,challenge:decode(json.challenge),
      user:{...json.user,id:decode(json.user.id)},
      excludeCredentials:(json.excludeCredentials||[]).map(item=>({...item,id:decode(item.id)}))};
  }
  function authenticationOptions(data){
    const json=data?.publicKey||data;
    if(typeof PublicKeyCredential.parseRequestOptionsFromJSON==="function"){
      return PublicKeyCredential.parseRequestOptionsFromJSON(json);
    }
    return {...json,challenge:decode(json.challenge),
      allowCredentials:(json.allowCredentials||[]).map(item=>({...item,id:decode(item.id)}))};
  }
  function serialize(credential){
    if(typeof credential.toJSON==="function")return credential.toJSON();
    const response=credential.response;
    const result={
      id:credential.id,
      rawId:encode(credential.rawId),
      type:"public-key",
      clientExtensionResults:credential.getClientExtensionResults?.()||{},
    };
    if(response.attestationObject){
      result.response={
        clientDataJSON:encode(response.clientDataJSON),
        attestationObject:encode(response.attestationObject),
        transports:response.getTransports?.()||[],
      };
    }else{
      result.response={
        clientDataJSON:encode(response.clientDataJSON),
        authenticatorData:encode(response.authenticatorData),
        signature:encode(response.signature),
        userHandle:response.userHandle?encode(response.userHandle):null,
      };
    }
    return result;
  }
  async function create(options){
    if(!supported())throw Error("需要使用 HTTPS 和支持 Passkey 的浏览器");
    const result=await navigator.credentials.create({publicKey:registrationOptions(options)});
    if(!result)throw Error("尚未创建 Passkey");
    return serialize(result);
  }
  async function get(options){
    if(!supported())throw Error("需要使用 HTTPS 和支持 Passkey 的浏览器");
    const result=await navigator.credentials.get({publicKey:authenticationOptions(options)});
    if(!result)throw Error("尚未选择 Passkey");
    return serialize(result);
  }
  window.RustDeskPasskeys={supported,create,get};
})();
