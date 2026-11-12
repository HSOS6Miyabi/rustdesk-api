package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
	"gorm.io/gorm"
)

type clientBuildSource struct {
	URL string
	SHA256 string
	Size int64
}

func buildClientHTTP() *http.Client {
	return &http.Client{
		Timeout:4*time.Minute,
		CheckRedirect:func(req *http.Request,via []*http.Request)error{
			if len(via)>=6{return errors.New("too many release download redirects")}
			hostname:=strings.ToLower(req.URL.Hostname())
			if req.URL.Scheme!="https" || (
				hostname!="github.com" &&
                hostname!="api.github.com" &&
				hostname!="release-assets.githubusercontent.com" &&
				hostname!="objects.githubusercontent.com") {
				return errors.New("client download redirected to an untrusted server")
			}
			return nil
		},
	}
}

func writeClientPackage(task *model.ClientBuildTask,source clientBuildSource,client *http.Client)(string,int64,error){
    return writeClientPackageContext(context.Background(),task,source,client)
}
func writeClientPackageContext(parent context.Context,task *model.ClientBuildTask,source clientBuildSource,client *http.Client)(string,int64,error){
	if task==nil||task.Id==0||source.Size<=0||source.Size>maxClientInstallerSize ||
        (source.SHA256!="" && len(source.SHA256)!=64) ||
		(!strings.HasSuffix(task.Filename,".exe") && !strings.HasSuffix(task.Filename,".zip")) ||
		len(task.Filename)>240 || strings.ContainsAny(task.Filename,"/\\") {
		return "",0,errors.New("invalid client download information")
	}
	folder:=filepath.Join(clientBuildRoot(),fmt.Sprintf("%d",task.Id))
	if err:=os.MkdirAll(folder,0700);err!=nil{return "",0,err}
	ctx,cancel:=context.WithTimeout(parent,4*time.Minute)
	defer cancel()
	request,err:=http.NewRequestWithContext(ctx,http.MethodGet,source.URL,nil)
	if err!=nil{return "",0,err}
	res,err:=client.Do(request)
	if err!=nil{return "",0,err}
	defer res.Body.Close()
	if res.StatusCode!=http.StatusOK{
		return "",0,fmt.Errorf("official client release returned HTTP %d",res.StatusCode)
	}
	tmp,err:=os.CreateTemp(folder,".client-build-*.tmp")
	if err!=nil{return "",0,err}
	tmpPath:=tmp.Name()
	defer os.Remove(tmpPath)
	hash:=sha256.New()
	count,copyErr:=io.Copy(io.MultiWriter(tmp,hash),io.LimitReader(res.Body,source.Size+1))
	if err:=tmp.Close();err!=nil{return "",0,err}
	if copyErr!=nil{return "",0,copyErr}
	if count!=source.Size{return "",0,fmt.Errorf("official client size mismatch: expected %d, got %d",source.Size,count)}
	digest:=hex.EncodeToString(hash.Sum(nil))
	if source.SHA256!="" && !strings.EqualFold(digest,source.SHA256){
        return "",0,errors.New("official client checksum does not match")
    }
    source.SHA256=digest
	if task.Platform=="windows-x64" || task.Platform=="windows-arm64" {
		output:=filepath.Join(folder,task.Filename)
		if err:=os.Rename(tmpPath,output);err!=nil{return "",0,err}
		if err:=os.Chmod(output,0600);err!=nil{return "",0,err}
		return digest,count,nil
	}
	return prepareClientArchive(task,source,tmpPath,folder)
}

func RecoverInterruptedClientBuilds(now int64)error{
	if DB==nil{return nil}
	return DB.Model(&model.ClientBuildTask{}).
		Where("status = ? AND updated_at <= ?", "building", time.Unix(now-600,0)).
		Updates(map[string]interface{}{
			"status":"failed", "error":"安装程序处理已中断，可重新提交任务",
			"finished_at":now,
		}).Error
}

func ProcessClientBuildQueue(limit int)(int,error){
    return processClientBuildQueueContext(context.Background(),limit)
}
func processClientBuildQueueContext(ctx context.Context,limit int)(int,error){
	if DB==nil{return 0,nil}
	if err:=RecoverInterruptedClientBuilds(time.Now().Unix());err!=nil{return 0,err}
	if limit<1||limit>2{limit=1}
	var queued []model.ClientBuildTask
	if err:=DB.Where("status = ?","pending").Order("id ASC").Limit(limit).
		Find(&queued).Error;err!=nil{return 0,err}
	complete:=0
	for _,task:=range queued {
        if err:=ctx.Err();err!=nil{return complete,err}
        now:=time.Now().Unix()
		claim:=DB.Model(&model.ClientBuildTask{}).Where("id = ? AND status = ?",task.Id,"pending").
			Updates(map[string]interface{}{"status":"building","started_at":now})
		if claim.Error!=nil{return complete,claim.Error}
		if claim.RowsAffected!=1{continue}
		workCtx,stopProgress:=context.WithCancel(ctx)
        progressDone:=make(chan struct{})
        go func(){
            defer close(progressDone)
            ticker:=time.NewTicker(30*time.Second)
            defer ticker.Stop()
            for {
                select {
                case <-workCtx.Done():return
                case <-ticker.C:
                    result:=DB.Model(&model.ClientBuildTask{}).
                        Where("id = ? AND status = ?",task.Id,"building").
                        UpdateColumn("updated_at",time.Now())
                    if result.Error!=nil&&Logger!=nil{
                        Logger.WithError(result.Error).Warn("client build progress update failed")
                    }
                    if result.Error!=nil||result.RowsAffected==0{return}
                }
            }
        }()
        source,err:=officialClientBuildSource(task.Platform,task.Version)
        var digest string
        var size int64
        if err==nil{digest,size,err=writeClientPackageContext(workCtx,&task,source,buildClientHTTP())}
        stopProgress()
        <-progressDone
        fields:=map[string]interface{}{"finished_at":time.Now().Unix()}
		if err!=nil {
			message:=err.Error()
			if len(message)>500{message=message[:500]}
			fields["status"]="failed"
			fields["error"]=message
		}else{
			fields["status"]="ready"
			fields["sha256"]=digest
			fields["bytes"]=size
			fields["error"]=""
			complete++
		}
		if err:=DB.Model(&model.ClientBuildTask{}).Where("id = ? AND status = ?",task.Id,"building").
			Updates(fields).Error;err!=nil{return complete,err}
	}
	return complete,nil
}

var clientBuildWake = make(chan struct{}, 1)

func wakeClientBuildWorker() {
	select {
	case clientBuildWake <- struct{}{}:
	default:
	}
}

// An idle builder sleeps until a task is submitted. Interrupted builds are
// recovered at their persisted deadline even when no new task is submitted.
func nextClientBuildDelay() (time.Duration, bool, error) {
	var pending model.ClientBuildTask
	err := DB.Select("id").Where("status = ?", "pending").Take(&pending).Error
	if err == nil {
		return 0, true, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return 0, false, err
	}
	var earliest model.ClientBuildTask
    err = DB.Model(&model.ClientBuildTask{}).Select("id,updated_at").
        Where("status = ?", "building").Order("updated_at ASC").Take(&earliest).Error
    if errors.Is(err,gorm.ErrRecordNotFound){return 0,false,nil}
    if err!=nil{return 0,false,err}
    delay:=time.Until(time.Time(earliest.UpdatedAt).Add(601*time.Second))
	if delay < 0 {
		delay = 0
	}
	return delay, true, nil
}

func StartClientBuildWorker(){
	launchServiceWorker(func(ctx context.Context) {
		for {
			if ctx.Err() != nil { return }
			if _, err := processClientBuildQueueContext(ctx,1); err != nil {
				if Logger != nil {
					Logger.WithError(err).Warn("client installer preparation failed")
				}
				if !waitForWorkContext(ctx, clientBuildWake, 10*time.Second, true) { return }
				continue
			}
			delay, scheduled, err := nextClientBuildDelay()
			if err != nil {
				if Logger != nil {
					Logger.WithError(err).Warn("client installer scheduling failed")
				}
				if !waitForWorkContext(ctx,clientBuildWake,10*time.Second,true) {return}
				continue
			}
			if !waitForWorkContext(ctx, clientBuildWake, delay, scheduled) { return }
		}
	})
}
