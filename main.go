// Command harness configures AI-agent harness artifacts (rules, skills, agents)
// across projects: it merges a shared library (~/.harness) with project-local
// artifacts (.agents) and generates AGENTS.md.
package main

import (
	"fmt"
	"os"

	"github.com/devstationtech/harness/internal/app"
	"github.com/devstationtech/harness/internal/selfupdate"
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
	// Remove any binary left aside by a previous Windows self-update (no-op
	// elsewhere).
	selfupdate.CleanupPrevious()

	command := ""
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "", "select":
		return app.Run(os.Stdout, version)
	case "init":
		return app.Init(os.Stdout)
	case "list", "ls":
		return app.List(os.Stdout)
	case "source":
		return app.Source(os.Stdout, args[1:])
	case "update":
		return app.Update(os.Stdout)
	case "search":
		return app.Search(os.Stdout, args[1:])
	case "upgrade":
		return app.Upgrade(os.Stdout)
	case "apply":
		return app.Apply(os.Stdout)
	case "vendor":
		return app.Vendor(os.Stdout, args[1:])
	case "self-update":
		return app.SelfUpdate(os.Stdout, version)
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
  harness source …   Manage artifact sources (add | list | remove)
  harness update     Refresh all sources and rebuild the search index
  harness search Q    Search artifacts across all sources (offline)
  harness upgrade    Re-resolve this project's selections to the latest
  harness apply      Reconcile this project from its committed harness.yaml
  harness vendor K/N Copy a shared/remote artifact into .agents (local override)
  harness self-update Update harness to the latest GitHub release
  harness version    Print the version
  harness help       Show this help

Sources:
  harness source add <git-url> [--name NAME] [--ref REF]
  harness source list
  harness source remove <name>

Artifacts live under ~/.harness (shared), .agents (project-local) and any
configured git source, using:
  skills/<name>/SKILL.md   rules/<name>/RULE.md   agents/<name>/AGENT.md
`)
}
