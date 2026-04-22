// internal/cli/output.go
package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/godspede/ghosthost/internal/admin"
)

// Format controls how command output is rendered.
type Format int

const (
	Human Format = iota
	JSON
)

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

func printList(w io.Writer, f Format, r admin.ListResponse) {
	if f == JSON {
		_ = json.NewEncoder(w).Encode(r)
		return
	}
	if len(r.Shares) == 0 {
		fmt.Fprintln(w, "(no active shares)")
		return
	}
	for _, e := range r.Shares {
		fmt.Fprintf(w, "%s  %s  %ds left\n  %s\n", e.ID, e.Name, e.Remaining, e.URL)
	}
}

func printStatus(w io.Writer, f Format, r admin.StatusResponse) {
	if f == JSON {
		_ = json.NewEncoder(w).Encode(r)
		return
	}
	fmt.Fprintf(w, "pid=%d port=%d active=%d uptime=%ds version=%s\n",
		r.PID, r.Port, r.ActiveCount, r.Uptime, r.Version)
}

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

func printOK(w io.Writer, f Format) {
	if f == JSON {
		_ = json.NewEncoder(w).Encode(admin.OKResponse{SchemaVersion: admin.SchemaVersion, OK: true})
		return
	}
	fmt.Fprintln(w, "ok")
}
