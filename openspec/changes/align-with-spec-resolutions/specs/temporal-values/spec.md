## MODIFIED Requirements

### Requirement: EDTF calendar subset
A calendar anchor SHALL be one of: a year `YYYY`; a month `YYYY-MM`; an ISO week `YYYY-Www`; a day `YYYY-MM-DD`; a neutral season `YYYY-21` to `YYYY-24`; a hemisphere-specific season `YYYY-25` to `YYYY-28` (Northern spring, summer, autumn, winter) or `YYYY-29` to `YYYY-32` (Southern); a quarter `YYYY-33` to `YYYY-36`; a bounded interval `A/B` between any two of these with A not after B; an open interval `../B` or `A/..`. Qualifiers `?`, `~`, `%` SHALL be refused. Normalisation SHALL upper-case `W`, zero-pad fields, and collapse `A/A` to `A`.

#### Scenario: Week accepted
- **WHEN** `--calendar 2026-W36` is given
- **THEN** the file carries `calendar: 2026-W36`

#### Scenario: Quarter accepted
- **WHEN** `--calendar 2026-35` is given
- **THEN** the file carries `calendar: 2026-35`

#### Scenario: Explicit Southern season accepted
- **WHEN** `--calendar 2026-29` is given
- **THEN** the file carries `calendar: 2026-29`

#### Scenario: Open interval accepted
- **WHEN** `--calendar ../2026-09` is given
- **THEN** the file carries `calendar: ../2026-09`

#### Scenario: Qualifier refused
- **WHEN** `--calendar 2026-09~` is given
- **THEN** the write is refused and the message names the qualifier

#### Scenario: Reversed interval refused
- **WHEN** `--calendar 2026-W38/2026-W36` is given
- **THEN** the write is refused

#### Scenario: Unknown code refused
- **WHEN** `--calendar 2026-40` is given
- **THEN** the write is refused naming the admitted season and quarter codes
