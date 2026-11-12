package router

import (
 "net/http"
 "net/http/httptest"
 "os"
 "path/filepath"
 "regexp"
 "strings"
 "testing"

 "github.com/gin-gonic/gin"
)

func TestConsoleAssetsChangeURLsAndRemainCacheable(t *testing.T) {
 gin.SetMode(gin.TestMode)
 root := t.TempDir()
 files := map[string]string{
  "index.html": `<html><head><link rel="stylesheet" href="/_admin/style.css"><script src="/_admin/client-local.js" defer></script><script src="/_admin/passkey.js" defer></script><script src="/_admin/app.js" defer></script></head><body></body></html>`,
  "style.css": ".base{color:red}", "app.js": "console.log('first');",
  "passkey.js": "console.log('passkey');", "client-local.js": "console.log('builder');",
  "favicon.svg": `<svg xmlns="http://www.w3.org/2000/svg"></svg>`,
 }
 for name, body := range files {
  if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0644); err != nil { t.Fatal(err) }
 }
 g := gin.New()
 h := consoleResourceHandler(root)
 g.GET("/_admin/*filepath", h)
 g.HEAD("/_admin/*filepath", h)
 call := func(url string, etag string) *httptest.ResponseRecorder {
  req:=httptest.NewRequest(http.MethodGet,url,nil)
  if etag!="" { req.Header.Set("If-None-Match",etag) }
  res:=httptest.NewRecorder()
  g.ServeHTTP(res,req)
  return res
 }
 index := call("/_admin/", "")
 if index.Code!=200 { t.Fatalf("console index: %d",index.Code) }
 if !strings.Contains(index.Header().Get("Cache-Control"),"must-revalidate") {t.Fatalf("index must be revalidated: %v",index.Header())}
 if strings.Contains(index.Header().Get("Cache-Control"),"no-store") {t.Fatal("index must remain cacheable")}
 if index.Header().Get("X-RustDesk-Console-Revision")=="" || !strings.Contains(index.Body.String(), `name="rustdesk-console-revision"`) {t.Fatal("missing resource revision")}
 assetPattern:=regexp.MustCompile(`/_admin/assets/[0-9a-f]{20}/app\.js`)
 oldURL:=assetPattern.FindString(index.Body.String())
 if oldURL=="" {t.Fatalf("app.js lacks fingerprinted URL: %s",index.Body.String())}
 cache := call(oldURL,"")
 if cache.Code!=200 || cache.Body.String()!=files["app.js"] {t.Fatalf("versioned asset response: %d %q",cache.Code,cache.Body.String())}
 if !strings.Contains(cache.Header().Get("Cache-Control"),"immutable") || !strings.Contains(cache.Header().Get("Cache-Control"),"31536000") {t.Fatalf("asset must retain long-lived cache: %v",cache.Header())}
 if call(oldURL,cache.Header().Get("ETag")).Code!=http.StatusNotModified {t.Fatal("asset ETag revalidation failed")}
 if call("/_admin/",index.Header().Get("ETag")).Code!=http.StatusNotModified {t.Fatal("index ETag revalidation failed")}
 if err:=os.WriteFile(filepath.Join(root,"app.js"),[]byte("console.log('second');"),0644);err!=nil{t.Fatal(err)}
 updated:=call("/_admin/",index.Header().Get("ETag"))
 if updated.Code!=200 {t.Fatalf("updated index must be delivered: %d",updated.Code)}
 newURL:=assetPattern.FindString(updated.Body.String())
 if newURL==oldURL || newURL=="" {t.Fatalf("asset URL did not change: %q => %q",oldURL,newURL)}
 if call(oldURL,"").Code!=http.StatusNotFound {t.Fatal("old fingerprint must not identify changed bytes")}
 if response:=call(newURL,""); response.Code!=200 || response.Body.String()!="console.log('second');" {t.Fatalf("new asset was not served: %d",response.Code)}
 if response:=call("/_admin/favicon.svg","");response.Code!=200 {t.Fatalf("other static files must remain accessible: %d",response.Code)}
 if response:=call("/_admin/index.html","");response.Code!=200 {t.Fatalf("index.html must be served: %d",response.Code)}
}
