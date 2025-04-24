package spec

// InstallSpec defines the v1 configuration schema for binstaller.
type InstallSpec struct {
	Schema             string             `yaml:"schema,omitempty"`          // Default: "v1"
	Name               string             `yaml:"name"`                      // Binary name
	Repo               string             `yaml:"repo"`                      // GitHub owner/repo (e.g., "owner/repo")
	DefaultVersion     string             `yaml:"default_version,omitempty"` // Default: "latest"
	Variant            *VariantConfig     `yaml:"variant,omitempty"`
	Asset              AssetConfig        `yaml:"asset"`
	Checksums          *ChecksumConfig    `yaml:"checksums,omitempty"`
	Attestation        *AttestationConfig `yaml:"attestation,omitempty"`
	Unpack             *UnpackConfig      `yaml:"unpack,omitempty"`
	SupportedPlatforms []Platform         `yaml:"supported_platforms,omitempty"`
}

// Platform defines a supported OS/Arch/Variant combination.
type Platform struct {
	OS      string `yaml:"os"`
	Arch    string `yaml:"arch"`
	Variant string `yaml:"variant,omitempty"`
}

// VariantConfig handles per-OS/ARCH variants (e.g., gnu vs musl).
type VariantConfig struct {
	Detect  *bool    `yaml:"detect,omitempty"` // Default: true
	Default string   `yaml:"default"`          // Fallback variant
	Choices []string `yaml:"choices,omitempty"`
}

// AssetConfig describes how to construct download URLs and names.
type AssetConfig struct {
	Template         string            `yaml:"template"`                    // Filename template
	DefaultExtension string            `yaml:"default_extension,omitempty"` // Default: ".tar.gz"
	Rules            []AssetRule       `yaml:"rules,omitempty"`
	NamingConvention *NamingConvention `yaml:"naming_convention,omitempty"`
}

// AssetRule defines overrides for specific platforms.
type AssetRule struct {
	When     PlatformCondition `yaml:"when"`
	Template string            `yaml:"template,omitempty"` // Optional override template
	OS       string            `yaml:"os,omitempty"`       // Optional override OS
	Arch     string            `yaml:"arch,omitempty"`     // Optional override ARCH
	Ext      string            `yaml:"ext,omitempty"`      // Optional override extension
}

// PlatformCondition specifies conditions for an AssetRule.
type PlatformCondition struct {
	OS      string `yaml:"os,omitempty"`
	Arch    string `yaml:"arch,omitempty"`
	Variant string `yaml:"variant,omitempty"`
}

// NamingConvention controls the casing of placeholders.
type NamingConvention struct {
	OS   string `yaml:"os,omitempty"`   // "lowercase" | "titlecase", Default: "lowercase"
	Arch string `yaml:"arch,omitempty"` // "lowercase", Default: "lowercase"
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
	if s.Variant != nil {
		if s.Variant.Detect == nil {
			detect := true
			s.Variant.Detect = &detect
		}
	}
	if s.Asset.DefaultExtension == "" {
		s.Asset.DefaultExtension = ".tar.gz"
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
