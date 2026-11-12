package admin

import (
	"encoding/csv"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lejianwen/rustdesk-api/v2/http/response"
	"github.com/lejianwen/rustdesk-api/v2/model"
	"github.com/lejianwen/rustdesk-api/v2/service"
)

type AuditManagement struct{}

func auditPage(c *gin.Context) (uint,uint,error) {
	page,err:=strconv.ParseUint(c.DefaultQuery("page","1"),10,32)
	if err!=nil{return 0,0,err}
	size,err:=strconv.ParseUint(c.DefaultQuery("page_size","20"),10,32)
	if err!=nil{return 0,0,err}
	if page==0{page=1}
	if size==0{size=20}
	if size>100{size=100}
	return uint(page),uint(size),nil
}

func (a *AuditManagement) Alarms(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	var alarmType *int
	if typ:=c.Query("typ");typ!=""{
		n,err:=strconv.Atoi(typ);if err!=nil{response.Fail(c,101,"无效的告警类型");return}
		alarmType=&n
	}
	peer:=strings.TrimSpace(c.Query("peer_id"))
	if len(peer)>128{response.Fail(c,101,"设备 ID 超出长度限制");return}
	list,total,err:=service.AllService.AuditService.AllAlarms(page,size,peer,alarmType,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"list":list,"total":total,"page":page,"page_size":size})
}

func (a *AuditManagement) MyAlarms(c *gin.Context) {
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	user:=service.AllService.UserService.CurUser(c)
	list,total,err:=service.AllService.AuditService.OwnAlarms(user.Id,page,size,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"list":list,"total":total,"page":page,"page_size":size})
}

func (a *AuditManagement) Operations(c *gin.Context){
	page,size,err:=auditPage(c)
	if err!=nil{response.Fail(c,101,"无效的页码");return}
	var userId uint
	if value:=c.Query("user_id");value!=""{
		id,e:=strconv.ParseUint(value,10,32)
		if e!=nil{response.Fail(c,101,"无效的用户 ID");return}
		userId=uint(id)
	}
	method:=strings.ToUpper(strings.TrimSpace(c.Query("method")))
	list,total,err:=service.AllService.AuditService.AllOperations(page,size,userId,method,c.Query("sort_by"),c.Query("sort_dir"))
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,gin.H{"list":list,"total":total,"page":page,"page_size":size})
}

func (a *AuditManagement) Retention(c *gin.Context){
	policy,err:=service.GetAuditRetention()
	if err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,policy)
}

func (a *AuditManagement) SaveRetention(c *gin.Context){
	var policy model.AuditRetentionPolicy
	if err:=c.ShouldBindJSON(&policy);err!=nil{response.Fail(c,101,err.Error());return}
	if err:=service.SaveAuditRetention(&policy);err!=nil{response.Fail(c,101,err.Error());return}
	response.Success(c,&policy)
}

func auditCleanupAdministrator(c *gin.Context) bool {
	user := service.AllService.UserService.CurUser(c)
	if user == nil || user.Id == 0 || !service.AllService.UserService.IsAdmin(user) {
		response.Fail(c, 403, "只有系统管理员可以清理日志")
		return false
	}
	return true
}

func (a *AuditManagement) CleanupPreview(c *gin.Context) {
	if !auditCleanupAdministrator(c) { return }
	days := 0
	if raw := c.Query("days"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil { response.Fail(c, 101, "无效的保留天数"); return }
		days = value
	}
	rows, err := service.PreviewLogCleanup(service.LogCleanupRequest{
		Kind: c.Query("kind"), Mode: c.Query("mode"), Days: days,
	}, time.Now())
	if err != nil { response.Fail(c, 101, err.Error()); return }
	var total int64
	for _, row := range rows { total += row.Eligible }
	response.Success(c, gin.H{"items": rows, "total": total})
}

func (a *AuditManagement) Cleanup(c *gin.Context) {
	if !auditCleanupAdministrator(c) { return }
	var request service.LogCleanupRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Fail(c, 101, "请填写有效的日志清理条件")
		return
	}
	rows, err := service.RunLogCleanup(request, time.Now())
	if err != nil { response.Fail(c, 101, err.Error()); return }
	var total int64
	for _, row := range rows { total += row.Deleted }
	response.Success(c, gin.H{"items": rows, "deleted": total})
}

func exportColumns(kind string) (interface{},[]string,error) {
	switch kind {
	case "audit_conn":
		return &model.AuditConn{},[]string{"id","peer_id","from_peer","from_account","to_account","from_user_id","to_user_id","ip","conn_id","session_id","conn_audit_ref","primary_auth","two_factor","type","close_time","created_at","note","note_updated_by"},nil
	case "audit_file":
		return &model.AuditFile{},[]string{"id","peer_id","from_peer","from_account","to_account","conn_id","path","type","ip","num","created_at"},nil
	case "audit_alarm":
		return &model.AuditAlarm{},[]string{"id","peer_id","typ","info","conn_id","conn_audit_ref","from_user_id","to_user_id","source_ip","created_at"},nil
	case "audit_operation":
		return &model.AuditOperation{},[]string{"id","user_id","account","method","path","client_ip","user_agent","http_status","success","created_at"},nil
	case "login_log":
		return &model.LoginLog{},[]string{"id","user_id","client","device_id","ip","type","platform","created_at"},nil
	default: return nil,nil,errors.New("unsupported export category")
	}
}

func csvSafe(value interface{}) string {
	if value==nil{return ""}
	var s string
	switch v:=value.(type){
	case []byte:s=string(v)
	case time.Time:s=v.Format(time.RFC3339)
	default:s=fmt.Sprint(v)
	}
	s=strings.TrimRight(s,"\x00")
	if len(s)>0 && strings.ContainsRune("=+-@\t\r\n",rune(s[0])) {
		s="'"+s
	}
	return s
}

// Export returns at most 10000 most recent entries in CSV format.
// A token is required in a request header and is never embedded in a download URL.
func (a *AuditManagement) Export(c *gin.Context){
	kind:=c.Query("kind")
	modelType,columns,err:=exportColumns(kind)
	if err!=nil{response.Fail(c,101,err.Error());return}
	query:=service.DB.Model(modelType).Select(strings.Join(columns,",")).Order("id DESC").Limit(10000)
	if from:=c.Query("from_id");from!="" {
		id,err:=strconv.ParseUint(from,10,64)
		if err!=nil{response.Fail(c,101,"无效的起始记录编号");return}
		query=query.Where("id < ?",id)
	}
	if start:=strings.TrimSpace(c.Query("from_date")); start!="" {
		date,e:=time.Parse("2006-01-02",start)
		if e!=nil{response.Fail(c,101,"无效的开始日期");return}
		query=query.Where("created_at >= ?",date)
	}
	if end:=strings.TrimSpace(c.Query("to_date")); end!="" {
		date,e:=time.Parse("2006-01-02",end)
		if e!=nil{response.Fail(c,101,"无效的结束日期");return}
		query=query.Where("created_at < ?",date.AddDate(0,0,1))
	}
	if peer:=strings.TrimSpace(c.Query("peer_id"));peer!="" {
		if len(peer)>128{response.Fail(c,101,"设备 ID 过长");return}
		switch kind{
		case "audit_conn","audit_file","audit_alarm":
			query=query.Where("peer_id LIKE ?","%"+peer+"%")
		default:
			response.Fail(c,101,"当前类别不支持设备筛选");return
		}
	}
	if fromPeer:=strings.TrimSpace(c.Query("from_peer"));fromPeer!="" {
		if len(fromPeer)>128{response.Fail(c,101,"来源设备 ID 过长");return}
		switch kind {
		case "audit_conn","audit_file":
			query=query.Where("from_peer LIKE ?","%"+fromPeer+"%")
		default:
			response.Fail(c,101,"当前类别不支持来源设备筛选");return
		}
	}
	if fromUser:=strings.TrimSpace(c.Query("from_user_id"));fromUser!=""{
		if kind!="audit_conn"&&kind!="audit_file"&&kind!="audit_alarm"{
			response.Fail(c,101,"当前类别不支持来源账户筛选");return
		}
		userID,e:=strconv.ParseUint(fromUser,10,32)
		if e!=nil||userID==0{response.Fail(c,101,"无效的来源账户编号");return}
		query=query.Where("from_user_id = ?",userID)
	}
	if action:=strings.TrimSpace(c.Query("action"));action!="" {
		if kind!="audit_conn" || (action!="new"&&action!="close") {
			response.Fail(c,101,"当前类别不支持此连接事件筛选");return
		}
		query=query.Where("action = ?",action)
	}
	// All filters are applied before the limited row query is executed.
	rows,err:=query.Rows()
	if err!=nil{response.Fail(c,101,err.Error());return}
	defer rows.Close()
	c.Header("Content-Type","text/csv; charset=utf-8")
	c.Header("Cache-Control","no-store")
	c.Header("X-Content-Type-Options","nosniff")
	c.Header("Content-Disposition",fmt.Sprintf("attachment; filename=%q",kind+".csv"))
	c.Status(http.StatusOK)
	writer:=csv.NewWriter(c.Writer)
	if err:=writer.Write(columns);err!=nil{return}
	for rows.Next(){
		values:=make([]interface{},len(columns))
		dest:=make([]interface{},len(columns))
		for i:=range values{dest[i]=&values[i]}
		if err:=rows.Scan(dest...);err!=nil{break}
		row:=make([]string,len(columns))
		for i:=range values{row[i]=csvSafe(values[i])}
		if err:=writer.Write(row);err!=nil{break}
	}
	writer.Flush()
	if err:=rows.Err();err!=nil && service.Logger!=nil{
		service.Logger.WithError(err).Warn("audit export failed")
	}
}
