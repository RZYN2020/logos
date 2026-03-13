package rule

// RuleStorageWithWatch extends RuleStorage with watch capabilities.
type RuleStorageWithWatch interface {
	RuleStorage
	// Close closes the storage connection.
	Close() error
}

// RuleStorageWithMutations extends RuleStorage with mutation capabilities.
type RuleStorageWithMutations interface {
	RuleStorage
	// PutRule stores a rule.
	PutRule(rule *Rule) error
	// DeleteRule deletes a rule.
	DeleteRule(ruleID string) error
}

// RuleStorageFull is the full rule storage interface with all capabilities.
type RuleStorageFull interface {
	RuleStorageWithWatch
	RuleStorageWithMutations
}
