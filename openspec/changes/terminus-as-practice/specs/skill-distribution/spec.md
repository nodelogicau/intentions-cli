## MODIFIED Requirements

### Requirement: The skill walks a workspace up to its termini by asking
The embedded skill SHALL state that every intention that is not a terminus reaches a firm terminus of the person's own, that a harness's first question in a workspace with no firm terminus is what the person means to keep doing, for its own sake, and that an `unserved` finding from `validate` or from a write is answered one intention at a time by asking what it is for and linking it, never by inventing a terminus. It SHALL state that a terminus the harness drafts is tentative and grounds nothing, that only the person firms a terminus with an author and no harness, and SHALL give the person that one command rather than run it. It SHALL state that a terminus is titled as a practice the person chooses to keep doing, not as who they are and not as a task, and that the test is whether the title could ever be done. The `init` conventions stub's Termini section SHALL say the same in a sentence.

#### Scenario: Skill states the walk-up
- **WHEN** the embedded skill is read
- **THEN** it contains a section on reaching a terminus that names the `unserved` code, says to ask rather than invent, and says a harness does not firm a terminus

#### Scenario: Conventions stub names firming
- **WHEN** `intentions init` writes `intentions.md`
- **THEN** its Termini section says a terminus grounds nothing until the person firms it

#### Scenario: Skill states the titling test
- **WHEN** the embedded skill is read
- **THEN** it says a terminus is titled as a practice, gives *following through* rather than *being someone who follows through*, and says the test is whether the title could ever be done

#### Scenario: Conventions stub states the practice convention
- **WHEN** `intentions init` writes `intentions.md`
- **THEN** its Termini section says a terminus is titled as a practice the person chooses to keep doing, and that the test is whether the title could ever be done

### Requirement: The skill says how a terminus changes
The embedded skill SHALL state that when a reason has changed the harness prefers superseding the terminus with a new one over rewording it in place, because the change stays visible and its dependents stay grounded; that a reason is withdrawn by setting its terminus tentative; that a terminus admits `abandoned` and `superseded` only; that retiring a terminus anything still rests on is refused; and that a harness retires only a tentative terminus it drafted itself. It SHALL state that a practice found to be several is recorded by superseding the terminus with each of them, `--superseded-by` repeated; that what rested on it stays grounded and `validate` reports each dependent as `split_terminus`; and that the harness asks the person which successor each dependent is for and never re-points one on its own. The verbs table's retire row SHALL note the terminus kinds and that `--superseded-by` repeats on a terminus.

#### Scenario: Skill states how a terminus changes
- **WHEN** the embedded skill is read
- **THEN** it says to supersede a changed reason, to set a terminus tentative to withdraw it, and never to retire one out from under what rests on it

#### Scenario: Skill states the split case
- **WHEN** the embedded skill is read
- **THEN** it says to record a split with a repeated `--superseded-by`, and to ask the person which successor each dependent is for instead of re-pointing
