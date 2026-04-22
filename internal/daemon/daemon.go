// internal/daemon/daemon.go
package daemon

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
	"github.com/godspede/ghosthost/internal/config"
	"github.com/godspede/ghosthost/internal/history"
	"github.com/godspede/ghosthost/internal/share"
	"github.com/godspede/ghosthost/internal/source"
	"github.com/godspede/ghosthost/internal/version"
)

type Clock interface{ Now() time.Time }

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }

type Core struct {
	cfg     config.Config
	secret  string
	hist    *history.Store
	clock   Clock
	started time.Time

	mu      sync.RWMutex
	byToken map[string]*share.Share
	byID    map[string]*share.Share

	onStop func()

	lastAdmin time.Time
}

func NewCore(cfg config.Config, secret string, hist *history.Store, stopFn func()) *Core {
	return &Core{
		cfg:     cfg,
		secret:  secret,
		hist:    hist,
		clock:   realClock{},
		started: time.Now(),
		byToken: map[string]*share.Share{},
		byID:    map[string]*share.Share{},
		onStop:  stopFn,
	}
}

func (c *Core) Secret() string { return c.secret }

func (c *Core) RestoreFromHistory() error {
	events, err := c.hist.Replay()
	if err != nil {
		return err
	}
	type state struct {
		s       share.Share
		revoked bool
		expired bool
	}
	byID := map[string]*state{}
	for _, ev := range events {
		switch ev.Op {
		case history.OpShare:
			d, _ := hex.DecodeString(ev.TokenHash)
			var digest [32]byte
			copy(digest[:], d)
			byID[ev.ID] = &state{s: share.Share{
				ID:          ev.ID,
				TokenDigest: digest,
				SrcPath:     ev.Src,
				DisplayName: ev.Name,
				CreatedAt:   ev.CreatedAt,
				ExpiresAt:   ev.ExpiresAt,
			}}
		case history.OpRevoke:
			if st, ok := byID[ev.ID]; ok {
				st.revoked = true
			}
		case history.OpExpire:
			if st, ok := byID[ev.ID]; ok {
				st.expired = true
			}
		}
	}
	now := c.clock.Now()
	for id, st := range byID {
		if st.revoked || st.expired {
			continue
		}
		if now.Before(st.s.ExpiresAt) {
			_ = c.hist.Append(history.Event{Op: history.OpExpire, ID: id, At: now, Reason: "restart"})
		}
	}
	return nil
}

func (c *Core) Share(req admin.ShareRequest) (admin.SharePayload, error) {
	abs, err := source.Resolve(req.SrcPath)
	if err != nil {
		return admin.SharePayload{}, err
	}
	name := req.DisplayName
	if name == "" {
		name = baseName(abs)
	}
	name, err = share.SanitizeDisplayName(name)
	if err != nil {
		return admin.SharePayload{}, err
	}
	ttl := time.Duration(req.TTLSeconds) * time.Second
	if ttl <= 0 {
		ttl = c.cfg.DefaultTTL
	}

	tok := share.NewToken()
	id := share.NewID()
	now := c.clock.Now()
	s := &share.Share{
		ID:          id,
		Token:       tok,
		TokenDigest: share.Digest(tok),
		SrcPath:     abs,
		DisplayName: name,
		CreatedAt:   now,
		ExpiresAt:   now.Add(ttl),
	}

	if err := c.hist.Append(history.Event{
		Op:        history.OpShare,
		ID:        id,
		TokenHash: hex.EncodeToString(s.TokenDigest[:]),
		Src:       abs,
		Name:      name,
		CreatedAt: now,
		ExpiresAt: s.ExpiresAt,
	}); err != nil {
		return admin.SharePayload{}, fmt.Errorf("history append: %w", err)
	}

	c.mu.Lock()
	c.byToken[tok] = s
	c.byID[id] = s
	c.lastAdmin = now
	c.mu.Unlock()

	return admin.SharePayload{
		SchemaVersion: admin.SchemaVersion,
		ID:            id,
		Token:         tok,
		URL:           c.buildURL(tok, name),
		ExpiresAt:     s.ExpiresAt,
	}, nil
}

func (c *Core) Revoke(id string) error {
	c.mu.Lock()
	s, ok := c.byID[id]
	if !ok {
		c.mu.Unlock()
		return errors.New("unknown id")
	}
	delete(c.byID, id)
	delete(c.byToken, s.Token)
	c.mu.Unlock()
	return c.hist.Append(history.Event{Op: history.OpRevoke, ID: id, At: c.clock.Now()})
}

func (c *Core) Reshare(id string) (admin.SharePayload, error) {
	events, err := c.hist.Replay()
	if err != nil {
		return admin.SharePayload{}, err
	}
	var src, name string
	for _, ev := range events {
		if ev.Op == history.OpShare && ev.ID == id {
			src, name = ev.Src, ev.Name
		}
	}
	if src == "" {
		return admin.SharePayload{}, errors.New("unknown id")
	}
	return c.Share(admin.ShareRequest{SrcPath: src, DisplayName: name})
}

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
		return admin.InfoPayload{}, admin.ErrNotFound
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

func (c *Core) List() admin.ListResponse {
	c.mu.RLock()
	defer c.mu.RUnlock()
	now := c.clock.Now()
	out := admin.ListResponse{SchemaVersion: admin.SchemaVersion}
	for _, s := range c.byID {
		if !s.Active(now) {
			continue
		}
		out.Shares = append(out.Shares, admin.ListEntry{
			ID:        s.ID,
			Name:      s.DisplayName,
			URL:       c.buildURL(s.Token, s.DisplayName),
			ExpiresAt: s.ExpiresAt,
			Remaining: int64(s.ExpiresAt.Sub(now).Seconds()),
		})
	}
	return out
}

func (c *Core) Status() admin.StatusResponse {
	c.mu.RLock()
	n := len(c.byID)
	c.mu.RUnlock()
	return admin.StatusResponse{
		SchemaVersion: admin.SchemaVersion,
		PID:           osGetpid(),
		Uptime:        int64(time.Since(c.started).Seconds()),
		Port:          c.cfg.Port,
		ActiveCount:   n,
		Version:       version.Version,
	}
}

func (c *Core) Stop() { c.onStop() }

func (c *Core) FindByToken(tok string) (*share.Share, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	s, ok := c.byToken[tok]
	if !ok {
		return nil, false
	}
	if !share.EqualDigest(share.Digest(tok), s.TokenDigest) {
		return nil, false
	}
	return s, true
}

func (c *Core) MarkExpired(id, reason string) {
	c.mu.Lock()
	s, ok := c.byID[id]
	if ok {
		delete(c.byID, id)
		delete(c.byToken, s.Token)
	}
	c.mu.Unlock()
	if ok {
		_ = c.hist.Append(history.Event{Op: history.OpExpire, ID: id, At: c.clock.Now(), Reason: reason})
	}
}

func (c *Core) ExpireDue(now time.Time) {
	c.mu.Lock()
	var dueIDs []string
	for id, s := range c.byID {
		if !s.Active(now) {
			dueIDs = append(dueIDs, id)
			delete(c.byID, id)
			delete(c.byToken, s.Token)
		}
	}
	c.mu.Unlock()
	for _, id := range dueIDs {
		_ = c.hist.Append(history.Event{Op: history.OpExpire, ID: id, At: now, Reason: "ttl"})
	}
}

func (c *Core) IdleFor(now time.Time) (time.Duration, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return now.Sub(c.lastAdmin), len(c.byID) == 0
}

func (c *Core) TouchAdmin() {
	c.mu.Lock()
	c.lastAdmin = c.clock.Now()
	c.mu.Unlock()
}

func (c *Core) buildURL(tok, name string) string {
	scheme := "http"
	if c.cfg.TLSCert != "" && c.cfg.TLSKey != "" {
		scheme = "https"
	}
	u := &url.URL{
		Scheme: scheme,
		Host:   fmt.Sprintf("%s:%d", c.cfg.Host, c.cfg.Port),
		Path:   "/t/" + tok + "/" + name,
	}
	return u.String()
}

func baseName(p string) string {
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' || p[i] == '\\' {
			return p[i+1:]
		}
	}
	return p
}
