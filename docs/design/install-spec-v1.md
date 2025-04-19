---
title: "InstallSpec v1 – Unified Installer Schema"
date: "2025-04-19"
author: "haya14busa"
collaborators: ["codex (OpenAI o3)", "GitHub Copilot (Gemini 2.5 Pro, o4-mini)"]
status: "draft"
parent: generic-installer-architecture.md
---

# InstallSpec v1 – Design Document (DRAFT)

This document is **part 2** of the *Generic Config‑Driven Installer* series.  It
defines **InstallSpec v1**, the first public, stable on‑disk schema that
`goinstaller` consumes to generate cross‑platform installer scripts (see
*[Architecture]*](generic-installer-architecture.md) for the high‑level design).

InstallSpec focuses on *what to install*; *how the file was produced*
(GoReleaser, hand‑crafted, Buildkite, …) is out of scope and handled by the
pluggable **Source Adapters** described in the architecture document.

The primary audience is maintainers of CLI tools who wish to publish GitHub
release assets that “just work” with a single, predictable `curl | sh`
one‑liner without constraining their build pipeline to GoReleaser.

## 1. Motivation & Background

`goinstaller` v0 only understands GoReleaser YAML.  That prevents support for
many projects that:

* hand‑craft release assets (Rust, Zig, C/C++ projects, …)
* use different naming conventions (e.g. `macOS` vs `darwin`, `x86_64` vs
  `amd64`)
* ship multiple vendor variants of the *same* OS/ARCH (e.g. `gnu`, `msvc`,
  `musl`)

To unlock those cases we introduce **InstallSpec**, a single document that
describes *what* to download and install.  *Where* the information came from
(GoReleaser YAML, GitHub API probing, CLI flags…) is handled by pluggable
“SourceAdapters” upstream.

## 2. Design Requirements

R1  Single text file (YAML/JSON) that end‑users can also hand‑edit.

R2  Concisely express common patterns; avoid having to enumerate every
    OS/ARCH/variant combination.

R3  Handle naming irregularities (capitalisation, aliases, vendor variants).

R4  Allow runtime auto‑detection *and* explicit override of variants.

R5  Provide machine validation for structure, defaults, enums.

R6  Remain VCS‑friendly (no generated binary blobs inside repo).

R7  Schema must be forward‑compatible: new, unknown fields must be safely
    ignored by an older `goinstaller` binary, while a newer binary can still
    understand old specs without a migration step.

## 3. InstallSpec v1 – High‑level Structure

```yaml
schema: v1                # omitted ⇒ v1
name: gh                  # binary name
repo: cli/cli             # GitHub owner/repo

default_version: latest   # optional fallback tag

variant:
  detect:  true           # runtime heuristic (default true)
  default: gnu            # value when detect fails
  choices: [gnu, msvc, musl]

asset:
  template: "${NAME}-v${VERSION}-${ARCH}-${OS}${EXT}"

  rules:                  # first match wins
    - when: { os: windows }
      ext:  ".zip"       # extension override only
    - when: { os: linux, arch: arm }
      template: "${NAME}-v${VERSION}-arm-unknown-${OS}-${VARIANT}${EXT}"
      ext:  ".tar.gz"

  os_alias:   { darwin: macOS, windows: Windows }
  arch_alias: { amd64: x86_64, arm64: aarch64 }

  naming_convention:      # how uname output is normalised
    os:   lowercase      # lowercase (darwin) | titlecase (Darwin)
    arch: lowercase      # lowercase (amd64, armv6)

checksums:
  template: "${NAME}-v${VERSION}-checksums.txt"
  algorithm: sha256

unpack:
  strip_components: 1
```

### 3.1 Placeholders recognised in templates

`${NAME}` `${VERSION}` `${OS}` `${ARCH}` `${EXT}` `${VARIANT}`

*   `${OS}`: Represents the target operating system. The casing (e.g., `linux` vs `Linux`) depends on `naming_convention.os`.
*   `${ARCH}`: Represents the target architecture. The specific value depends on `naming_convention.arch` and `arch_alias`. It might be a standard Go architecture name (e.g., `amd64`, `arm64`) or a more specific one (e.g., `armv6`) derived from system information and aliases.

Placeholders are replaced *verbatim* after all aliasing and naming‑convention
normalisation has taken place.  They are always replaced as plain strings; no
shell quoting is attempted inside the template – the caller (usually
`goinstaller`) is responsible for quoting paths when executing commands.

### 3.2 Asset resolution flow

1. Canonicalise OS/ARCH according to `naming_convention`.
2. Apply alias maps.
3. Decide `VARIANT` using: CLI flag → auto detection → default.
4. Walk `asset.rules`; first matching `when` wins.
5. Combine `template` & `ext` overrides, then substitute placeholders.

## 4. Worked Example

```yaml
# mycli-installspec.yml (abridged)

name: mycli
repo: acme/mycli

asset:
  template: "${NAME}-v${VERSION}-${OS}-${ARCH}${EXT}"

checksums:
  template: "${NAME}-v${VERSION}-checksums.txt"
```

If a user executes the generated installer on **macOS arm64** requesting
version `v2.3.4`, resolution proceeds as follows:

1. `OS` / `ARCH` normalise to `darwin` / `arm64` (Go convention).
2. No `asset.rules` match; default `.tar.gz` is kept from the global `EXT`.
3. Placeholders are substituted →

   `mycli-v2.3.4-darwin-arm64.tar.gz`

4. The checksum file becomes →

   `mycli-v2.3.4-checksums.txt`

Running on **Windows amd64** yields

`mycli-v2.3.4-windows-amd64.zip` because the extension override rule in Section
3 applies.


## 5. Schema definition (CUE)

```cue
// InstallSpec defines the on-disk schema for the installer configuration.
InstallSpec: {
  // schema version (SemVer): bump major for breaking changes.
  schema?: "v1" | *"v1"

  // name of the binary to install.
  // example: "mytool"
  name:    string

  // GitHub owner/repo containing the releases.
  // Must match '<owner>/<repo>'.
  // example: "cli/cli"
  repo:    =~"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+"

  // optional list of supported OS/ARCH (and variant) combinations
  // if omitted, all detected platforms are attempted and missing assets cause failure.
  // example: [{os: "linux", arch: "amd64"}, {os: "darwin", arch: "arm64"}]
  supported_platforms?: [...{
    // operating system name
    // example: "linux"
    os:      string
    // architecture name
    // example: "amd64"
    arch:    string
    // optional variant for this platform (e.g., "gnu", "musl")
    // if omitted, global variant settings apply
    variant?: string
  }] & >=1
  // example: [{os: "linux", arch: "amd64", variant: "musl"}, {os: "darwin", arch: "arm64"}]

  // default_version tags if not specified at runtime.
  // example: "v1.2.3" or "latest"
  default_version?: string | *"latest"

  // variant handles per-OS/ARCH variants (e.g., gnu vs musl).
  variant?: {
    // enable runtime detection of variant
    // if false, use default.
    detect?:  bool | *true

    // fallback variant when detection fails.
    // example: "gnu"
    default:  string

    // allowed variant values.
    // example: ["gnu", "musl"]
    choices?: [...string] & >=1
  }

  // asset describes how to construct download URLs and names.
  asset: {
    // file name template with placeholders:
    // ${NAME},${VERSION},${OS},${ARCH},${EXT},${VARIANT}
    // example: "${NAME}-v${VERSION}-${OS}-${ARCH}.tar.gz"
    template: string

    // rules for per-platform overrides; first match wins.
    rules?: [...{
      when: { os?: string, arch?: string, variant?: string }
      // optional override template
      template?: string
      // optional override extension
      ext?:      string
    }]

    // map system os names to schema os placeholder
    // example: { darwin: "macOS" }
    os_alias?:   { [string]: string }

    // map system arch names to schema arch placeholder
    // example: { armv6: "armv6l" }
    arch_alias?: { [string]: string }

    // control casing of placeholders
    naming_convention?: {
      // lowercase ("linux") or titlecase ("Linux").
      os:   "lowercase" | "titlecase" | *"lowercase"
      // lowercase only ("amd64", "armv6").
      arch: "lowercase" | *"lowercase"
    }
  }

  // verify checksums or signatures
  checksums?: {
    // name of checksum file
    // example: "${NAME}-v${VERSION}-checksums.txt"
    template:  string

    // supported checksum algorithm (script currently only supports sha256)
    algorithm?: "sha256" | *"sha256"
  }

  // attestation settings (GitHub 'gh attestation verify')
  attestation?: {
    // enable attestation verification feature
    // corresponds to CLI flag --enable-gh-attestation
    enabled?:           bool   | *false
    // require attestation verification to succeed, else fail install
    // corresponds to CLI flag --require-attestation
    require?:           bool   | *false
    // additional flags passed to 'gh attestation verify'
    // corresponds to --gh-attestation-verify-flags
    verify_flags?:      string
  }

  // unpack controls how archives are extracted
  unpack?: {
    // strip leading path components when extracting
    // maps to 'tar --strip-components=<n>'
    strip_components?: int | *0
  }
}
```
