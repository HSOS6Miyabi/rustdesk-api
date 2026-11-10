package service

import (
	"sync"
	"time"
)

var pendingPushEvents sync.WaitGroup

// FlushPendingEvents waits until accepted in-memory events have been written
// to their durable delivery tables. Delivery itself can resume after restart.
func FlushPendingEvents(timeout time.Duration) bool {
	if timeout<=0 {timeout=8*time.Second}
	finished:=make(chan struct{})
	go func(){
		pendingPushEvents.Wait()
		close(finished)
	}()
	timer:=time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-finished:return true
	case <-timer.C:return false
	}
}
