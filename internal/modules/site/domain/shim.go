package domain

import (
	"encoding/json"
	"strings"
)

// shimMarker tells Localize a page already carries the script.
const shimMarker = "data-site-shim"

// shimScript runs first in every page and gives the address back to links the page builds while it runs: the
// attributes and properties that load or open something, fetch, XMLHttpRequest and the history. It leaves a
// link alone when it is not a root link or already carries the address, so an app that knows its address works
// as before. The address is also in window.__SITE_BASE__, for a router that needs a base. Location.assign and
// replace cannot be wrapped by a page; apps use a relative address there. A page here has an opaque origin, so a
// request that asks for credentials would be refused by the browser against the wildcard CORS answer; fetch and
// XMLHttpRequest are made to ask for none (the proxy keeps the app's cookies for it instead).
const shimScript = `(function(){var P=__PREFIX__;
function fix(u){if(typeof u!=="string"||u.charAt(0)!=="/"||u.charAt(1)==="/")return u;
if(u===P||u.indexOf(P+"/")===0||u.indexOf(P+"?")===0||u.indexOf(P+"#")===0)return u;return P+u;}
window.__SITE_BASE__=P;
var names={href:1,src:1,action:1,poster:1,formaction:1,data:1};
var set=Element.prototype.setAttribute;
Element.prototype.setAttribute=function(n,v){return set.call(this,n,names[String(n).toLowerCase()]?fix(v):v);};
[[HTMLAnchorElement,"href"],[HTMLAreaElement,"href"],[HTMLLinkElement,"href"],[HTMLBaseElement,"href"],
[HTMLScriptElement,"src"],[HTMLImageElement,"src"],[HTMLIFrameElement,"src"],[HTMLSourceElement,"src"],
[HTMLMediaElement,"src"],[HTMLEmbedElement,"src"],[HTMLFormElement,"action"],[HTMLVideoElement,"poster"]]
.forEach(function(p){var d=Object.getOwnPropertyDescriptor(p[0].prototype,p[1]);
if(!d||!d.set)return;Object.defineProperty(p[0].prototype,p[1],{configurable:true,enumerable:d.enumerable,get:d.get,
set:function(v){d.set.call(this,fix(v));}});});
var f=window.fetch;
if(f)window.fetch=function(i,o){
if(o&&o.credentials==="include")o=Object.assign({},o,{credentials:"same-origin"});
else if(window.Request&&i instanceof Request&&i.credentials==="include")i=new Request(i,{credentials:"same-origin"});
return f.call(this,typeof i==="string"?fix(i):i,o);};
var wc=Object.getOwnPropertyDescriptor(XMLHttpRequest.prototype,"withCredentials");
if(wc&&wc.set)Object.defineProperty(XMLHttpRequest.prototype,"withCredentials",{configurable:true,
enumerable:wc.enumerable,get:wc.get,set:function(){wc.set.call(this,false);}});
var x=XMLHttpRequest.prototype.open;
XMLHttpRequest.prototype.open=function(m,u){var a=Array.prototype.slice.call(arguments);
a[1]=fix(String(u));return x.apply(this,a);};
["pushState","replaceState"].forEach(function(k){var h=history[k];
history[k]=function(s,t,u){return h.call(this,s,t,u==null?u:fix(String(u)));};});
var w=window.open;
window.open=function(u){var a=Array.prototype.slice.call(arguments);a[0]=fix(a[0]);return w.apply(this,a);};
if(window.EventSource){var E=window.EventSource;window.EventSource=function(u,o){return new E(fix(String(u)),o);};
window.EventSource.prototype=E.prototype;}
})();`

// shimTag is the script element for a page opened at prefix.
func shimTag(prefix string) string {
	quoted, _ := json.Marshal(prefix)

	return "<script " + shimMarker + ">" + strings.Replace(shimScript, "__PREFIX__", string(quoted), 1) + "</script>"
}
