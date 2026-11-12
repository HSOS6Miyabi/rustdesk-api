package router

import (
 "crypto/sha256"
 "encoding/hex"
 "errors"
 "net/http"
 "os"
 "path/filepath"
 "strings"

 "github.com/gin-gonic/gin"
)

var consoleAssets = []string{"style.css", "client-local.js", "passkey.js", "app.js"}

func consoleFileHash(content []byte) string {
 digest := sha256.Sum256(content)
 return hex.EncodeToString(digest[:])[:20]
}

func readConsoleAssets(root string) (map[string][]byte, map[string]string, string, error) {
 contents := make(map[string][]byte, len(consoleAssets))
 hashes := make(map[string]string, len(consoleAssets))
 var revision strings.Builder
 for _, name := range consoleAssets {
  data, err := os.ReadFile(filepath.Join(root, name))
  if err != nil { return nil, nil, "", err }
  contents[name] = data
  hashes[name] = consoleFileHash(data)
  revision.WriteString(hashes[name])
 }
 return contents, hashes, consoleFileHash([]byte(revision.String())), nil
}

func serveConsoleIndex(c *gin.Context, root string) {
 template, err := os.ReadFile(filepath.Join(root, "index.html"))
 if err != nil { c.Status(http.StatusInternalServerError); return }
 _, hashes, revision, err := readConsoleAssets(root)
 if err != nil { c.Status(http.StatusInternalServerError); return }
 document := string(template)
 for _, name := range consoleAssets {
  plain := "/_admin/" + name
  addressed := "/_admin/assets/" + hashes[name] + "/" + name
  document = strings.ReplaceAll(document, plain, addressed)
 }
 document = strings.Replace(document, "</head>",
  `<meta name="rustdesk-console-revision" content="`+revision+`">`+"\n</head>", 1)
 payload := []byte(document)
 etag := `"` + consoleFileHash(payload) + `"`
 c.Header("Cache-Control", "private, max-age=0, must-revalidate")
 c.Header("ETag", etag)
 c.Header("X-RustDesk-Console-Revision", revision)
 if c.GetHeader("If-None-Match") == etag { c.Status(http.StatusNotModified); return }
 c.Data(http.StatusOK, "text/html; charset=utf-8", payload)
}

func consoleResourceHandler(root string) gin.HandlerFunc {
 return func(c *gin.Context) {
  name := strings.TrimPrefix(c.Param("filepath"), "/")
  if name == "" || name == "index.html" { serveConsoleIndex(c, root); return }
  if strings.HasPrefix(name, "assets/") {
   fragments := strings.Split(name, "/")
   if len(fragments) != 3 { c.Status(http.StatusNotFound); return }
   expectedHash, assetName := fragments[1], fragments[2]
   valid := false
   for _, allowed := range consoleAssets { if assetName == allowed { valid = true; break } }
   if !valid { c.Status(http.StatusNotFound); return }
   content, err := os.ReadFile(filepath.Join(root, assetName))
   if err != nil {
    if errors.Is(err, os.ErrNotExist) { c.Status(http.StatusNotFound) } else { c.Status(http.StatusInternalServerError) }
    return
   }
   actualHash := consoleFileHash(content)
   if expectedHash != actualHash { c.Status(http.StatusNotFound); return }
   etag := `"` + actualHash + `"`
   c.Header("Cache-Control", "public, max-age=31536000, immutable")
   c.Header("ETag", etag)
   c.Header("X-Content-Type-Options", "nosniff")
   if c.GetHeader("If-None-Match") == etag { c.Status(http.StatusNotModified); return }
   contentType := "application/javascript; charset=utf-8"
   if filepath.Ext(assetName) == ".css" { contentType = "text/css; charset=utf-8" }
   c.Data(http.StatusOK, contentType, content)
   return
  }
  c.Header("Cache-Control", "public, max-age=3600, must-revalidate")
  http.StripPrefix("/_admin/", http.FileServer(http.Dir(root))).ServeHTTP(c.Writer, c.Request)
 }
}
