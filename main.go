package main

import (
	"fmt"
	"os"

	"github.com/hyper0x/cmdguard/internal/msg"
	subcmd "github.com/hyper0x/cmdguard/internal/subcmd"
)

var version = "dev"  // set via -ldflags at build time
var commit = "none"  // set via -ldflags at build time

func main() {
	if len(os.Args) < 2 {
		fmt.Print(msg.MainHelp)
		os.Exit(1)
	}

	sub := os.Args[1]

	switch sub {
	case "help", "--help":
		fmt.Print(msg.MainHelp)
	case "version", "--version":
		printVersion()
	case "init":
		subcmd.RunInit(os.Args[2:])
	case "list":
		subcmd.RunList(os.Args[2:])
	case "undo":
		subcmd.RunUndo(os.Args[2:])
	case "vault":
		subcmd.RunVault(os.Args[2:])
	case "config":
		subcmd.RunConfig(os.Args[2:])
	default:
		// sub is a command name like rm, mv, chmod
		guardCmd := sub
		args := os.Args[2:]
		subcmd.Version = version
		subcmd.Commit = commit
		subcmd.RunGuard(guardCmd, args)
	}
}

func printVersion() {
	if version == "dev" {
		fmt.Printf("cmdguard %s (commit: %s)\n", version, commit)
	} else {
		fmt.Printf("cmdguard %s\n", version)
	}
}
