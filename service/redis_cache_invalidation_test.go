package service

import "testing"

func TestRedisCacheInvalidationMessages(t *testing.T) {
    readPolicy := func() uint64 {
        policyActivity.Lock()
        defer policyActivity.Unlock()
        return policyActivity.generation
    }
    readPush := func() uint64 {
        pushActivity.Lock()
        defer pushActivity.Unlock()
        return pushActivity.generation
    }
    readNotification := func() uint64 {
        administrativeNotificationActivity.Lock()
        defer administrativeNotificationActivity.Unlock()
        return administrativeNotificationActivity.generation
    }
    firstPolicy := readPolicy()
    firstPush := readPush()
    firstNotification := readNotification()
    applyRemoteCacheInvalidation("other-node:policy")
    if readPolicy() != firstPolicy+1 || readPush()!=firstPush || readNotification()!=firstNotification {
        t.Fatal("policy message changed the wrong cache")
    }
    applyRemoteCacheInvalidation("other-node:push")
    applyRemoteCacheInvalidation("other-node:notification")
    if readPush()!=firstPush+1 || readNotification()!=firstNotification+1 {
        t.Fatal("push or notification cache did not refresh")
    }
    applyRemoteCacheInvalidation(streamRedisNode+":policy")
    applyRemoteCacheInvalidation("other-node:unrecognized")
    applyRemoteCacheInvalidation("malformed")
    if readPolicy()!=firstPolicy+1 {
        t.Fatal("local or malformed cache invalidation must be ignored")
    }
}
