package service

import (
	"context"
	"time"

	"github.com/lejianwen/rustdesk-api/v2/lib/cache"
)

// Expired file-cache entries are cleaned only when this backend is configured.
func StartFileCacheMaintenance(fileCache *cache.FileCache) {
	if fileCache == nil { return }
	launchServiceWorker(func(ctx context.Context) {
		cleanup := func() {
			if err := fileCache.Gc(); err != nil && Logger != nil {
				Logger.WithError(err).Warn("file cache cleanup failed")
			}
		}
		cleanup()
		ticker := time.NewTicker(24*time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done(): return
			case <-ticker.C: cleanup()
			}
		}
	})
}
