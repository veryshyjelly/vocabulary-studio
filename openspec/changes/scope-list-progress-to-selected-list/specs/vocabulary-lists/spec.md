# Spec Delta

## ADDED Requirements

### Requirement: List enrichment ownership
The list view SHALL display progress only for its current list/account selection and ignore obsolete detail, progress, error, and loading completions.

#### Scenario: Responses finish out of order
- **WHEN** list A progress arrives after list B is selected and loaded
- **THEN** list B retains its own progress

#### Scenario: Select another list
- **WHEN** a different list is opened while previous progress is displayed
- **THEN** old progress clears until the current list's progress is available

#### Scenario: Leave or change account
- **WHEN** a pending request completes after navigation away or account change
- **THEN** it cannot restore obsolete view state
