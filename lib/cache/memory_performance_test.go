package cache

import (
	"strings"
	"testing"
)

func TestMemoryCacheReplacementCountsOnlyCurrentValue(t *testing.T) {
	c := NewMemoryCache(24)
	defer c.stopEviction()
	for i := 0; i < 100; i++ {
		if err := c.Set("same", strings.Repeat("x", 10), 60); err != nil {
			t.Fatal(err)
		}
	}
	if c.usedBytes != 16 {
		t.Fatalf("replacement used %d bytes, expected 16", c.usedBytes)
	}
	if err := c.Set("extra", "y", 60); err != nil {
		t.Fatal(err)
	}
	var result string
	if err := c.Get("same", &result); err != nil || result != strings.Repeat("x", 10) {
		t.Fatalf("replacement was evicted: %q %v", result, err)
	}
	if c.usedBytes > c.maxBytes {
		t.Fatalf("memory budget exceeded: %d > %d", c.usedBytes, c.maxBytes)
	}
}

func TestMemoryCacheEvictsAfterGrowingReplacement(t *testing.T) {
	c := NewMemoryCache(24)
	defer c.stopEviction()
	for _, key := range []string{"a", "b"} {
		if err := c.Set(key, "x", 60); err != nil { t.Fatal(err) }
	}
	if err := c.Set("a", strings.Repeat("z", 19), 60); err != nil { t.Fatal(err) }
	if _, found := c.data["b"]; found { t.Fatal("least recently used entry was not evicted") }
	var actual string
	if err := c.Get("a", &actual); err != nil || actual != strings.Repeat("z", 19) {
		t.Fatalf("replacement lost: %q %v", actual, err)
	}
}
