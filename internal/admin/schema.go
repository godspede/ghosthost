// internal/admin/schema.go
package admin

import "time"

// SchemaVersion is bumped only on breaking changes to admin/JSON responses.
const SchemaVersion = "1"

type SharePayload struct {
	SchemaVersion string    `json:"schema_version"`
	ID            string    `json:"id"`
	Token         string    `json:"token"`
	URL           string    `json:"url"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// InfoPayload augments SharePayload with src_path and created_at, returned by
// GET /info. Token is intentionally included (via embedded SharePayload) because
// /info is on the Bearer-auth-gated admin API, same as /share and /list; callers
// already have the secret and can obtain the token via /list anyway.
type InfoPayload struct {
	SharePayload
	SrcPath   string    `json:"src_path"`
	CreatedAt time.Time `json:"created_at"`
}

type ListEntry struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
	Remaining int64     `json:"remaining_seconds"`
}

type ListResponse struct {
	SchemaVersion string      `json:"schema_version"`
	Shares        []ListEntry `json:"shares"`
}

type StatusResponse struct {
	SchemaVersion string `json:"schema_version"`
	PID           int    `json:"pid"`
	Uptime        int64  `json:"uptime_seconds"`
	Port          int    `json:"port"`
	ActiveCount   int    `json:"active_count"`
	Version       string `json:"version"`
}

type OKResponse struct {
	SchemaVersion string `json:"schema_version"`
	OK            bool   `json:"ok"`
}

type ShareRequest struct {
	SrcPath     string `json:"src_path"`
	DisplayName string `json:"display_name,omitempty"`
	TTLSeconds  int64  `json:"ttl_seconds,omitempty"`
}

type IDRequest struct {
	ID string `json:"id"`
}
