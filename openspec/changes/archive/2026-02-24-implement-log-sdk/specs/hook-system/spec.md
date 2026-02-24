## ADDED Requirements

### Requirement: Hook system SHALL support filtering by log level
The Logger SHALL provide LevelHook to filter logs based on minimum level.

#### Scenario: LevelHook filters lower level logs
- **WHEN** LevelHook(LevelInfo) is added and DEBUG log is emitted
- **THEN** the DEBUG log SHALL be filtered out

#### Scenario: LevelHook allows higher level logs
- **WHEN** LevelHook(LevelInfo) is added and ERROR log is emitted
- **THEN** the ERROR log SHALL be recorded

### Requirement: Hook system SHALL support filtering by line number
The Logger SHALL provide LineHook to filter logs based on source line range.

#### Scenario: LineHook filters logs outside range
- **WHEN** LineHook(100, 200) is added and log from line 50 is emitted
- **THEN** the log SHALL be filtered out

#### Scenario: LineHook allows logs within range
- **WHEN** LineHook(100, 200) is added and log from line 150 is emitted
- **THEN** the log SHALL be recorded

### Requirement: Hook interface SHALL support custom filters
The Hook interface SHALL allow users to implement custom filtering logic.

#### Scenario: Custom Hook implementation
- **WHEN** user implements Hook interface with custom OnLog logic
- **THEN** the custom filter SHALL be applied to all logs

#### Scenario: Multiple Hooks chain
- **WHEN** multiple Hooks are added via AddHook
- **THEN** all Hooks SHALL be evaluated in order, returning false from any Hook filters the log

### Requirement: Logger SHALL support adding and chaining Hooks
The Logger SHALL provide AddHook method for registering filters.

#### Scenario: AddHook returns new Logger instance
- **WHEN** user calls `log.AddHook(hook)`
- **THEN** a new Logger instance SHALL be returned with the hook attached

#### Scenario: Original Logger remains unchanged
- **WHEN** AddHook is called on a Logger
- **THEN** the original Logger SHALL NOT have the hook attached
