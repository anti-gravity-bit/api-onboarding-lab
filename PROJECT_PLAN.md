# API Onboarding Lab — project plan

Independent portfolio project for a senior Go and Python backend engineer based in India. It demonstrates tested API onboarding and webhook integration reliability. It is personal work. It uses public and synthetic data only. The MVP is sized for two focused days.

The service is a local demo. It is not a production deployment, and this plan does not claim production readiness, customers, benchmarks, or certifications.

## Phase 0 discovery (2026-10-19)

| Check | Result |
| --- | --- |
| Project directory | `/home/abir-antigravity/workspace/api-onboarding-lab` |
| Book tree | Empty `books/golang-intro/api-onboarding-lab` removed. Book files were not edited. |
| Go on PATH | `go1.26.1` at `/usr/local/go/bin/go`. `GOTOOLCHAIN=auto`. |
| Current stable Go | `go1.27.2` is the newest stable on <https://go.dev/dl/?mode=json>, fetched 2026-10-19. `go1.26.9` is also listed as stable. |
| Pin | Module will set `go 1.27.2` and `toolchain go1.27.2`. The first `go` command may download that toolchain. Download success is not claimed until a later log shows it. |
| Git | 2.43.0. Identity visible from the book repo config: Abir Anhad, `abir@anhad.io`. |
| GitHub CLI | `gh` is not installed. |
| GitHub username | Personal OSS remotes use `git@github.com:anti-gravity-bit/...` for envlock, goholic, halide, and sambodhi. Treat `anti-gravity-bit` as the observed account, and confirm it before creating a repository. |
| Target repo | `git ls-remote git@github.com:anti-gravity-bit/api-onboarding-lab.git` returned `Repository not found`. SSH reached GitHub. The repo is absent or not visible to this key. It was not created. |
| Docker | 29.9.1. Compose v5.6.0. |
| Codex | 0.162.1, ChatGPT login, model `gpt-5.6-sol`. `codex exec` supports `-C`, `-s workspace-write`, `--json`, `-o`, and `--skip-git-repo-check`. |
| AGY | 1.4.2. Models include `gemini-3.8-flash-high`. Claude model names are listed and must not be used. |
| Grok | 1.0.62. |
| staticcheck | 2026.1 (v0.7.0), on PATH. |
| golangci-lint | 2.13.2, built with go1.26.1, on PATH. |
| govulncheck | Not installed. |
| Secret scanners | `gitleaks` and `trufflehog` are not installed. |
| OpenAPI CLI | `spectral` and `redocly` are not installed. |
| Three-agent bus | None. Seats run through the local CLIs in `AGENTS.md`. |

## Decisions

- Standard library only. HTTP stack is `net/http`.
- Persistence is one process-local memory store. It is not durable, not shared across processes, and not a production database.
- Webhook duplicates return success with `duplicate: true`, so a retrying sender stops. They do not return 409.
- Dedupe key is the event id. A reused id with a different body is still a duplicate. The first accepted event wins. This limitation stays in the docs.
- The webhook receiver does not look up the order. Signature, freshness, schema, and dedupe are the feature.
- Default listen address on the host is `127.0.0.1:8080`. The container sets `ADDR=:8080` and does not publish a host port by itself. Compose, if added, publishes only on `127.0.0.1`.
- Example credentials in `.env.example` are public fixtures. They are long enough to pass startup checks and are not secret.
- License is undecided. Until a `LICENSE` file exists, no license is granted.
- Module path, pending account confirmation: `github.com/anti-gravity-bit/api-onboarding-lab`.

## Recommended license

MIT. It is short, familiar to client reviewers, and matches a small demo they may copy. Apache-2.0 is the alternative if a patent grant is wanted. Neither file will be added until you choose.

## Work sequence

1. Freeze the contract. Done in `CONTRACT.md` and `api/openapi.yaml`.
2. Codex writes failing tests, then the server, examples, Dockerfile, and Compose. Small local commits. No push.
3. Grok re-runs the tests and the quickstart from a clean checkout and rejects any claim the transcript does not support.
4. AGY writes the docs site against that behavior and tests the terminal snippets.
5. Quality commands run locally: `gofmt`, `go vet`, `go test -race`, coverage, staticcheck, golangci-lint, govulncheck once installed, OpenAPI check, and a manual secret scan.
6. CI workflow files are added with pinned action SHAs looked up at that time, minimal permissions, and no untrusted pull-request token.
7. Stop for approval before `git push`, repository creation, Pages, or any public API deploy.

## Blockers

- License approval.
- Confirmation of the GitHub account and permission to create the public repository.
- `gh` is missing, so repository creation from the CLI needs either an installed `gh` auth flow, the GitHub website, or another approved API. No personal access token will be requested in a prompt.
- Pages and the live docs URL do not exist yet. The README must not link to them or show a CI badge until those exist.
- govulncheck, spectral, and a dedicated secret scanner are not installed yet.
