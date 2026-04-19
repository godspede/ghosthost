package server

import (
	"net/http"
	"strings"
	"time"

	"github.com/godspede/ghosthost/internal/share"
)

type Lookup interface {
	FindByToken(token string) (*share.Share, bool)
	MarkExpired(id, reason string)
}

func New(l Lookup) http.Handler {
	mux := http.NewServeMux()
	h := &handler{lookup: l, now: time.Now}
	mux.HandleFunc("/t/", h.serveToken)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(404) })
	return mux
}

type handler struct {
	lookup Lookup
	now    func() time.Time
}

func (h *handler) serveToken(w http.ResponseWriter, r *http.Request) {
	parts := strings.SplitN(strings.TrimPrefix(r.URL.Path, "/t/"), "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		w.WriteHeader(404)
		return
	}
	tok := parts[0]

	s, ok := h.lookup.FindByToken(tok)
	if !ok {
		w.WriteHeader(404)
		return
	}
	now := h.now()
	if !s.Active(now) {
		h.lookup.MarkExpired(s.ID, "ttl")
		w.WriteHeader(404)
		return
	}

	f, err := osOpen(s.SrcPath)
	if err != nil {
		h.lookup.MarkExpired(s.ID, "src_missing")
		w.WriteHeader(404)
		return
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		h.lookup.MarkExpired(s.ID, "src_missing")
		w.WriteHeader(404)
		return
	}

	mode := share.Inline
	if r.URL.Query().Get("dl") == "1" {
		mode = share.Attachment
	}
	disp, err := share.ContentDispositionHeader(s.DisplayName, mode)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	if serveVideoWrapperReal(w, r, s) {
		return
	}
	if serveAudioWrapperReal(w, r, s) {
		return
	}

	ct := contentTypeFor(s.DisplayName)
	if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	w.Header().Set("Content-Disposition", disp)
	w.Header().Set("Cache-Control", "private, no-store")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Referrer-Policy", "no-referrer")

	http.ServeContent(w, r, s.DisplayName, fi.ModTime(), f)
}

