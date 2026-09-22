## ADDED Requirements

### Requirement: The skill walks a workspace up to its termini by asking
The embedded skill SHALL state that every intention that is not a terminus reaches a firm terminus of the person's own, that a harness's first question in a workspace with no firm terminus is who the person is trying to be, and that an `unserved` finding from `validate` or from a write is answered one intention at a time by asking what it is for and linking it, never by inventing a terminus. It SHALL state that a terminus the harness drafts is tentative and grounds nothing, that only the person firms a terminus with an author and no harness, and SHALL give the person that one command rather than run it. The `init` conventions stub's Termini section SHALL say the same in a sentence.

#### Scenario: Skill states the walk-up
- **WHEN** the embedded skill is read
- **THEN** it contains a section on reaching a terminus that names the `unserved` code, says to ask rather than invent, and says a harness does not firm a terminus

#### Scenario: Conventions stub names firming
- **WHEN** `intentions init` writes `intentions.md`
- **THEN** its Termini section says a terminus grounds nothing until the person firms it
