## Context

Wording only, following the specification's `terminus-as-practice`. Nothing here changes what any verb accepts, refuses or writes.

## Decisions

- **Messages change with the convention.** A message that says "who the person is" or "a self" teaches the old convention every time it's shown, so the five that describe a terminus are reworded. Rule codes don't change.
- **Fixtures stay.** Test titles such as "being someone who follows through" are inputs, and validation never reads wording. Rewriting forty fixtures would churn the diff for nothing. One spec scenario that uses such a title as an input stays too.
- **The installed skill copy is regenerated with `make skill`**, so the committed stamp doesn't churn.

## Risks / Trade-offs

- [A harness that cached the old skill keeps advising the noun form] → It still validates. The next `skill install` after release replaces it.
