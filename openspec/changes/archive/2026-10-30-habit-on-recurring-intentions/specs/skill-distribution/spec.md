## ADDED Requirements

### Requirement: The skill says habit is the person's
The embedded skill SHALL state that a recurring intention the person no longer plans may be held as habit; that the harness may suggest habituation, for example after a run of fulfilled instances, and never sets it; that the harness may clear it when a habit has lapsed and the person wants the plan back; that a habit's unplaced instances are dormant and not to be resolved; and that habit needs a `intentions/0.2` workspace. The verbs table SHALL carry `--habit` and `--clear-habit`.

#### Scenario: Skill states the habit rule
- **WHEN** the embedded skill is read
- **THEN** it says the harness suggests habituation and the person sets it, and that clearing is the harness's to do when asked
