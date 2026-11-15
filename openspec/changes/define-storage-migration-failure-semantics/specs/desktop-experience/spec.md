# Spec Delta

## ADDED Requirements

### Requirement: Preference publication follows persistence
The desktop SHALL publish candidate preferences and emit settings:changed only after their persistence succeeds.

#### Scenario: Preference write fails
- **WHEN** saving a candidate theme or default-view preference fails
- **THEN** previous live values remain selected and no settings success event is emitted

#### Scenario: Preference write succeeds
- **WHEN** valid candidate preferences are saved successfully
- **THEN** the returned preferences, live values, persisted values, and settings event agree
