// Package actions provides helper functions for building custom actions.
//
// This package is deprecated. All built-in actions are now defined
// directly in the parent rule package to avoid circular imports.
//
// Use the helper functions in this package to build custom actions:
//   - GetStringConfig
//   - GetIntConfig
//   - GetFloat64Config
//   - GetStringSliceConfig
package actions

import "github.com/log-system/logos/pkg/rule"

// GetStringConfig safely gets a string config value.
func GetStringConfig(config map[string]interface{}, key string, defaultValue string) string {
	return rule.GetStringConfig(config, key, defaultValue)
}

// GetIntConfig safely gets an int config value.
func GetIntConfig(config map[string]interface{}, key string, defaultValue int) int {
	return rule.GetIntConfig(config, key, defaultValue)
}

// GetFloat64Config safely gets a float64 config value.
func GetFloat64Config(config map[string]interface{}, key string, defaultValue float64) float64 {
	return rule.GetFloat64Config(config, key, defaultValue)
}

// GetStringSliceConfig safely gets a string slice config value.
func GetStringSliceConfig(config map[string]interface{}, key string, defaultValue []string) []string {
	return rule.GetStringSliceConfig(config, key, defaultValue)
}
