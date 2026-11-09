package router

import (
	"github.com/gin-gonic/gin"
	_ "github.com/lejianwen/rustdesk-api/v2/docs/admin"
	"github.com/lejianwen/rustdesk-api/v2/global"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/admin/my"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func Init(g *gin.Engine) {

	//swagger
	//g.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	if global.Config.App.ShowSwagger == 1 {
		g.GET("/admin/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.InstanceName("admin")))
	}

	adg := g.Group("/api/admin")
	LoginBind(adg)
	adg.POST("/user/register", (&admin.User{}).Register)

	ConfigBind(adg)

	adg.Use(middleware.BackendUserAuth(),middleware.RecordOperations())
	//FileBind(adg)
	UserBind(adg)
	TOTPBind(adg)
	GroupBind(adg)
	TagBind(adg)
	AddressBookBind(adg)
	PeerBind(adg)
	admin.DeviceDeploymentBind(adg)
	DeviceSessionsBind(adg)
	APITokenBind(adg)
	MailBind(adg)
	InvitationBind(adg)
	OauthBind(adg)
	LoginLogBind(adg)
	AuditBind(adg)
	AddressBookCollectionBind(adg)
	AddressBookCollectionRuleBind(adg)
	UserTokenBind(adg)

	//deprecated by ConfigBind
	//rs := &admin.Rustdesk{}
	//adg.GET("/server-config", rs.ServerConfig)
	//adg.GET("/app-config", rs.AppConfig)
	//deprecated end

	ShareRecordBind(adg)
	MyBind(adg)

	RustdeskCmdBind(adg)
	DeviceGroupBind(adg)
	ContactBind(adg)
	SharedDevicesBind(adg)
	PersonalGroupBind(adg)
	MyAuditBind(adg)
	AuditManagementBind(adg)
	DevicePolicyBind(adg)
	//访问静态文件
	//g.StaticFS("/upload", http.Dir(global.Config.Gin.ResourcesPath+"/upload"))
}

func RustdeskCmdBind(adg *gin.RouterGroup) {
	cont := &admin.Rustdesk{}
	rg := adg.Group("/rustdesk")
	rg.POST("/sendCmd", cont.SendCmd)
	rg.GET("/cmdList", cont.CmdList)
	rg.POST("/cmdDelete", cont.CmdDelete)
	rg.POST("/cmdCreate", cont.CmdCreate)
}
func LoginBind(rg *gin.RouterGroup) {
	cont := &admin.Login{}
	rg.POST("/login", cont.Login)
	rg.POST("/password/forgot", (&admin.PasswordRecovery{}).Request)
	rg.POST("/password/reset", (&admin.PasswordRecovery{}).Complete)
	rg.POST("/invite/accept", (&admin.Invitation{}).Accept)
	rg.GET("/captcha", cont.Captcha)
	rg.POST("/logout", cont.Logout)
	rg.GET("/login-options", cont.LoginOptions)
	rg.POST("/oidc/auth", cont.OidcAuth)
	rg.GET("/oidc/auth-query", cont.OidcAuthQuery)
}

func UserBind(rg *gin.RouterGroup) {
	aR := rg.Group("/user")
	{
		cont := &admin.User{}
		aR.GET("/current", cont.Current)
		aR.POST("/changeCurPwd", cont.ChangeCurPwd)
		aR.POST("/myOauth", cont.MyOauth)
		//aR.GET("/myPeer", cont.MyPeer)
		aR.POST("/groupUsers", cont.GroupUsers)
	}
	aRP := rg.Group("/user").Use(middleware.AdminPrivilege())
	{
		cont := &admin.User{}
		aRP.GET("/list", cont.List)
		aRP.GET("/detail/:id", cont.Detail)
		aRP.POST("/create", cont.Create)
		aRP.POST("/update", cont.Update)
		aRP.POST("/delete", cont.Delete)
		aRP.POST("/changePwd", cont.UpdatePassword)
	}
}

func GroupBind(rg *gin.RouterGroup) {
	aR := rg.Group("/group").Use(middleware.AdminPrivilege())
	{
		cont := &admin.Group{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.GET("/members", cont.Members)
		aR.POST("/members/batch", cont.BatchMembers)
		aR.GET("/anchors", cont.Anchors)
		aR.POST("/anchors", cont.SetAnchors)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func DeviceGroupBind(rg *gin.RouterGroup) {
	aR := rg.Group("/device_group").Use(middleware.AdminPrivilege())
	{
		cont := &admin.DeviceGroup{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func TagBind(rg *gin.RouterGroup) {
	aR := rg.Group("/tag").Use(middleware.AdminPrivilege())
	{
		cont := &admin.Tag{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}

func AddressBookBind(rg *gin.RouterGroup) {
	aR := rg.Group("/address_book")
	{
		cont := &admin.AddressBook{}
		aR.POST("/shareByWebClient", cont.ShareByWebClient)

		arp := aR.Use(middleware.AdminPrivilege())
		arp.GET("/list", cont.List)
		//arp.GET("/detail/:id", cont.Detail)
		arp.POST("/create", cont.Create)
		arp.POST("/update", cont.Update)
		arp.POST("/delete", cont.Delete)
		arp.POST("/batchCreate", cont.BatchCreate)
		arp.POST("/batchCreateFromPeers", cont.BatchCreateFromPeers)
		arp.GET("/export", (&admin.AddressBookCSV{}).AdminExport)
		arp.POST("/import", (&admin.AddressBookCSV{}).AdminImport)
		arp.GET("/trash/list", (&admin.AddressBookTrash{}).AdminList)
		arp.POST("/trash/restore", (&admin.AddressBookTrash{}).AdminRestore)
		arp.POST("/trash/purge", (&admin.AddressBookTrash{}).AdminPurge)

	}
}
func PeerBind(rg *gin.RouterGroup) {
	aR := rg.Group("/peer")
	aR.POST("/simpleData", (&admin.Peer{}).SimpleData)
	aR.Use(middleware.AdminPrivilege())
	{
		cont := &admin.Peer{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
		aR.POST("/batchDelete", cont.BatchDelete)
	}
}

func OauthBind(rg *gin.RouterGroup) {
	aR := rg.Group("/oauth")
	{
		cont := &admin.Oauth{}
		aR.POST("/confirm", cont.Confirm)
		aR.POST("/bind", cont.ToBind)
		aR.POST("/bindConfirm", cont.BindConfirm)
		aR.POST("/unbind", cont.Unbind)
		aR.GET("/info", cont.Info)
	}
	arp := aR.Use(middleware.AdminPrivilege())
	{
		cont := &admin.Oauth{}
		arp.GET("/list", cont.List)
		arp.GET("/detail/:id", cont.Detail)
		arp.POST("/create", cont.Create)
		arp.POST("/update", cont.Update)
		arp.POST("/delete", cont.Delete)

	}

}
func LoginLogBind(rg *gin.RouterGroup) {
	cont := &admin.LoginLog{}
	aR := rg.Group("/login_log").Use(middleware.AdminPrivilege())
	aR.GET("/list", cont.List)
	aR.POST("/delete", cont.Delete)
	aR.POST("/batchDelete", cont.BatchDelete)
}
func AuditBind(rg *gin.RouterGroup) {
	cont := &admin.Audit{}
	aR := rg.Group("/audit_conn").Use(middleware.AdminPrivilege())
	aR.GET("/list", cont.ConnList)
	aR.POST("/note", cont.UpdateNote)
	aR.POST("/delete", cont.ConnDelete)
	aR.POST("/batchDelete", cont.BatchConnDelete)
	afR := rg.Group("/audit_file").Use(middleware.AdminPrivilege())
	afR.GET("/list", cont.FileList)
	afR.POST("/delete", cont.FileDelete)
	afR.POST("/batchDelete", cont.BatchFileDelete)
}
func AddressBookCollectionBind(rg *gin.RouterGroup) {
	aR := rg.Group("/address_book_collection").Use(middleware.AdminPrivilege())
	{
		cont := &admin.AddressBookCollection{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}

}
func AddressBookCollectionRuleBind(rg *gin.RouterGroup) {
	aR := rg.Group("/address_book_collection_rule").Use(middleware.AdminPrivilege())
	{
		cont := &admin.AddressBookCollectionRule{}
		aR.GET("/list", cont.List)
		aR.GET("/detail/:id", cont.Detail)
		aR.POST("/create", cont.Create)
		aR.POST("/update", cont.Update)
		aR.POST("/delete", cont.Delete)
	}
}
func UserTokenBind(rg *gin.RouterGroup) {
	aR := rg.Group("/user_token").Use(middleware.AdminPrivilege())
	cont := &admin.UserToken{}
	aR.GET("/list", cont.List)
	aR.POST("/delete", cont.Delete)
	aR.POST("/batchDelete", cont.BatchDelete)
}
func ConfigBind(rg *gin.RouterGroup) {
	aR := rg.Group("/config")
	rs := &admin.Config{}

	aR.GET("/admin", rs.AdminConfig)

	aR.Use(middleware.BackendUserAuth())
	aR.GET("/server", rs.ServerConfig)
	aR.GET("/app", rs.AppConfig)

}

/*
func FileBind(rg *gin.RouterGroup) {
	aR := rg.Group("/file")
	{
		cont := &admin.File{}
		aR.POST("/notify", cont.Notify)
		aR.OPTIONS("/oss_token", nil)
		aR.OPTIONS("/upload", nil)
		aR.GET("/oss_token", cont.OssToken)
		aR.POST("/upload", cont.Upload)
	}
}*/

func MyBind(rg *gin.RouterGroup) {
	{
		cont := &my.ShareRecord{}
		rg.GET("/my/share_record/list", cont.List)
		rg.POST("/my/share_record/delete", cont.Delete)
		rg.POST("/my/share_record/batchDelete", cont.BatchDelete)
	}

	{
		cont := &my.AddressBook{}
		rg.GET("/my/address_book/list", cont.List)
		rg.POST("/my/address_book/create", cont.Create)
		rg.POST("/my/address_book/update", cont.Update)
		rg.POST("/my/address_book/delete", cont.Delete)
		rg.POST("/my/address_book/batchCreateFromPeers", cont.BatchCreateFromPeers)
		rg.POST("/my/address_book/batchUpdateTags", cont.BatchUpdateTags)
		rg.GET("/my/address_book/export", (&admin.AddressBookCSV{}).MyExport)
		rg.POST("/my/address_book/import", (&admin.AddressBookCSV{}).MyImport)
		rg.GET("/my/address_book/trash/list", (&admin.AddressBookTrash{}).MyList)
		rg.POST("/my/address_book/trash/restore", (&admin.AddressBookTrash{}).MyRestore)
		rg.POST("/my/address_book/trash/purge", (&admin.AddressBookTrash{}).MyPurge)
	}

	{
		cont := &my.Tag{}
		rg.GET("/my/tag/list", cont.List)
		rg.POST("/my/tag/create", cont.Create)
		rg.POST("/my/tag/update", cont.Update)
		rg.POST("/my/tag/delete", cont.Delete)
	}

	{
		cont := &my.AddressBookCollection{}
		rg.GET("/my/address_book_collection/list", cont.List)
		rg.POST("/my/address_book_collection/create", cont.Create)
		rg.POST("/my/address_book_collection/update", cont.Update)
		rg.POST("/my/address_book_collection/delete", cont.Delete)
	}

	{
		cont := &my.AddressBookCollectionRule{}
		rg.GET("/my/address_book_collection_rule/list", cont.List)
		rg.POST("/my/address_book_collection_rule/create", cont.Create)
		rg.POST("/my/address_book_collection_rule/update", cont.Update)
		rg.POST("/my/address_book_collection_rule/delete", cont.Delete)
	}
	{
		cont := &my.Peer{}
		rg.GET("/my/peer/list", cont.List)

	}

	{
		cont := &my.LoginLog{}
		rg.GET("/my/login_log/list", cont.List)
		rg.POST("/my/login_log/delete", cont.Delete)
		rg.POST("/my/login_log/batchDelete", cont.BatchDelete)
	}
}

func ShareRecordBind(rg *gin.RouterGroup) {
	aR := rg.Group("/share_record").Use(middleware.AdminPrivilege())
	{
		cont := &admin.ShareRecord{}
		aR.GET("/list", cont.List)
		aR.POST("/delete", cont.Delete)
		aR.POST("/batchDelete", cont.BatchDelete)
	}

}

func ContactBind(rg *gin.RouterGroup) {
	r := rg.Group("/contact")
	c := &admin.Contact{}
	r.GET("/list", c.List)
	r.POST("/add", c.Add)
	r.POST("/delete", c.Delete)
}

func SharedDevicesBind(rg *gin.RouterGroup) {
	rg.GET("/shared_devices/list", (&admin.SharedDevices{}).List)
	rg.GET("/shared_devices/sources", (&admin.SharedDevices{}).Sources)
}

func PersonalGroupBind(rg *gin.RouterGroup) {
	owner := &admin.MyGroup{}
	r := rg.Group("/my/group")
	r.GET("/list", owner.List)
	r.GET("/detail/:id", owner.Detail)
	r.POST("/create", owner.Create)
	r.POST("/update", owner.Update)
	r.POST("/delete", owner.Delete)
	r.GET("/members", owner.Members)
	r.POST("/members/batch", owner.BatchMembers)
	r.GET("/anchors", owner.Anchors)
	r.POST("/anchors", owner.SetAnchors)
	r.GET("/user/lookup", owner.UserLookup)
}

func MyAuditBind(rg *gin.RouterGroup) {
	audit := &admin.MyAudit{}
	rg.GET("/my/audit_conn/list", audit.Connections)
	rg.GET("/my/audit_file/list", audit.Files)
}

func AuditManagementBind(rg *gin.RouterGroup) {
	handler:=&admin.AuditManagement{}
	rg.GET("/my/audit_alarm/list",handler.MyAlarms)
	rg.GET("/audit_alarm/list",middleware.AdminPrivilege(),handler.Alarms)
	rg.GET("/audit_operation/list",middleware.AdminPrivilege(),handler.Operations)
	adm:=rg.Group("/audit").Use(middleware.AdminPrivilege())
	adm.GET("/alarms",handler.Alarms)
	adm.GET("/operations",handler.Operations)
	adm.GET("/retention",handler.Retention)
	adm.POST("/retention",handler.SaveRetention)
	adm.GET("/export",handler.Export)
}

func DevicePolicyBind(rg *gin.RouterGroup) {
	handler:=&admin.DevicePolicy{}
	adm:=rg.Group("/device_policy").Use(middleware.AdminPrivilege())
	adm.GET("/list",handler.List)
	adm.POST("/create",handler.Create)
	adm.POST("/update",handler.Update)
	adm.POST("/delete",handler.Delete)
	adm.GET("/assignment/list",handler.AssignmentList)
	adm.POST("/assignment/create",handler.AssignmentCreate)
	adm.POST("/assignment/update",handler.AssignmentUpdate)
	adm.POST("/assignment/delete",handler.AssignmentDelete)
	adm.GET("/user_assignment/list",handler.UserAssignmentList)
	adm.POST("/user_assignment/create",handler.UserAssignmentCreate)
	adm.POST("/user_assignment/update",handler.UserAssignmentUpdate)
	adm.POST("/user_assignment/delete",handler.UserAssignmentDelete)
	adm.GET("/group_assignment/list",handler.GroupAssignmentList)
	adm.POST("/group_assignment/create",handler.GroupAssignmentCreate)
	adm.POST("/group_assignment/update",handler.GroupAssignmentUpdate)
	adm.POST("/group_assignment/delete",handler.GroupAssignmentDelete)
}

func DeviceSessionsBind(rg *gin.RouterGroup) {
	handler:=&admin.DeviceSessions{}
	r:=rg.Group("/session").Use(middleware.AdminPrivilege())
	r.GET("/list",handler.List)
	r.POST("/disconnect",handler.Disconnect)
	r.GET("/disconnect/list",handler.Requests)
}

func APITokenBind(rg *gin.RouterGroup){
	handler:=&admin.APITokens{}
	r:=rg.Group("/api_token").Use(middleware.AdminPrivilege())
	r.GET("/list",handler.List)
	r.POST("/create",handler.Create)
	r.POST("/delete",handler.Delete)
}

func MailBind(rg *gin.RouterGroup) {
	controller:=&admin.Mail{}
	r:=rg.Group("/mail").Use(middleware.AdminPrivilege())
	r.GET("/settings",controller.Settings)
	r.POST("/settings",controller.Save)
	r.POST("/test",controller.Test)
}

func InvitationBind(rg *gin.RouterGroup){
	controller:=&admin.Invitation{}
	r:=rg.Group("/invite").Use(middleware.AdminPrivilege())
	r.GET("/list",controller.List)
	r.POST("/create",controller.Create)
	r.POST("/revoke",controller.Revoke)
}

func TOTPBind(rg *gin.RouterGroup){
	controller:=&admin.TOTP{}
	r:=rg.Group("/mfa")
	r.GET("/status",controller.Status)
	r.POST("/setup",controller.Setup)
	r.POST("/enable",controller.Enable)
	r.POST("/disable",controller.Disable)
	r.POST("/recovery/regenerate",controller.RegenerateRecoveryCodes)
}
