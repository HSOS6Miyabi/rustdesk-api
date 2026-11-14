package router

import (
    "net/http"
    "path/filepath"

    "github.com/gin-gonic/gin"
    "github.com/lejianwen/rustdesk-api/v2/global"
    "github.com/lejianwen/rustdesk-api/v2/http/controller/web"
    "github.com/lejianwen/rustdesk-api/v2/http/middleware"
)

func WebInit(g *gin.Engine) {
    index := &web.Index{}
    g.GET("/", index.Index)

    if global.Config.App.WebClient == 1 {
        protected := middleware.WebClientSessionAuth()
        g.GET("/webclient-config/index.js", protected, index.ConfigJs)
        g.Group("/webclient", protected).StaticFS("/", http.Dir(
            filepath.Join(global.Config.Gin.ResourcesPath, "web")))
        g.Group("/webclient2", protected).StaticFS("/", http.Dir(
            filepath.Join(global.Config.Gin.ResourcesPath, "web2")))
    }

    console := consoleResourceHandler(filepath.Join(global.Config.Gin.ResourcesPath, "admin"))
    g.GET("/_admin/*filepath", console)
    g.HEAD("/_admin/*filepath", console)
}
