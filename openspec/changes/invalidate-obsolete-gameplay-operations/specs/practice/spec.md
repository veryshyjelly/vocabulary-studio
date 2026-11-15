# Spec Delta

## ADDED Requirements

### Requirement: Gameplay result ownership
The desktop SHALL apply gameplay results only while their initiating account and mode session remain current, and SHALL clear all local gameplay modes on logout even if persistent cleanup fails.

#### Scenario: Old quiz finishes late
- **WHEN** quiz A submission completes after quiz B becomes current
- **THEN** quiz B state is not replaced

#### Scenario: Start finishes after logout
- **WHEN** an old account's gameplay start response arrives after logout
- **THEN** the result cannot reactivate gameplay

#### Scenario: Persistent cleanup fails
- **WHEN** logout clears local authentication but clearing its store fails
- **THEN** trainer, Bee and quiz become inactive and cleanup failure is reported separately
