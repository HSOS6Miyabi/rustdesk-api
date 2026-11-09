package router

import (
	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/controller/admin"
	"github.com/lejianwen/rustdesk-api/v2/http/middleware"
)

// AutomationRoutes exposes selected existing management handlers to automation.
// Each method has its own API-token scope; browser sessions cannot use these routes.
func AutomationRoutes(engine *gin.Engine) {
	r := engine.Group("/api/automation/v1")
	read := func(path, permission string, handler gin.HandlerFunc) {
		r.GET(path, middleware.AutomationTokenAuth(permission), handler)
	}
	write := func(path, permission string, handler gin.HandlerFunc) {
		r.POST(path, middleware.AutomationTokenAuth(permission),
			middleware.RecordOperations(), handler)
	}
	peers := &admin.Peer{}
	read("/devices", "devices.read", peers.List)
	read("/devices/:id", "devices.read", peers.Detail)
	write("/devices/create", "devices.manage", peers.Create)
	write("/devices/update", "devices.manage", peers.Update)
	write("/devices/transfer", "devices.manage", peers.Transfer)
	write("/devices/delete", "devices.manage", peers.Delete)
	write("/devices/batch-delete", "devices.manage", peers.BatchDelete)

	deployments := &admin.DeviceDeployment{}
	read("/deployments", "deployments.read", deployments.List)
	write("/deployments/status", "deployments.manage", deployments.SetStatus)

	relays := &admin.RelayNodes{}
	read("/relays", "relays.read", relays.List)
	write("/relays/create", "relays.manage", relays.Create)
	write("/relays/update", "relays.manage", relays.Update)
	write("/relays/delete", "relays.manage", relays.Delete)

	sessions := &admin.DeviceSessions{}
	read("/sessions", "sessions.read", sessions.List)
	read("/sessions/disconnects", "sessions.read", sessions.Requests)
	write("/sessions/disconnect", "sessions.disconnect", sessions.Disconnect)

	controls := &admin.ControlRoles{}
	assignments := &admin.ControlAssignments{}
	read("/control-roles", "control_roles.read", controls.List)
	read("/control-assignments", "control_roles.read", assignments.List)
	read("/control-roles/effective", "control_roles.read", controls.Effective)
	write("/control-roles/create", "control_roles.manage", controls.Create)
	write("/control-roles/update", "control_roles.manage", controls.Update)
	write("/control-roles/delete", "control_roles.manage", controls.Delete)
	write("/control-assignments/create", "control_roles.manage", assignments.Create)
	write("/control-assignments/update", "control_roles.manage", assignments.Update)
	write("/control-assignments/delete", "control_roles.manage", assignments.Delete)

	rules := &admin.AccessRules{}
	read("/access-rules", "access_rules.read", rules.List)
	write("/access-rules/create", "access_rules.manage", rules.Create)
	write("/access-rules/update", "access_rules.manage", rules.Update)
	write("/access-rules/delete", "access_rules.manage", rules.Delete)

	policies := &admin.DevicePolicy{}
	read("/policies", "policies.read", policies.List)
	read("/policies/assignments", "policies.read", policies.AssignmentList)
	read("/policies/user-assignments", "policies.read", policies.UserAssignmentList)
	read("/policies/group-assignments", "policies.read", policies.GroupAssignmentList)
	write("/policies/create", "policies.manage", policies.Create)
	write("/policies/update", "policies.manage", policies.Update)
	write("/policies/delete", "policies.manage", policies.Delete)

	audits := &admin.Audit{}
	read("/audits/connections", "audits.read", audits.ConnList)
	read("/audits/files", "audits.read", audits.FileList)
}
