// internal/admin/handler_test.go
package admin

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakeCore struct {
	secret string
}

func (f *fakeCore) Secret() string { return f.secret }
func (f *fakeCore) Share(ShareRequest) (SharePayload, error) {
	return SharePayload{SchemaVersion: SchemaVersion, ID: "i", Token: "t", URL: "u",
		ExpiresAt: time.Unix(0, 0)}, nil
}
func (f *fakeCore) Revoke(id string) error { return nil }
func (f *fakeCore) Reshare(id string) (SharePayload, error) {
	return SharePayload{SchemaVersion: SchemaVersion, ID: id}, nil
}
func (f *fakeCore) Info(query string) (InfoPayload, error) {
	return InfoPayload{
		SharePayload: SharePayload{SchemaVersion: SchemaVersion, ID: "i", Token: "t", URL: "u"},
		SrcPath:      "/abs/path",
	}, nil
}
func (f *fakeCore) List() ListResponse     { return ListResponse{SchemaVersion: SchemaVersion} }
func (f *fakeCore) Status() StatusResponse { return StatusResponse{SchemaVersion: SchemaVersion, PID: 1} }
func (f *fakeCore) Stop()                  {}

func req(method, path string, body interface{}, bearer string) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, path, &buf)
	if bearer != "" {
		r.Header.Set("Authorization", "Bearer "+bearer)
	}
	r.Header.Set("Content-Type", "application/json")
	return r
}

func TestAuth(t *testing.T) {
	core := &fakeCore{secret: "s3cret"}
	h := NewHandler(core)
	cases := []struct {
		bearer string
		want   int
	}{
		{"", http.StatusUnauthorized},
		{"wrong", http.StatusUnauthorized},
		{"s3cret", http.StatusOK},
	}
	for _, c := range cases {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req("GET", "/status", nil, c.bearer))
		if w.Code != c.want {
			t.Errorf("bearer %q: got %d, want %d", c.bearer, w.Code, c.want)
		}
	}
}

func TestShareEndpoint(t *testing.T) {
	core := &fakeCore{secret: "s"}
	h := NewHandler(core)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req("POST", "/share", ShareRequest{SrcPath: `C:\a.mp4`}, "s"))
	if w.Code != http.StatusOK {
		t.Fatalf("code %d body=%s", w.Code, w.Body.String())
	}
	var p SharePayload
	if err := json.NewDecoder(w.Body).Decode(&p); err != nil {
		t.Fatal(err)
	}
	if p.SchemaVersion != "1" {
		t.Errorf("missing schema_version")
	}
}

func TestMalformedBody(t *testing.T) {
	core := &fakeCore{secret: "s"}
	h := NewHandler(core)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("POST", "/share", strings.NewReader("{not json"))
	r.Header.Set("Authorization", "Bearer s")
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("want 400, got %d", w.Code)
	}
}
