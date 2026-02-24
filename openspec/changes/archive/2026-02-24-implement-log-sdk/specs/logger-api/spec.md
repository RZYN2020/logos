## ADDED Requirements

### Requirement: Logger SHALL support traditional printf-style API
The Logger SHALL provide traditional logging methods compatible with standard log package usage.

#### Scenario: Printf with format string
- **WHEN** user calls `log.Printf("Hello %s", "world")`
- **THEN** the message SHALL be formatted and logged with INFO level

#### Scenario: Println with arguments
- **WHEN** user calls `log.Println("message", "args")`
- **THEN** the arguments SHALL be concatenated and logged with INFO level

#### Scenario: Print without newline
- **WHEN** user calls `log.Print("message")`
- **THEN** the message SHALL be logged with INFO level without automatic newline

### Requirement: Logger SHALL support strongly-typed chain-style API
The Logger SHALL provide fluent interface methods for structured logging similar to Zap/ZeroLog.

#### Scenario: Chain-style Info logging
- **WHEN** user calls `log.Info("message").Str("key", "value").Send()`
- **THEN** the message SHALL be logged with structured fields

#### Scenario: Chain-style with multiple field types
- **WHEN** user chains Str, Int, Int64, Float64, Bool methods
- **THEN** all fields SHALL be included in the log entry

#### Scenario: Chain-style logging without Send
- **WHEN** user calls chain methods without calling Send()
- **THEN** the log SHALL NOT be emitted until Send() is called

### Requirement: Logger SHALL support structured fields
The Logger SHALL support key-value pair fields in both API styles.

#### Scenario: Traditional style with fields
- **WHEN** user calls `log.Info("msg", logger.F("key", "value"))`
- **THEN** the field SHALL be serialized with the log entry

#### Scenario: With fields inheritance
- **WHEN** user creates logger with `log.With(logger.F("k", "v"))`
- **THEN** all subsequent logs from child logger SHALL include the field

### Requirement: Logger SHALL support all standard log levels
The Logger SHALL provide methods for Debug, Info, Warn, Error, Fatal, and Panic levels.

#### Scenario: Debug level logging
- **WHEN** user calls `log.Debug("debug msg")`
- **THEN** the log SHALL have level field set to "DEBUG"

#### Scenario: Error level logging
- **WHEN** user calls `log.Error("error msg")`
- **THEN** the log SHALL have level field set to "ERROR"

#### Scenario: Fatal level behavior
- **WHEN** user calls `log.Fatal("fatal msg").Send()`
- **THEN** the log SHALL be emitted and application SHALL exit

### Requirement: Log entries SHALL include metadata
All log entries SHALL include timestamp, level, message, service, cluster, pod identifiers.

#### Scenario: Log entry with service metadata
- **WHEN** a log is created with Config{ServiceName: "my-service"}
- **THEN** the log entry SHALL include service field with value "my-service"

#### Scenario: Log entry with trace context
- **WHEN** trace_id and span_id are provided in context
- **THEN** the log entry SHALL include trace_id and span_id fields
