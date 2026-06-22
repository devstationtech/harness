// Command harness configures AI-agent harness artifacts (rules, skills, agents)
// across projects: it merges a shared library (~/.harness) with project-local
// artifacts (.agents) and generates AGENTS.md.
package main

import (
	"fmt"
	"os"

	"github.com/devstationtech/harness/internal/app"
)

// version is overridden at build time via -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "harness: "+err.Error())
		os.Exit(1)
	}
}

func run(args []string) error {
	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "", "select":
		return app.Run(os.Stdout)
	case "init":
		return app.Init(os.Stdout)
	case "list", "ls":
		return app.List(os.Stdout)
	case "version", "--version", "-v":
		fmt.Fprintf(os.Stdout, "harness %s\n", version)
		return nil
	case "help", "--help", "-h":
		printUsage(os.Stdout)
		return nil
	default:
		printUsage(os.Stderr)
		return fmt.Errorf("unknown command %q", command)
	}
}

func printUsage(out *os.File) {
	fmt.Fprint(out, `harness — configure AI-agent artifacts across projects

Usage:
  harness            Select artifacts for the current project (interactive)
  harness init       Create and seed the shared library (~/.harness)
  harness list       List the merged catalog as plain text
  harness version    Print the version
  harness help       Show this help

Artifacts live under ~/.harness (shared) and .agents (project-local), using:
  skills/<name>/SKILL.md   rules/<name>/RULE.md   agents/<name>/AGENT.md
`)
}
