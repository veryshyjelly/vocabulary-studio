# Spec Delta

## ADDED Requirements

### Requirement: Explicit dashboard availability
The dashboard SHALL distinguish unavailable sections from successful empty results, retain successful data, expose safe failure status, and permit retry.

#### Scenario: One section fails
- **WHEN** one component request fails while others succeed
- **THEN** successful data remains and the failed section is unavailable rather than empty

#### Scenario: Account verification unavailable
- **WHEN** account verification fails because of transport or upstream availability
- **THEN** the dashboard reports unavailable verification rather than proof of sign-out

#### Scenario: Retry recovers
- **WHEN** dashboard retry succeeds after a transient component failure
- **THEN** successful sections replace their unavailable status
