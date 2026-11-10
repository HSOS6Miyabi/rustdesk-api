package utils

import (
	"testing"
	"time"
)

func TestDisabledIPBanAvoidsGlobalMutex(t *testing.T) {
	limiter := NewLoginLimiter(SecurityPolicy{CaptchaThreshold: 3, BanThreshold: 0})
	defer close(limiter.cleanupStop)
	if limiter.BanEnabled() { t.Fatal("IP blocking was unexpectedly enabled") }
	limiter.mu.Lock()
	done := make(chan bool, 1)
	go func() { done <- limiter.IsIPBanned("198.51.100.42") }()
	select {
	case blocked := <-done:
		if blocked { t.Fatal("disabled IP blocking rejected a request") }
	case <-time.After(200 * time.Millisecond):
		limiter.mu.Unlock()
		t.Fatal("disabled IP blocking waited on the global mutex")
	}
	limiter.mu.Unlock()
}

func TestEnabledIPBanStillRejectsRepeatedFailures(t *testing.T) {
	limiter := NewLoginLimiter(SecurityPolicy{CaptchaThreshold: 3, BanThreshold: 2})
	defer close(limiter.cleanupStop)
	if !limiter.BanEnabled() { t.Fatal("IP blocking was unexpectedly disabled") }
	for i := 0; i < 2; i++ {
		limiter.RecordFailedAttempt("198.51.100.43")
	}
	if !limiter.IsIPBanned("198.51.100.43") {
		t.Fatal("enabled IP blocking did not reject repeated login failures")
	}
}
