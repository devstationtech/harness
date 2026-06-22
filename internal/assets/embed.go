// Package assets embeds the static resources shipped inside the harness binary:
// the AGENTS.md template and the seed artifacts written into a fresh shared
// library by `harness init`.
package assets

import (
	"embed"
	_ "embed"
)

// AgentsTemplate is the text/template source for generated AGENTS.md files.
//
//go:embed templates/agents.md.tmpl
var AgentsTemplate string

// SeedFS holds the seed artifacts laid out exactly as they should appear under
// the shared library root (skills/, rules/, agents/).
//
//go:embed all:seed
var SeedFS embed.FS

// SeedRoot is the directory prefix inside SeedFS that contains the library tree.
const SeedRoot = "seed"
