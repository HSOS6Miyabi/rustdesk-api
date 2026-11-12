package service

import (
 "strings"
 "testing"
 "gorm.io/driver/sqlite"
 "gorm.io/gorm"
 "github.com/lejianwen/rustdesk-api/v2/model"
)

func TestApplyColumnSortOrdersAcrossPages(t *testing.T){
 db,err:=gorm.Open(sqlite.Open("file:sort-query-test?mode=memory&cache=shared"),&gorm.Config{})
 if err!=nil{t.Fatal(err)}
 if err:=db.AutoMigrate(&model.User{});err!=nil{t.Fatal(err)}
 for _,username:=range []string{"Zulu","Charlie","Alpha","Echo","Bravo"}{
  if err:=db.Create(&model.User{Username:username,Nickname:username}).Error;err!=nil{t.Fatal(err)}
 }
 for _,tc:=range []struct{page int;want []string}{
  {0,[]string{"Alpha","Bravo"}},{1,[]string{"Charlie","Echo"}},{2,[]string{"Zulu"}},
 }{
  var users []model.User
  query:=db.Model(&model.User{}).Order("id DESC")
  ApplyColumnSort(query,"username","asc")
  if err:=query.Offset(tc.page*2).Limit(2).Find(&users).Error;err!=nil{t.Fatal(err)}
  if len(users)!=len(tc.want){t.Fatalf("page %d returned %d rows",tc.page,len(users))}
  for i,u:=range users{if u.Username!=tc.want[i]{t.Fatalf("page %d item %d: got %q want %q",tc.page,i,u.Username,tc.want[i])}}
 }
}

func TestApplyColumnSortRejectsSQLExpressions(t *testing.T){
 db,err:=gorm.Open(sqlite.Open("file:sort-query-injection?mode=memory&cache=shared"),&gorm.Config{DryRun:true})
 if err!=nil{t.Fatal(err)}
 query:=db.Model(&model.User{}).Order("id DESC")
 ApplyColumnSort(query,"username; DROP TABLE users","desc")
 var users []model.User
 result:=query.Find(&users)
 if strings.Contains(result.Statement.SQL.String(),"DROP TABLE"){t.Fatalf("unsafe sort clause: %s",result.Statement.SQL.String())}
}
