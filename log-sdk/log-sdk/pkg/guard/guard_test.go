package guard

import (
	"testing"
	"time"
)

func TestTokenBucketGuard(t *testing.T) {
	g := NewTokenBucketGuard(2, 1, time.Second)
	if !g.Allow() {
		t.Fatal("should allow")
	}
	if !g.Allow() {
		t.Fatal("should allow")
	}
	if g.Allow() {
		t.Fatal("should not allow")
	}
}
