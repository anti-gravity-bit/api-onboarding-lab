# Test evidence provenance

These files are command transcripts captured on 2026-10-19 after Go downloaded
toolchain `go1.27.2`. `go version` in `go-test.txt` is part of that run.

`go-test.txt` and `go-test-race.txt` use `go test -count=1`. Vet output is in
`go-vet.txt`. `staticcheck.txt`, `golangci-lint.txt`, and `govulncheck.txt`
record those tools. `quickstart.txt` is the loopback run of the README commands
against `go run ./cmd/labd` on `127.0.0.1:8080`. A failing transcript is not a pass.
