# Agent seats

There is no shared three-agent API. Grok drives the work and invokes the local CLIs below one seat at a time. Parallel work is limited to separate git worktrees. Do not put passwords, tokens, or webhook secrets in prompts, logs, or commits.

Verified on 2026-10-19. Do not invent extra flags.

## Grok

Orchestrator in this session. Owns requirements, the contract, task order, and integration review.

- Binary: `~/.grok/bin/grok` 1.0.62.

## Codex

Implements Go, tests, example clients, Dockerfile, and CI files.

- Binary: `/home/linuxbrew/.linuxbrew/bin/codex` 0.162.1.
- Login status checked: logged in with ChatGPT. The model for this project is `gpt-5.6-sol`.
- Working root must be this repository.

```bash
codex exec \
  -m gpt-5.6-sol \
  -s workspace-write \
  -C /home/abir-antigravity/workspace/api-onboarding-lab \
  -o build/codex-last.md \
  --json \
  - < prompts/codex-phase2.md
```

`--skip-git-repo-check` is a real flag on this Codex build. Use it only when the directory is not yet a git repository.

## AGY

Writes the documentation site, copy, accessibility, and an independent QA pass against the frozen contract. Do not point AGY at its listed Claude models.

- Binary: `~/.local/bin/agy` 1.4.2.
- Model list checked on 2026-10-19 includes `gemini-3.8-flash-high`.
- `--effort low` conflicts with that model. Use `--effort high`.
- Put `--model` and `--effort` before `--print`. `--print` consumes the next argument.

```bash
agy \
  --model gemini-3.8-flash-high \
  --effort high \
  --mode accept-edits \
  --add-dir /home/abir-antigravity/workspace/api-onboarding-lab \
  --print="$(cat prompts/agy-phase3.md)"
```

AGY runs only after the Go tests for the contract pass. It may not change route behavior to make the docs easier.

## Out of scope for every seat

- The Go intro book, employer trees, and anything under `workspace/fiverros`, `workspace/go/anhad`, or `workspace/backups`.
- `git push`, GitHub repository creation, Pages, and public deployment.
- A `LICENSE` file until the maintainer chooses one.
- Manus, and any commercial deliverable path whose license is not already cleared.
