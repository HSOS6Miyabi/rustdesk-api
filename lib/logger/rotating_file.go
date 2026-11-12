package logger

import (
    "fmt"
    "os"
    "path/filepath"
    "regexp"
    "strings"
    "sync"
    "time"
)

const maxLogFileSize int64 = 32 << 20

var logArchiveName = regexp.MustCompile(`^[0-9]{8}-[0-9]{6}-[0-9]{3}$`)

// rotatingFileWriter keeps the currently written log at the configured path.
// A previous day's file and a full 32 MiB file are moved into dated archives.
type rotatingFileWriter struct {
    mu sync.Mutex
    path string
    file *os.File
    day string
    size int64
    now func() time.Time
}

func newRotatingFileWriter(path string, clock func() time.Time) (*rotatingFileWriter, error) {
    if clock == nil { clock = time.Now }
    if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil { return nil, err }
    file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0640)
    if err != nil { return nil, err }
    info, err := file.Stat()
    if err != nil { file.Close(); return nil, err }
    day := clock().Local().Format("20060102")
    if info.Size() != 0 { day = info.ModTime().Local().Format("20060102") }
    return &rotatingFileWriter{path:path,file:file,size:info.Size(),day:day,now:clock}, nil
}

func (w *rotatingFileWriter) Write(data []byte) (int,error) {
    w.mu.Lock()
    defer w.mu.Unlock()
    if w.file == nil { return 0, os.ErrClosed }
    now := w.now().Local()
    if w.size > 0 && (now.Format("20060102") != w.day ||
        (w.size+int64(len(data)) > maxLogFileSize && w.size >= maxLogFileSize/2)) {
        if err := w.rotate(now); err != nil { return 0, err }
    }
    n,err := w.file.Write(data)
    w.size += int64(n)
    return n,err
}

func (w *rotatingFileWriter) rotate(now time.Time) error {
    info,err := w.file.Stat()
    if err != nil { return err }
    // Use the former file's date, so an archived log from an old server
    // deployment remains eligible for its original retention deadline.
    archiveTime := info.ModTime().Local()
    if archiveTime.After(now) { archiveTime = now }
    suffix := archiveTime.Format("20060102-150405")
    var archive string
    for index:=0;index<1000;index++ {
        candidate:=fmt.Sprintf("%s.%s-%03d",w.path,suffix,index)
        if _,err:=os.Lstat(candidate);os.IsNotExist(err) {
            archive=candidate
            break
        } else if err!=nil { return err }
    }
    if archive=="" {return fmt.Errorf("cannot reserve a log archive name")}
    if err := w.file.Close(); err != nil { return err }
    if err := os.Rename(w.path,archive);err!=nil {
        w.file, _ = os.OpenFile(w.path,os.O_WRONLY|os.O_CREATE|os.O_APPEND,0640)
        return err
    }
    file,err := os.OpenFile(w.path,os.O_WRONLY|os.O_CREATE|os.O_APPEND,0640)
    if err != nil {return err}
    w.file = file
    w.day = now.Format("20060102")
    w.size = 0
    return nil
}

func (w *rotatingFileWriter) Close() error {
    w.mu.Lock()
    defer w.mu.Unlock()
    if w.file==nil {return nil}
    err:=w.file.Close()
    w.file=nil
    return err
}

// ProcessLogArchives counts or removes archived log files belonging to the
// configured logger path. The active log file is never removed.
func ProcessLogArchives(path string, now time.Time, days int, remove bool) (int64,error) {
    if path=="" {return 0,nil}
    if days<0||days>3650 {return 0,fmt.Errorf("invalid server log retention days")}
    dir,base:=filepath.Dir(path),filepath.Base(path)
    entries,err:=os.ReadDir(dir)
    if os.IsNotExist(err) {return 0,nil}
    if err!=nil{return 0,err}
    cutoff:=now.Add(-time.Duration(days)*24*time.Hour)
    var total int64
    for _,entry:=range entries {
        if !strings.HasPrefix(entry.Name(),base+".")||!entry.Type().IsRegular(){continue}
        suffix:=strings.TrimPrefix(entry.Name(),base+".")
        if !logArchiveName.MatchString(suffix){continue}
        archivedAt,err:=time.ParseInLocation("20060102-150405",suffix[:15],time.Local)
        if err!=nil {continue}
        // days=0 is the explicit "all archives" option.
        if days!=0&&!archivedAt.Before(cutoff){continue}
        if remove {
            if err:=os.Remove(filepath.Join(dir,entry.Name()));err!=nil {
                return total,err
            }
        }
        total++
    }
    return total,nil
}
