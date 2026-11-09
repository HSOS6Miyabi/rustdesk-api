package service

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/lejianwen/rustdesk-api/v2/model"
)

func clientShellQuote(value string) string {
	return "'" + strings.ReplaceAll(value,"'","'\\''") + "'"
}

func clientShellScript(platform,filename string,settings clientServerSettings)(string,string,string,error){
	if !validateClientHost(settings.Host) || !validateClientAPI(settings.API) ||
		(settings.Relay!=""&&!validateClientHost(settings.Relay)) {
		return "","","",errors.New("client configuration contains an invalid address")
	}
	publicKey,err:=base64.StdEncoding.DecodeString(settings.Key)
	if err!=nil||len(publicKey)!=32 {
		return "","","",errors.New("client configuration contains an invalid public key")
	}
	var scriptName,readme,script string
	switch platform {
	case "linux-x64","linux-arm64":
		if !strings.HasSuffix(filename,".deb") {return "","","",errors.New("invalid Linux package")}
		scriptName="install-and-configure.sh"
		script="#!/bin/sh\nset -eu\ncd \"$(dirname \"$0\")\"\n"+
			"if ! command -v apt-get >/dev/null 2>&1; then\n"+
			"  echo '该安装文件适用于 Debian 和 Ubuntu。其他发行版请使用官方安装说明。' >&2\n  exit 1\nfi\n"+
			"sudo apt-get install -y ./"+filename+"\n"+
			"RUSTDESK_BIN=$(command -v rustdesk || true)\n"+
			"if [ -z \"$RUSTDESK_BIN\" ]; then\n"+
			"  echo '找不到已安装的 rustdesk 命令。' >&2\n  exit 1\nfi\n"
		readme="RustDesk 官方 Linux 安装文件及服务器配置\n\n"+
			"1. 在 Debian/Ubuntu 上解压所有文件。\n"+
			"2. 打开终端，进入解压目录，运行 sh ./install-and-configure.sh。\n"+
			"3. 安装完成后打开 RustDesk，检查服务器地址并测试连接。\n"+
			"如需使用其他 Linux 发行版，请从 RustDesk 官方网站获取相应安装程序。\n"
	case "macos-x64","macos-arm64":
		if !strings.HasSuffix(filename,".dmg") {return "","","",errors.New("invalid macOS package")}
		scriptName="configure-rustdesk.command"
		script="#!/bin/sh\nset -eu\n"+
			"RUSTDESK_BIN=/Applications/RustDesk.app/Contents/MacOS/RustDesk\n"+
			"if [ ! -x \"$RUSTDESK_BIN\" ]; then\n"+
			"  echo '请先打开随附的 DMG，将 RustDesk.app 拖入应用程序目录，再运行本脚本。' >&2\n  exit 1\nfi\n"
		readme="RustDesk 官方 macOS 安装文件及服务器配置\n\n"+
			"1. 解压所有文件，打开随附的 DMG。\n"+
			"2. 将 RustDesk.app 拖入“应用程序”目录。\n"+
			"3. 在终端运行 sh ./configure-rustdesk.command，按系统要求授予权限。\n"+
			"4. 打开 RustDesk，检查服务器地址并测试远程连接。\n"+
			"脚本通过官方命令行设置服务器地址，不修改 DMG 或应用程序文件。\n"
	default:
		return "","","",errors.New("unsupported provisioning script platform")
	}
	bin := "\"$RUSTDESK_BIN\""
	for _,option:=range []struct{name,value string}{
		{"custom-rendezvous-server",settings.Host},
		{"key",settings.Key},
		{"api-server",settings.API},
		{"relay-server",settings.Relay},
	} {
		if option.value!="" {
			value:=clientShellQuote(option.value)
			script+="sudo "+bin+" --option "+option.name+" "+value+"\n"
			script+="ACTUAL=$(sudo "+bin+" --option "+option.name+")\n"
			script+="if [ \"$ACTUAL\" != "+value+" ]; then\n"+
				"  echo '无法确认 RustDesk 的 "+option.name+" 配置，请检查服务是否已安装并启动。' >&2\n"+
				"  exit 1\nfi\n"
		}
	}
	script+="echo 'RustDesk 服务器设置已保存并已核对。'\n"
	return scriptName,script,readme,nil
}

func prepareClientArchive(task *model.ClientBuildTask,source clientBuildSource,installerPath,folder string)(string,int64,error){
	if task==nil||task.Id==0||!strings.HasSuffix(task.Filename,".zip"){
		return "",0,errors.New("invalid client archive task")
	}
	var settings clientServerSettings
	if err:=json.Unmarshal([]byte(task.ConfigSnapshot),&settings);err!=nil{
		return "",0,err
	}
	sourceURL,err:=url.Parse(source.URL)
	if err!=nil{return "",0,err}
	filename:=path.Base(sourceURL.Path)
	if filename=="."||filename=="/"||strings.ContainsAny(filename,"/\\") {
		return "",0,errors.New("invalid official installer filename")
	}
	scriptName,script,readme,err:=clientShellScript(task.Platform,filename,settings)
	if err!=nil{return "",0,err}
	readme+="\n官方安装文件来源："+source.URL+"\n官方文件 SHA-256："+source.SHA256+"\n"+
		"本压缩包内安装文件内容保持原样。安装前请核查文件与所用操作系统。\n"
	tmp,err:=os.CreateTemp(folder,".client-package-*.tmp")
	if err!=nil{return "",0,err}
	tmpPath:=tmp.Name()
	defer os.Remove(tmpPath)
	zw:=zip.NewWriter(tmp)
	addFile:=func(name string,mode os.FileMode,reader io.Reader)error{
		if name==""||strings.ContainsAny(name,"/\\") {return errors.New("invalid archive entry")}
		header:=&zip.FileHeader{Name:name,Method:zip.Store}
		header.SetMode(mode)
		target,err:=zw.CreateHeader(header)
		if err!=nil{return err}
		_,err=io.Copy(target,reader)
		return err
	}
	installer,err:=os.Open(installerPath)
	if err!=nil{zw.Close();tmp.Close();return "",0,err}
	err=addFile(filename,0644,installer)
	installer.Close()
	if err==nil{err=addFile(scriptName,0755,strings.NewReader(script))}
	if err==nil{err=addFile("README.txt",0644,strings.NewReader(readme))}
	closeError:=zw.Close()
	fileError:=tmp.Close()
	if err!=nil{return "",0,err}
	if closeError!=nil{return "",0,closeError}
	if fileError!=nil{return "",0,fileError}
	archive,err:=os.Open(tmpPath)
	if err!=nil{return "",0,err}
	stat,err:=archive.Stat()
	if err!=nil{archive.Close();return "",0,err}
	if stat.Size()<=0||stat.Size()>100<<20 {
		archive.Close()
		return "",0,errors.New("client package size is invalid")
	}
	hash:=sha256.New()
	_,err=io.Copy(hash,archive)
	archive.Close()
	if err!=nil{return "",0,err}
	output:=filepath.Join(folder,task.Filename)
	if err:=os.Rename(tmpPath,output);err!=nil{return "",0,err}
	if err:=os.Chmod(output,0600);err!=nil{return "",0,err}
	return hex.EncodeToString(hash.Sum(nil)),stat.Size(),nil
}
