## ADDED Requirements

### Requirement: A placement's zone is outside every projection
Normalisation SHALL drop a placement start's time-zone suffix along with its offset, hashing the instant in UTC. Adding, changing or removing a zone SHALL leave every version unchanged and lapse no acknowledgement.

#### Scenario: Zone added
- **WHEN** a placement's `start` gains `[Australia/Melbourne]` with the same offset
- **THEN** the object's version is unchanged
