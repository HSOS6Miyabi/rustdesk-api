package service

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func validClientProfile() *model.ClientBuildProfile {
	return &model.ClientBuildProfile{
		Name:"Office support",Platform:"windows-x64",Version:"1.5.1",
		Host:"desk.example.com:21116",
		PublicKey:base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7},32)),
		APIServer:"https://desk.example.com",RelayServer:"relay.example.com:21117",
	}
}

func TestClientBuildSettingsAndImmutableTask(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
	profile:=validClientProfile()
	if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
	name,err:=BuildClientFilename(profile)
	if err!=nil || !strings.HasPrefix(name,"rustdesk-licensed-")||!strings.HasSuffix(name,".exe"){
		t.Fatalf("invalid official RustDesk custom-client filename: %s %v",name,err)
	}
	encoded:=strings.TrimSuffix(strings.TrimPrefix(name,"rustdesk-licensed-"),".exe")
	runes:=[]byte(encoded)
	for i,j:=0,len(runes)-1;i<j;i,j=i+1,j-1{runes[i],runes[j]=runes[j],runes[i]}
	raw,err:=base64.RawURLEncoding.DecodeString(string(runes))
	if err!=nil{t.Fatal(err)}
	var embedded clientServerSettings
	if err:=json.Unmarshal(raw,&embedded);err!=nil{t.Fatal(err)}
	if embedded.Host!=profile.Host||embedded.Key!=profile.PublicKey||
		embedded.API!=profile.APIServer||embedded.Relay!=profile.RelayServer {
		t.Fatalf("embedded server profile mismatch: %+v",embedded)
	}
	task,err:=QueueClientBuild(profile.Id,3)
	if err!=nil||task.Status!="pending"||task.Filename!=name{t.Fatalf("build was not queued: %+v %v",task,err)}
	profile.Host="second.example.com:21116"
	if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
	var frozen model.ClientBuildTask
	if err:=DB.First(&frozen,task.Id).Error;err!=nil{t.Fatal(err)}
	if !strings.Contains(frozen.ConfigSnapshot,"desk.example.com:21116") ||
		strings.Contains(frozen.ConfigSnapshot,"second.example.com") {
		t.Fatalf("queued build changed after profile edit: %+v",frozen)
	}
	records,total,err:=ListClientBuildTasks(1,20)
	if err!=nil||total!=1||len(records)!=1{t.Fatalf("build history unavailable: %d %v",total,err)}
	if err:=RetryClientBuild(task.Id);err==nil{t.Fatal("pending job was incorrectly retried")}
	if err:=DB.Model(task).Update("status","failed").Error;err!=nil{t.Fatal(err)}
	if err:=RetryClientBuild(task.Id);err!=nil{t.Fatal(err)}
	if err:=DB.First(&frozen,task.Id).Error;err!=nil || frozen.Status!="pending"{
		t.Fatalf("failed build did not return to the queue: %+v %v",frozen,err)
	}
	if err:=DropClientBuildProfile(profile.Id);err!=nil{t.Fatal(err)}
	if records,total,err:=ListClientBuildTasks(1,20);err!=nil||total!=1||len(records)!=1{
		t.Fatalf("deleting a profile destroyed build history: %d %v",total,err)
	}
}

func TestClientBuildRejectsUnsupportedAndMaliciousSettings(t *testing.T) {
	for _,modify:=range []func(*model.ClientBuildProfile){
		func(p *model.ClientBuildProfile){p.Platform="android"},
		func(p *model.ClientBuildProfile){p.Version="unknown"},
		func(p *model.ClientBuildProfile){p.Host="desk.example.com\r\nInvalid: header"},
		func(p *model.ClientBuildProfile){p.Host="https://evil.example.com/path"},
		func(p *model.ClientBuildProfile){p.PublicKey="short"},
		func(p *model.ClientBuildProfile){p.APIServer="https://admin:password@desk.example.com"},
		func(p *model.ClientBuildProfile){p.RelayServer="relay.example.com,api=attacker"},
		func(p *model.ClientBuildProfile){p.Name=""}, 
	}{
		p:=validClientProfile()
		modify(p)
		if err:=ValidateClientBuildProfile(p);err==nil{
			t.Fatalf("invalid custom profile was accepted: %+v",p)
		}
	}
	for _,platform:=range []string{"windows-x64","windows-arm64","linux-x64","linux-arm64","macos-x64","macos-arm64"}{
		name,err:=officialClientAssetName(platform,"1.5.3")
		if err!=nil{t.Fatal(err)}
		source,err:=officialSourceFromRelease(platform,"1.5.3",officialRustDeskRelease{
			TagName:"1.5.3",
			Assets:[]officialRustDeskAsset{{
				Name:name,Size:256, Digest:"sha256:"+strings.Repeat("a",64),
				BrowserDownloadURL:"https://github.com/rustdesk/rustdesk/releases/download/1.5.3/"+name,
			}},
		})
		if err!=nil||source.Size!=256||source.SHA256!=strings.Repeat("a",64){
			t.Fatalf("official installer metadata is invalid for %s: %+v %v",platform,source,err)
		}
	}
	if _,err:=officialClientAssetName("android","1.5.3");err==nil{
		t.Fatal("unsupported installer platform was accepted")
	}
}

func TestClientBuildPackagesOfficialBytesWithoutModification(t *testing.T) {
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
	t.Setenv("RUSTDESK_CLIENT_BUILD_DIR",t.TempDir())
	profile:=validClientProfile()
	if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
	task,err:=QueueClientBuild(profile.Id,5)
	if err!=nil{t.Fatal(err)}
	fixture:=[]byte("mock signed RustDesk Windows executable content")
	digest:=sha256.Sum256(fixture)
	hash:=hex.EncodeToString(digest[:])
	mock:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.Header().Set("Content-Type","application/octet-stream")
		_,_ = w.Write(fixture)
	}))
	defer mock.Close()
	source:=clientBuildSource{URL:mock.URL,SHA256:hash,Size:int64(len(fixture))}
	returned,size,err:=writeClientPackage(task,source,mock.Client())
	if err!=nil||returned!=hash||size!=int64(len(fixture)){
		t.Fatalf("client artifact generation failed: %s %d %v",returned,size,err)
	}
	path:=filepath.Join(clientBuildRoot(),fmtTaskID(task.Id),task.Filename)
	stored,err:=os.ReadFile(path)
	if err!=nil{t.Fatal(err)}
	if !bytes.Equal(stored,fixture){t.Fatal("building changed the official installer contents")}
	if err:=DB.Model(task).Updates(map[string]interface{}{
		"status":"ready","sha256":hash,"bytes":size,
	}).Error;err!=nil{t.Fatal(err)}
	pathFromService,name,err:=ClientBuildFile(task.Id)
	if err!=nil||name!=task.Filename||pathFromService!=path{
		t.Fatalf("authenticated file lookup failed: %s %s %v",pathFromService,name,err)
	}
	source.SHA256=strings.Repeat("0",64)
	if _,_,err:=writeClientPackage(task,source,mock.Client());err==nil{
		t.Fatal("invalid official download checksum was accepted")
	}
	after,err:=os.ReadFile(path)
	if err!=nil||!bytes.Equal(after,fixture){t.Fatal("checksum rejection modified the installed artifact")}

	tampered:=append([]byte(nil),fixture...)
	tampered[len(tampered)-1]^=1
	if err:=os.WriteFile(path,tampered,0600);err!=nil{t.Fatal(err)}
	if _,_,err:=ClientBuildFile(task.Id);err==nil{
		t.Fatal("download accepted a client package whose bytes changed after preparation")
	}
	if err:=os.WriteFile(path,fixture,0600);err!=nil{t.Fatal(err)}
	if _,_,err:=ClientBuildFile(task.Id);err!=nil{t.Fatal(err)}
	openFile,downloadName,downloadSize,err:=OpenVerifiedClientBuild(task.Id)
	if err!=nil||downloadName!=task.Filename||downloadSize!=int64(len(fixture)){
		t.Fatalf("verified streaming file is unavailable: %s %d %v",downloadName,downloadSize,err)
	}
	renamed:=path+".original"
	if err:=os.Rename(path,renamed);err!=nil{t.Fatal(err)}
	if err:=os.WriteFile(path,[]byte("different file"),0600);err!=nil{t.Fatal(err)}
	streamed,err:=io.ReadAll(openFile)
	openFile.Close()
	if err!=nil||!bytes.Equal(streamed,fixture){
		t.Fatal("installer streaming reopened an unverified replacement path")
	}
	if err:=os.Remove(path);err!=nil{t.Fatal(err)}
	if err:=os.Rename(renamed,path);err!=nil{t.Fatal(err)}
	link:=path+".backup"
	if err:=os.Rename(path,link);err!=nil{t.Fatal(err)}
	if err:=os.Symlink(link,path);err!=nil{t.Fatal(err)}
	if _,_,err:=ClientBuildFile(task.Id);err==nil{
		t.Fatal("download accepted a symlink instead of a verified package")
	}
}

func fmtTaskID(id uint)string{return strconv.FormatUint(uint64(id),10)}


func TestInterruptedClientInstallerTasksCanBeRetried(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
	profile:=validClientProfile()
	if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
	expired,err:=QueueClientBuild(profile.Id,3)
	if err!=nil{t.Fatal(err)}
	recent,err:=QueueClientBuild(profile.Id,3)
	if err!=nil{t.Fatal(err)}
	now:=time.Now().Unix()
	for _,entry:=range []struct{ID uint;started int64}{
		{expired.Id,now-900},{recent.Id,now-60},
	}{
		if err:=DB.Model(&model.ClientBuildTask{}).Where("id = ?",entry.ID).
            Updates(map[string]interface{}{"status":"building","started_at":entry.started}).Error;err!=nil{t.Fatal(err)}
        if err:=DB.Model(&model.ClientBuildTask{}).Where("id = ?",entry.ID).
            UpdateColumn("updated_at",time.Unix(entry.started,0)).Error;err!=nil{t.Fatal(err)}
    }
	if err:=RecoverInterruptedClientBuilds(now);err!=nil{t.Fatal(err)}
	var oldTask,newTask model.ClientBuildTask
	if err:=DB.First(&oldTask,expired.Id).Error;err!=nil{t.Fatal(err)}
	if err:=DB.First(&newTask,recent.Id).Error;err!=nil{t.Fatal(err)}
	if oldTask.Status!="failed"||oldTask.FinishedAt!=now||oldTask.Error==""{
		t.Fatalf("abandoned build task remains blocked: %+v",oldTask)
	}
	if newTask.Status!="building"||newTask.Error!=""{
		t.Fatalf("a recent build was interrupted incorrectly: %+v",newTask)
	}
	if err:=RetryClientBuild(oldTask.Id);err!=nil{t.Fatal(err)}
	if err:=DB.First(&oldTask,expired.Id).Error;err!=nil{t.Fatal(err)}
	if oldTask.Status!="pending"||oldTask.StartedAt!=0||oldTask.Error!=""{
		t.Fatalf("interrupted installer was not returned to the queue: %+v",oldTask)
	}
}
