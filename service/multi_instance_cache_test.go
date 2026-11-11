package service

import (
    "testing"
    "time"
)

func TestOptionalFeatureCacheRefreshDependsOnMultiInstanceMode(t *testing.T){
    t.Setenv("RUSTDESK_API_MULTI_INSTANCE_CACHE","")
    if got:=featureCacheRefreshInterval();got!=30*time.Second {
        t.Fatalf("single instance interval: %v",got)
    }
    t.Setenv("RUSTDESK_API_MULTI_INSTANCE_CACHE","1")
    if got:=featureCacheRefreshInterval();got!=5*time.Second {
        t.Fatalf("multi instance interval: %v",got)
    }
}
