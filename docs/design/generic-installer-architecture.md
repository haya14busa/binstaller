---
title: "InstallSpec‑Driven Installer Architecture"
date: "2025-04-19"
author: "goinstaller Team"
version: "0.2.0"
status: "draft"
---

# Generic Config‑Driven Installer Architecture

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
  - checksum definition (file, algorithm)
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

## 8. Implementation Roadmap
1. Phase 1: Extract InstallSpec & SourceAdapter interface; migrate current GoReleaser code
2. Phase 2: Wire CLI flag parsing to select SourceAdapter
3. Phase 3: Implement GitHub Probe adapter (API calls, naming heuristics)
4. Phase 4: Add File & Flags adapters
5. Phase 5: Update templates, tests, examples, docs

## 9. Risks & Mitigations
- Naming inference may require pattern flags for edge cases
- GitHub rate limits: support unauthenticated vs token flows
- Template versioning: allow per‑project overrides and locking