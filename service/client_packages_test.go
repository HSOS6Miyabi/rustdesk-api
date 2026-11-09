package service

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func TestOfficialLinuxAndMacInstallerPackagesPreserveOriginalFiles(t *testing.T){
	cleanup:=setupGroupDB(t)
	defer cleanup()
	if err:=DB.AutoMigrate(&model.ClientBuildProfile{},&model.ClientBuildTask{});err!=nil{t.Fatal(err)}
	t.Setenv("RUSTDESK_CLIENT_BUILD_DIR",t.TempDir())
	original:=[]byte("offline official installer fixture for packaging tests")
	sum:=sha256.Sum256(original)
	checksum:=hex.EncodeToString(sum[:])
	mock:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
		w.WriteHeader(http.StatusOK)
		_,_=w.Write(original)
	}))
	defer mock.Close()
	for _,tc:=range []struct{platform,asset,script string}{
		{"linux-x64","rustdesk-1.5.1-x86_64.deb","install-and-configure.sh"},
		{"linux-arm64","rustdesk-1.5.1-aarch64.deb","install-and-configure.sh"},
		{"macos-x64","rustdesk-1.5.1-x86_64.dmg","configure-rustdesk.command"},
		{"macos-arm64","rustdesk-1.5.1-aarch64.dmg","configure-rustdesk.command"},
	}{
		t.Run(tc.platform,func(t *testing.T){
			profile:=validClientProfile()
			profile.Platform=tc.platform
			if err:=SaveClientBuildProfile(profile);err!=nil{t.Fatal(err)}
			task,err:=QueueClientBuild(profile.Id,9)
			if err!=nil||!strings.HasSuffix(task.Filename,".zip"){
				t.Fatalf("configuration package was not queued: %+v %v",task,err)
			}
			source:=clientBuildSource{
				URL:mock.URL+"/"+tc.asset,SHA256:checksum,Size:int64(len(original)),
			}
			digest,size,err:=writeClientPackage(task,source,mock.Client())
			if err!=nil||size<=int64(len(original))||len(digest)!=64{
				t.Fatalf("package generation failed: %d %s %v",size,digest,err)
			}
			location:=filepath.Join(clientBuildRoot(),fmtTaskID(task.Id),task.Filename)
			zr,err:=zip.OpenReader(location)
			if err!=nil{t.Fatal(err)}
			items:=map[string][]byte{}
			for _,entry:=range zr.File {
				r,err:=entry.Open()
				if err!=nil{t.Fatal(err)}
				data,err:=io.ReadAll(r)
				r.Close()
				if err!=nil{t.Fatal(err)}
				items[entry.Name]=data
			}
			zr.Close()
			if !bytes.Equal(items[tc.asset],original) {
				t.Fatal("official installer bytes changed inside the custom package")
			}
			script:=items[tc.script]
			for _,required:=range []string{
				"custom-rendezvous-server","--option key","--option api-server",
				"--option relay-server",profile.Host,profile.PublicKey,profile.APIServer,
			}{
				if !bytes.Contains(script,[]byte(required)){
					t.Fatalf("configuration script lacks %q",required)
				}
			}
			if !bytes.Contains(items["README.txt"],[]byte(checksum)){
				t.Fatal("package instructions do not include the official installer checksum")
			}
			if runtime.GOOS!="windows" {
				command:=exec.Command("sh","-n")
				command.Stdin=bytes.NewReader(script)
				if out,err:=command.CombinedOutput();err!=nil{
					t.Fatalf("client setup script has invalid shell syntax: %v %s",err,out)
				}
			}
			if runtime.GOOS=="linux" && strings.HasPrefix(tc.platform,"linux") {
				directory:=t.TempDir()
				binaryDir:=filepath.Join(directory,"bin")
				if err:=os.MkdirAll(binaryDir,0700);err!=nil{t.Fatal(err)}
				for name,body:=range map[string]string{
					"sudo":"#!/bin/sh\nprintf '%s\\n' \"$*\" >> \"$RUSTDESK_INSTALL_TEST_LOG\"\n",
					"rustdesk":"#!/bin/sh\nexit 0\n",
				}{
					if err:=os.WriteFile(filepath.Join(binaryDir,name),[]byte(body),0755);err!=nil{t.Fatal(err)}
				}
				installScript:=filepath.Join(directory,tc.script)
				if err:=os.WriteFile(installScript,script,0755);err!=nil{t.Fatal(err)}
				logfile:=filepath.Join(directory,"commands.txt")
				command:=exec.Command("sh",installScript)
				command.Dir=directory
				command.Env=append(os.Environ(),
					"PATH="+binaryDir+string(os.PathListSeparator)+os.Getenv("PATH"),
					"RUSTDESK_INSTALL_TEST_LOG="+logfile)
				if output,err:=command.CombinedOutput();err!=nil{
					t.Fatalf("simulated Linux setup did not finish: %v %s",err,output)
				}
				data,err:=os.ReadFile(logfile)
				if err!=nil{t.Fatal(err)}
				commands:=string(data)
				for _,required:=range []string{
					"apt-get install -y ./"+tc.asset,
					"--option custom-rendezvous-server "+profile.Host,
					"--option key "+profile.PublicKey,
					"--option api-server "+profile.APIServer,
					"--option relay-server "+profile.RelayServer,
				}{
					if !strings.Contains(commands,required){
						t.Fatalf("simulated setup omitted %q",required)
					}
				}
				if strings.Count(strings.TrimSpace(commands),"\n")!=4{
					t.Fatalf("simulated setup ran an unexpected number of commands: %s",commands)
				}
			}
			if err:=DB.Model(task).Updates(map[string]interface{}{
				"status":"ready","sha256":digest,"bytes":size,
			}).Error;err!=nil{t.Fatal(err)}
			if _,_,err:=ClientBuildFile(task.Id);err!=nil{t.Fatal(err)}
			// Corruption of a packaged installer must also invalidate downloads.
			file,err:=os.OpenFile(location,os.O_WRONLY|os.O_APPEND,0600)
			if err!=nil{t.Fatal(err)}
			_,err=file.Write([]byte("changed"))
			file.Close()
			if err!=nil{t.Fatal(err)}
			if _,_,err:=ClientBuildFile(task.Id);err==nil{
				t.Fatal("modified configuration archive was accepted for download")
			}
		})
	}
}

func TestClientSetupScriptQuotesPublicConfiguration(t *testing.T){
	settings:=clientServerSettings{
		Host:"desk.example.com:21116",
		Key:validClientProfile().PublicKey,
		API:"https://desk.example.com",
		Relay:"relay.example.com:21117",
	}
	for _,platform:=range []string{"linux-x64","macos-arm64"}{
		filename:="rustdesk-1.5.1-x86_64.deb"
		if strings.HasPrefix(platform,"macos"){filename="rustdesk-1.5.1-aarch64.dmg"}
		_,script,_,err:=clientShellScript(platform,filename,settings)
		if err!=nil{t.Fatal(err)}
		if strings.Count(script," --option ")!=4{
			t.Fatalf("expected four official client configuration commands: %s",script)
		}
	}
	if !strings.Contains(clientShellQuote("a'b"),"'\\''"){
		t.Fatal("shell configuration values are not correctly quoted")
	}
}
