package datasource

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/aquaproj/aqua/v2/pkg/config/registry"
	"github.com/haya14busa/goinstaller/pkg/spec"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

// AquaRegistryAdapter implements SourceAdapter for Aqua registry YAML files.
type AquaRegistryAdapter struct {
	reader io.Reader // Used for stdin, file, etc.
	repo   string    // Used for GitHub fetch, e.g. "owner/name"
	ref    string    // GitHub ref (commit SHA or "HEAD"), default "HEAD"
}

// NewAquaRegistryAdapterFromReader creates an adapter from an io.Reader (stdin, file, etc.).
func NewAquaRegistryAdapterFromReader(reader io.Reader) SourceAdapter {
	return &AquaRegistryAdapter{reader: reader}
}

// NewAquaRegistryAdapterFromRepo creates an adapter that fetches the registry YAML from GitHub.
// If ref is empty, "HEAD" is used.
func NewAquaRegistryAdapterFromRepo(repo string, ref string) SourceAdapter {
	if ref == "" {
		ref = "HEAD"
	}
	return &AquaRegistryAdapter{repo: repo, ref: ref}
}

// GenerateInstallSpecs parses the registry config and returns InstallSpecs for supported packages.
// GenerateInstallSpec parses the registry config and returns the first InstallSpec for a supported package.
// TODO: Support returning multiple InstallSpecs if needed.
func (a *AquaRegistryAdapter) GenerateInstallSpec(ctx context.Context) (*spec.InstallSpec, error) {
	var r io.Reader
	if a.reader != nil {
		r = a.reader
	} else if a.repo != "" {
		// Fetch from GitHub
		ref := a.ref
		if ref == "" {
			ref = "HEAD"
		}
		url := "https://raw.githubusercontent.com/aquaproj/aqua-registry/" + ref + "/pkgs/" + a.repo + "/registry.yaml"
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return nil, errors.New("failed to fetch registry.yaml from GitHub: " + resp.Status)
		}
		r = resp.Body
	} else {
		return nil, errors.New("no input source provided")
	}

	// Parse YAML into Aqua's official struct
	var regConfig registry.Config
	dec := yaml.NewDecoder(r)
	if err := dec.Decode(&regConfig); err != nil {
		return nil, err
	}

	// Implement mapping/filtering logic from regConfig.Packages to InstallSpec
	for _, pkg := range regConfig.PackageInfos {
		if pkg.Type != "github_release" {
			continue
		}

		// Map fields to InstallSpec
		installSpec := &spec.InstallSpec{}

		// Name: pkg.Name or first files.name
		if pkg.Name != "" {
			installSpec.Name = pkg.Name
		} else if len(pkg.Files) > 0 && pkg.Files[0].Name != "" {
			installSpec.Name = pkg.Files[0].Name
		}

		// Repo: repo_owner/repo_name
		if pkg.RepoOwner != "" && pkg.RepoName != "" {
			installSpec.Repo = pkg.RepoOwner + "/" + pkg.RepoName
		}

		// Asset.Template
		installSpec.Asset.Template = pkg.Asset

		// SupportedPlatforms
		installSpec.SupportedPlatforms = convertSupportedEnvs(pkg.SupportedEnvs)

		if pkg.Checksum != nil {
			installSpec.Checksums = &spec.ChecksumConfig{
				Template:  pkg.Checksum.Asset,
				Algorithm: pkg.Checksum.Algorithm,
			}
		}

		// AssetRule.Binaries: all files.name
		binaries := make([]spec.Binary, 0, len(pkg.Files))
		for _, f := range pkg.Files {
			if f.Name != "" {
				binaries = append(binaries, spec.Binary{Name: f.Name, Path: f.Name})
			}
		}
		if len(binaries) > 0 {
			installSpec.Asset.Rules = append(installSpec.Asset.Rules, spec.AssetRule{
				Binaries: binaries,
			})
		}
		// TODO: Map overrides, format_overrides, and other fields as needed

		return installSpec, nil // Return the first valid InstallSpec
	}

	return nil, errors.New("no valid github_release package found in registry")
}

// convertSupportedEnvs converts registry.SupportedEnvs to []spec.Platform.
func convertSupportedEnvs(envs registry.SupportedEnvs) []spec.Platform {
	var platforms []spec.Platform
	for _, env := range envs {
		parts := strings.SplitN(env, "/", 2)
		if len(parts) == 2 {
			platforms = append(platforms, spec.Platform{OS: parts[0], Arch: parts[1]})
		}
	}
	return platforms
}
