// internal/admin/handler.go
package admin

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"
)

// Core is the subset of the daemon that the admin handler needs.
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

// NewHandler returns the admin HTTP handler.
func NewHandler(c Core) http.Handler {
	mux := http.NewServeMux()
	h := &handler{core: c}
	mux.HandleFunc("/share", h.auth(h.share))
	mux.HandleFunc("/revoke", h.auth(h.revoke))
	mux.HandleFunc("/reshare", h.auth(h.reshare))
	mux.HandleFunc("/list", h.auth(h.list))
	mux.HandleFunc("/status", h.auth(h.status))
	mux.HandleFunc("/stop", h.auth(h.stop))
	return mux
}

type handler struct{ core Core }

func (h *handler) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		want := h.core.Secret()
		got := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if len(got) == 0 || subtle.ConstantTimeCompare([]byte(got), []byte(want)) != 1 {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		next(w, r)
	}
}

func writeJSON(w http.ResponseWriter, code int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (h *handler) share(w http.ResponseWriter, r *http.Request) {
	var req ShareRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p, err := h.core.Share(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) revoke(w http.ResponseWriter, r *http.Request) {
	var req IDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	if err := h.core.Revoke(req.ID); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, OKResponse{SchemaVersion: SchemaVersion, OK: true})
}

func (h *handler) reshare(w http.ResponseWriter, r *http.Request) {
	var req IDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.ID == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	p, err := h.core.Reshare(req.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func (h *handler) list(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.core.List())
}

func (h *handler) status(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, h.core.Status())
}

func (h *handler) stop(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, OKResponse{SchemaVersion: SchemaVersion, OK: true})
	go h.core.Stop()
}
