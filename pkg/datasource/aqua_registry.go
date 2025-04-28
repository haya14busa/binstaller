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

// hasTrueConstraint returns true if the constraint is "" or "true".
func hasTrueConstraint(constraint string) bool {
	return constraint == "" || constraint == "true"
}

// mapToInstallSpec maps a registry.PackageInfo to a *spec.InstallSpec.
func mapToInstallSpec(p registry.PackageInfo) (*spec.InstallSpec, error) {
	installSpec := &spec.InstallSpec{}
	if p.Name != "" {
		installSpec.Name = p.Name
	} else if len(p.Files) > 0 && p.Files[0].Name != "" {
		installSpec.Name = p.Files[0].Name
	}
	if p.RepoOwner != "" && p.RepoName != "" {
		installSpec.Repo = p.RepoOwner + "/" + p.RepoName
	}
	converted, err := ConvertAquaTemplateToInstallSpec(p.Asset)
	if err != nil {
		return nil, err
	}
	installSpec.Asset.Template = converted
	if !strings.HasSuffix(converted, "${EXT}") {
		installSpec.Asset.Template += "${EXT}"
	}
	installSpec.Asset.DefaultExtension = formatToExtension(p.Format)
	installSpec.SupportedPlatforms = convertSupportedEnvs(p.SupportedEnvs)
	if p.Checksum != nil {
		convertedChecksum, err := ConvertAquaTemplateToInstallSpec(p.Checksum.Asset)
		if err != nil {
			return nil, err
		}
		installSpec.Checksums = &spec.ChecksumConfig{
			Template:  convertedChecksum,
			Algorithm: p.Checksum.Algorithm,
		}
	}
	binaries := make([]spec.Binary, 0, len(p.Files))
	for _, f := range p.Files {
		if f.Name != "" {
			path := f.Src
			if path == "" {
				path = f.Name
			} else {
				evaluated, err := ConvertAquaTemplateToInstallSpec(path)
				if err != nil {
					return nil, err
				}
				path = evaluated
			}
			binaries = append(binaries, spec.Binary{Name: f.Name, Path: path})
		}
	}
	if len(binaries) > 0 {
		installSpec.Asset.Binaries = binaries
	}

	// Convert FormatOverrides to Asset.Rules
	for _, fo := range p.FormatOverrides {
		if fo == nil {
			continue
		}
		rule := spec.AssetRule{
			When: spec.PlatformCondition{OS: fo.GOOS},
			Ext:  formatToExtension(fo.Format),
		}
		installSpec.Asset.Rules = append(installSpec.Asset.Rules, rule)
	}
	// Convert Replacements to Asset.Rules
	for k, v := range p.Replacements {
		rule := spec.AssetRule{}
		if isOS(k) {
			rule.When.OS = k
			rule.OS = v
		} else {
			rule.When.Arch = k
			rule.Arch = v
		}
		installSpec.Asset.Rules = append(installSpec.Asset.Rules, rule)
	}

	return installSpec, nil
}

// isOS returns true if the string is a known GOOS value (target OS for Go builds).
// List generated from: go tool dist list -json | jq -r '.[].GOOS' | sort -u (as of 2025-04-28)
// aix, android, darwin, dragonfly, freebsd, illumos, ios, js, linux, netbsd, openbsd, plan9, solaris, wasip1, windows
func isOS(s string) bool {
	switch s {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "illumos", "ios", "js", "linux", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows":
		return true
	}
	return false
}

// GenerateInstallSpec parses the Aqua registry config and returns the first valid InstallSpec for a supported package.
// Currently, only packages of type "github_release" are supported.
// If version overrides are present, the first valid override is returned.
// Returns an error if no valid package is found or if template conversion fails.
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

		// Main package: only if VersionConstraints is empty or "true"
		if hasTrueConstraint(pkg.VersionConstraints) {
			spec, err := mapToInstallSpec(*pkg)
			if err != nil {
				return nil, err
			}
			return spec, nil
		}

		// version_overrides: only those with VersionConstraints "true"
		for _, vo := range pkg.VersionOverrides {
			if hasTrueConstraint(vo.VersionConstraints) {
				// Map override fields onto a copy of pkg, then map to InstallSpec
				override := mergeVersionOverride(*pkg, *vo)
				spec, err := mapToInstallSpec(override)
				if err != nil {
					return nil, err
				}
				return spec, nil
			}
		}
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
