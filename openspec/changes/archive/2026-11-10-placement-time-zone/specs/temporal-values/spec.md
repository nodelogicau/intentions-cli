## MODIFIED Requirements

### Requirement: Placement values
A PLACEMENT read from a file SHALL have `start` as an RFC 3339 datetime with offset, optionally followed by one RFC 9557 time-zone suffix holding a bare IANA zone name (`2026-09-15T10:00:00+10:00[Australia/Melbourne]`), or a calendar day, a `duration`, and optionally one `location` URI. When `start` is a calendar day the duration SHALL be a whole number of days and `start` SHALL carry no suffix. An offset-form suffix, any other suffix tag, more than one suffix, and the critical flag `!` SHALL be errors. A suffix after `Z` SHALL be admitted. The offset SHALL fix the instant; nothing SHALL move a placement to follow its zone. `start` SHALL be written back exactly as read. No other datetime field SHALL admit a suffix, and an error SHALL name the field.

#### Scenario: All-day with fractional duration rejected
- **WHEN** a file carries `placement: {start: 2026-09-21, duration: PT4H}`
- **THEN** `validate` reports an error

#### Scenario: Datetime placement accepted
- **WHEN** a file carries `placement: {start: 2026-09-15T10:00:00+10:00, duration: PT1H}`
- **THEN** `validate` reports nothing for it

#### Scenario: Zoned placement accepted
- **WHEN** a file carries `placement: {start: 2026-09-15T10:00:00+10:00[Australia/Melbourne], duration: PT1H}`
- **THEN** `validate` reports nothing for it and a rewrite keeps the start byte for byte, unquoted

#### Scenario: Suffix forms refused
- **WHEN** a placement's `start` ends in `[+10:00]`, `[u-ca=hebrew]`, `[!Australia/Melbourne]`, or two suffixes
- **THEN** `validate` reports an error naming the suffix

#### Scenario: Suffix on a calendar day
- **WHEN** a file carries `placement: {start: "2026-09-21[Australia/Melbourne]", duration: P1D}`
- **THEN** `validate` reports an error

#### Scenario: Suffix on a timestamp
- **WHEN** an object's `timestamp` carries `[Australia/Melbourne]`
- **THEN** `validate` reports an error naming `timestamp` and saying a suffix is admitted only on a placement's `start`

#### Scenario: UTC with a zone
- **WHEN** a placement's `start` is `2026-09-15T00:00:00Z[Australia/Melbourne]`
- **THEN** `validate` reports nothing for it
