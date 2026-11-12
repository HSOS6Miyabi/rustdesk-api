package my

import (
 "strconv"
 "strings"
 "unicode/utf8"

 "github.com/gin-gonic/gin"
 "github.com/lejianwen/rustdesk-api/v2/http/response"
 "github.com/lejianwen/rustdesk-api/v2/model"
 "github.com/lejianwen/rustdesk-api/v2/service"
 "gorm.io/gorm"
)

type Directory struct{}

type publicUser struct {
 Id uint `json:"id"`
 Username string `json:"username"`
 Nickname string `json:"nickname"`
}

type publicDevice struct {
 RowId uint `json:"row_id"`
 Id string `json:"id"`
 Alias string `json:"alias"`
 Hostname string `json:"hostname"`
 Os string `json:"os"`
 Username string `json:"username"`
 UserId uint `json:"user_id"`
 OwnerUsername string `json:"owner_username"`
 LastOnlineTime int64 `json:"last_online_time"`
 LastOnlineIp string `json:"last_online_ip"`
 Note string `json:"note"`
 Owned bool `json:"owned" gorm:"-"`
}

func searchTerm(value string) (string,bool) {
 value=strings.TrimSpace(value)
 if utf8.RuneCountInString(value)<2 || utf8.RuneCountInString(value)>100{return "",false}
 return value,true
}
func likeKeyword(keyword string) string {
 esc:=strings.NewReplacer("!","!!","%","!%","_","!_")
 return "%"+esc.Replace(strings.ToLower(keyword))+"%"
}

func (h *Directory) Users(c *gin.Context) {
 keyword,ok:=searchTerm(c.Query("keyword"))
 if !ok {response.Fail(c,101,"请输入至少两个字符，最多 100 个字符");return}
 user:=service.AllService.UserService.CurUser(c)
 if user==nil||user.Id==0 {response.Fail(c,403,"请先登录");return}
 like:=likeKeyword(keyword)
 id,_:=strconv.ParseUint(keyword,10,64)
 query:=service.DB.Model(&model.User{}).Where("status = ?",model.COMMON_STATUS_ENABLE)
 query=query.Where("(LOWER(username) LIKE ? ESCAPE '!' OR LOWER(nickname) LIKE ? ESCAPE '!' OR LOWER(email) LIKE ? ESCAPE '!' OR id = ?)",like,like,like,id)
 var users []publicUser
 // The results expose public names only, including when the search matched an email.
 if err:=query.Select("id, username, nickname").Order("username ASC, id ASC").Limit(30).Scan(&users).Error;err!=nil{
  response.Fail(c,101,"用户搜索失败");return
 }
 if users==nil{users=[]publicUser{}}
 response.Success(c,gin.H{"list":users,"total":len(users)})
}

func (h *Directory) User(c *gin.Context) {
 value:=strings.TrimSpace(c.Query("id"))
 if value=="" || len(value)>100{response.Fail(c,101,"用户编号无效");return}
 current:=service.AllService.UserService.CurUser(c)
 if current==nil||current.Id==0{response.Fail(c,403,"请先登录");return}
 query:=service.DB.Model(&model.User{}).Where("status = ?",model.COMMON_STATUS_ENABLE)
 id,err:=strconv.ParseUint(value,10,64)
 if err==nil && id>0{query=query.Where("id = ?",id)}else{query=query.Where("username = ?",value)}
 var result publicUser
 if err:=query.Select("id,username,nickname").Take(&result).Error;err!=nil||result.Id==0{
  response.Fail(c,404,"没有找到可以查看的用户");return
 }
 response.Success(c,result)
}

func directoryDeviceQuery(user *model.User) (*gorm.DB,error) {
 owners,err:=service.SharedDeviceOwnerIDs(user)
 if err!=nil{return nil,err}
 visible:=append([]uint{user.Id},owners...)
 return service.DB.Table("peers AS p").Joins("LEFT JOIN users AS u ON u.id = p.user_id").Where("p.user_id IN ?",visible),nil
}

func (h *Directory) Devices(c *gin.Context) {
 keyword,ok:=searchTerm(c.Query("keyword"))
 if !ok {response.Fail(c,101,"请输入至少两个字符，最多 100 个字符");return}
 user:=service.AllService.UserService.CurUser(c)
 if user==nil||user.Id==0{response.Fail(c,403,"请先登录");return}
 query,err:=directoryDeviceQuery(user)
 if err!=nil{response.Fail(c,101,"设备搜索失败");return}
 like:=likeKeyword(keyword)
 query=query.Where("(LOWER(p.id) LIKE ? ESCAPE '!' OR LOWER(p.alias) LIKE ? ESCAPE '!' OR LOWER(p.hostname) LIKE ? ESCAPE '!' OR LOWER(p.os) LIKE ? ESCAPE '!' OR LOWER(p.username) LIKE ? ESCAPE '!' OR LOWER(p.last_online_ip) LIKE ? ESCAPE '!' OR LOWER(p.note) LIKE ? ESCAPE '!' OR LOWER(u.username) LIKE ? ESCAPE '!')",
 like,like,like,like,like,like,like,like)
 var rows []publicDevice
 if err:=query.Select("p.row_id, p.id, p.alias, p.hostname, p.os, p.username, p.user_id, u.username AS owner_username, p.last_online_time, p.last_online_ip, p.note").
  Order("p.last_online_time DESC, p.row_id DESC").Limit(50).Scan(&rows).Error;err!=nil{
  response.Fail(c,101,"设备搜索失败");return
 }
 if rows==nil{rows=[]publicDevice{}}
 for i:=range rows {rows[i].Owned=rows[i].UserId==user.Id}
 response.Success(c,gin.H{"list":rows,"total":len(rows)})
}

func (h *Directory) Device(c *gin.Context) {
 id:=strings.TrimSpace(c.Query("id"))
 if id==""||len(id)>253{response.Fail(c,101,"设备 ID 无效");return}
 user:=service.AllService.UserService.CurUser(c)
 if user==nil||user.Id==0{response.Fail(c,403,"请先登录");return}
 query,err:=directoryDeviceQuery(user)
 if err!=nil{response.Fail(c,101,"查询设备失败");return}
 var rows []publicDevice
 if err:=query.Select("p.row_id, p.id, p.alias, p.hostname, p.os, p.username, p.user_id, u.username AS owner_username, p.last_online_time, p.last_online_ip, p.note").
  Where("p.id = ?",id).Limit(20).Scan(&rows).Error;err!=nil{
  response.Fail(c,101,"查询设备失败");return
 }
 if len(rows)==0{response.Fail(c,404,"该设备不属于您的设备或共享设备");return}
 result:=rows[0]
 for _,item:=range rows{if item.UserId==user.Id{result=item;break}}
 result.Owned=result.UserId==user.Id
 response.Success(c,result)
}
