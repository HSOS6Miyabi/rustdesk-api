package cache

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileCacheGcRemovesOnlyExpiredEntries(t *testing.T) {
	dir := t.TempDir()
	cache := NewFileCache()
	cache.SetDir(dir)
	if err := cache.Set("expired", "old", 1); err != nil { t.Fatal(err) }
	expired := cache.fileName("expired")
	oldTime := time.Now().Add(-time.Hour)
	if err := os.Chtimes(expired, oldTime, oldTime); err != nil { t.Fatal(err) }
	if err := cache.Set("live", "present", 60); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("keep"), 0600); err != nil { t.Fatal(err) }
	otherHash := strings.Repeat("f", 32)
	if err := os.Mkdir(filepath.Join(dir, otherHash), 0700); err != nil { t.Fatal(err) }
	if err := cache.Gc(); err != nil { t.Fatal(err) }
	if _, err := os.Stat(expired); !os.IsNotExist(err) { t.Fatalf("expired entry was retained: %v", err) }
	for _, path := range []string{cache.fileName("live"), filepath.Join(dir, "notes.txt"),
		filepath.Join(dir, otherHash)} {
		if _, err := os.Stat(path); err != nil { t.Fatalf("unrelated cache file was removed: %s %v", path, err) }
	}
}
