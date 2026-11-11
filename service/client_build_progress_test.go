package service

import (
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestClientBuildHeartbeatProtectsLongTask(t *testing.T){
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
    now:=time.Now().UTC()
    task:=&model.ClientBuildTask{Status:"building",StartedAt:now.Add(-25*time.Minute).Unix(),Filename:"test.exe"}
    if err:=DB.Create(task).Error;err!=nil{t.Fatal(err)}
    if err:=RecoverInterruptedClientBuilds(now.Unix());err!=nil{t.Fatal(err)}
    var status model.ClientBuildTask
    if err:=DB.First(&status,task.Id).Error;err!=nil{t.Fatal(err)}
    if status.Status!="building"{t.Fatalf("active long-running task was cancelled: %q",status.Status)}
    expired:=now.Add(-12*time.Minute)
    if err:=DB.Model(task).UpdateColumn("updated_at",expired).Error;err!=nil{t.Fatal(err)}
    if err:=RecoverInterruptedClientBuilds(now.Unix());err!=nil{t.Fatal(err)}
    if err:=DB.First(&status,task.Id).Error;err!=nil{t.Fatal(err)}
    if status.Status!="failed"{t.Fatalf("abandoned task was not recovered: %q",status.Status)}
}
