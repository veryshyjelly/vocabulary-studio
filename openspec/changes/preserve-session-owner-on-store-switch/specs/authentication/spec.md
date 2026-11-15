# Spec Delta

## ADDED Requirements

### Requirement: Storage switches preserve active session ownership
After a successful backend switch, the application SHALL preserve active trainer state and route subsequent persistence, including persistence from existing trainer sessions, through the selected backend.

#### Scenario: Continue after switching to memory
- **WHEN** a file-backed active trainer switches to memory and later refreshes credentials
- **THEN** practice state is preserved and the old session file is not recreated

#### Scenario: Switch to the current mode
- **WHEN** the requested backend is already selected
- **THEN** the existing client and active practice state are retained
