You are the Codex implementation seat for API Onboarding Lab.

Work only inside /home/abir-antigravity/workspace/api-onboarding-lab.
Do not read or edit the Go intro book, employer trees, home credentials, or other repositories.
Do not run git push, gh repo create, or any deploy.
Do not add a LICENSE file.
Do not print or commit real secrets. The only credentials are the public fixtures in .env.example.
Do not use Manus.

Read CONTRACT.md, ACCEPTANCE.md, and api/openapi.yaml. They are frozen. If a test cannot be written without changing them, stop and write the conflict in build/codex-conflict.md instead of improvising.

Implement in this order, with one local git commit per step:

1. go.mod module github.com/anti-gravity-bit/api-onboarding-lab with `go 1.27.2` and `toolchain go1.27.2`. Standard library only.
2. Failing tests that cover every row in ACCEPTANCE.md, including concurrent duplicate delivery under go test -race. Use the public fixture secrets and an injected clock.
3. Production code that makes those tests pass:
   - cmd/labd
   - internal/config
   - internal/httpapi
   - internal/store (in-memory, not durable)
   - webhooksig (HMAC-SHA256, raw body, constant-time compare, timestamp window)
4. examples/go-client and examples/curl that match the contract, with timeouts and status checks.
5. Dockerfile multi-stage, CGO_ENABLED=0, non-root numeric user, no secrets in the image. docker-compose.yml publishes 127.0.0.1:8080 only.
6. Makefile targets: test, test-race, cover, vet, fmt.

Run the tests yourself. Save the unmodified command output in evidence/go-test.txt and evidence/go-test-race.txt. If a command fails, leave the output and fix the code. Do not hand-write a passing transcript.

Commit only this repository. Do not claim the demo is production-ready in comments or docs.
