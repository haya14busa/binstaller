package datasource

import (
	"context"

	"github.com/haya14busa/goinstaller/pkg/spec"
)

// DetectInput provides the necessary information for a SourceAdapter to detect
// the installation specification. Fields might be optional depending on the adapter.
type DetectInput struct {
	// Path to a local configuration file (e.g., .goreleaser.yml, .binstaller.yml)
	FilePath string
	// GitHub repository owner/name (e.g., "cli/cli")
	Repo string
	// Specific tag or ref to inspect (e.g., "v2.45.0")
	Tag string
	// User-provided asset pattern template
	AssetPattern string
	// Other CLI flags or parameters needed for detection
	Flags map[string]string
}

// SourceAdapter defines the interface for detecting and generating an InstallSpec
// from various sources like GoReleaser config, GitHub releases, or CLI flags.
type SourceAdapter interface {
	// Detect attempts to generate an InstallSpec based on the provided input.
	Detect(ctx context.Context, input DetectInput) (*spec.InstallSpec, error)
}
