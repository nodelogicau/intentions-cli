## ADDED Requirements

### Requirement: Habit
A recurring intention MAY carry `habit: true`, meaning the person holds it as habit: its context cues it and it needs no plan. `intention add --habit` and `intention edit --habit` SHALL set it, and `intention edit --clear-habit` SHALL clear it. `habit` SHALL be admitted only on an intention carrying `cadence`; a write whose result carries `habit` without `cadence` SHALL be refused with exit code 2, and clearing `cadence` on a habit SHALL be refused naming `--clear-habit`.

A write that sets `habit` on an intention that did not carry it SHALL be refused with exit code 2, and nothing written:
- when its source carries a harness (rule `harness_habit`), whatever `--policy` it names, naming the act as the person's;
- in a workspace whose `format` is `intentions/0.1` (rule `habit_format`), naming `intentions migrate`.

A write that clears `habit`, or keeps an existing `habit` while changing something else, SHALL be accepted from any source; whether the person set `habit` is checked at write time only.

Habit SHALL change nothing about the intention's own standing: it stays unretired, keeps its `stability`, and remains subject to Every intention reaches a terminus. Habituation and reverting SHALL write nothing to existing instances. An instance is *dormant* when it is unplaced and unretired and its `instance-of` target carries `habit` in a `intentions/0.2` workspace. A placed instance of a habit is not dormant. In a `intentions/0.1` workspace a `habit` found on an object SHALL have no effect on generation, resolution, `unresolved` or consistency.

#### Scenario: The person habituates
- **WHEN** `intention edit int_R --habit` is run with an author and no harness on a recurring intention in a 0.2 workspace
- **THEN** the file carries `habit: true` after `cadence`, the version changes, the intention stays active, and no instance file changes

#### Scenario: A harness habituates
- **WHEN** `intention edit int_R --habit --harness claude` is run
- **THEN** the command exits with code 2 with rule `harness_habit`, naming the act as the person's, and the file is unchanged

#### Scenario: A harness habituates under a policy
- **WHEN** `intention add --habit --harness claude --policy int_P` is run, int_P being a firm policy of the subject
- **THEN** the command exits with code 2 with rule `harness_habit`

#### Scenario: A harness reverts
- **WHEN** `intention edit int_R --clear-habit --harness claude` is run on a habit
- **THEN** the write is accepted, its unplaced instances are no longer dormant, and the next `generate` creates instances for occurrences in the horizon that have none

#### Scenario: A harness edits a habit
- **WHEN** `intention edit int_R --description "after coffee" --harness claude` is run on a habit
- **THEN** the write is accepted and `habit` is unchanged

#### Scenario: Habit on a one-off
- **WHEN** `intention add --title X --duration PT1H --habit` is run without `--cadence`
- **THEN** the command exits with code 2 naming `habit` and `cadence`

#### Scenario: Habit in a 0.1 workspace
- **WHEN** the person runs `intention edit int_R --habit` in a workspace whose `format` is `intentions/0.1`
- **THEN** the command exits with code 2 with rule `habit_format`, naming `intentions migrate`

#### Scenario: A habit outlives its reason
- **WHEN** in a 0.2 workspace the person sets tentative the only firm terminus a habit reaches
- **THEN** `validate` reports the habit as `unserved` at error level
