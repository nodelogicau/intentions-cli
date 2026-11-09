## MODIFIED Requirements

### Requirement: The skill gives the calendar hold workflow
The embedded skill SHALL tell the harness:
- a busy hold for a placed intention with no `parties` is recorded with `intention edit --external <system>:<uid>`; for an intention with `parties` the hold belongs on its commitment;
- the hold carries only the placement's start and duration (in all-day form for an all-day placement), `placement.location` where it is a URI, the title, `TRANSP:OPAQUE` / `freeBusyStatus: busy`, and `STATUS:TENTATIVE` or `CONFIRMED` from stability;
- before importing an event, look its UID up with `intention list --external <uid>`, and import nothing that matches;
- after a replacing selection, move the same event;
- after an `abandoned` or `superseded` retirement, release the hold without writing to the workspace; after `fulfilled`, leave it;
- a `hold_imported` warning means a commitment carries the intention's hold link: cancel the commitment when it is the hold imported before it was recognised, or clear the intention's link when the event is someone else's, and ask the person when it is unclear which.

#### Scenario: Skill states the hold workflow
- **WHEN** `intentions skill show` is run
- **THEN** the output names `--external`, the hold's fixed shape, the `list --external` lookup before import, what to do with the hold on each retirement kind, and both fixes for `hold_imported`
