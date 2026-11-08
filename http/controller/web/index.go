package web

import (
	"encoding/json"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
)

type Index struct {}

func (i *Index) Index(c *gin.Context) {
	c.Redirect(302, "/_admin/")
}

// ConfigJs delivers only public browser connection settings.
func (i *Index) ConfigJs(c *gin.Context) {
	values, err := json.Marshal(gin.H{
		"api_server": global.Config.Rustdesk.ApiServer,
		"ws_host": global.Config.Rustdesk.WsHost,
		"magic_queryonline": global.Config.Rustdesk.WebclientMagicQueryonline,
	})
	if err != nil {
		c.Status(500)
		return
	}
	script := fmt.Sprintf(`(function () {
		const settings = %s;
		localStorage.setItem("api-server", settings.api_server);
		localStorage.setItem("wc-api-server", settings.api_server);
		window.webclient_magic_queryonline = settings.magic_queryonline;
		window.ws_host = settings.ws_host;
	})();`, string(values))
	c.Header("Content-Type", "application/javascript; charset=utf-8")
	c.Header("Cache-Control", "no-store")
	c.String(200, script)
}
