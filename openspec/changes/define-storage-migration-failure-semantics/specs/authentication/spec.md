# Spec Delta

## ADDED Requirements

### Requirement: Consistent storage migration outcomes
Before migration commits, failures SHALL preserve the previous selection. After commit, old-backend cleanup failure SHALL retain the new selection and report unresolved cleanup explicitly.

#### Scenario: Preference commit fails
- **WHEN** destination staging succeeds but saving the new selection fails
- **THEN** the old backend remains selected and failed staged-data cleanup is reported

#### Scenario: Old store cleanup fails
- **WHEN** the new selection commits but old-backend cleanup fails
- **THEN** the new backend stays selected and cleanup can be retried with visible feedback

#### Scenario: Switches overlap
- **WHEN** two storage switches execute concurrently
- **THEN** commits are serialized and live selection agrees with durable preferences
