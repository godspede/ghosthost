// internal/history/events.go
package history

import "time"

type OpKind string

const (
	OpShare  OpKind = "share"
	OpRevoke OpKind = "revoke"
	OpExpire OpKind = "expire"
)

// Event is the on-disk representation of a history event. Exactly one of
// the op-specific fields is populated, determined by Op.
type Event struct {
	Op        OpKind    `json:"op"`
	ID        string    `json:"id"`
	At        time.Time `json:"at,omitempty"`         // revoke/expire
	TokenHash string    `json:"token_hash,omitempty"` // share (hex)
	Src       string    `json:"src,omitempty"`        // share
	Name      string    `json:"name,omitempty"`       // share
	CreatedAt time.Time `json:"created_at,omitempty"` // share
	ExpiresAt time.Time `json:"expires_at,omitempty"` // share
	Reason    string    `json:"reason,omitempty"`     // expire
}
