// internal/cli/commands.go
package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/godspede/ghosthost/internal/admin"
	"github.com/godspede/ghosthost/internal/share"
	"github.com/godspede/ghosthost/internal/source"
)

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

	const usageLine = "usage: share <path>... [--ttl 24h] [--as name] [--anon] [--verbose] [--yes]"
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(o.stderr, usageLine)
		return ExitUsage
	}
	paths := fs.Args()
	if len(paths) == 0 {
		fmt.Fprintln(o.stderr, usageLine)
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
		var name string
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
			// Print any URLs already issued so the user can still revoke them, then the error.
			printShares(o.stdout, o.format, payloads, *verbose)
			fmt.Fprintf(o.stderr, "share %s: %v\n", paths[i], err)
			return ExitGeneric
		}
		payloads = append(payloads, p)
	}

	printShares(o.stdout, o.format, payloads, *verbose)
	return ExitOK
}

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

func cmdList(ctx context.Context, args []string, o *globalOpts) int {
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, "daemon:", err)
		return ExitDaemon
	}
	r, err := c.List(ctx)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitGeneric
	}
	printList(o.stdout, o.format, r)
	return ExitOK
}

func cmdHistory(ctx context.Context, args []string, o *globalOpts) int {
	_ = ctx
	path := filepath.Join(o.cfg.DataDir, "history.jsonl")
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitGeneric
	}
	if o.format == JSON {
		o.stdout.Write(b)
		return ExitOK
	}
	fmt.Fprintln(o.stdout, string(b))
	return ExitOK
}

func cmdReshare(ctx context.Context, args []string, o *globalOpts) int {
	if len(args) != 1 {
		fmt.Fprintln(o.stderr, "usage: reshare <id>")
		return ExitUsage
	}
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitDaemon
	}
	p, err := c.Reshare(ctx, args[0])
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitNotFound
	}
	// reshare is always verbose — the user needs the new id to revoke it later.
	printShare(o.stdout, o.format, p, true)
	return ExitOK
}

func cmdRevoke(ctx context.Context, args []string, o *globalOpts) int {
	if len(args) != 1 {
		fmt.Fprintln(o.stderr, "usage: revoke <id>")
		return ExitUsage
	}
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitDaemon
	}
	if err := c.Revoke(ctx, args[0]); err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitNotFound
	}
	printOK(o.stdout, o.format)
	return ExitOK
}

func cmdStatus(ctx context.Context, args []string, o *globalOpts) int {
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitDaemon
	}
	r, err := c.Status(ctx)
	if err != nil {
		fmt.Fprintln(o.stderr, err)
		return ExitGeneric
	}
	printStatus(o.stdout, o.format, r)
	return ExitOK
}

func cmdStop(ctx context.Context, args []string, o *globalOpts) int {
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		printOK(o.stdout, o.format)
		return ExitOK
	}
	_ = c.Stop(ctx)
	printOK(o.stdout, o.format)
	return ExitOK
}
