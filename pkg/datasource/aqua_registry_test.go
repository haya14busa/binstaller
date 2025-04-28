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
        src: gh
    supported_envs:
      - linux/amd64
      - darwin/amd64
      - windows/amd64
    checksum:
      type: github_release
      asset: "checksums.txt"
      algorithm: sha256
    format: tar.gz
`

const sampleAquaYAMLChecksumTemplate = `
packages:
  - name: gh
    type: github_release
    repo_owner: cli
    repo_name: cli
    asset: "gh_{{.Version}}_{{.OS}}_{{.Arch}}.tar.gz"
    files:
      - name: gh
        src: gh
    supported_envs:
      - linux/amd64
    checksum:
      type: github_release
      asset: "{{.Asset}}.sha256"
      algorithm: sha256
    format: tar.gz
`

func newTestInstallSpec(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAML))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func newTestInstallSpecChecksumTemplate(t *testing.T) *spec.InstallSpec {
	t.Helper()
	adapter := NewAquaRegistryAdapterFromReader(strings.NewReader(sampleAquaYAMLChecksumTemplate))
	installSpec, err := adapter.GenerateInstallSpec(context.Background())
	if err != nil {
		t.Fatalf("GenerateInstallSpec failed: %v", err)
	}
	return installSpec
}

func TestAquaRegistryAdapter_Name(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "gh"
	if installSpec.Name != want {
		t.Errorf("Name: got %q, want %q", installSpec.Name, want)
	}
}

func TestAquaRegistryAdapter_Repo(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "cli/cli"
	if installSpec.Repo != want {
		t.Errorf("Repo: got %q, want %q", installSpec.Repo, want)
	}
}

func TestAquaRegistryAdapter_AssetTemplate(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := "gh_${TAG}_${OS}_${ARCH}.tar.gz${EXT}"
	if installSpec.Asset.Template != want {
		t.Errorf("Asset.Template: got %q, want %q", installSpec.Asset.Template, want)
	}
}

func TestAquaRegistryAdapter_DefaultExtension(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := ".tar.gz"
	if installSpec.Asset.DefaultExtension != want {
		t.Errorf("Asset.DefaultExtension: got %q, want %q", installSpec.Asset.DefaultExtension, want)
	}
}

func TestAquaRegistryAdapter_SupportedPlatforms(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	want := []spec.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "windows", Arch: "amd64"},
	}
	if len(installSpec.SupportedPlatforms) != len(want) {
		t.Fatalf("SupportedPlatforms: got %d, want %d", len(installSpec.SupportedPlatforms), len(want))
	}
	for i, p := range want {
		if installSpec.SupportedPlatforms[i] != p {
			t.Errorf("SupportedPlatforms[%d]: got %+v, want %+v", i, installSpec.SupportedPlatforms[i], p)
		}
	}
}

func TestAquaRegistryAdapter_Checksums(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	if installSpec.Checksums == nil {
		t.Fatal("Checksums: got nil, want non-nil")
	}
	if installSpec.Checksums.Template != "checksums.txt" {
		t.Errorf("Checksums.Template: got %q, want %q", installSpec.Checksums.Template, "checksums.txt")
	}
	if installSpec.Checksums.Algorithm != "sha256" {
		t.Errorf("Checksums.Algorithm: got %q, want %q", installSpec.Checksums.Algorithm, "sha256")
	}
}

func TestAquaRegistryAdapter_Checksums_TemplateWithAsset(t *testing.T) {
	installSpec := newTestInstallSpecChecksumTemplate(t)
	want := "${ASSET_FILENAME}.sha256"
	if installSpec.Checksums.Template != want {
		t.Errorf("Checksums.Template: got %q, want %q", installSpec.Checksums.Template, want)
	}
}

func TestAquaRegistryAdapter_Binaries(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	binaries := installSpec.Asset.Binaries
	if len(binaries) != 1 || binaries[0].Name != "gh" || binaries[0].Path != "gh" {
		t.Errorf("Asset.Binaries: got %+v, want [{Name: \"gh\", Path: \"gh\"}]", binaries)
	}
}

func TestAquaRegistryAdapter_AssetRules_Empty(t *testing.T) {
	installSpec := newTestInstallSpec(t)
	if len(installSpec.Asset.Rules) != 0 {
		t.Errorf("Asset.Rules: got %+v, want empty", installSpec.Asset.Rules)
	}
}

// Additional tests for FormatOverrides and Replacements can be added with custom YAML samples.
