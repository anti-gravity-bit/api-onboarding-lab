# Tasks

Owners: Grok (contract and review), Codex (Go and CI), AGY (docs and QA). Status is local until a public step is approved.

| ID | Status | Owner | Task |
| --- | --- | --- | --- |
| T0 | done | Grok | Discovery, plan, acceptance matrix, agent seats |
| T1 | done | Grok | Freeze routes, schemas, webhook rules, and failure semantics |
| T2 | done | Codex | Acceptance tests for every row in `ACCEPTANCE.md` |
| T3 | done | Codex | Server, in-memory store, and webhook verifier. Stdlib only |
| T4 | done | Codex | Go client, curl scripts, Dockerfile, and loopback Compose |
| T5 | done | Grok | Re-ran tests, race detector, and quickstart with Go 1.27.2 |
| T6 | pending | AGY | Docs site: Overview, Quickstart, Webhook security, API reference, Examples, Troubleshooting, Architecture, Testing, Limitations |
| T7 | pending | Grok | Audit the docs against `CONTRACT.md` and the test transcript |
| T8 | pending | Codex | `gofmt`, `go vet`, coverage, staticcheck, golangci-lint, govulncheck, OpenAPI lint |
| T9 | pending | Codex | GitHub Actions with pinned SHAs and minimal permissions. Do not enable the workflow on a remote until push is approved |
| T10 | done | Grok | MIT license added after explicit approval. Publication remains blocked |
| T11 | blocked | Grok | Create the GitHub repo, push, open the PR, and confirm remote CI |
| T12 | blocked | Grok | Pages only after approval. Record the real URL and commit SHA |
| T13 | pending | Grok | `CASE_STUDY.md` and the polished README, using executed commands only |

T2 is the first implementation task. Tests come before handlers. Codex commits locally and does not push.
