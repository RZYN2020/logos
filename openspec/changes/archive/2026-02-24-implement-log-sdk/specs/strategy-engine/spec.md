## ADDED Requirements

### Requirement: Strategy engine SHALL evaluate rules for log filtering
The Strategy Engine SHALL evaluate conditions against log entries and apply actions.

#### Scenario: Rule matches level condition
- **WHEN** a rule with condition.level="ERROR" is configured and ERROR log is emitted
- **THEN** the rule action SHALL be applied

#### Scenario: Rule matches service condition
- **WHEN** a rule with condition.service="api" is configured and matching log is emitted
- **THEN** the rule action SHALL be applied

### Requirement: Strategy engine SHALL support sampling
The Strategy Engine SHALL support probabilistic sampling of log entries.

#### Scenario: Sampling rate filters logs
- **WHEN** action.sampling=0.5 is configured
- **THEN** approximately 50% of matching logs SHALL be recorded

#### Scenario: Full sampling records all logs
- **WHEN** action.sampling=1.0 is configured
- **THEN** all matching logs SHALL be recorded

### Requirement: Strategy engine SHALL support ETCD configuration
The Strategy Engine SHALL load and watch configuration from ETCD.

#### Scenario: Load initial configuration
- **WHEN** Engine is created with ETCD endpoints
- **THEN** initial strategies SHALL be loaded from ETCD

#### Scenario: Hot reload on configuration change
- **WHEN** ETCD configuration is updated
- **THEN** the new configuration SHALL be applied without restart

### Requirement: Strategy engine SHALL handle ETCD unavailability
The Strategy Engine SHALL gracefully handle ETCD connection failures.

#### Scenario: ETCD unavailable on startup
- **WHEN** ETCD is unreachable during Engine creation
- **THEN** Engine SHALL use default allow-all strategy and log warning

#### Scenario: ETCD disconnection during watch
- **WHEN** ETCD watch connection is lost
- **THEN** Engine SHALL continue with last known configuration
