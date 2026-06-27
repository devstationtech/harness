package app

import (
	"fmt"
	"io"
	"strings"

	"github.com/devstationtech/harness/internal/artifact"
	"github.com/devstationtech/harness/internal/vendor"
)

// Vendor copies a shared or remote artifact into the project's .agents directory
// so it is committed and overrides the shared one — useful for an open-source
// project that wants specific skills available to contributors without sharing
// the maintainer's whole library. It takes a "<kind>/<name>" argument.
func Vendor(out io.Writer, args []string) error {
	if len(args) < 1 {
		return fmt.Errorf("usage: harness vendor <kind>/<name> (e.g. skill/api-designer)")
	}
	kindText, name, ok := strings.Cut(args[0], "/")
	if !ok || kindText == "" || name == "" {
		return fmt.Errorf("usage: harness vendor <kind>/<name> (e.g. skill/api-designer)")
	}
	kind, ok := artifact.ParseKind(kindText)
	if !ok {
		return fmt.Errorf("unknown kind %q", kindText)
	}

	cat, projectRoot, _, err := loadCatalog()
	if err != nil {
		return err
	}
	resolved, ok := cat.Find(artifact.Identity{Kind: kind, Name: name})
	if !ok {
		return fmt.Errorf("no %s named %q in the catalog", kindText, name)
	}
	if resolved.Source == artifact.SourceLocal {
		fmt.Fprintf(out, "%s/%s is already local.\n", kindText, name)
		return nil
	}

	if _, _, err := vendor.Vendor(resolved, projectRoot); err != nil {
		return err
	}
	fmt.Fprintf(out, "Vendored %s/%s into .agents — it now overrides the shared one.\n", kindText, name)
	fmt.Fprintln(out, "Run `harness` (or `harness apply`) to update AGENTS.md.")
	return nil
}
