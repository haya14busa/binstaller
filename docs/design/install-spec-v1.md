---
title: "InstallSpec v1 – Unified Installer Schema"
date: "2025-04-19"
author: "haya14busa"
collaborators: ["OpenAI o3"]
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
    os:   go             # go | uname | title
    arch: go             # go | uname

checksums:
  template: "${NAME}-v${VERSION}-checksums.txt"
  algorithm: sha256

unpack:
  strip_components: 1
```

### 3.1 Placeholders recognised in templates

`${NAME}` `${VERSION}` `${OS}` `${ARCH}` `${EXT}` `${VARIANT}`

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

CUE was selected over Protocol Buffers / JSON‑Schema because:

* schema, defaults and validation live in *one* concise file;
* YAML/JSON can be directly imported/merged;
* Go code generation is officially supported;
* no runtime library is required for YAML reading – decoding still happens via
  the generated Go structs.

```cue
// installspect.cue – abridged

InstallSpec: {
  schema?: "v1" | *"v1"
  name:    string
  repo:    =~"[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+"

  default_version?: string | *"latest"

  variant?: {
    detect?:  bool | *true
    default:  string
    choices?: [...string] & >=1
  }

  asset: {
    template: string & =~".*\\${VERSION}.*"

    rules?: [...{
      when: { os?: string, arch?: string, variant?: string }
      template?: string
      ext?:      string
    }]

    os_alias?:   { [string]: string }
    arch_alias?: { [string]: string }

    naming_convention?: {
      os:   "go" | "uname" | "title" | *"go"
      arch: "go" | "uname" | *"go"
    }
  }

  checksums?: {
    template:  string
    algorithm?: "sha256" | "sha512" | *"sha256"
    per_asset?: bool | *false
  }

  unpack?: {
    strip_components?: int | *0
  }
}
```

## 6. Future Work / Open Questions

1. Per‑rule `bin` / `bin_path` overrides – do we need them in v1?
2. Support for non‑GitHub hosts (GitLab, self‑hosted, S3).
3. What should be the default `strip_components` (0 vs 1)?
4. Formalise semantic‑versioning story for the schema itself (`schema: v1`
   vs `schema: v1alpha1`, transition period, feature‑guarding, etc.).

---

*Please leave comments directly on this document; improvements welcome.*
