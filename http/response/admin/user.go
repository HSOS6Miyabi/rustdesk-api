package admin

import "github.com/lejianwen/rustdesk-api/v2/model"

type LoginPayload struct {
	Id uint `json:"id"`
	GroupID uint `json:"group_id"`
	Username   string   `json:"username"`
	Email      string   `json:"email"`
	Avatar     string   `json:"avatar"`
	Token      string   `json:"token"`
	RouteNames []string `json:"route_names"`
	Nickname   string   `json:"nickname"`
	HasLocalPassword bool `json:"has_local_password"`
}

func (lp *LoginPayload) FromUser(user *model.User) {
	lp.Id = user.Id
	lp.GroupID = user.GroupId
	lp.Username = user.Username
	lp.Email = user.Email
	lp.Avatar = user.Avatar
	lp.Nickname = user.Nickname
	lp.HasLocalPassword = user.Password != ""
}

type UserOauthItem struct {
	Op     string `json:"op"`
	Status int    `json:"status"`
}
