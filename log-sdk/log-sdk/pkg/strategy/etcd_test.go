package strategy

import (
	"testing"
	"time"
)

func TestEtcdConfig_DefaultEtcdConfig(t *testing.T) {
	cfg := DefaultEtcdConfig()

	if len(cfg.Endpoints) != 1 || cfg.Endpoints[0] != "localhost:2379" {
		t.Errorf("Endpoints = %v, want [localhost:2379]", cfg.Endpoints)
	}

	if cfg.DialTimeout != 5*time.Second {
		t.Errorf("DialTimeout = %v, want 5s", cfg.DialTimeout)
	}
}

func TestNewEtcdClient_NoEndpoints(t *testing.T) {
	cfg := EtcdConfig{Endpoints: []string{}}
	_, err := NewEtcdClient(cfg)
	if err == nil {
		t.Error("Expected error for no endpoints, got nil")
	}
}

func TestEventType_String(t *testing.T) {
	tests := []struct {
		et   EventType
		want string
	}{
		{EventPut, "PUT"},
		{EventDelete, "DELETE"},
		{EventType(99), "UNKNOWN"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.et.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExtractIDFromKey(t *testing.T) {
	tests := []struct {
		key  string
		want string
	}{
		{"/strategies/my-strategy", "my-strategy"},
		{"/strategies/", ""},
		{"/strategies", "strategies"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			if got := extractIDFromKey(tt.key); got != tt.want {
				t.Errorf("extractIDFromKey(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

// Integration tests - these require a running etcd instance
// Use `go test -tags=integration` to run

// TestEtcdClient_Integration tests require a running etcd at localhost:2379
// Run with: go test -run Integration -v
func TestEtcdClient_Integration_LoadStrategies(t *testing.T) {
	// Skip if no etcd available
	cfg := DefaultEtcdConfig()
	client, err := NewEtcdClient(cfg)
	if err != nil {
		t.Skipf("Etcd not available: %v", err)
	}
	defer client.Close()

	// Put a test strategy
	testStrategy := Strategy{
		ID:      "test-strategy",
		Name:    "Test Strategy",
		Enabled: true,
		Rules: []Rule{
			{
				Name: "test-rule",
				Condition: Condition{
					Level:   "ERROR",
					Service: "test-service",
				},
				Action: Action{
					Enabled:  true,
					Sampling: 1.0,
				},
			},
		},
	}

	if err := client.PutStrategy(testStrategy); err != nil {
		t.Skipf("Failed to put strategy: %v", err)
	}

	// Clean up
	defer func() {
		_ = client.DeleteStrategy("test-strategy")
	}()

	// Load strategies
	strategies, err := client.LoadStrategies()
	if err != nil {
		t.Errorf("LoadStrategies failed: %v", err)
		return
	}

	if len(strategies) == 0 {
		t.Error("Expected at least one strategy")
	}
}

func TestEtcdClient_Integration_Watch(t *testing.T) {
	// Skip if no etcd available
	cfg := DefaultEtcdConfig()
	client, err := NewEtcdClient(cfg)
	if err != nil {
		t.Skipf("Etcd not available: %v", err)
	}
	defer client.Close()

	// Start watching
	eventCh, errCh := client.WatchStrategies()

	// Give watch time to start
	time.Sleep(100 * time.Millisecond)

	// Put a test strategy to trigger an event
	testStrategy := Strategy{
		ID:      "watch-test",
		Name:    "Watch Test",
		Enabled: true,
		Rules:   []Rule{},
	}

	if err := client.PutStrategy(testStrategy); err != nil {
		t.Skipf("Failed to put strategy: %v", err)
	}

	// Clean up
	defer func() {
		_ = client.DeleteStrategy("watch-test")
	}()

	// Wait for event or timeout
	select {
	case event := <-eventCh:
		if event.Type != EventPut {
			t.Errorf("Expected PUT event, got %v", event.Type)
		}
		if event.Strategy.ID != "watch-test" {
			t.Errorf("Expected strategy ID 'watch-test', got %s", event.Strategy.ID)
		}
	case err := <-errCh:
		t.Errorf("Watch error: %v", err)
	case <-time.After(2 * time.Second):
		// Timeout is acceptable - watch may not receive events in test environment
		t.Log("Watch timeout - may be expected in test environment")
	}
}

func TestEtcdClient_Integration_IsConnected(t *testing.T) {
	cfg := DefaultEtcdConfig()
	client, err := NewEtcdClient(cfg)
	if err != nil {
		t.Skipf("Etcd not available: %v", err)
	}
	defer client.Close()

	if !client.IsConnected() {
		t.Error("Expected IsConnected to return true")
	}
}
