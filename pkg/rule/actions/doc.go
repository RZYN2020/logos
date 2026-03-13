// Package actions provides built-in actions for the unified rule engine.
//
// Flow Control Actions:
//   - keep: Keep the log entry and terminate rule evaluation
//   - drop: Drop the log entry and terminate rule evaluation
//   - sample: Sample logs at a configurable rate
//
// Transformation Actions:
//   - mask: Mask sensitive data using regex or full field masking
//   - truncate: Truncate field values to a maximum length
//   - extract: Extract substrings using regex capture groups
//   - rename: Rename fields
//   - remove: Remove one or more fields
//   - set: Set a field to a specific value
//
// Metadata Actions:
//   - mark: Add metadata marks to log entries
package actions
