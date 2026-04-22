# PR2 — Multi-file share + `--anon` — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Prerequisite:** PR1 (`docs/superpowers/plans/2026-04-21-pr1-share-minimal-output-and-info.md`) must be merged. This plan assumes `printShare(w, f, p, verbose bool)` and `ghosthost info` already exist.

**Goal:** Make `ghosthost share` variadic (1..N file paths in one invocation) with atomic pre-flight validation, add `--anon` to replace each display-name with a random 6-char base32 slug preserving extension, and move `--json` to always emit a JSON array.

**Architecture:** The CLI accepts `<path>...`, resolves every path via `source.Resolve` before any RPC, and refuses the whole batch on any error. Daemon stays single-file — the CLI issues N sequential `POST /share` requests. `--anon` is applied in the CLI by computing the anonymized display name and passing it via the existing `ShareRequest.DisplayName` field. Output collapses from per-payload to a slice-first API: `printShares(w, f, payloads, verbose)`.

**Tech Stack:** Go 1.25. Reuses `internal/share` base32 machinery for the slug generator. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-21-share-multifile-and-minimal-output-design.md`

---

## File Structure

- **Create** `internal/share/anon.go` — `AnonDisplayName(srcPath string) string` returning `<6-char-base32-lowercase><ext>` (ext preserved, empty if src has none).
- **Create** `internal/share/anon_test.go` — verifies length, charset, extension preservation.
- **Modify** `internal/cli/commands.go` — `cmdShare` becomes variadic; add `--anon` and `--yes` flags; atomic validation loop; loop over paths issuing N `Share` RPCs; collect payloads; call new `printShares`.
- **Modify** `internal/cli/output.go` — add `printShares(w, f, payloads, verbose)`. JSON mode always encodes a slice. Human mode prints one URL per line when not verbose; one rich block per payload separated by blank lines when verbose.
- **Modify** `internal/cli/cli.go:82-95` — update `usage` string to document `<path>...`, `--anon`, `--yes`.
- **Modify** `internal/cli/commands_test.go` — add tests for variadic parsing, atomic validation, `--as` + N>1 rejection, cap behavior, `--anon` slug shape, output formats. (Note: file was created in PR1 Task 6.)
- **Modify** `README.md` — update share synopsis, add `--anon` doc, note `--json` array breaking change.
- **Modify** `CLAUDE.md` — update §8 to note multi-file behavior.
- **Modify** `skills/ghosthost/SKILL.md` — guidance on using variadic form.

---

## Task 1: `AnonDisplayName` helper

**Files:**
- Create: `internal/share/anon.go`
- Create: `internal/share/anon_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/share/anon_test.go`:

```go
package share

import (
	"regexp"
	"strings"
	"testing"
)

func TestAnonDisplayName_Shape(t *testing.T) {
	cases := []struct {
		src     string
		wantExt string
	}{
		{"secret.pdf", ".pdf"},
		{"C:\\a\\b\\IMG_0001.PNG", ".png"}, // ext lowercased
		{"/etc/hosts", ""},                  // no ext
		{"archive.tar.gz", ".gz"},           // last ext only, by design
		{"noext", ""},
		{".bashrc", ""},                     // leading dot, no extension by filepath.Ext rules
	}
	slugRe := regexp.MustCompile(`^[a-z2-7]{6}$`)
	for _, c := range cases {
		got := AnonDisplayName(c.src)
		base := got
		if c.wantExt != "" {
			if !strings.HasSuffix(got, c.wantExt) {
				t.Errorf("AnonDisplayName(%q) = %q, want suffix %q", c.src, got, c.wantExt)
			}
			base = strings.TrimSuffix(got, c.wantExt)
		}
		if !slugRe.MatchString(base) {
			t.Errorf("AnonDisplayName(%q) = %q, slug %q does not match [a-z2-7]{6}", c.src, got, base)
		}
	}
}

func TestAnonDisplayName_Unique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		s := AnonDisplayName("x.txt")
		if seen[s] {
			t.Fatalf("collision after %d iterations: %q", i, s)
		}
		seen[s] = true
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/share/ -run TestAnonDisplayName -v`
Expected: FAIL — `AnonDisplayName` undefined.

- [ ] **Step 3: Implement `AnonDisplayName`**

Create `internal/share/anon.go`:

```go
// internal/share/anon.go
package share

import (
	"crypto/rand"
	"path/filepath"
	"strings"
)

// AnonDisplayName returns a 6-char lowercase-base32 slug, followed by the
// lowercased extension of srcPath (if any). Example: "secret.PDF" -> "k9vm3q.pdf".
// The source file's original name and path are never included.
func AnonDisplayName(srcPath string) string {
	var b [4]byte // 32 bits -> 7 base32 chars, we take 6
	if _, err := rand.Read(b[:]); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	slug := strings.ToLower(b32.EncodeToString(b[:]))[:6]
	ext := strings.ToLower(filepath.Ext(srcPath))
	return slug + ext
}
```

(`b32` is the package-level `base32.StdEncoding.WithPadding(base32.NoPadding)` defined in `internal/share/token.go:11` — reuse it.)

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/share/ -run TestAnonDisplayName -v`
Expected: PASS for both subtests.

- [ ] **Step 5: Verify the slug survives `SanitizeDisplayName`**

Run this ad-hoc snippet (or drop into the test as `TestAnonDisplayName_PassesSanitize`):

```go
func TestAnonDisplayName_PassesSanitize(t *testing.T) {
	for i := 0; i < 20; i++ {
		n := AnonDisplayName("x.pdf")
		if _, err := SanitizeDisplayName(n); err != nil {
			t.Fatalf("AnonDisplayName %q failed sanitize: %v", n, err)
		}
	}
}
```

Run: `go test ./internal/share/ -run TestAnonDisplayName_PassesSanitize -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/share/anon.go internal/share/anon_test.go
git commit -m "feat(share): AnonDisplayName (6-char base32 slug + ext)"
```

---

## Task 2: `printShares` — slice-first output

**Files:**
- Modify: `internal/cli/output.go`
- Modify: `internal/cli/commands_test.go` (new tests)

- [ ] **Step 1: Write failing tests for `printShares`**

Append to `internal/cli/commands_test.go`:

```go
func TestPrintShares_DefaultMultiple(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{
		{URL: "http://h/t/a/one.png"},
		{URL: "http://h/t/b/two.png"},
		{URL: "http://h/t/c/three.png"},
	}
	printShares(&buf, Human, payloads, false)
	want := "http://h/t/a/one.png\nhttp://h/t/b/two.png\nhttp://h/t/c/three.png\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestPrintShares_DefaultSingle(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{URL: "http://h/t/a/x.png"}}
	printShares(&buf, Human, payloads, false)
	if buf.String() != "http://h/t/a/x.png\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestPrintShares_Verbose(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{
		{URL: "http://h/t/a/one.png", ID: "aaa11111", ExpiresAt: time.Unix(2_000_000_000, 0)},
		{URL: "http://h/t/b/two.png", ID: "bbb22222", ExpiresAt: time.Unix(2_000_000_000, 0)},
	}
	printShares(&buf, Human, payloads, true)
	out := buf.String()
	if strings.Count(out, "URL:") != 2 {
		t.Errorf("want 2 URL: lines, got: %q", out)
	}
	if !strings.Contains(out, "aaa11111") || !strings.Contains(out, "bbb22222") {
		t.Errorf("missing ids: %q", out)
	}
}

func TestPrintShares_JSONIsArray(t *testing.T) {
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{SchemaVersion: "1", ID: "aaa11111"}}
	printShares(&buf, JSON, payloads, false)
	s := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		t.Errorf("want JSON array, got: %q", s)
	}
	if !strings.Contains(s, `"id":"aaa11111"`) {
		t.Errorf("missing id in array: %q", s)
	}
}

func TestPrintShares_JSONArrayEvenForSingle(t *testing.T) {
	// PR2 breaking change: single-file --json is now a one-element array.
	var buf bytes.Buffer
	payloads := []admin.SharePayload{{SchemaVersion: "1", ID: "only"}}
	printShares(&buf, JSON, payloads, false)
	s := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(s, "[") {
		t.Errorf("single-file --json must be an array, got: %q", s)
	}
}
```

- [ ] **Step 2: Run to verify they fail**

Run: `go test ./internal/cli/ -run TestPrintShares -v`
Expected: FAIL — `printShares` undefined.

- [ ] **Step 3: Implement `printShares`**

Append to `internal/cli/output.go`:

```go
func printShares(w io.Writer, f Format, payloads []admin.SharePayload, verbose bool) {
	if f == JSON {
		// PR2: always a JSON array, even for a single share.
		_ = json.NewEncoder(w).Encode(payloads)
		return
	}
	if !verbose {
		for _, p := range payloads {
			fmt.Fprintln(w, p.URL)
		}
		return
	}
	for i, p := range payloads {
		if i > 0 {
			fmt.Fprintln(w)
		}
		fmt.Fprintf(w, "URL:     %s\n", p.URL)
		fmt.Fprintf(w, "ID:      %s\n", p.ID)
		fmt.Fprintf(w, "Expires: %s (%s from now)\n", p.ExpiresAt.Format(time.RFC3339),
			time.Until(p.ExpiresAt).Round(time.Second))
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `go test ./internal/cli/ -run TestPrintShares -v`
Expected: PASS for all five subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/output.go internal/cli/commands_test.go
git commit -m "feat(cli): printShares — slice-first output, JSON is always an array"
```

---

## Task 3: Variadic `cmdShare` with atomic validation

**Files:**
- Modify: `internal/cli/commands.go:25-56` (replace `cmdShare`)
- Modify: `internal/cli/cli.go:82-95` (update `usage`)
- Modify: `internal/cli/commands_test.go` (add validation tests)

The new behavior:

1. Accept `<path>...` (1..N).
2. If N == 0: print usage, exit `ExitUsage`.
3. If N > 64 and `--yes` not set: print error, exit `ExitUsage`.
4. If N > 1 and `--as` set: print error, exit `ExitUsage`.
5. Resolve every path via `source.Resolve` up front. If any fail, print `<path>: <err>` one per line to stderr and exit `ExitSourceBad`. Zero RPCs made.
6. For each resolved path, compute display name (`--as` value if N==1 and set; else `AnonDisplayName(abs)` if `--anon`; else `filepath.Base(abs)`).
7. Issue N sequential `c.Share(...)` calls. On first error, print any payloads already received (in URL-only form to stdout), then the error to stderr, exit `ExitGeneric`.
8. Print all successful payloads via `printShares`.

- [ ] **Step 1: Write failing validation tests**

Append to `internal/cli/commands_test.go`. These tests exercise `cmdShare` directly by constructing a `globalOpts` — study the existing `cli_test.go` for the setup pattern. If helpers are missing, create a minimal one:

```go
func newTestOpts(t *testing.T) (*globalOpts, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var out, errb bytes.Buffer
	o := &globalOpts{
		stdout: &out,
		stderr: &errb,
		format: Human,
		// cfg is intentionally zero; tests that need a daemon connection
		// will short-circuit before reaching EnsureDaemon.
	}
	return o, &out, &errb
}

func TestCmdShare_NoArgs(t *testing.T) {
	o, _, errb := newTestOpts(t)
	code := cmdShare(context.Background(), nil, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "usage:") {
		t.Errorf("stderr missing usage: %q", errb.String())
	}
}

func TestCmdShare_AsWithMultiple_Rejected(t *testing.T) {
	o, _, errb := newTestOpts(t)
	// Use real temp files so --as validation runs before source.Resolve.
	a := writeTempFile(t, "a")
	b := writeTempFile(t, "b")
	code := cmdShare(context.Background(), []string{"--as", "x", a, b}, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "--as") {
		t.Errorf("stderr missing --as hint: %q", errb.String())
	}
}

func TestCmdShare_CapWithoutYes_Rejected(t *testing.T) {
	o, _, errb := newTestOpts(t)
	args := []string{}
	for i := 0; i < 65; i++ {
		args = append(args, writeTempFile(t, "x"))
	}
	code := cmdShare(context.Background(), args, o)
	if code != ExitUsage {
		t.Errorf("code = %d, want ExitUsage", code)
	}
	if !strings.Contains(errb.String(), "64") {
		t.Errorf("stderr should mention the 64 cap: %q", errb.String())
	}
}

func TestCmdShare_AtomicValidation_BadPath(t *testing.T) {
	o, _, errb := newTestOpts(t)
	good := writeTempFile(t, "hello")
	bad := filepath.Join(t.TempDir(), "does-not-exist.txt")
	code := cmdShare(context.Background(), []string{good, bad}, o)
	if code != ExitSourceBad {
		t.Errorf("code = %d, want ExitSourceBad", code)
	}
	// The bad path must be reported; no RPC should have been made.
	if !strings.Contains(errb.String(), "does-not-exist.txt") {
		t.Errorf("stderr missing bad path: %q", errb.String())
	}
}
```

Helper:

```go
func writeTempFile(t *testing.T, body string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "f-"+randString()+".txt")
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func randString() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
```

Add imports: `context`, `os`, `path/filepath`, `crypto/rand`, `encoding/hex`.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run TestCmdShare -v`
Expected: all four FAIL — current `cmdShare` only accepts one arg.

- [ ] **Step 3: Rewrite `cmdShare`**

Replace the body of `cmdShare` in `internal/cli/commands.go`:

```go
const shareMaxBatch = 64

func cmdShare(ctx context.Context, args []string, o *globalOpts) int {
	fs := flag.NewFlagSet("share", flag.ContinueOnError)
	fs.SetOutput(o.stderr)
	ttl := fs.Duration("ttl", o.cfg.DefaultTTL, "time-to-live")
	displayName := fs.String("as", "", "display/download name override (requires exactly one file)")
	verbose := fs.Bool("verbose", false, "print rich output (URL, id, expiry) instead of just the URL")
	fs.BoolVar(verbose, "v", false, "shorthand for --verbose")
	anon := fs.Bool("anon", false, "replace each display-name with a random slug preserving extension")
	yes := fs.Bool("yes", false, "confirm batches larger than 64 files")

	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(o.stderr, "usage: share <path>... [--ttl 24h] [--as name] [--anon] [--verbose] [--yes]")
		return ExitUsage
	}
	paths := fs.Args()
	if len(paths) == 0 {
		fmt.Fprintln(o.stderr, "usage: share <path>... [--ttl 24h] [--as name] [--anon] [--verbose] [--yes]")
		return ExitUsage
	}
	if len(paths) > 1 && *displayName != "" {
		fmt.Fprintln(o.stderr, "--as requires exactly one file")
		return ExitUsage
	}
	if len(paths) > shareMaxBatch && !*yes {
		fmt.Fprintf(o.stderr, "refusing to create more than %d shares in one invocation; pass --yes to override\n", shareMaxBatch)
		return ExitUsage
	}

	// Atomic pre-flight: resolve every path first; if any fail, report all, zero shares created.
	absPaths := make([]string, len(paths))
	var failed bool
	for i, p := range paths {
		abs, err := source.Resolve(p)
		if err != nil {
			fmt.Fprintf(o.stderr, "%s: %v\n", p, err)
			failed = true
			continue
		}
		absPaths[i] = abs
	}
	if failed {
		return ExitSourceBad
	}

	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, "daemon:", err)
		return ExitDaemon
	}

	payloads := make([]admin.SharePayload, 0, len(absPaths))
	for i, abs := range absPaths {
		name := ""
		switch {
		case *displayName != "" && len(absPaths) == 1:
			name = *displayName
		case *anon:
			name = share.AnonDisplayName(abs)
		default:
			name = filepath.Base(abs)
		}
		p, err := c.Share(ctx, admin.ShareRequest{
			SrcPath:     abs,
			DisplayName: name,
			TTLSeconds:  int64(ttl.Seconds()),
		})
		if err != nil {
			// Print the URLs that succeeded so the user can still revoke them,
			// then surface the error. Partial shares are acceptable because
			// pre-flight validation already passed.
			printShares(o.stdout, o.format, payloads, *verbose)
			fmt.Fprintf(o.stderr, "share %s: %v\n", paths[i], err)
			return ExitGeneric
		}
		payloads = append(payloads, p)
	}

	printShares(o.stdout, o.format, payloads, *verbose)
	return ExitOK
}
```

Add imports: `"github.com/godspede/ghosthost/internal/share"` (already present? verify).

- [ ] **Step 4: Update `usage` in `internal/cli/cli.go`**

Change the `share` line in the `usage` constant to:

```
  share <path>... [--ttl 24h] [--as name] [--anon] [--verbose] [--yes]
```

- [ ] **Step 5: Run the full CLI test suite**

Run: `go test ./internal/cli/ -v`
Expected: all tests pass (new variadic validation tests, plus all PR1 tests still green).

- [ ] **Step 6: Commit**

```bash
git add internal/cli/commands.go internal/cli/cli.go internal/cli/commands_test.go
git commit -m "feat(cli): variadic share with --anon, --yes, atomic validation"
```

---

## Task 4: Integration-style test — variadic share end-to-end

**Files:**
- Modify: `internal/cli/commands_test.go`

This test spins up an `httptest.Server` mimicking the daemon's `/share` endpoint and verifies N files → N URLs in argv order. It locks in the happy path.

- [ ] **Step 1: Write the failing test**

Append to `internal/cli/commands_test.go`:

```go
func TestCmdShare_MultipleFiles_HappyPath(t *testing.T) {
	t.Skip("requires daemon bootstrap wiring; implement if a test server hook exists, otherwise cover via manual smoke test")
}
```

The CLI bootstraps the daemon via `EnsureDaemon`, which is hard to stub without invasive refactoring. If the codebase already provides an injection point (check `internal/cli/daemon_bootstrap.go`), implement the test fully. Otherwise leave as `t.Skip` with a clear note and rely on the manual smoke test in Task 6.

- [ ] **Step 2: Inspect the bootstrap code**

Run: `cat internal/cli/daemon_bootstrap.go`

If `EnsureDaemon` can be overridden via a package-level var or interface, replace the `t.Skip` with a real test that:

- Starts an `httptest.Server` with a `/share` handler that echoes the `SrcPath`.
- Overrides the bootstrap to return a `Client` pointing at that server.
- Runs `cmdShare` with 3 temp-file paths.
- Asserts stdout has exactly 3 lines and each URL contains the expected source filename (or slug when `--anon` is set).

If no injection point exists, DO NOT ADD ONE in this PR — leave the `t.Skip` with a one-line comment explaining why (no bootstrap override available), and rely on the manual smoke test in Task 6 to cover the happy path. Adding a hook purely for tests is out of scope here.

- [ ] **Step 3: Run to confirm state**

Run: `go test ./internal/cli/ -run TestCmdShare_MultipleFiles_HappyPath -v`
Expected: either PASS (if implemented) or SKIP (if not).

- [ ] **Step 4: Commit**

```bash
git add internal/cli/commands_test.go
git commit -m "test(cli): multi-file share happy-path integration (or placeholder)"
```

---

## Task 5: Update documentation

**Files:**
- Modify: `README.md`
- Modify: `CLAUDE.md`
- Modify: `skills/ghosthost/SKILL.md`

- [ ] **Step 1: Update `README.md`**

Locate the `share` synopsis and usage example (was updated in PR1 Task 8). Update to show:

```
ghosthost share <path>... [--ttl 24h] [--as name] [--anon] [--verbose] [--yes]
```

Add a "Sharing multiple files at once" subsection near the existing share example:

```bash
$ ghosthost share a.png b.png c.png
http://homepc.tail-4a9c2e.ts.net:8750/t/abc.../a.png
http://homepc.tail-4a9c2e.ts.net:8750/t/def.../b.png
http://homepc.tail-4a9c2e.ts.net:8750/t/ghi.../c.png
```

Add an "Anonymizing filenames" subsection:

```bash
$ ghosthost share --anon secret-tax-return.pdf
http://homepc.tail-4a9c2e.ts.net:8750/t/abc.../k9vm3q.pdf
```

Note the behavior rules:
- Validation is atomic: any bad path aborts the whole batch.
- `--as` requires exactly one file.
- `--anon` preserves the file's extension; the recipient sees only the random slug.
- Batches over 64 files require `--yes`.

Add a **Breaking changes** callout (near the top of a "Changes in this release" section, or in the CHANGELOG if one exists):

> **Breaking:** `ghosthost share --json` now always emits a JSON array (previously a bare `SharePayload` object). Single-file invocations return a one-element array. Callers parsing the output must update to read `result[0]`.

- [ ] **Step 2: Update `CLAUDE.md`**

In §8, note the new default behavior for multi-file drops. Suggested insertion near §8's end:

> **Multiple files:** `ghosthost share a.png b.png c.png` creates three independent shares in one invocation and prints one URL per line. This is the preferred form when dropping a batch of files — it avoids multiple permission prompts. Pass `--anon` to randomize each URL's filename segment while preserving the extension.

Update any remaining examples that pass a single path to `share` — they still work unchanged, but if the §8 proof-of-install shows `--json` parsing, update the consumer to read an array (`jq '.[0].url'` instead of `jq '.url'`).

- [ ] **Step 3: Update `skills/ghosthost/SKILL.md`**

Add or update guidance:

> **When the user wants to share multiple local files, pass them all in a single `ghosthost share` invocation:** `ghosthost share file1.png file2.png file3.png`. This avoids multiple permission prompts. Every file gets its own URL on its own line of stdout.
>
> **Anonymize the URL filename with `--anon`** when the original filename might be sensitive. The extension is preserved for correct browser handling; only the filename is replaced with a random 6-char slug.
>
> **`--json` output is always a JSON array** — even for a single file. Parse `result[0].url` for single-file cases.

- [ ] **Step 4: Verify docs are self-consistent**

Run: `grep -rn "ghosthost share" README.md CLAUDE.md skills/ghosthost/SKILL.md`
Expected: no examples contradict the new variadic / array-JSON behavior.

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md skills/ghosthost/SKILL.md
git commit -m "docs: variadic share, --anon, and --json array shape"
```

---

## Task 6: Final verification

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 2: `go vet`**

Run: `go vet ./...`
Expected: no output.

- [ ] **Step 3: Manual smoke test — multi-file**

Run:
```bash
for i in 1 2 3; do printf "file %d\n" $i > /tmp/gh-multi-$i.txt; done
ghosthost share /tmp/gh-multi-1.txt /tmp/gh-multi-2.txt /tmp/gh-multi-3.txt
```
Expected: exactly 3 URL lines on stdout, in argv order, no stderr chatter. Each URL resolves via `curl -I` to `200 OK`.

- [ ] **Step 4: Manual smoke test — `--anon`**

Run:
```bash
ghosthost share --anon /tmp/gh-multi-1.txt
```
Expected: the URL's final segment is `<6-char-slug>.txt`, NOT `gh-multi-1.txt`. Copy the URL and verify `curl` returns `file 1`.

- [ ] **Step 5: Manual smoke test — atomic validation**

Run:
```bash
ghosthost share /tmp/gh-multi-1.txt /tmp/does-not-exist.txt; echo "exit=$?"
```
Expected: stderr mentions `does-not-exist.txt`, `exit=6` (`ExitSourceBad`), and `ghosthost list` still shows no new shares.

- [ ] **Step 6: Manual smoke test — `--as` + multi rejected**

Run:
```bash
ghosthost share --as renamed /tmp/gh-multi-1.txt /tmp/gh-multi-2.txt; echo "exit=$?"
```
Expected: stderr `--as requires exactly one file`, `exit=2` (`ExitUsage`).

- [ ] **Step 7: Manual smoke test — cap**

Run:
```bash
FILES=$(for i in $(seq 1 65); do printf '/tmp/gh-cap-%d.txt ' $i; touch /tmp/gh-cap-$i.txt; done)
ghosthost share $FILES; echo "exit=$?"
```
Expected: stderr mentions `64`, `exit=2`. Then:
```bash
ghosthost share --yes $FILES
```
Expected: 65 URL lines on stdout.

- [ ] **Step 8: Manual smoke test — `--json` is always an array**

Run:
```bash
ghosthost --json share /tmp/gh-multi-1.txt | jq 'type'
```
Expected: `"array"`.

- [ ] **Step 9: Clean up test files**

```bash
rm -f /tmp/gh-multi-*.txt /tmp/gh-cap-*.txt
ghosthost list --json | jq -r '.shares[].id' | xargs -I{} ghosthost revoke {}
```

- [ ] **Step 10: Open the PR**

Push and open a PR titled "share: variadic + `--anon` + JSON array shape" with:
- Link to the spec (`docs/superpowers/specs/2026-04-21-share-multifile-and-minimal-output-design.md`).
- A "Breaking changes" section listing the `--json` array shift.
- Manual-test results from Steps 3-8.
