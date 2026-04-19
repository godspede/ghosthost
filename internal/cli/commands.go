// internal/cli/commands.go
package cli

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/godspede/ghosthost/internal/admin"
	"github.com/godspede/ghosthost/internal/source"
)

var commands = map[string]subcmd{
	"share":   cmdShare,
	"list":    cmdList,
	"history": cmdHistory,
	"reshare": cmdReshare,
	"revoke":  cmdRevoke,
	"status":  cmdStatus,
	"stop":    cmdStop,
}

func cmdShare(ctx context.Context, args []string, o *globalOpts) int {
	fs := flag.NewFlagSet("share", flag.ContinueOnError)
	fs.SetOutput(o.stderr)
	ttl := fs.Duration("ttl", o.cfg.DefaultTTL, "time-to-live")
	displayName := fs.String("as", "", "display/download name override")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 {
		fmt.Fprintln(o.stderr, "usage: share <path> [--ttl 24h] [--as name]")
		return ExitUsage
	}
	path := fs.Arg(0)
	abs, err := source.Resolve(path)
	if err != nil {
		fmt.Fprintln(o.stderr, "source:", err)
		return ExitSourceBad
	}
	c, err := EnsureDaemon(o.cfg.DataDir, o.cfgPath)
	if err != nil {
		fmt.Fprintln(o.stderr, "daemon:", err)
		return ExitDaemon
	}
	name := *displayName
	if name == "" {
		name = filepath.Base(abs)
	}
	p, err := c.Share(ctx, admin.ShareRequest{SrcPath: abs, DisplayName: name, TTLSeconds: int64(ttl.Seconds())})
	if err != nil {
		fmt.Fprintln(o.stderr, "share:", err)
		return ExitGeneric
	}
	printShare(o.stdout, o.format, p)
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
	printShare(o.stdout, o.format, p)
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
