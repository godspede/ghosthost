# PR1 — Share minimal output + `info` lookup — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Reshape `ghosthost share`'s default human output to a single URL line, and add a new `ghosthost info <arg>` subcommand that resolves a URL, path, token, or id back to the share's metadata.

**Architecture:** The daemon owns all input-normalization for `info`. The CLI URL-encodes the raw argument and forwards it; the daemon parses URL-path/token/id forms, looks up against existing `byToken`/`byID` maps, and returns an `InfoPayload` that embeds `SharePayload` with two extra fields (`src_path`, `created_at`). The `share` command's default output becomes a bare URL; `--verbose` restores today's rich block. `--json` for `share` is unchanged in this PR (still a bare `SharePayload`).

**Tech Stack:** Go 1.25, `net/http`, `encoding/json`, `flag`. No new dependencies.

**Spec:** `docs/superpowers/specs/2026-04-21-share-multifile-and-minimal-output-design.md`

---

## File Structure

- **Modify** `internal/admin/schema.go` — add `InfoPayload` type.
- **Create** `internal/admin/info.go` — pure `ParseInfoQuery(raw string) (token, id string, err error)` function that normalizes the four accepted input forms. Returns exactly one of `token`/`id` populated, never both.
- **Create** `internal/admin/info_test.go` — table-test the parser against every input form and malformed cases.
- **Modify** `internal/admin/handler.go` — add `Info(query string) (InfoPayload, error)` to the `Core` interface, register `GET /info` handler, wire it up to call `core.Info(r.URL.Query().Get("q"))`.
- **Modify** `internal/admin/handler_test.go` — add `Info` to `fakeCore`, add `TestInfoEndpoint` covering the four input forms and 404 on miss.
- **Modify** `internal/daemon/daemon.go` — implement `Core.Info(query string) (admin.InfoPayload, error)`. Uses `admin.ParseInfoQuery`, looks up in `byToken` then `byID`, constructs the payload.
- **Modify** `internal/daemon/daemon_test.go` — add a test for `Core.Info` that exercises live/expired/revoked/unknown.
- **Modify** `internal/cli/client.go` — add `Info(ctx, arg string) (admin.InfoPayload, error)`.
- **Modify** `internal/cli/output.go` — change `printShare` to take a `verbose bool`; default path prints URL only. Add `printInfo`.
- **Modify** `internal/cli/commands.go` — add `--verbose` / `-v` flag to `cmdShare`; thread it into `printShare`. Add `cmdInfo`. Register `info` in the `commands` map.
- **Modify** `internal/cli/cli.go` — update the `usage` string to document `info`.
- **Create** `internal/cli/commands_test.go` — golden tests for minimal-vs-verbose share output and for `cmdInfo` dispatch. (Does not currently exist.)
- **Modify** `README.md` — update share-output examples.
- **Modify** `CLAUDE.md` — update §8 hello-world example output and the render guidance to match the new minimal stdout.
- **Modify** `skills/ghosthost/SKILL.md` — note that human-mode `share` prints URL only; use `--json` or `info` for metadata.

---

## Task 1: Pure `ParseInfoQuery` normalization

**Files:**
- Create: `internal/admin/info.go`
- Create: `internal/admin/info_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/admin/info_test.go`:

```go
package admin

import "testing"

func TestParseInfoQuery(t *testing.T) {
	const (
		tok = "aaaabbbbccccddddeeeeffffgg" // 26 chars, lowercase base32
		id  = "abcdefgh"                    // 8 chars, lowercase base32
	)
	cases := []struct {
		name        string
		in          string
		wantToken   string
		wantID      string
		wantErr     bool
	}{
		{"bare token", tok, tok, "", false},
		{"bare id", id, "", id, false},
		{"path-only with name", "/t/" + tok + "/foo.png", tok, "", false},
		{"path-only no name", "/t/" + tok, tok, "", false},
		{"path-only no leading slash", "t/" + tok + "/foo.png", tok, "", false},
		{"full URL", "http://host:8750/t/" + tok + "/foo.png", tok, "", false},
		{"https URL", "https://host/t/" + tok, tok, "", false},
		{"empty", "", "", "", true},
		{"junk", "not a real thing", "", "", true},
		{"URL without /t/", "http://host/other/thing", "", "", true},
		{"token wrong length", "abc", "", "", true},
		{"token wrong chars", "1!!!aaaabbbbccccddddeeefff", "", "", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tok, id, err := ParseInfoQuery(c.in)
			if (err != nil) != c.wantErr {
				t.Fatalf("err = %v, wantErr = %v", err, c.wantErr)
			}
			if tok != c.wantToken || id != c.wantID {
				t.Errorf("got token=%q id=%q, want token=%q id=%q",
					tok, id, c.wantToken, c.wantID)
			}
		})
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/admin/ -run TestParseInfoQuery -v`
Expected: FAIL with "undefined: ParseInfoQuery".

- [ ] **Step 3: Implement `ParseInfoQuery`**

Create `internal/admin/info.go`:

```go
// internal/admin/info.go
package admin

import (
	"errors"
	"net/url"
	"regexp"
	"strings"
)

// Token is 26 chars of lowercase base32 (RFC 4648 a-z2-7, no padding).
var tokenRe = regexp.MustCompile(`^[a-z2-7]{26}$`)

// ID is 8 chars of lowercase base32.
var idRe = regexp.MustCompile(`^[a-z2-7]{8}$`)

// pathRe captures the token from /t/<token> or /t/<token>/<name>.
var pathRe = regexp.MustCompile(`(?:^|/)t/([a-z2-7]{26})(?:/.*)?$`)

// ParseInfoQuery normalizes one of {full URL, path /t/<tok>/<name>, bare token, bare id}
// into either a token (26-char base32) or an id (8-char base32). Exactly one
// of the returned values is populated on success.
func ParseInfoQuery(raw string) (token, id string, err error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", "", errors.New("empty query")
	}

	// URL with scheme: parse, extract path.
	if u, perr := url.Parse(s); perr == nil && u.Scheme != "" && u.Host != "" {
		if m := pathRe.FindStringSubmatch(u.Path); m != nil {
			return m[1], "", nil
		}
		return "", "", errors.New("URL does not contain a /t/<token> path")
	}

	// Path form, with or without leading slash.
	if m := pathRe.FindStringSubmatch(s); m != nil {
		return m[1], "", nil
	}

	// Bare token.
	if tokenRe.MatchString(s) {
		return s, "", nil
	}

	// Bare id.
	if idRe.MatchString(s) {
		return "", s, nil
	}

	return "", "", errors.New("not a recognizable URL, path, token, or id")
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/admin/ -run TestParseInfoQuery -v`
Expected: PASS for all 12 subtests.

- [ ] **Step 5: Commit**

```bash
git add internal/admin/info.go internal/admin/info_test.go
git commit -m "feat(admin): ParseInfoQuery normalizes URL/path/token/id"
```

---

## Task 2: `InfoPayload` schema type

**Files:**
- Modify: `internal/admin/schema.go` (after line 15, the end of `SharePayload`)

- [ ] **Step 1: Add the type**

Insert into `internal/admin/schema.go` immediately after the `SharePayload` struct definition:

```go
// InfoPayload augments SharePayload with src_path and created_at, returned by
// GET /info.
type InfoPayload struct {
	SharePayload
	SrcPath   string    `json:"src_path"`
	CreatedAt time.Time `json:"created_at"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/admin/`
Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add internal/admin/schema.go
git commit -m "feat(admin): add InfoPayload type"
```

---

## Task 3: Extend `Core` interface + daemon implementation of `Info`

**Files:**
- Modify: `internal/admin/handler.go:12-20` (Core interface)
- Modify: `internal/daemon/daemon.go` (add `Info` method after `Reshare`, ~line 190)
- Modify: `internal/daemon/daemon_test.go` (add test for `Info`)

- [ ] **Step 1: Add `Info` to the `Core` interface**

In `internal/admin/handler.go`, extend the `Core` interface:

```go
type Core interface {
	Secret() string
	Share(ShareRequest) (SharePayload, error)
	Revoke(id string) error
	Reshare(id string) (SharePayload, error)
	Info(query string) (InfoPayload, error)
	List() ListResponse
	Status() StatusResponse
	Stop()
}
```

- [ ] **Step 2: Add a failing test for `Core.Info`**

Check whether `internal/daemon/daemon_test.go` exists first:

Run: `ls internal/daemon/`

If `daemon_test.go` exists, append to it. Otherwise create it. Inspect adjacent files for how `Core` is constructed in tests (look for `NewCore` usage and any helper like `newTestCore`). Use the same construction here.

Append this test (adapt the setup helper's name to match what you find in existing tests):

```go
func TestCore_Info(t *testing.T) {
	c := newTestCore(t) // use whatever helper the existing tests use
	// create a share
	abs := writeTempFile(t, "hello")
	p, err := c.Share(admin.ShareRequest{SrcPath: abs, DisplayName: "hello.txt"})
	if err != nil {
		t.Fatal(err)
	}

	// lookup by token
	got, err := c.Info(p.Token)
	if err != nil {
		t.Fatalf("Info(token): %v", err)
	}
	if got.ID != p.ID || got.SrcPath != abs {
		t.Errorf("Info(token) = %+v, want id=%s src=%s", got, p.ID, abs)
	}

	// lookup by id
	got, err = c.Info(p.ID)
	if err != nil {
		t.Fatalf("Info(id): %v", err)
	}
	if got.URL != p.URL {
		t.Errorf("Info(id).URL = %q, want %q", got.URL, p.URL)
	}

	// lookup by path-only URL
	got, err = c.Info("/t/" + p.Token + "/hello.txt")
	if err != nil {
		t.Fatalf("Info(path): %v", err)
	}
	if got.ID != p.ID {
		t.Errorf("Info(path).ID = %q, want %q", got.ID, p.ID)
	}

	// unknown
	if _, err := c.Info("zzzzzzzz"); err == nil {
		t.Error("Info(unknown id) should error")
	}

	// revoked -> not found
	if err := c.Revoke(p.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.Info(p.ID); err == nil {
		t.Error("Info on revoked id should error")
	}
}
```

If `newTestCore` and `writeTempFile` helpers don't exist, write them locally at the top of the test file. Example:

```go
func writeTempFile(t *testing.T, contents string) string {
	t.Helper()
	dir := t.TempDir()
	p := filepath.Join(dir, "f.txt")
	if err := os.WriteFile(p, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}
```

If a `newTestCore` doesn't exist, model one after `daemon.NewCore` plus a temp `history.Store`. Look at how `TestShareEndpoint` in `internal/admin/handler_test.go` or existing tests in `internal/daemon/` initialize cores.

- [ ] **Step 3: Run the test to verify it fails**

Run: `go test ./internal/daemon/ -run TestCore_Info -v`
Expected: FAIL with "c.Info undefined" (or build failure because the interface expects it).

- [ ] **Step 4: Implement `Core.Info`**

In `internal/daemon/daemon.go`, add after the `Reshare` method (around line 190):

```go
func (c *Core) Info(query string) (admin.InfoPayload, error) {
	tok, id, err := admin.ParseInfoQuery(query)
	if err != nil {
		return admin.InfoPayload{}, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := c.clock.Now()
	var s *share.Share
	if tok != "" {
		s = c.byToken[tok]
		if s != nil && !share.EqualDigest(share.Digest(tok), s.TokenDigest) {
			s = nil
		}
	} else {
		s = c.byID[id]
	}
	if s == nil || !s.Active(now) {
		return admin.InfoPayload{}, errors.New("not found")
	}
	return admin.InfoPayload{
		SharePayload: admin.SharePayload{
			SchemaVersion: admin.SchemaVersion,
			ID:            s.ID,
			Token:         s.Token,
			URL:           c.buildURL(s.Token, s.DisplayName),
			ExpiresAt:     s.ExpiresAt,
		},
		SrcPath:   s.SrcPath,
		CreatedAt: s.CreatedAt,
	}, nil
}
```

- [ ] **Step 5: Update the `fakeCore` in handler_test.go**

In `internal/admin/handler_test.go`, add an `Info` method to `fakeCore` (around line 29):

```go
func (f *fakeCore) Info(query string) (InfoPayload, error) {
	return InfoPayload{
		SharePayload: SharePayload{SchemaVersion: SchemaVersion, ID: "i", Token: "t", URL: "u"},
		SrcPath:      "/abs/path",
	}, nil
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `go test ./internal/daemon/ -run TestCore_Info -v && go build ./...`
Expected: PASS on `TestCore_Info`; full build succeeds.

- [ ] **Step 7: Commit**

```bash
git add internal/admin/handler.go internal/admin/handler_test.go internal/daemon/daemon.go internal/daemon/daemon_test.go
git commit -m "feat(daemon): Core.Info resolves URL/path/token/id to InfoPayload"
```

---

## Task 4: Wire up `GET /info` admin endpoint

**Files:**
- Modify: `internal/admin/handler.go:22-33` (route registration) and add handler method
- Modify: `internal/admin/handler_test.go` (add `TestInfoEndpoint`)

- [ ] **Step 1: Write the failing test**

Append to `internal/admin/handler_test.go`:

```go
func TestInfoEndpoint(t *testing.T) {
	core := &fakeCore{secret: "s"}
	h := NewHandler(core)

	// hit: daemon returns an InfoPayload
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/info?q=aaaabbbbccccddddeeeeffffgg", nil)
	r.Header.Set("Authorization", "Bearer s")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("code=%d body=%s", w.Code, w.Body.String())
	}
	var p InfoPayload
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.SrcPath == "" {
		t.Error("InfoPayload.SrcPath should be populated")
	}

	// miss: daemon returns error -> 404
	missCore := &fakeCore{secret: "s", infoErr: true}
	h = NewHandler(missCore)
	w = httptest.NewRecorder()
	r = httptest.NewRequest("GET", "/info?q=zzzzzzzz", nil)
	r.Header.Set("Authorization", "Bearer s")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusNotFound {
		t.Errorf("miss: want 404, got %d", w.Code)
	}
}
```

Then extend `fakeCore`:

```go
type fakeCore struct {
	secret  string
	infoErr bool
}

func (f *fakeCore) Info(query string) (InfoPayload, error) {
	if f.infoErr {
		return InfoPayload{}, errors.New("not found")
	}
	return InfoPayload{
		SharePayload: SharePayload{SchemaVersion: SchemaVersion, ID: "i", Token: "t", URL: "u"},
		SrcPath:      "/abs/path",
	}, nil
}
```

(Replace the existing `Info` method added in Task 3 with this extended version. Add `"errors"` to imports.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/admin/ -run TestInfoEndpoint -v`
Expected: FAIL with 404 (route not registered → serves mux default 404, but test expects 200 for hit case).

- [ ] **Step 3: Register the route and add the handler**

In `internal/admin/handler.go`, in `NewHandler` after the `/stop` line:

```go
mux.HandleFunc("/info", h.auth(h.info))
```

Then add the handler method:

```go
func (h *handler) info(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	p, err := h.core.Info(q)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/admin/ -v`
Expected: all tests pass, including `TestInfoEndpoint`.

- [ ] **Step 5: Commit**

```bash
git add internal/admin/handler.go internal/admin/handler_test.go
git commit -m "feat(admin): GET /info endpoint"
```

---

## Task 5: Client `Info` method

**Files:**
- Modify: `internal/cli/client.go` (add `Info` after `Reshare`, ~line 74)
- Modify: `internal/cli/client_test.go` (add a test)

- [ ] **Step 1: Write the failing test**

Inspect existing tests in `internal/cli/client_test.go` to see the httptest server setup pattern. Append a test that mirrors the existing style (likely uses `httptest.NewServer` with a handler that returns canned JSON):

```go
func TestClient_Info(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/info" {
			t.Errorf("path = %q, want /info", r.URL.Path)
		}
		if q := r.URL.Query().Get("q"); q != "abcd1234" {
			t.Errorf("q = %q, want abcd1234", q)
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(admin.InfoPayload{
			SharePayload: admin.SharePayload{SchemaVersion: "1", ID: "abcd1234"},
			SrcPath:      "/x",
		})
	}))
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	port, _ := strconv.Atoi(u.Port())
	c := NewClient(port, "")
	c.BaseURL = srv.URL // override loopback assumption

	got, err := c.Info(context.Background(), "abcd1234")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "abcd1234" || got.SrcPath != "/x" {
		t.Errorf("got %+v", got)
	}
}
```

Add any missing imports (`net/url`, `strconv`, `context`, `encoding/json`, `net/http`, `net/http/httptest`, `github.com/godspede/ghosthost/internal/admin`).

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/cli/ -run TestClient_Info -v`
Expected: FAIL with "c.Info undefined".

- [ ] **Step 3: Implement `Client.Info`**

In `internal/cli/client.go`, after `Reshare`:

```go
func (c *Client) Info(ctx context.Context, arg string) (admin.InfoPayload, error) {
	var p admin.InfoPayload
	err := c.do(ctx, "GET", "/info?q="+url.QueryEscape(arg), nil, &p)
	return p, err
}
```

Add `"net/url"` to imports.

- [ ] **Step 4: Run the test to verify it passes**

Run: `go test ./internal/cli/ -run TestClient_Info -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/cli/client.go internal/cli/client_test.go
git commit -m "feat(cli): Client.Info"
```

---

## Task 6: Minimal share output + `--verbose` flag

**Files:**
- Modify: `internal/cli/output.go:21-30` (`printShare`)
- Modify: `internal/cli/commands.go:25-56` (`cmdShare` — add `-v`/`--verbose`)
- Create: `internal/cli/commands_test.go` (golden tests for both output modes)

- [ ] **Step 1: Write the failing golden test**

Create `internal/cli/commands_test.go`:

```go
package cli

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
)

func TestPrintShare_MinimalDefault(t *testing.T) {
	var buf bytes.Buffer
	p := admin.SharePayload{
		SchemaVersion: "1",
		ID:            "abc12345",
		Token:         "t",
		URL:           "http://h/t/t/x",
		ExpiresAt:     time.Unix(2_000_000_000, 0),
	}
	printShare(&buf, Human, p, false)
	got := buf.String()
	want := "http://h/t/t/x\n"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestPrintShare_Verbose(t *testing.T) {
	var buf bytes.Buffer
	p := admin.SharePayload{URL: "http://h/t/t/x", ID: "abc12345", ExpiresAt: time.Unix(2_000_000_000, 0)}
	printShare(&buf, Human, p, true)
	got := buf.String()
	if !strings.Contains(got, "URL:") || !strings.Contains(got, "ID:") || !strings.Contains(got, "Expires:") {
		t.Errorf("verbose output missing fields: %q", got)
	}
}

func TestPrintShare_JSONUnchanged(t *testing.T) {
	var buf bytes.Buffer
	p := admin.SharePayload{SchemaVersion: "1", ID: "abc12345", Token: "tok", URL: "u"}
	printShare(&buf, JSON, p, false)
	if !strings.Contains(buf.String(), `"id":"abc12345"`) {
		t.Errorf("JSON output missing id: %q", buf.String())
	}
	// JSON output is a bare object in PR1 (not an array yet).
	if strings.HasPrefix(strings.TrimSpace(buf.String()), "[") {
		t.Errorf("PR1 JSON should be a bare object, got array: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `go test ./internal/cli/ -run TestPrintShare -v`
Expected: COMPILE FAIL — `printShare` takes 3 args, not 4.

- [ ] **Step 3: Update `printShare`**

Replace the existing `printShare` in `internal/cli/output.go`:

```go
func printShare(w io.Writer, f Format, p admin.SharePayload, verbose bool) {
	if f == JSON {
		_ = json.NewEncoder(w).Encode(p)
		return
	}
	if !verbose {
		fmt.Fprintln(w, p.URL)
		return
	}
	fmt.Fprintf(w, "URL:     %s\n", p.URL)
	fmt.Fprintf(w, "ID:      %s\n", p.ID)
	fmt.Fprintf(w, "Expires: %s (%s from now)\n", p.ExpiresAt.Format(time.RFC3339),
		time.Until(p.ExpiresAt).Round(time.Second))
}
```

- [ ] **Step 4: Update all callers of `printShare`**

`printShare` is called in three places in `internal/cli/commands.go` — `cmdShare`, `cmdReshare`. Update them:

In `cmdShare` (around line 25), add a `--verbose` flag and thread it:

```go
func cmdShare(ctx context.Context, args []string, o *globalOpts) int {
	fs := flag.NewFlagSet("share", flag.ContinueOnError)
	fs.SetOutput(o.stderr)
	ttl := fs.Duration("ttl", o.cfg.DefaultTTL, "time-to-live")
	displayName := fs.String("as", "", "display/download name override")
	verbose := fs.Bool("verbose", false, "print rich output (URL, id, expiry) instead of just the URL")
	fs.BoolVar(verbose, "v", false, "shorthand for --verbose")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprintln(o.stderr, "usage: share <path> [--ttl 24h] [--as name] [--verbose]")
		return ExitUsage
	}
	// ... unchanged middle ...
	printShare(o.stdout, o.format, p, *verbose)
	return ExitOK
}
```

In `cmdReshare` (around line 104), pass `true` to preserve the rich reshare output:

```go
printShare(o.stdout, o.format, p, true)
```

(Rationale: `reshare` is an interactive re-creation — the user ran a command explicitly to get a URL back with metadata. Keeping it verbose matches its intent. If this feels wrong, see spec note on `reshare` — default it to `false` to match `share`. For this PR, keep `reshare` verbose.)

- [ ] **Step 5: Run all CLI tests**

Run: `go test ./internal/cli/ -v`
Expected: all tests pass, including the three new `TestPrintShare_*` tests.

- [ ] **Step 6: Commit**

```bash
git add internal/cli/output.go internal/cli/commands.go internal/cli/commands_test.go
git commit -m "feat(cli): default share output is URL only; --verbose restores rich block"
```

---

## Task 7: `cmdInfo` subcommand

**Files:**
- Modify: `internal/cli/commands.go` (add `cmdInfo`, register in map)
- Modify: `internal/cli/output.go` (add `printInfo`)
- Modify: `internal/cli/cli.go:82-95` (update `usage` string)
- Modify: `internal/cli/commands_test.go` (add tests)

- [ ] **Step 1: Write the failing test for `printInfo`**

Append to `internal/cli/commands_test.go`:

```go
func TestPrintInfo_Human(t *testing.T) {
	var buf bytes.Buffer
	p := admin.InfoPayload{
		SharePayload: admin.SharePayload{
			SchemaVersion: "1",
			ID:            "abc12345",
			URL:           "http://h/t/tok/x",
			ExpiresAt:     time.Unix(2_000_000_000, 0),
		},
		SrcPath:   "/abs/x",
		CreatedAt: time.Unix(1_000_000_000, 0),
	}
	printInfo(&buf, Human, p)
	out := buf.String()
	for _, s := range []string{"URL:", "ID:", "Src:", "Created:", "Expires:", "abc12345", "/abs/x"} {
		if !strings.Contains(out, s) {
			t.Errorf("missing %q in: %q", s, out)
		}
	}
}

func TestPrintInfo_JSON(t *testing.T) {
	var buf bytes.Buffer
	p := admin.InfoPayload{
		SharePayload: admin.SharePayload{SchemaVersion: "1", ID: "abc12345"},
		SrcPath:      "/abs/x",
	}
	printInfo(&buf, JSON, p)
	if !strings.Contains(buf.String(), `"src_path":"/abs/x"`) {
		t.Errorf("missing src_path: %q", buf.String())
	}
}
```

- [ ] **Step 2: Run the test to verify it fails**

Run: `go test ./internal/cli/ -run TestPrintInfo -v`
Expected: FAIL — `printInfo` undefined.

- [ ] **Step 3: Add `printInfo`**

In `internal/cli/output.go`, append:

```go
func printInfo(w io.Writer, f Format, p admin.InfoPayload) {
	if f == JSON {
		_ = json.NewEncoder(w).Encode(p)
		return
	}
	fmt.Fprintf(w, "URL:     %s\n", p.URL)
	fmt.Fprintf(w, "ID:      %s\n", p.ID)
	fmt.Fprintf(w, "Src:     %s\n", p.SrcPath)
	fmt.Fprintf(w, "Created: %s\n", p.CreatedAt.Format(time.RFC3339))
	fmt.Fprintf(w, "Expires: %s (%s from now)\n", p.ExpiresAt.Format(time.RFC3339),
		time.Until(p.ExpiresAt).Round(time.Second))
}
```

- [ ] **Step 4: Add `cmdInfo` and register it**

In `internal/cli/commands.go`, add to the `commands` map:

```go
var commands = map[string]subcmd{
	"share":   cmdShare,
	"info":    cmdInfo,
	"list":    cmdList,
	"history": cmdHistory,
	"reshare": cmdReshare,
	"revoke":  cmdRevoke,
	"status":  cmdStatus,
	"stop":    cmdStop,
}
```

Append the command:

```go
func cmdInfo(ctx context.Context, args []string, o *globalOpts) int {
	if len(args) != 1 {
		fmt.Fprintln(o.stderr, "usage: info <url-or-path-or-token-or-id>")
		return ExitUsage
	}
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, "daemon:", err)
		return ExitDaemon
	}
	p, err := c.Info(ctx, args[0])
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitNotFound
	}
	printInfo(o.stdout, o.format, p)
	return ExitOK
}
```

- [ ] **Step 5: Update the `usage` string**

In `internal/cli/cli.go`, update the `usage` constant's `Commands:` block to include `info`:

```go
const usage = `ghosthost — temporary file sharing over HTTP

Commands:
  share <path> [--ttl 24h] [--as name] [--verbose]
  info <url-or-path-or-token-or-id>
  list
  history [--limit N]
  reshare <id>
  revoke <id>
  status
  stop

Global flags:
  --config <path>   override config.toml location
  --json            emit machine-readable JSON`
```

- [ ] **Step 6: Run the full CLI test suite**

Run: `go test ./internal/cli/ -v`
Expected: all tests pass, including the two new `TestPrintInfo_*` tests.

- [ ] **Step 7: Commit**

```bash
git add internal/cli/commands.go internal/cli/output.go internal/cli/cli.go internal/cli/commands_test.go
git commit -m "feat(cli): info subcommand"
```

---

## Task 8: Update documentation

**Files:**
- Modify: `README.md` — share-output examples
- Modify: `CLAUDE.md` — §8 hello-world proof-of-install
- Modify: `skills/ghosthost/SKILL.md` — guidance on default vs verbose output

- [ ] **Step 1: Update `README.md`**

Run: `grep -n "URL:\|Expires:\|ID:" README.md` — find every example that shows the old rich output for `share`.

For each such example, replace the rich block with a single-line URL, and add a nearby sentence like:

> Human-mode `share` prints just the URL. Pass `--verbose` for the full id + expiry block, or look up an active share later with `ghosthost info <url-or-id>`.

Add a new "Info lookup" subsection near the existing `share`/`list` documentation, with one example:

```bash
$ ghosthost info http://homepc.tail-4a9c2e.ts.net:8750/t/k3n.../hello.txt
URL:     http://homepc.tail-4a9c2e.ts.net:8750/t/k3n.../hello.txt
ID:      8f2b1c04
Src:     /home/you/hello.txt
Created: 2026-04-21T13:14:15Z
Expires: 2026-04-22T13:14:15Z (23h59m59s from now)
```

- [ ] **Step 2: Update `CLAUDE.md` §8**

In `CLAUDE.md`, the §8 hello-world proof-of-install currently describes the CLI JSON output. Keep that path (it uses `--json`). But add a short paragraph noting the new default human output:

> **Human-mode output (no `--json`):** `ghosthost share <path>` prints one line: the URL. No id, no expiry, no hints. Use `--verbose` if you want the old rich block, or run `ghosthost info <url-or-id>` to retrieve metadata for an active share.

No other changes — the existing guidance already passes `--json` for structured data, which is unchanged in this PR.

- [ ] **Step 3: Update `skills/ghosthost/SKILL.md`**

Run: `cat skills/ghosthost/SKILL.md` to review current content. Insert or update a "Reading share output" section:

> The `ghosthost share` command in human mode prints exactly one line: the share URL, no metadata. If you need the id (for later `revoke`) or expiry, use `ghosthost share --json` (still emits `SharePayload`) or call `ghosthost info <arg>` after the fact. The `info` command accepts any of: the full URL, just the URL path (`/t/<token>/<name>`), the bare token, or the bare id.

- [ ] **Step 4: Verify the docs are self-consistent**

Run: `grep -rn "ghosthost share" README.md CLAUDE.md skills/ghosthost/SKILL.md | head -40` — scan every mention to confirm no stale rich-output examples remain in the default (human) path.

- [ ] **Step 5: Commit**

```bash
git add README.md CLAUDE.md skills/ghosthost/SKILL.md
git commit -m "docs: default share output is URL only; document info command"
```

---

## Task 9: Final verification

- [ ] **Step 1: Run the full test suite**

Run: `go test ./...`
Expected: all tests pass.

- [ ] **Step 2: Run `go vet` and (if present) `golangci-lint run`**

Run:
```bash
go vet ./...
golangci-lint run 2>/dev/null || true
```
Expected: no output from `go vet`; lint clean if lint is configured.

- [ ] **Step 3: Manual smoke test — minimal share**

Run:
```bash
printf 'hello\n' > /tmp/ghosthost-pr1.txt
ghosthost share /tmp/ghosthost-pr1.txt
```
Expected: one line of output, a bare URL ending in `/ghosthost-pr1.txt`.

- [ ] **Step 4: Manual smoke test — `--verbose` restores rich block**

Run: `ghosthost share --verbose /tmp/ghosthost-pr1.txt`
Expected: three-line block with `URL:`, `ID:`, `Expires:`.

- [ ] **Step 5: Manual smoke test — `info` with each input form**

Capture the URL from the minimal share above. Then:

```bash
URL=$(ghosthost share /tmp/ghosthost-pr1.txt)
ghosthost info "$URL"                               # full URL
ghosthost info "$(echo "$URL" | sed -E 's|^[^/]*//[^/]+||')"  # path only
# also try bare token (extract the token from the URL path segment after /t/)
# and bare id (from `ghosthost list`)
```

Expected: each resolves to an `InfoPayload` block with matching ID/URL.

- [ ] **Step 6: Manual smoke test — `info` on unknown returns non-zero**

Run:
```bash
ghosthost info zzzzzzzz; echo "exit=$?"
```
Expected: "not found" on stderr, `exit=5` (`ExitNotFound`).

- [ ] **Step 7: Clean up + commit any follow-ups**

If manual testing turned up issues, fix them with small commits. Otherwise, nothing to commit.

- [ ] **Step 8: Open the PR**

The branch is ready. Push and open a PR titled "share: minimal default output + `info` subcommand" with the spec link in the body.
