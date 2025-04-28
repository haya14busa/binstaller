package datasource

import (
	"context"
	"strings"
	"testing"

	"github.com/haya14busa/goinstaller/pkg/spec"
)

const sampleAquaYAML = `
packages:
  - name: gh
    type: github_release
    repo_owner: cli
    repo_name: cli
    asset: "gh_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
    files:
      - name: gh
        src: "{{.AssetWithoutExt}}_bin"
    supported_envs:
      - linux/amd64
    checksum:
      type: github_release
      asset: "{{.AssetWithoutExt}}.sha256"
      algorithm: sha256
    format: tar.gz
`

func newTestInstallSpecWithAssetWithoutExt(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAML))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func TestAquaRegistryAdapter_AssetWithoutExt(t *testing.T) {
	installSpec := newTestInstallSpecWithAssetWithoutExt(t)
	wantChecksum := "gh_${TAG}_${OS}_${ARCH}.tar.gz.sha256"
	if installSpec.Checksums.Template != wantChecksum {
		t.Errorf("Checksums.Template: got %q, want %q", installSpec.Checksums.Template, wantChecksum)
	}
	binaries := installSpec.Asset.Binaries
	wantBinaryPath := "gh_${TAG}_${OS}_${ARCH}.tar.gz_bin"
	if len(binaries) != 1 || binaries[0].Path != wantBinaryPath {
		t.Errorf("Asset.Binaries[0].Path: got %q, want %q", binaries[0].Path, wantBinaryPath)
	}
}
