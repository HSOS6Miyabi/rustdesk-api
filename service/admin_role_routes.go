package service

import "strings"

// AdminPermissionForRoute returns the permission required for an existing
// management route. Unlisted routes remain accessible only to full administrators.
func AdminPermissionForRoute(method,path string)string{
	p:=strings.TrimPrefix(path,"/api/admin")
	method=strings.ToUpper(method)
	if !strings.HasPrefix(p,"/"){return ""}
	read:=method=="GET"
	write:=method=="POST"||method=="PUT"||method=="DELETE"
	if !read&&!write{return ""}
	switch {
	case p=="/user/list"||p=="/user/detail/:id":
		if read{return "users.view"}
	case p=="/user/create":
		if write{return "users.create"}
	case p=="/user/delete":
		if write{return "users.delete"}
	case p=="/user/update":
		if write{return "users.edit"}
	case p=="/user/changePwd":
		if write{return "users.edit_password"}
	case p=="/user/mfa/reset":
		if write{return "users.manage_2fa"}
	case p=="/user_token/list":
		if read{return "users.force_logout"}
	case p=="/user_token/delete"||p=="/user_token/batchDelete":
		if write{return "users.force_logout"}
	case p=="/peer/list"||p=="/peer/export"||p=="/peer/detail/:id":
		if read{return "devices.view"}
	case p=="/peer/create":
		if write{return "devices.create"}
	case p=="/peer/update":
		if write{return "devices.edit_info"}
	case p=="/peer/transfer":
		if write{return "devices.transfer"}
	case p=="/peer/delete"||p=="/peer/batchDelete":
		if write{return "devices.delete"}
	case strings.HasPrefix(p,"/group/"):
		if read{return "user_groups.view"}
		if write{return "user_groups.edit"}
	case strings.HasPrefix(p,"/device_group/"):
		if read{return "device_groups.view"}
		if write{return "device_groups.edit"}
	case strings.HasPrefix(p,"/audit_conn/")||strings.HasPrefix(p,"/audit_file/"):
		if read{return "audits.view"}
		if write{return "audits.edit"}
	case p=="/audit_alarm/list"||p=="/audit_operation/list"||
		p=="/audit/alarms"||p=="/audit/operations"||p=="/audit/export":
		if read{return "audits.view"}
	case p=="/audit/retention":
		if read{return "audits.view"}
		if write{return "audits.edit"}
		case strings.HasPrefix(p,"/device_policy/"):
		if read{return "strategies.view"}
		if write{return "strategies.edit"}
	case strings.HasPrefix(p,"/control_role/")||strings.HasPrefix(p,"/control_assignment/"):
		if read{return "control_roles.view"}
		if write{return "control_roles.edit"}
	case strings.HasPrefix(p,"/access_rule/"):
		if read{return "access_rules.view"}
		if write{return "access_rules.edit"}
	case strings.HasPrefix(p,"/relay/"):
		if read{return "relay_nodes.view"}
		if write{return "relay_nodes.edit"}
	case strings.HasPrefix(p,"/session/"):
		if read{return "sessions.view"}
		if write{return "sessions.edit"}
	case strings.HasPrefix(p,"/api_token/"):
		if read{return "api_tokens.view"}
		if write{return "api_tokens.edit"}
	case strings.HasPrefix(p,"/mail/"):
		if read{return "smtp.view"}
		if write{return "smtp.edit"}
	case strings.HasPrefix(p,"/notification/"):
		if read{return "notifications.view"}
		if write{return "notifications.edit"}
	case strings.HasPrefix(p,"/invite/"):
		if read||write{return "users.create"}
	case strings.HasPrefix(p,"/device_deployment/"):
		if read{return "device_deployment.view"}
		if write{return "device_deployment.edit"}
	case strings.HasPrefix(p,"/address_book/")||
		strings.HasPrefix(p,"/address_book_collection/")||
		strings.HasPrefix(p,"/address_book_collection_rule/")||
		strings.HasPrefix(p,"/tag/"):
		if read{return "address_books.view"}
		if write{return "address_books.edit"}
	case strings.HasPrefix(p,"/login_log/"):
		if read{return "audits.view"}
		if write{return "audits.edit"}
	case strings.HasPrefix(p,"/oauth/"):
		// Identity provider settings are reserved for full administrators.
		return ""
	}
	return ""
}
