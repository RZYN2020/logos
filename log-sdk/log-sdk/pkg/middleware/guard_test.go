package middleware

import (
	"testing"
	"time"
	"github.com/log-system/log-sdk/pkg/guard"
)

func TestTokenBucketGuard(t *testing.T) {
	g := guard.NewTokenBucketGuard(2, 1, time.Second)
	if !g.Allow() {
		t.Fatal("should allow")
	}
}
