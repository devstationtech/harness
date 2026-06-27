// Package compose derives, for a selected set of artifacts, how abstract skills
// bind to the capabilities that implement their contracts — the interface/trait
// composition at the heart of harness's reusable design library.
package compose

import (
	"sort"

	"github.com/devstationtech/harness/internal/artifact"
)

// Binding pairs one contract of an abstract skill with the capability chosen to
// implement it, plus any other selected providers that were shadowed.
type Binding struct {
	Contract   string
	Capability artifact.Identity
	Shadowed   []artifact.Identity
}

// Composition is the result of binding one abstract skill's contracts.
type Composition struct {
	Abstract artifact.Identity
	Bindings []Binding
	Unbound  []string
}

// Complete reports whether every contract was bound.
func (c Composition) Complete() bool { return len(c.Unbound) == 0 }

// Bind computes a composition for every selected abstract skill. A contract is
// bound to a selected capability that implements the abstract and provides the
// contract; when several qualify, the highest precedence (local over shared)
// then the first by name wins and the rest are shadowed; a contract with no
// provider is left unbound.
func Bind(selected []artifact.Artifact) []Composition {
	var abstracts, capabilities []artifact.Artifact
	for _, a := range selected {
		switch {
		case a.IsAbstract():
			abstracts = append(abstracts, a)
		case a.IsCapability():
			capabilities = append(capabilities, a)
		}
	}

	sort.SliceStable(capabilities, func(i, j int) bool {
		if rankI, rankJ := sourceRank(capabilities[i].Source), sourceRank(capabilities[j].Source); rankI != rankJ {
			return rankI < rankJ
		}
		return capabilities[i].Name < capabilities[j].Name
	})
	sort.SliceStable(abstracts, func(i, j int) bool { return abstracts[i].Name < abstracts[j].Name })

	compositions := make([]Composition, 0, len(abstracts))
	for _, abstract := range abstracts {
		compositions = append(compositions, bindContracts(abstract, capabilities))
	}
	return compositions
}

// bindContracts resolves each contract of one abstract skill against the
// capabilities (already in precedence order).
func bindContracts(abstract artifact.Artifact, capabilities []artifact.Artifact) Composition {
	composition := Composition{Abstract: abstract.Identity()}
	for _, contract := range abstract.Contracts {
		var providers []artifact.Artifact
		for _, capability := range capabilities {
			if capability.Implements == abstract.Name && contains(capability.Provides, contract) {
				providers = append(providers, capability)
			}
		}
		if len(providers) == 0 {
			composition.Unbound = append(composition.Unbound, contract)
			continue
		}
		binding := Binding{Contract: contract, Capability: providers[0].Identity()}
		for _, shadowed := range providers[1:] {
			binding.Shadowed = append(binding.Shadowed, shadowed.Identity())
		}
		composition.Bindings = append(composition.Bindings, binding)
	}
	return composition
}

// sourceRank ranks a project-local artifact above a shared or remote one, so a
// local capability wins a contract over a shared provider.
func sourceRank(source artifact.Source) int {
	if source == artifact.SourceLocal {
		return 0
	}
	return 1
}

func contains(values []string, target string) bool {
	for _, v := range values {
		if v == target {
			return true
		}
	}
	return false
}
