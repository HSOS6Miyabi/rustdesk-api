package logger

import (
    "os"
    "path/filepath"
    "testing"
    "time"
)

func TestLogFilesRotateAndArchivesFollowRetention(t *testing.T) {
    folder:=t.TempDir()
    path:=filepath.Join(folder,"server.log")
    first:=time.Date(2026,time.October,1,11,0,0,0,time.Local)
    current:=first
    writer,err:=newRotatingFileWriter(path,func()time.Time{return current})
    if err!=nil{t.Fatal(err)}
    defer writer.Close()
    if _,err:=writer.Write([]byte("day one\n"));err!=nil{t.Fatal(err)}
    if err:=os.Chtimes(path,first,first);err!=nil{t.Fatal(err)}

    current=first.Add(25*time.Hour)
    if _,err:=writer.Write([]byte("day two\n"));err!=nil{t.Fatal(err)}
    active,err:=os.ReadFile(path)
    if err!=nil||string(active)!="day two\n"{
        t.Fatalf("current log is incorrect: %q %v",active,err)
    }
    archive:=path+".20261001-110000-000"
    saved,err:=os.ReadFile(archive)
    if err!=nil||string(saved)!="day one\n"{t.Fatalf("archived log mismatch: %q %v",saved,err)}
    unrelated:=path+".backup"
    if err:=os.WriteFile(unrelated,[]byte("preserve"),0600);err!=nil{t.Fatal(err)}

    count,err:=ProcessLogArchives(path,first.Add(26*time.Hour),30,false)
    if err!=nil||count!=0{t.Fatalf("too recent for retention: %d %v",count,err)}
    count,err=ProcessLogArchives(path,first.Add(32*24*time.Hour),30,false)
    if err!=nil||count!=1{t.Fatalf("wrong archived file count: %d %v",count,err)}
    count,err=ProcessLogArchives(path,first.Add(32*24*time.Hour),30,true)
    if err!=nil||count!=1{t.Fatalf("wrong archived file deletion count: %d %v",count,err)}
    if _,err:=os.Stat(archive);!os.IsNotExist(err){t.Fatalf("old archive remains: %v",err)}
    if _,err:=os.Stat(path);err!=nil{t.Fatalf("active log was deleted: %v",err)}
    if _,err:=os.Stat(unrelated);err!=nil{t.Fatalf("unrelated file was deleted: %v",err)}
}

func TestLogArchivesAllAndSymlinkProtection(t *testing.T) {
    folder:=t.TempDir()
    path:=filepath.Join(folder,"server.log")
    archived:=path+".20261001-020304-000"
    if err:=os.WriteFile(archived,[]byte("archive"),0600);err!=nil{t.Fatal(err)}
    if err:=os.WriteFile(path,[]byte("active"),0600);err!=nil{t.Fatal(err)}
    link:=path+".20261001-020304-001"
    if err:=os.Symlink(path,link);err!=nil{t.Fatal(err)}
    count,err:=ProcessLogArchives(path,time.Now(),0,true)
    if err!=nil||count!=1{t.Fatalf("unexpected archive cleanup: %d %v",count,err)}
    if _,err:=os.Stat(path);err!=nil{t.Fatalf("active file was removed: %v",err)}
    if _,err:=os.Lstat(link);err!=nil{t.Fatalf("symlink should not be followed: %v",err)}
}
