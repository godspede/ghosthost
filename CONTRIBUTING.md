# Contributing to ghosthost

Thanks for your interest. Small project, light process.

## Branches

- Open PRs against `main`.
- No direct pushes to `main` except from maintainers.

## Code style

- Go 1.22+. `go fmt ./...` must be clean.
- `go vet ./...` and `go test ./...` must pass.
- Prefer stdlib over new dependencies. Adding a dep needs a short
  justification in the PR description.

## Commit messages

Conventional Commits style, e.g.:

    feat(server): support range requests
    fix(daemon): release lockfile on SIGINT
    docs: clarify first-run flow

## Running tests

    go test ./...
    go test -race ./...

End-to-end smoke test (builds binary, spawns daemon, hits real HTTP):

    GH_SMOKE=1 go test -tags=smoke ./internal/smoke/...

Fuzz the Content-Disposition builder:

    go test -run=^$ -fuzz=FuzzContentDispositionHeader -fuzztime=30s ./internal/share/

## Reporting security issues

See [SECURITY.md](SECURITY.md). Do not file public issues for
vulnerabilities.
