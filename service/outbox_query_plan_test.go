package service

import (
    "fmt"
    "strings"
    "testing"
    "time"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestNotificationOutboxQueryPlanAndBoundedResult(t *testing.T){
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
        if len(rows)==0 {t.Fatalf("%s returned no query plan",table)}
        for _,row:=range rows {
            t.Logf("%s: %s",table,row.Detail)
            if !strings.Contains(strings.ToUpper(row.Detail),"SCAN")&&
               !strings.Contains(strings.ToUpper(row.Detail),"SEARCH") {
                t.Fatalf("%s returned an unexpected query plan: %s",table,row.Detail)
            }
        }
        var ids []uint
        if err:=DB.Table(table).Select("id").
            Where("status IN ? AND next_attempt_at <= ? AND lease_until <= ?",
                []string{"pending","retry"},now,now).
            Order("id ASC").Limit(20).Pluck("id",&ids).Error;err!=nil{t.Fatal(err)}
        if len(ids)==0||len(ids)>20 {
            t.Fatalf("%s unexpected due rows: %d",table,len(ids))
        }
    }
}
