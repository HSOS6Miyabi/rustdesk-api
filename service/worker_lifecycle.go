package service

import (
	"context"
	"sync"
	"time"
)

// The event admission lock prevents a WaitGroup.Add from racing with shutdown.
var eventAdmission = struct {
	sync.RWMutex
	closed bool
}{}

var serviceWorkers = struct {
	sync.Mutex
	context context.Context
	cancel context.CancelFunc
	pending sync.WaitGroup
	stopping bool
}{}

func launchServiceWorker(run func(context.Context)) {
	serviceWorkers.Lock()
	if serviceWorkers.stopping {
		serviceWorkers.Unlock()
		return
	}
	if serviceWorkers.context == nil {
		serviceWorkers.context, serviceWorkers.cancel = context.WithCancel(context.Background())
	}
	ctx := serviceWorkers.context
	serviceWorkers.pending.Add(1)
	serviceWorkers.Unlock()
	go func() {
		defer serviceWorkers.pending.Done()
		run(ctx)
	}()
}

// StopServiceWorkers rejects new in-memory events, flushes accepted heartbeat
// events, and waits for all database-using background workers to finish.
func StopServiceWorkers(timeout time.Duration) bool {
	if timeout <= 0 { timeout = 30 * time.Second }
	start := time.Now()
	eventAdmission.Lock()
	eventAdmission.closed = true
	eventAdmission.Unlock()
	flushed := FlushPendingEvents(timeout)
	serviceWorkers.Lock()
	serviceWorkers.stopping = true
	if serviceWorkers.cancel != nil { serviceWorkers.cancel() }
	serviceWorkers.Unlock()
	finished := make(chan struct{})
	go func() {
		serviceWorkers.pending.Wait()
		close(finished)
	}()
	left := timeout - time.Since(start)
	if left < 0 { left = 0 }
	timer := time.NewTimer(left)
	defer timer.Stop()
	select {
	case <-finished:
		return flushed
	case <-timer.C:
		return false
	}
}

// waitForWorkContext never polls an empty queue. Cancellation interrupts both
// sleeping workers and workers waiting for a persisted deadline.
func waitForWorkContext(ctx context.Context, wake <-chan struct{},
	delay time.Duration, hasDeadline bool) bool {
	if ctx.Err() != nil { return false }
	if !hasDeadline {
		select {
		case <-ctx.Done(): return false
		case <-wake: return ctx.Err() == nil
		}
	}
	if delay <= 0 { return ctx.Err() == nil }
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done(): return false
	case <-wake: return ctx.Err() == nil
	case <-timer.C: return ctx.Err() == nil
	}
}
