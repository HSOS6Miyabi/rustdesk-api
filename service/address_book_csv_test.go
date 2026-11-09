package service

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strconv"
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestAddressBookCSVImportExportAndPreservePasswords(t *testing.T) {
	cleanup := setupGroupDB(t)
	defer cleanup()
	if err := DB.AutoMigrate(&model.AddressBook{}, &model.AddressBookCollection{}); err != nil { t.Fatal(err) }
	users := seedGroups(t)
	owner, other := users["member"], users["other"]
	book := &model.AddressBookCollection{UserId:owner.Id,Name:"工作设备"}
	if err := DB.Create(book).Error; err != nil { t.Fatal(err) }
	stored := &model.AddressBook{
		Id:"desk-000001",UserId:owner.Id,CollectionId:book.Id,
		Alias:"旧名称",Password:"existing-password",Hash:"existing-hash",
		Tags:[]byte(`["old"]`),
	}
	if err := DB.Create(stored).Error; err != nil { t.Fatal(err) }
	source := "id,alias,hostname,username,platform,collection_id,tags,note,force_always_relay\n" +
		"desk-000001,新名称,主机A,guest,Windows," +
		strconv.FormatUint(uint64(book.Id),10)+",工作;远程,设备维护,true\n" +
		"desk-000002,'=SUM(1+1),主机B,admin,Linux,"+strconv.FormatUint(uint64(book.Id),10)+",工作,备注,false\n"
	result, err := ImportAddressBookCSV(owner.Id,strings.NewReader(source))
	if err != nil { t.Fatal(err) }
	if result.Created!=1 || result.Updated!=1 { t.Fatalf("unexpected import summary: %+v",result) }
	repeated, err := ImportAddressBookCSV(owner.Id,strings.NewReader(source))
	if err != nil || repeated.Created!=0 || repeated.Updated!=2 { t.Fatalf("import is not repeatable: %+v %v",repeated,err) }
	var existing model.AddressBook
	if err:=DB.Where("user_id = ? AND id = ?",owner.Id,"desk-000001").First(&existing).Error;err!=nil{t.Fatal(err)}
	if existing.Password!="existing-password" || existing.Hash!="existing-hash" ||
		existing.Alias!="新名称" || !existing.ForceAlwaysRelay {
		t.Fatalf("previous connection credentials were changed: %+v",existing)
	}
	var tags []string
	if err:=json.Unmarshal(existing.Tags,&tags);err!=nil || len(tags)!=2 || tags[0]!="工作"{
		t.Fatalf("tags were not saved: %v %v",tags,err)
	}
	var buffer bytes.Buffer
	if err:=ExportAddressBookCSV(owner.Id,&buffer);err!=nil{t.Fatal(err)}
	if !strings.HasPrefix(buffer.String(),"\ufeff"){t.Fatal("missing UTF-8 signature")}
	if strings.Contains(buffer.String(),"existing-password")||strings.Contains(buffer.String(),"existing-hash"){
		t.Fatal("sensitive connection settings were exported")
	}
	reader:=csv.NewReader(strings.NewReader(strings.TrimPrefix(buffer.String(),"\ufeff")))
	records,err:=reader.ReadAll()
	if err!=nil{t.Fatal(err)}
	if len(records)!=3 || records[2][1]!="'=SUM(1+1)"{
		t.Fatalf("spreadsheet formulas were not neutralized: %+v",records)
	}
	if outcome,err:=ImportAddressBookCSV(other.Id,strings.NewReader(source));err==nil{
		t.Fatalf("foreign collection import was accepted: %+v",outcome)
	}
	var total int64
	if err:=DB.Model(&model.AddressBook{}).Where("user_id = ?",other.Id).Count(&total).Error;err!=nil||total!=0{
		t.Fatalf("foreign account was changed: %d %v",total,err)
	}
}

func TestAddressBookCSVValidationIsAtomic(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.AddressBook{},&model.AddressBookCollection{});err!=nil{t.Fatal(err)}
	owner:=seedGroups(t)["member"]
	for _,test:=range []string{
		"id,alias\ndesk-001,new\ndesk-001,duplicate\n",
		"id,collection_id\ndesk-001,0\ndesk-002,99999\n",
		"alias,hostname\nmissing,device\n",
		"id,unknown_column\ndesk-001,abc\n",
		"id,force_always_relay\ndesk-001,definitely\n",
		"id,alias\ndesk-001,valid\ndesk-002",
	}{
		if _,err:=ImportAddressBookCSV(owner.Id,strings.NewReader(test));err==nil{
			t.Fatalf("invalid CSV was accepted: %q",test)
		}
		var count int64
		if err:=DB.Model(&model.AddressBook{}).Where("user_id = ?",owner.Id).Count(&count).Error;err!=nil||count!=0{
			t.Fatalf("invalid CSV partially changed database: %d %v",count,err)
		}
	}
}

