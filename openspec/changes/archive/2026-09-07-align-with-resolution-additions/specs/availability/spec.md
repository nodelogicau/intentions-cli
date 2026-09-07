## MODIFIED Requirements

### Requirement: Scope
`scope` SHALL be `personal`, `organisation`, or `public`, and SHALL default to `personal`. `availability edit --scope` SHALL only widen: `personal` to `organisation` or `public`, `organisation` to `public`. Scope SHALL govern which resolutions may use the availability as supply: a `personal` availability is visible only when its `subject` is the intention's `subject` or one of its `parties`, while `organisation` and `public` availability is visible to a resolver whose `resolver.scope` is at or narrower than the availability's scope.

#### Scenario: Widen accepted
- **WHEN** `availability edit avl_A --scope organisation` is run on a `personal` availability
- **THEN** the file carries `scope: organisation`

#### Scenario: Narrowing refused
- **WHEN** `availability edit avl_A --scope personal` is run on an `organisation` availability
- **THEN** the command exits with code 2

#### Scenario: Personal stays personal
- **WHEN** an organisation workspace holds Rob's `personal` availability and Ada resolves an intention that does not involve Rob
- **THEN** it is not visible as supply
