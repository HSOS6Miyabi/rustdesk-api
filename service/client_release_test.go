package service

import (
    "bytes"
    "context"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "os"
    "path/filepath"
    "strings"
    "testing"

    "github.com/lejianwen/rustdesk-api/v2/model"
)

func officialReleaseFixture(version, platform, digest string) officialRustDeskRelease {
    name,_:=officialClientAssetName(platform,version)
    return officialRustDeskRelease{
        TagName:version,
        Assets:[]officialRustDeskAsset{{
            Name:name,
            Size:128,
            Digest:digest,
            BrowserDownloadURL:"https://github.com/rustdesk/rustdesk/releases/download/"+version+"/"+name,
        }},
    }
}

func TestOfficialClientReleaseAssetSelection(t *testing.T) {
    for _,version:=range []string{"1.4.5","1.5.1","1.5.3"} {
        for _,platform:=range officialClientPlatforms {
            fixture:=officialReleaseFixture(version,platform,"sha256:"+strings.Repeat("b",64))
            source,err:=officialSourceFromRelease(platform,version,fixture)
            if err!=nil{t.Fatalf("%s %s: %v",version,platform,err)}
            if source.Size!=128||source.SHA256!=strings.Repeat("b",64)||
                !strings.HasPrefix(source.URL,"https://github.com/rustdesk/rustdesk/releases/download/"+version+"/"){
                t.Fatalf("unexpected official source: %+v",source)
            }
        }
    }
    old:=officialReleaseFixture("1.3.9","windows-x64","")
    source,err:=officialSourceFromRelease("windows-x64","1.3.9",old)
    if err!=nil||source.SHA256!=""{t.Fatalf("legacy release should be supported: %+v %v",source,err)}
    for _,bad:=range []string{"1.5.3/../archive","latest","v1.5.3","1.5.3-dev","1.5.3?x"}{
        if validOfficialClientVersion(bad){t.Fatalf("invalid version accepted: %q",bad)}
    }
    if _,err:=officialSourceFromRelease("windows-arm64","1.3.9",old);err==nil{
        t.Fatal("missing official Windows ARM64 installer was accepted")
    }
    unsigned:=officialReleaseFixture("1.5.3","linux-x64","")
    unsigned.Prerelease=true
    if _,err:=officialSourceFromRelease("linux-x64","1.5.3",unsigned);err==nil{
        t.Fatal("prerelease unexpectedly accepted")
    }
    corrupted:=officialReleaseFixture("1.5.3","linux-x64","sha256:invalid")
    if _,err:=officialSourceFromRelease("linux-x64","1.5.3",corrupted);err==nil{
        t.Fatal("incorrect official digest was accepted")
    }
    anotherHost:=officialReleaseFixture("1.5.3","linux-x64","")
    anotherHost.Assets[0].BrowserDownloadURL="https://example.test/rustdesk.deb"
    if _,err:=officialSourceFromRelease("linux-x64","1.5.3",anotherHost);err==nil{
        t.Fatal("third-party download URL was accepted")
    }
}

func TestOfficialClientVersionCatalog(t *testing.T){
    release:=officialReleaseFixture("1.5.3","windows-x64","sha256:"+strings.Repeat("a",64))
    second,_:=officialClientAssetName("linux-x64","1.5.3")
    release.Assets=append(release.Assets,officialRustDeskAsset{
        Name:second,Size:1024,Digest:"sha256:"+strings.Repeat("c",64),
        BrowserDownloadURL:"https://github.com/rustdesk/rustdesk/releases/download/1.5.3/"+second,
    })
    old:=officialReleaseFixture("1.4.5","linux-arm64","")
    rejected:=officialReleaseFixture("1.6.0","windows-x64","")
    rejected.Prerelease=true
    versions:=versionsFromOfficialReleases([]officialRustDeskRelease{release,old,rejected,release})
    if len(versions)!=2||versions[0].Version!="1.5.3"||
        len(versions[0].Platforms)!=2||versions[1].Version!="1.4.5"{
        t.Fatalf("invalid versions catalog: %+v",versions)
    }
}

func TestOfficialClientReleaseHTTPMetadata(t *testing.T){
    release:=officialReleaseFixture("1.5.3","linux-x64","sha256:"+strings.Repeat("f",64))
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        if r.URL.Path!="/tags/1.5.3"{http.NotFound(w,r);return}
        w.Header().Set("Content-Type","application/json")
        json.NewEncoder(w).Encode(release)
    }))
    defer server.Close()
    source,err:=officialClientBuildSourceWithURL(context.Background(),server.Client(),server.URL,"linux-x64","1.5.3")
    if err!=nil||source.Size!=128{t.Fatalf("cannot read official release JSON: %+v %v",source,err)}
    if _,err=officialClientBuildSourceWithURL(context.Background(),server.Client(),server.URL,"linux-x64","1.0.0");err==nil{
        t.Fatal("non-existent release must fail")
    }
}

func TestOfficialLegacyClientDownloadRecordsComputedChecksum(t *testing.T) {
    cleanup:=setupGroupDB(t)
    defer cleanup()
    if err:=DB.AutoMigrate(&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
    t.Setenv("RUSTDESK_CLIENT_BUILD_DIR",t.TempDir())
    profile:=validClientProfile()
    profile.Version="1.3.9"
    if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
    task,err:=QueueClientBuild(profile.Id,5)
    if err!=nil{t.Fatal(err)}
    file:=append([]byte{'M','Z'},bytes.Repeat([]byte{7},254)...)
    sum:=sha256.Sum256(file)
    want:=hex.EncodeToString(sum[:])
    server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
        w.WriteHeader(http.StatusOK)
        w.Write(file)
    }))
    defer server.Close()
    source:=clientBuildSource{URL:server.URL,Size:int64(len(file))}
    digest,size,err:=writeClientPackage(task,source,server.Client())
    if err!=nil||digest!=want||size!=int64(len(file)){
        t.Fatalf("legacy download checksum was not recorded: %s %d %v",digest,size,err)
    }
    if err:=DB.Model(task).Updates(map[string]interface{}{
        "status":"ready","sha256":digest,"bytes":size,
    }).Error;err!=nil{t.Fatal(err)}
    actual:=filepath.Join(clientBuildRoot(),fmtTaskID(task.Id),task.Filename)
    contents,err:=os.ReadFile(actual)
    if err!=nil||!bytes.Equal(contents,file){t.Fatalf("official EXE bytes changed: %v",err)}
    if _,_,err:=ClientBuildFile(task.Id);err!=nil{t.Fatal(err)}
}
