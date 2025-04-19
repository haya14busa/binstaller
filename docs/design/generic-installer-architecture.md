---
title: "InstallSpec‑Driven Installer Architecture"
date: "2025-04-19"
author: "goinstaller Team"
version: "0.2.0"
status: "draft"
---

# Generic Config‑Driven Installer Architecture

> 📄 **Document series overview**
> This file lays out the *architecture* of a generic, config‑driven installer
> pipeline.  The concrete specification of the on‑disk schema referred to in
> this document lives in **[InstallSpec v1](install-spec-v1.md)** which should be
> read together with this file.

## 1. Background & Motivation
Today, `goinstaller` only supports reading a GoReleaser YAML (`.goreleaser.yml`) to generate a shell installer script.
Many projects either do not use GoReleaser or have custom asset naming conventions and release workflows.
We need a pluggable, data‑source‑agnostic architecture that—given minimal inputs via CLI flags, manual config, GitHub API, etc.—can generate the same installer logic without being tightly coupled to GoReleaser.

## 2. Goals
- Extract a **pure InstallSpec** that fully describes name, version placeholder, supported platforms, asset templates, checksum/verification settings, etc.
- Define a **SourceAdapter** interface to populate that InstallSpec from any origin.
- Maintain a single **ScriptGenerator** component that transforms InstallSpec → installer code (shell, PowerShell, …).
- Preserve existing GoReleaser flow as one SourceAdapter implementation and default path for backward compatibility.

## 3. High‑Level Architecture

```text
┌──────────────┐    ┌───────────────┐    ┌───────────────────┐
│ SourceAdapter│ ─> │  InstallSpec │ ─> │  ScriptBuilder   │ ─> install.sh
└──────────────┘    └───────────────┘    └───────────────────┘
```

- **SourceAdapter**: interface `Detect(ctx, in DetectInput) (InstallSpec, error)`
  - goreleaserAdapter (existing `.goreleaser.yml`)
  - githubProbeAdapter (GitHub Releases API, asset inspection)
  - flagsAdapter (CLI flags for name, patterns, etc.)
  - fileAdapter (user‑supplied `install-spec.yaml`)
- **InstallSpec**: Go struct/YAML schema that holds:
  - `name`, `repo`, `version` placeholder
  - per‑platform archives (`os`, `arch`, `asset` template, `bin`)
  - checksum definition (file, algorithm, embedded checksums)
  - attestation verification settings
  - unpack options (strip components)
- **ScriptBuilder**: generates installer scripts
  - powered by Go `text/template` (+ sprig)
  - supports POSIX shell & PowerShell; template sets are pluggable
  - injects download, checksum verify, attestation, retry, flags

## 4. Example InstallSpec Schema (YAML)
```yaml
name: mytool
version: 1.2.3
download:
  base_url: https://github.com/OWNER/REPO/releases/download/v{{version}}
assets:
  - os: linux
    arch: amd64
    file: mytool_{{version}}_linux_amd64.tar.gz
    checksum: SHA256SUMS
  - os: darwin
    arch: arm64
    file: mytool_{{version}}_darwin_arm64.tar.gz
verify:
  checksum: true
  attestation: true
  embedded_checksums:  # Pre-verified checksums embedded in the script
    v1.2.3:
      - filename: "mytool_1.2.3_linux_amd64.tar.gz"
        hash: "1234567890abcdef..."
      - filename: "mytool_1.2.3_darwin_arm64.tar.gz"
        hash: "abcdef1234567890..."
templates:
  shell: default_install.sh.tmpl
  powershell: default_install.ps1.tmpl
```

## 5. CLI UX
```bash
# Default (GoReleaser)
goinstaller generate \
  --source goreleaser \
  --file .goreleaser.yml \
  --output install.sh

# GitHub Releases
goinstaller generate \
  --source github \
  --repo owner/repo \
  --tag v1.2.3 \
  --asset-pattern "{{name}}_{{version}}_{{os}}_{{arch}}.tar.gz" \
  --output install.sh

# Manual config
goinstaller generate \
  --source manual \
  --config myconfig.yml \
  --output install.sh

# Pure flags
goinstaller generate \
  --source cli \
  --name mytool \
  --version 0.9.0 \
  --base-url https://example.com/downloads \
  --asset linux/amd64=mytool_{{version}}_linux_amd64.tgz \
  --output install.sh

# With pre-verified checksums
goinstaller generate \
  --source github \
  --repo owner/repo \
  --tag v1.2.3 \
  --pre-verify-checksums \
  --output install.sh
```

## 6. Integration with Existing Code
```
goinstaller/
├── cmd/goinstaller/main.go       # add --source flag & dispatcher
├── internal/
│   ├── datasource/               # new package
│   │   ├── interface.go          # SourceAdapter interface & options
│   │   ├── goreleaser.go         # existing logic moved here
│   │   ├── github.go             # GitHub probe implementation
│   │   ├── flags.go              # flags → InstallSpec
│   │   └── file.go               # install-spec.yaml parser
│   ├── config/                   # InstallSpec struct + YAML schema
│   └── shell/                    # existing generator refactored as ScriptBuilder
│       ├── generator.go
│       └── templates/
└── pkg/
    └── verify/                   # checksum & attestation helpers
```

## 7. Compatibility & Migration
- Default (no `--source`): legacy GoReleaser path
- Existing flags deprecated in favor of explicit `--source`
- Support overrides via CLI and manual YAML

## 8. Embedded Checksums Benefits

The embedded checksums feature provides several significant advantages:

### 8.1 Performance Improvements
- **Reduced HTTP Requests**: Eliminates the need to download separate checksum files during installation, reducing the number of HTTP requests by at least one per installation.
- **Faster Installations**: Installation completes more quickly, especially on slower networks, as there's no need to wait for additional checksum file downloads.
- **Optimized Verification Flow**: When the installer script itself is verified with attestation, the embedded checksums can be trusted implicitly, allowing the binary verification process to be streamlined or even skipped in certain scenarios, further accelerating the installation process.

### 8.2 Reliability Enhancements
- **Offline Installation Support**: Enables completely offline installations once the installer script and binary are downloaded, as no additional network requests for checksum files are needed.
- **Reduced Network Dependency**: Less susceptible to temporary network issues or checksum file server unavailability.
- **Consistent Verification**: Ensures the same checksums are used for verification regardless of network conditions or changes to remote checksum files.

### 8.3 Security Considerations
- **Pre-verified Integrity**: Checksums can be pre-verified by the script generator using `gh attestation verify` or other secure methods before embedding.
- **Tamper Resistance**: Makes it harder for attackers to substitute malicious checksums, as they would need to modify the installer script itself (which could also be signed or verified).
- **Audit Trail**: Provides a clear record of which checksums were used for verification at the time the installer was generated.
- **Trust Chain**: When the installer script is verified with attestation, the embedded checksums inherit this trust, creating a stronger end-to-end security model.

### 8.4 User Experience
- **Simplified Installation**: Users don't need to worry about checksum file availability or format.
- **Consistent Behavior**: Installation process behaves the same way across different environments and network conditions.
- **Transparent Verification**: Users can inspect the embedded checksums in the installer script before running it.

This feature is particularly valuable for enterprise environments with strict security policies, air-gapped systems, or deployments in regions with unreliable internet connectivity.

## 9. Implementation Roadmap
1. Phase 1: Extract InstallSpec & SourceAdapter interface; migrate current GoReleaser code
2. Phase 2: Wire CLI flag parsing to select SourceAdapter
3. Phase 3: Implement GitHub Probe adapter (API calls, naming heuristics)
4. Phase 4: Add File & Flags adapters
5. Phase 5: Update templates, tests, examples, docs
6. Phase 6: Implement embedded checksums feature for offline installation

## 10. Risks & Mitigations
- Naming inference may require pattern flags for edge cases
- GitHub rate limits: support unauthenticated vs token flows
- Template versioning: allow per‑project overrides and locking
- Embedded checksums: ensure proper validation of pre-verified checksums
