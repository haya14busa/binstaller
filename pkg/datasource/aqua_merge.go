package datasource

import "github.com/aquaproj/aqua/v2/pkg/config/registry"

// mergeVersionOverride merges override fields from vo into pkg and returns a new PackageInfo.
func mergeVersionOverride(pkg registry.PackageInfo, vo registry.VersionOverride) registry.PackageInfo {
	merged := pkg

	// Only map fields that exist and are not pointers, or handle pointers with nil checks.
	if vo.Asset != "" {
		merged.Asset = vo.Asset
	}
	if vo.Format != "" {
		merged.Format = vo.Format
	}
	if vo.Files != nil {
		merged.Files = vo.Files
	}
	if vo.SupportedEnvs != nil {
		merged.SupportedEnvs = vo.SupportedEnvs
	}
	if vo.Checksum != nil {
		merged.Checksum = vo.Checksum
	}
	if vo.FormatOverrides != nil {
		merged.FormatOverrides = vo.FormatOverrides
	}
	if vo.Overrides != nil {
		merged.Overrides = vo.Overrides
	}
	// Merge Replacements: vo takes precedence over pkg
	if vo.Replacements != nil {
		if merged.Replacements == nil {
			merged.Replacements = make(map[string]string)
		}
		for k, v := range vo.Replacements {
			merged.Replacements[k] = v
		}
	}
	return merged
}
