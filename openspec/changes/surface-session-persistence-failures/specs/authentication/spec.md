# Spec Delta

## ADDED Requirements

### Requirement: Observable persistence failure and recovery
Successful web authentication whose selected persistent store fails SHALL expose safe persistence-failure status separately from invalid credentials, retain authenticated memory state, and support retry without silently changing backends.

#### Scenario: Login cannot save
- **WHEN** web authentication succeeds but the selected file or keyring save fails
- **THEN** the caller can identify unsaved authentication without receiving credentials in diagnostic data

#### Scenario: Retry succeeds
- **WHEN** a user retries saving and the selected backend succeeds
- **THEN** the persistence warning clears without changing backend

#### Scenario: Refreshed token cannot be persisted
- **WHEN** token acquisition succeeds but saving the refreshed session snapshot fails
- **THEN** the caller can inspect the persistence failure without losing valid in-memory authentication or receiving credentials in diagnostic data

#### Scenario: CLI login cannot save
- **WHEN** explicit CLI login authenticates but cannot persist its requested file
- **THEN** stderr reports persistence failure, exit status is nonzero, and JSON stdout remains valid
