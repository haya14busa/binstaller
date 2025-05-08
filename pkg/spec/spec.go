package spec

import "strings"

// InstallSpec defines the v1 configuration schema for binstaller.
type InstallSpec struct {
	Schema             string             `yaml:"schema,omitempty"`          // Default: "v1"
	Name               string             `yaml:"name,omitempty"`            // Optiona. Binary name
	Repo               string             `yaml:"repo"`                      // GitHub owner/repo (e.g., "owner/repo")
	DefaultVersion     string             `yaml:"default_version,omitempty"` // Default: "latest"
	DefaultBinDir      string             `yaml:"default_bin_dir,omitempty"` // Default: "${BINSTALLER_BIN} or ${HOME}/.local/bin"
	Asset              AssetConfig        `yaml:"asset"`
	Checksums          *ChecksumConfig    `yaml:"checksums,omitempty"`
	Attestation        *AttestationConfig `yaml:"attestation,omitempty"`
	Unpack             *UnpackConfig      `yaml:"unpack,omitempty"`
	SupportedPlatforms []Platform         `yaml:"supported_platforms,omitempty"`
}

// Platform defines a supported OS/Arch combination.
type Platform struct {
	OS   string `yaml:"os"`
	Arch string `yaml:"arch"`
}

// AssetConfig describes how to construct download URLs and names.
type AssetConfig struct {
	Template         string            `yaml:"template"` // Filename template
	DefaultExtension string            `yaml:"default_extension,omitempty"`
	Binaries         []Binary          `yaml:"binaries,omitempty"` // binary name and path
	Rules            []AssetRule       `yaml:"rules,omitempty"`
	NamingConvention *NamingConvention `yaml:"naming_convention,omitempty"`
	ArchEmulation    *ArchEmulation    `yaml:"arch_emulation,omitempty"`
}

// AssetRule defines overrides for specific platforms.
type AssetRule struct {
	When     PlatformCondition `yaml:"when"`
	Template string            `yaml:"template,omitempty"` // Optional override template
	OS       string            `yaml:"os,omitempty"`       // Optional override OS
	Arch     string            `yaml:"arch,omitempty"`     // Optional override ARCH
	Ext      string            `yaml:"ext,omitempty"`      // Optional override extension
	Binaries []Binary          `yaml:"binaries,omitempty"` // Optional override binary name and path
}

// Binary defines overrides for specific binary namd and path to binary from extracted directory
type Binary struct {
	Name string `yaml:"name"`
	Path string `yaml:"path"`
}

// PlatformCondition specifies conditions for an AssetRule.
type PlatformCondition struct {
	OS   string `yaml:"os,omitempty"`
	Arch string `yaml:"arch,omitempty"`
}

// NamingConvention controls the casing of placeholders.
type NamingConvention struct {
	OS   string `yaml:"os,omitempty"`   // "lowercase" | "titlecase", Default: "lowercase"
	Arch string `yaml:"arch,omitempty"` // "lowercase", Default: "lowercase"
}

// ArchEmulation controls options of arch emulation.
type ArchEmulation struct {
	Rosetta2 bool `yaml:"rosetta2,omitempty"` // If true, use amd64 as ARCH instead of arm64 if Rosetta2 is available
}

// ChecksumConfig defines how to verify checksums.
type ChecksumConfig struct {
	Template          string                        `yaml:"template"`                     // Checksum filename template
	Algorithm         string                        `yaml:"algorithm,omitempty"`          // Default: "sha256"
	EmbeddedChecksums map[string][]EmbeddedChecksum `yaml:"embedded_checksums,omitempty"` // Keyed by version string
}

// EmbeddedChecksum holds pre-verified checksum information.
type EmbeddedChecksum struct {
	Filename  string `yaml:"filename"`            // Asset filename
	Hash      string `yaml:"hash"`                // Checksum hash
	Algorithm string `yaml:"algorithm,omitempty"` // Optional override
}

// AttestationConfig defines settings for attestation verification.
type AttestationConfig struct {
	Enabled     *bool  `yaml:"enabled,omitempty"`      // Default: false
	Require     *bool  `yaml:"require,omitempty"`      // Default: false
	VerifyFlags string `yaml:"verify_flags,omitempty"` // Additional flags for 'gh attestation verify'
}

// UnpackConfig controls how archives are extracted.
type UnpackConfig struct {
	StripComponents *int `yaml:"strip_components,omitempty"` // Default: 0
}

// Default values for pointers
func (s *InstallSpec) SetDefaults() {
	if s.Schema == "" {
		s.Schema = "v1"
	}
	if s.DefaultVersion == "" {
		s.DefaultVersion = "latest"
	}
	if s.DefaultBinDir == "" {
		s.DefaultBinDir = "${BINSTALLER_BIN:-${HOME}/.local/bin}"
	}
	if s.Asset.NamingConvention == nil {
		s.Asset.NamingConvention = &NamingConvention{}
	}
	if s.Asset.NamingConvention.OS == "" {
		s.Asset.NamingConvention.OS = "lowercase"
	}
	if s.Asset.NamingConvention.Arch == "" {
		s.Asset.NamingConvention.Arch = "lowercase"
	}
	if s.Name == "" && s.Repo != "" {
		sp := strings.SplitN(s.Repo, "/", 2)
		if len(sp) == 2 {
			s.Name = sp[1]
		}
	}
	if s.Asset.Binaries == nil && s.Name != "" {
		if s.Asset.DefaultExtension != "" {
			s.Asset.Binaries = []Binary{
				{Name: s.Name, Path: s.Name},
			}
		} else {
			s.Asset.Binaries = []Binary{
				{Name: s.Name, Path: "${ASSET_FILENAME}"},
			}
		}
	}
	if s.Checksums != nil {
		if s.Checksums.Algorithm == "" {
			s.Checksums.Algorithm = "sha256"
		}
		for version, checksums := range s.Checksums.EmbeddedChecksums {
			for i := range checksums {
				if checksums[i].Algorithm == "" {
					checksums[i].Algorithm = s.Checksums.Algorithm
				}
			}
			s.Checksums.EmbeddedChecksums[version] = checksums // Assign back if modified
		}
	}
	if s.Attestation != nil {
		if s.Attestation.Enabled == nil {
			enabled := false
			s.Attestation.Enabled = &enabled
		}
		if s.Attestation.Require == nil {
			require := false
			s.Attestation.Require = &require
		}
	}
}
