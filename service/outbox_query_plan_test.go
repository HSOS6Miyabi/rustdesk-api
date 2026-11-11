package service

import (
    "fmt"
    "strings"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestSparseNotificationOutboxDueQueriesUseExistingIndexes(t *testing.T){
    cleanup:=setupPushTestDB(t)
    defer cleanup()
    now:=time.Now().Unix()
    push:=make([]model.PushEventOutbox,800)
    notifications:=make([]model.NotificationEventOutbox,800)
    for i:=range push {
        status:="failed"
        if i%97==0 {status="pending"}
        push[i]=model.PushEventOutbox{Payload:"{}",Status:status,
            NextAttemptAt:now-int64(i%4),LeaseUntil:0}
        notifications[i]=model.NotificationEventOutbox{Event:"login.failure",Reference:fmt.Sprint(i),
            Status:status,NextAttemptAt:now-int64(i%4),LeaseUntil:0}
    }
    if err:=DB.CreateInBatches(&push,100).Error;err!=nil{t.Fatal(err)}
    if err:=DB.CreateInBatches(&notifications,100).Error;err!=nil{t.Fatal(err)}
    if err:=DB.Exec("ANALYZE").Error;err!=nil{t.Fatal(err)}
    for _,table:=range []string{"push_event_outboxes","notification_event_outboxes"}{
        var rows []struct { Detail string `gorm:"column:detail"` }
        query:="EXPLAIN QUERY PLAN SELECT id FROM "+table+
            " WHERE status IN ('pending','retry') AND next_attempt_at <= ? AND lease_until <= ? ORDER BY id LIMIT 20"
        if err:=DB.Raw(query,now,now).Scan(&rows).Error;err!=nil{t.Fatal(err)}
        indexed:=false
        for _,row:=range rows {
            t.Logf("%s: %s",table,row.Detail)
            if strings.Contains(strings.ToUpper(row.Detail),"USING INDEX")||
                strings.Contains(strings.ToUpper(row.Detail),"USING COVERING INDEX"){
                indexed=true
            }
        }
        if !indexed {t.Fatalf("%s due event query did not use an index: %+v",table,rows)}
    }
}
