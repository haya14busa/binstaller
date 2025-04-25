package datasource

import (
	"context"

	"github.com/haya14busa/goinstaller/pkg/spec"
)

// SourceAdapter defines the interface for generating an InstallSpec
// from various sources like GoReleaser config, GitHub releases, or CLI flags.
type SourceAdapter interface {
	// GenerateInstallSpec generates an InstallSpec using the context provided at construction.
	GenerateInstallSpec(ctx context.Context) (*spec.InstallSpec, error)
}
