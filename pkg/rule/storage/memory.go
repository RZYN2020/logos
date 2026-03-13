package storage

import (
	"sync"

	"github.com/log-system/logos/pkg/rule"
)

// MemoryStorage is an in-memory rule storage for testing.
type MemoryStorage struct {
	mu      sync.RWMutex
	rules   map[string]*rule.Rule
	watches []chan<- *WatchEvent
}

// WatchEvent represents a watch event.
type WatchEvent struct {
	Type WatchEventType
	Rule *rule.Rule
}

// WatchEventType is the type of watch event.
type WatchEventType string

const (
	WatchEventPut    WatchEventType = "put"
	WatchEventDelete WatchEventType = "delete"
)

// NewMemoryStorage creates a new in-memory storage.
func NewMemoryStorage() *MemoryStorage {
	return &MemoryStorage{
		rules:   make(map[string]*rule.Rule),
		watches: make([]chan<- *WatchEvent, 0),
	}
}

// LoadRules loads all rules from storage.
func (s *MemoryStorage) LoadRules() ([]*rule.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*rule.Rule, 0, len(s.rules))
	for _, r := range s.rules {
		rules = append(rules, r)
	}
	return rules, nil
}

// GetRule gets a single rule by ID.
func (s *MemoryStorage) GetRule(ruleID string) (*rule.Rule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, ok := s.rules[ruleID]
	if !ok {
		return nil, ruleNotFoundError{ruleID: ruleID}
	}
	return rule, nil
}

// PutRule stores a rule.
func (s *MemoryStorage) PutRule(r *rule.Rule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules[r.ID] = r

	// Notify watchers
	for _, ch := range s.watches {
		select {
		case ch <- &WatchEvent{Type: WatchEventPut, Rule: r}:
		default:
			// Channel full, skip
		}
	}

	return nil
}

// DeleteRule deletes a rule.
func (s *MemoryStorage) DeleteRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.rules, ruleID)

	// Notify watchers
	for _, ch := range s.watches {
		select {
		case ch <- &WatchEvent{Type: WatchEventDelete, Rule: &rule.Rule{ID: ruleID}}:
		default:
			// Channel full, skip
		}
	}

	return nil
}

// Watch watches for rule changes.
func (s *MemoryStorage) Watch(ch chan<- *WatchEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.watches = append(s.watches, ch)
}

// Unwatch stops watching.
func (s *MemoryStorage) Unwatch(ch chan<- *WatchEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, c := range s.watches {
		if c == ch {
			s.watches = append(s.watches[:i], s.watches[i+1:]...)
			break
		}
	}
}

// SetRules sets multiple rules at once.
func (s *MemoryStorage) SetRules(rules []*rule.Rule) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.rules = make(map[string]*rule.Rule)
	for _, r := range rules {
		s.rules[r.ID] = r
	}
}

// ruleNotFoundError is returned when a rule is not found.
type ruleNotFoundError struct {
	ruleID string
}

func (e ruleNotFoundError) Error() string {
	return "rule not found: " + e.ruleID
}

// Ensure MemoryStorage implements RuleStorage interface
var _ rule.RuleStorage = (*MemoryStorage)(nil)
