// cmd/ghosthost/main.go
package main

import (
	"os"

	"github.com/godspede/ghosthost/internal/cli"
)

func main() {
	os.Exit(cli.Run(os.Args[1:], os.Stdout, os.Stderr))
}
