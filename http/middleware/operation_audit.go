package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

const maxResponseSample = 4096

type auditResponseWriter struct {
	gin.ResponseWriter
	sample bytes.Buffer
}

func (w *auditResponseWriter) capture(b []byte) {
	if w.sample.Len() >= maxResponseSample {return}
	remaining := maxResponseSample - w.sample.Len()
	if len(b) > remaining {b = b[:remaining]}
	w.sample.Write(b)
}
func (w *auditResponseWriter) Write(b []byte) (int,error) {
	w.capture(b)
	return w.ResponseWriter.Write(b)
}
func (w *auditResponseWriter) WriteString(s string) (int,error) {
	w.capture([]byte(s))
	return w.ResponseWriter.WriteString(s)
}

// RecordOperations logs all authenticated mutations, including rejected attempts,
// without recording passwords, access tokens or request bodies.
func RecordOperations() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodPost,http.MethodPut,http.MethodDelete,http.MethodPatch:
		default:
			c.Next();return
		}
		user := service.AllService.UserService.CurUser(c)
		if user == nil || user.Id == 0 {c.Next();return}
		writer := &auditResponseWriter{ResponseWriter:c.Writer}
		c.Writer=writer
		c.Next()
		status := writer.Status()
		success := status >= 200 && status < 300
		var response struct{Code *int `json:"code"`}
		if json.Unmarshal(writer.sample.Bytes(),&response)==nil && response.Code!=nil {
			success=success && *response.Code==0
		}
		path:=c.FullPath()
		if path=="" {path=c.Request.URL.Path}
		agent:=c.Request.UserAgent()
		if len(agent)>255 {agent=agent[:255]}
		if len(path)>255 {path=path[:255]}
		entry:=&model.AuditOperation{
			UserId:user.Id,Account:user.Username,Method:strings.ToUpper(c.Request.Method),
			Path:path,ClientIp:c.ClientIP(),UserAgent:agent,
			HttpStatus:status,Success:success,
		}
		if err:=service.DB.Create(entry).Error;err!=nil && service.Logger!=nil {
			service.Logger.WithError(err).Warn("failed recording management operation")
		}
	}
}
