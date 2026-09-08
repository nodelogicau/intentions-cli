## ADDED Requirements

### Requirement: A workspace never writes availability to make a resolution succeed
An availability is an assertion about a particular's capacity, so a workspace SHALL NOT write one for a party in order that a resolution succeed. A party the workspace holds no availability for is untracked rather than unavailable, and resolution treats them as unconstrained; the answer to an intention that will not resolve is to report what stands in the way, never to record a fact nobody asserted.

#### Scenario: No manufactured supply
- **WHEN** an intention naming an external party will not resolve
- **THEN** what stands in the way is reported and no availability whose subject is that party is written

#### Scenario: Declaring capacity narrows it
- **WHEN** a party who was untracked declares one availability
- **THEN** they become tracked, and an intention naming them resolves only within that availability, which is stricter than before they declared it
