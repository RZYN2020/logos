## ADDED Requirements

### Requirement: Async producer SHALL batch log messages
The Producer SHALL batch multiple log messages before sending to Kafka.

#### Scenario: Batch size triggers send
- **WHEN** batch size reaches configured threshold (e.g., 100)
- **THEN** the batch SHALL be sent to Kafka

#### Scenario: Timeout triggers partial batch
- **WHEN** timeout expires with fewer than batch size messages
- **THEN** the partial batch SHALL be sent to Kafka

### Requirement: Async producer SHALL handle backpressure
The Producer SHALL implement backpressure when buffer is full.

#### Scenario: Buffer full drops message
- **WHEN** buffer is full and new message arrives
- **THEN** the message SHALL be dropped and error returned

#### Scenario: Non-blocking send
- **WHEN** buffer is full
- **THEN** Send() SHALL return immediately without blocking caller

### Requirement: Async producer SHALL support graceful shutdown
The Producer SHALL flush remaining messages on close.

#### Scenario: Close flushes pending messages
- **WHEN** Close() is called with pending messages
- **THEN** all pending messages SHALL be sent before returning

#### Scenario: Close timeout handling
- **WHEN** flush takes longer than timeout
- **THEN** remaining messages MAY be dropped

### Requirement: Async producer SHALL handle Kafka failures
The Producer SHALL handle Kafka connection failures gracefully.

#### Scenario: Kafka temporarily unavailable
- **WHEN** Kafka is unreachable
- **THEN** messages SHALL accumulate in buffer up to max size

#### Scenario: Fallback to console when configured
- **WHEN** Kafka send fails and FallbackToConsole is true
- **THEN** messages SHALL be printed to console
