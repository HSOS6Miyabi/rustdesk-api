package config

import (
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
)

func TestBootstrapSettingsKeepPrivateLastWorkingCopy(t *testing.T) {
    path:=filepath.Join(t.TempDir(),"nested","bootstrap.json")
    first:=BootstrapSettings{Version:1,Overrides:map[string]SettingOverride{
        "app.internal-key":{Enabled:true,Value:"test-encryption-key-123456"},
    }}
    second:=BootstrapSettings{Version:2,Overrides:map[string]SettingOverride{
        "app.internal-key":{Enabled:true,Value:"test-encryption-key-next-123456"},
    }}
    if err:=writeBootstrapSettingsFile(path,first);err!=nil{t.Fatal(err)}
    if _,err:=os.Stat(path+".previous");!os.IsNotExist(err){
        t.Fatalf("unexpected previous settings file before replacement: %v",err)
    }
    if err:=writeBootstrapSettingsFile(path,second);err!=nil{t.Fatal(err)}
    for file,want:=range map[string]int64{path:2,path+".previous":1}{
        stat,err:=os.Stat(file)
        if err!=nil{t.Fatal(err)}
        if stat.Mode().Perm()!=0600 {
            t.Fatalf("settings file %q is readable by other accounts: %o",file,stat.Mode().Perm())
        }
        raw,err:=os.ReadFile(file)
        if err!=nil{t.Fatal(err)}
        var saved BootstrapSettings
        if err:=json.Unmarshal(raw,&saved);err!=nil{t.Fatal(err)}
        if saved.Version!=want {t.Fatalf("%q has version %d, want %d",file,saved.Version,want)}
    }
}
