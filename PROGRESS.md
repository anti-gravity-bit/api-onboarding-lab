# Progress

Local portfolio work for API Onboarding Lab. This log records what was checked, not what was intended.

## 2026-10-19

- Created `/home/abir-antigravity/workspace/api-onboarding-lab`.
- Removed the empty directory that had been created inside the Go intro book. The book tree was not otherwise modified.
- Phase 0 discovery is recorded in `PROJECT_PLAN.md`.
- API behavior is frozen in `CONTRACT.md` and `api/openapi.yaml`. No server is implemented yet, so no runtime test transcript exists.
- Maintainer chose MIT. `LICENSE` added locally. No remote was added.
- GitHub repository was not created, pushed, or configured.
- Local git repository initialized. Planning commit `b760f40` on `main`. No remote.
- Codex phase 2 finished without a commit because its sandbox could not write `.git`. The process exit code was 1 for that reason. The implementation was reviewed locally afterward.
- Public repository `https://github.com/anti-gravity-bit/api-onboarding-lab` exists. It is public, default branch `main`, and its only commit as of this check is `a191bd31fd0be4af0cc84c2b66371e0cd5247fdd` ("Initial commit"), containing `LICENSE` with copyright Abir Saarkar. Local work has not been pushed.
- Local `LICENSE` copyright line was changed from Abir Anhad to Abir Saarkar so it matches that published file.
- `go test -count=1 ./...` and `go test -count=1 -race ./...` passed with downloaded toolchain `go1.27.2`. `go mod tidy` removes a redundant `toolchain go1.27.2` line, so the pin is the `go 1.27.2` line.
- `staticcheck` 2026.2.2, `golangci-lint` 2.15.0, and `govulncheck` v1.9.0 reported no findings. The PATH copies of staticcheck 2026.1 and golangci-lint 2.13.2 cannot read this module. Transcripts are in `evidence/`.
- A loopback quickstart was captured in `evidence/quickstart.txt`.

## Blocked on approval

- LICENSE choice.
- Creating or pushing `api-onboarding-lab`.
- Changing GitHub settings, merging to main, enabling Pages, or deploying the API.
