## 1. Wording

- [x] 1.1 Embedded skill: first question, titling rule, split example, bare adoption, "What do you mean to keep doing?"
- [x] 1.2 `init` stub Termini section and its test assertion
- [x] 1.3 Messages: `terminus_fulfilled` (policy.go, rules.go), `bare_adoption` and `serves_target` (desire.go, validate.go), unserved fix (grounding.go)
- [x] 1.4 README split and bare-adoption examples; CHANGELOG entry under Changed

## 2. Verify

- [x] 2.1 `make skill` regenerates the installed copy
- [x] 2.2 `go test ./...` and `openspec validate --all --strict` pass
