// internal/cli/cli.go
package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/godspede/ghosthost/internal/config"
	"github.com/godspede/ghosthost/internal/daemon"
)

// Exit codes, documented in README.
const (
	ExitOK        = 0
	ExitGeneric   = 1
	ExitUsage     = 2
	ExitConfig    = 3
	ExitDaemon    = 4
	ExitNotFound  = 5
	ExitSourceBad = 6
)

type subcmd func(ctx context.Context, args []string, opts *globalOpts) int

type globalOpts struct {
	cfgPath string
	format  Format
	stdout  io.Writer
	stderr  io.Writer
	cfg     config.Config
}

// Run dispatches the CLI. argv excludes the program name.
func Run(argv []string, stdout, stderr io.Writer) int {
	opts := &globalOpts{
		cfgPath: config.DefaultConfigPath(),
		stdout:  stdout,
		stderr:  stderr,
	}
	fs := flag.NewFlagSet("ghosthost", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.StringVar(&opts.cfgPath, "config", opts.cfgPath, "path to config.toml")
	jsonFlag := fs.Bool("json", false, "emit JSON output")
	if err := fs.Parse(argv); err != nil {
		return ExitUsage
	}
	if *jsonFlag {
		opts.format = JSON
	}
	rest := fs.Args()
	if len(rest) == 0 {
		fmt.Fprintln(stderr, usage)
		return ExitUsage
	}
	cmd, rest := rest[0], rest[1:]

	if cmd == "daemon" {
		return runDaemon(opts)
	}

	cfg, err := config.Load(opts.cfgPath)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return ExitConfig
	}
	opts.cfg = cfg

	sc, ok := commands[cmd]
	if !ok {
		fmt.Fprintln(stderr, "unknown command:", cmd)
		fmt.Fprintln(stderr, usage)
		return ExitUsage
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return sc(ctx, rest, opts)
}

const usage = `ghosthost — temporary file sharing over HTTP

Commands:
  share <path> [--ttl 24h] [--as name]
  list
  history [--limit N]
  reshare <id>
  revoke <id>
  status
  stop

Global flags:
  --config <path>   override config.toml location
  --json            emit machine-readable JSON`

func runDaemon(opts *globalOpts) int {
	cfg, err := config.Load(opts.cfgPath)
	if err != nil {
		fmt.Fprintln(opts.stderr, err)
		return ExitConfig
	}
	if err := daemon.Run(cfg); err != nil {
		fmt.Fprintln(opts.stderr, err)
		return ExitGeneric
	}
	return ExitOK
}
