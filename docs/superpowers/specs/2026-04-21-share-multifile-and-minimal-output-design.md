# Design: minimal `share` output + multi-file share

**Date:** 2026-04-21
**Status:** Approved (brainstorm)
**Shipping as:** one spec, two sequential PRs

## Motivation

Two related frictions in `ghosthost share`:

1. **Token noise.** The default human output of `share` emits a multi-line block (URL + id + expiry + hints). In the common case — Claude calls `share` on behalf of a user — that block is pure overhead; only the URL is load-bearing. The rest can be looked up on demand.
2. **Permission-prompt friction.** Sharing N files requires N invocations, and each invocation is a separate permission prompt in Claude Code. A single variadic invocation collapses that into one prompt.

The two changes touch the same output contract and must be designed together, but land better as two small PRs than one coordinated diff.

## Scope

- **In scope:** CLI argument shape, default output format, new `info` lookup command, optional filename anonymization, `--json` array shape.
- **Out of scope:** directory/recursive sharing, glob expansion inside ghosthost, history-walking `info`, per-file TTLs, batch RPC endpoint on the daemon.

---

## PR1 — Minimal output + `info` lookup

### CLI changes

**`ghosthost share <path>`**

- Default human output becomes a single line: the URL, trailing newline, nothing else.
- New flag `--verbose` / `-v` restores the current rich block (URL, id, expires, revoke hint).
- `--json` output is unchanged in PR1 (still a bare `SharePayload` object). PR2 turns it into an array.

**`ghosthost info <arg>`** (new subcommand)

Accepts any of:

- Full URL: `http://host:port/t/<token>/<name>`
- Path-only URL: `/t/<token>/<name>` or `t/<token>/<name>`
- Bare token (26-char base32)
- Bare id (8-char base32; exact match, consistent with `revoke`)

Normalization rules (in order):

1. If input parses as a URL with a non-empty host, extract `Path`. Otherwise treat input as path-or-identifier.
2. If the resulting string matches `^/?t/([a-z2-7]{26})(/.*)?$`, extract the token.
3. Else if the string is 26 chars of lowercase base32 (uppercase rejected), treat as a bare token.
4. Else treat as an id / id-prefix and resolve against `byID`.
5. No match → error.

Output:

- Human (default): a rich block — URL, id, src_path, created_at, expires_at.
- `--json`: a new `InfoPayload` type that embeds `SharePayload` and adds `src_path`, `created_at`.

Revoked, expired, and unknown shares all surface identically as "not found" — no `revoked` field in the response, and no information leak about which tokens previously existed.

Exit codes:

- `ExitOK` on hit.
- `ExitNotFound` (existing constant, same as `revoke` on a missing id) on any miss: unknown, expired, or revoked.

### Daemon changes

- New admin endpoint: `GET /info?q=<arg>` with the existing Bearer-token auth.
- The daemon owns all normalization. The CLI URL-encodes the raw `<arg>` and forwards it; no client-side parsing. Keeps the resolution rules in one place.
- Resolves against existing `byToken` and `byID` maps. No history walking. Expired shares are GC'd from these maps already; they surface as not-found.
- Returns `InfoPayload` or `404` with a JSON error body.

### Data model

- `Share` struct: no change.
- New `admin.InfoPayload`:
  ```go
  type InfoPayload struct {
      SharePayload              // embeds schema_version, id, token, url, expires_at
      SrcPath    string    `json:"src_path"`
      CreatedAt  time.Time `json:"created_at"`
  }
  ```
- `schema_version` on `SharePayload` remains `"1"`. `InfoPayload` inherits it.

### Tests (PR1)

- `internal/cli/commands_test.go`: new tests covering
  - default minimal-output format (single line, trailing newline, no stderr chatter)
  - `--verbose` restores rich output
  - `info` with each of the four input forms
  - `info` on unknown / expired / revoked returns `ExitNotFound` (exit 5)
- `internal/admin/handler_test.go`: `TestInfoEndpoint` covering URL/path/token/id normalization and miss behavior.
- Golden-output test for the new minimal stdout format.

### Documentation (PR1)

- `README.md` and `CLAUDE.md` §8: update the "what `share` prints" example. The hello-world proof still works but now the one-line stdout is literally the URL.
- `skills/ghosthost/SKILL.md`: update guidance — if the skill needs id/expiry, it must pass `--json` (current) or `--verbose`, or call `info` after the fact. Explicitly document this change.

---

## PR2 — Multi-file share + `--anon`

### CLI changes

**`ghosthost share <path>...`** (variadic, 1..N)

- **Atomic validation:** resolve all paths via `source.Resolve` up front. On any failure, print every failure to stderr (one per line, `<path>: <reason>`) and exit non-zero. Zero shares created.
- **Cap:** refuse batches larger than 64 with a clear error. `--yes` overrides.
- **`--as` rejected when N ≠ 1** with a clear error ("`--as` requires exactly one file").
- **`--ttl`** applies uniformly to every file in the batch.
- **`--anon`:** for each share, replace the display-name with a random 6-char base32 slug, preserving the source file's extension (`secret.pdf` → `k9vm3q.pdf`). Applies to single and multi-file alike.
- **No glob expansion, no directory expansion.** Directories continue to be rejected by `source.Resolve`.

### Output shapes

**Default (human):** one URL per line on stdout, argv order, nothing else.

```
http://host:8750/t/abc.../a.png
http://host:8750/t/def.../b.png
http://host:8750/t/ghi.../c.png
```

**`--verbose`:** one rich block per file, blank-line separated.

**`--json`:** a JSON array of `SharePayload`, argv order. **Single-file `--json` also becomes a one-element array** for consistency — this is a breaking change to the current JSON contract and must be called out.

### Daemon changes

None structural. The CLI issues N sequential `POST /share` requests after pre-flight validation passes. Rationale:

- Pre-flight validation already happens client-side in `source.Resolve`, so the remaining daemon-side work per file is small and unlikely to fail once validation passes.
- A batch `POST /shares` endpoint would double the admin handler surface for marginal atomicity value — once pre-flight passes, the realistic failure modes are "daemon is sick," where atomicity doesn't help the user.

**Mid-batch RPC failure handling:** if a request fails mid-batch, the CLI prints URLs already issued (so the user can still revoke), prints the error to stderr, and exits non-zero. Partial shares remain — acceptable because the user can see them in the output and in `list`.

### Anonymization details

- Slug: 6 characters, base32 alphabet `A-Z2-7` (~30 bits of noise). Enough to be unambiguous in a listing; token (26 chars) is what actually provides secrecy.
- Extension preservation: `filepath.Ext(src)` lowercased; if empty, the slug has no extension.
- The anonymized display-name is what's sanitized, stored in `Share.DisplayName`, and used for Content-Disposition. The original filename is never sent to the recipient in any form.
- `--anon` is mutually compatible with `--as` in the single-file case (`--as` wins — explicit user intent beats random). When `N > 1`, `--as` is rejected anyway, so no conflict.

### Tests (PR2)

- `internal/cli/commands_test.go`:
  - variadic parsing (N=1, N=3, N=0 rejected)
  - `--as` + N>1 rejection
  - cap behavior: 64 OK, 65 rejected, 65 + `--yes` OK
  - `--anon` slug shape, extension preservation, applied to every file in a batch
  - atomic validation: mixed good/bad paths → zero shares created, all errors reported
  - default multi-file stdout is exactly N lines
- `internal/admin/handler_test.go`: no change to `/share`.
- Golden-output tests for the multi-file `--json` array shape.

### Documentation (PR2)

- `README.md`, `CLAUDE.md`, `INSTALL.md`: updated synopsis, `--anon` documentation, note about the `--json` array breaking change.
- `skills/ghosthost/SKILL.md`: update to use variadic form when the user drops multiple files.

---

## Breaking changes summary

1. **PR1:** default human output of `share` loses id and expiry. Mitigation: `--verbose` or `info`.
2. **PR2:** `--json` output of `share` becomes an array even for single-file. Mitigation: callers update to read `result[0]`.

Both are called out in release notes. The Claude skill lands updated in the same PR as the change it's responding to.

## Error-handling principles

- All input validation happens in the CLI, before any RPC. Failures are collected and reported together; one bad path doesn't mask others.
- `info` miss / expired / revoked / unknown all map to `ExitNotFound` (exit 5) with a uniform message. No information leak about which shares *used* to exist.
- Daemon 5xx mid-batch: print what succeeded, print the error, exit non-zero.

## Non-goals

- Directory / recursive sharing — out of scope. If the user wants a directory, they zip it first.
- Glob expansion inside ghosthost — the shell is responsible. On Windows, users use `Get-ChildItem *.png | %{ ghosthost share $_ }` or similar.
- Per-file TTLs in batch mode — no syntax, no demand.
- Batch RPC endpoint on the daemon — marginal value, doubles handler surface.
- History-walking `info` — only live shares resolve.
