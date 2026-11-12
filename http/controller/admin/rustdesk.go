package admin

import (
    "errors"
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/request/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type Rustdesk struct {
}

type RustdeskCmd struct {
	Cmd    string `json:"cmd"`
	Option string `json:"option"`
	Target string `json:"target"`
}

func (r *Rustdesk) CmdList(c *gin.Context) {
	q := &admin.PageQuery{}
	if err := c.ShouldBindQuery(q); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	res := service.AllService.ServerCmdService.List(q.Page, 9999)
	//在列表前添加系统命令
	list := make([]*model.ServerCmd, 0)
	list = append(list, model.SysIdServerCmds...)
	list = append(list, model.SysRelayServerCmds...)
	list = append(list, res.ServerCmds...)
	res.ServerCmds = list
	response.Success(c, res)
}

func (r *Rustdesk) CmdDelete(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	if f.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError"))
		return
	}

	ex := service.AllService.ServerCmdService.Info(f.Id)
	if ex.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}

	err := service.AllService.ServerCmdService.Delete(ex)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}
func (r *Rustdesk) CmdCreate(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	err := service.AllService.ServerCmdService.Create(f)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func (r *Rustdesk) CmdUpdate(c *gin.Context) {
	f := &model.ServerCmd{}
	if err := c.ShouldBindJSON(f); err != nil {
		response.Fail(c, 101, response.TranslateMsg(c, "ParamsError")+err.Error())
		return
	}
	errList := global.Validator.ValidStruct(c, f)
	if len(errList) > 0 {
		response.Fail(c, 101, errList[0])
		return
	}
	ex := service.AllService.ServerCmdService.Info(f.Id)
	if ex.Id == 0 {
		response.Fail(c, 101, response.TranslateMsg(c, "ItemNotFound"))
		return
	}
	err := service.AllService.ServerCmdService.Update(f)
	if err != nil {
		response.Fail(c, 101, err.Error())
		return
	}
	response.Success(c, nil)
}

func executeRustdeskCommand(command *RustdeskCmd) (string, error) {
    if command.Cmd == "" || command.Target == "" {
        return "", errors.New("请选择服务器和管理操作")
    }
    if err := validateRustdeskCommand(command.Target, command.Cmd, command.Option); err != nil {
        return "", err
    }
    port := 0
    switch command.Target {
    case model.ServerCmdTargetIdServer:
        port = global.Config.Admin.IdServerPort - 1
    case model.ServerCmdTargetRelayServer:
        port = global.Config.Admin.RelayServerPort
    default:
        return "", errors.New("服务器类型无效")
    }
    return service.AllService.ServerCmdService.SendCmd(port, command.Cmd, command.Option)
}

func (r *Rustdesk) SendCmd(c *gin.Context) {
    command := &RustdeskCmd{}
    if err := c.ShouldBindJSON(command); err != nil {
        response.Fail(c, 101, "请求参数格式无效")
        return
    }
    output, err := executeRustdeskCommand(command)
    if err != nil {
        response.Fail(c, 101, err.Error())
        return
    }
    response.Success(c, output)
}

// ExecuteCmd returns parsed data for the graphical console.
func (r *Rustdesk) ExecuteCmd(c *gin.Context) {
    command := &RustdeskCmd{}
    if err := c.ShouldBindJSON(command); err != nil {
        response.Fail(c, 101, "请求参数格式无效")
        return
    }
    output, err := executeRustdeskCommand(command)
    if err != nil {
        response.Fail(c, 101, err.Error())
        return
    }
    response.Success(c, parseRustdeskReply(command.Target, command.Cmd, command.Option, output))
}
