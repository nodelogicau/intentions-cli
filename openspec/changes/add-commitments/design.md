## Context

The COMMITMENT object, its party statuses and its single retirement kind are already modelled and validated; `select` writes commitments and the resolver already treats a retired one as free time. What is missing is the acts.

## Decisions

### D1. Status is recorded, never decided
The specification is unusually firm here: a party's status is a deontic fact about their will, requiring sincerity, and software SHALL never set it. There is no policy analogous to `auto_firm`, and none is added. The guard is the same one `acknowledge` uses: the act needs an author, the verb changes exactly one party entry, and the tool description says it is only ever on the person's word. A harness may relay the person's answer, as it relays every other act, and the source records that it did.

### D2. Which party
`--party` defaults to `defaults.subject`, the workspace owner, which is the only entry a local act may speak for. A workspace with no default subject must name the party. The party must already be on the commitment; the verbs never add one. Answering for another party belongs to an imported iTIP reply, which this version does not implement, so the refusal says so.

### D3. Cancellation clears the placement
`cancel` appends the terminal `retired: {kind: cancelled}` record and, when the commitment names an intention that is still placed and unretired, clears that intention's placement in the same act. The intention keeps its window, its duration and its stability, and reappears in `unresolved`. The RESOLUTION record is left untouched as history, exactly as `select --replace` leaves the record it supersedes. Cancelling does not retire the intention.

### D4. Idempotence and refusals
Setting a status the party already holds is refused rather than written, because the projection would not change and the file would gain a new source and timestamp saying nothing. A retired commitment refuses every act. Cancelling a cancelled commitment is refused by the existing retirement rule.

### D5. No create, no edit
The two origins are a resolution and an import. There is no verb to write a commitment by hand, and none to edit one: `transparent` arrives only by import, and the fields a person can change are the party statuses and the retirement.

## Risks / Trade-offs

- **A declined commitment still occupies time.** The resolver counts every unretired, opaque commitment as occupancy, with `accepted` raising the reconsideration cost. Nothing in the specification says what a `declined` party means for the subject's own time, or whether the intention behind it should be freed. This change does not guess: `decline` records the answer and leaves occupancy alone, and the question goes upstream as SPEC-FEEDBACK item 24.
- Cancelling a commitment whose intention another commitment also fulfils would clear a placement that the other still rests on. The specification does not contemplate two commitments for one intention, and `select` never writes them; the verb clears the placement only when the intention is unretired and placed, and `check` reports anything left inconsistent.
