# Binstaller Implementation Plan

This plan outlines the steps to implement the generic, config-driven installer architecture based on the design documents:
- `docs/design/generic-installer-architecture.md`
- `docs/design/install-spec-v1.md`

## Phase 1: Core Structure & GoReleaser Adapter

-   [x] Create `pkg/spec` directory.
-   [x] Define `InstallSpec` struct in `pkg/spec/spec.go` based on the CUE definition in `install-spec-v1.md`.
-   [x] Create `pkg/datasource` directory.
-   [x] Define `SourceAdapter` interface in `pkg/datasource/datasource.go`.
-   [x] Create `pkg/datasource/goreleaser.go`.
-   [x] Refactor existing GoReleaser parsing logic (from `source.go`?) into `goreleaserAdapter` implementing `SourceAdapter`.
    -   [x] Implement basic mapping from `goreleaser.config.Project` to `spec.InstallSpec`.
    -   [x] Handle `--name` and `--repo` overrides in `goreleaserAdapter`.
    -   [ ] Implement logic to infer `name` from directory if missing in config.
    -   [ ] Implement logic to infer `repo` from git remote if loading local file.
    -   [ ] Improve `translateTemplate` to handle more GoReleaser template syntax (e.g., common conditionals).
    -   [ ] Ensure format overrides without explicit format strings are handled gracefully or mapped to defaults.
-   [ ] Add basic validation helpers in `pkg/spec`.

## Phase 2: Two-Step Workflow Commands (`init` & `gen`)

-   [x] Create `cmd/binst` directory.
-   [x] Set up basic CLI structure in `cmd/binst/main.go` (using Cobra or similar). (Includes basic logging).
-   [x] Implement `binst init` command:
    -   [x] Add `--source` flag (initially supporting `goreleaser`).
    -   [x] Add `--file` flag for GoReleaser source.
    -   [x] Add `--repo` flag.
    -   [ ] Add `--tag` flag. (Tag is primarily for GitHub source, less relevant for file source).
    -   [x] Add `--output` flag (defaulting to stdout).
    -   [x] Implement logic to call `goreleaserAdapter` and output `InstallSpec` YAML.
    -   [x] Add `--name` flag for name override.
    -   [ ] Add dependencies between flags (e.g., --file required if --source goreleaser and no --repo).
-   [x] Create `internal/shell` directory.
-   [x] Refactor script generation logic (from `shell_godownloader.go`, `shellfn.go`?) into `ScriptBuilder` in `internal/shell`. (Implemented `Generate` function).
    -   [x] Move shell function library to `internal/shell/functions.go`.
    -   [x] Embed shell functions into the generated script.
    -   [x] Ensure generated script performs dynamic OS/Arch detection at runtime.
    -   [x] Ensure generated script performs dynamic Version/Tag resolution at runtime (using `github_release`).
    -   [x] Ensure generated script performs dynamic asset filename resolution at runtime (using `substitute_placeholders`).
    -   [x] Ensure generated script performs dynamic checksum filename resolution at runtime.
    -   [x] Ensure generated script constructs correct download URLs at runtime.
    -   [x] Ensure generated script performs download and checksum verification.
    -   [x] Ensure generated script performs extraction and installation.
    -   [x] Ensure generated script handles `strip_components`.
    -   [x] Ensure generated script handles naming conventions (lowercase/titlecase) for OS/Arch.
    -   [ ] Implement shell logic to apply asset rules (template/ext overrides) based on runtime OS/Arch/Variant.
    -   [ ] Implement shell logic to apply aliases (os_alias, arch_alias) based on runtime OS/Arch.
    -   [ ] Implement shell logic to handle variants.
    -   [ ] Implement shell logic to look up embedded checksums.
    -   [ ] Implement shell logic for attestation verification.
-   [x] Implement `binst gen` command:
    -   [x] Add `--config` flag (defaulting to `.binstaller.yml`).
    -   [x] Add `--output` flag (defaulting to stdout).
    -   [x] Implement logic to read `InstallSpec` from config file.
    -   [x] Implement logic to call `ScriptBuilder` and output the installer script.

## Phase 3: GitHub Probe Adapter

-   [ ] Create `pkg/datasource/github.go`.
-   [ ] Implement `githubProbeAdapter` using GitHub Releases API.
    -   [ ] Handle API authentication (token vs unauthenticated).
    -   [ ] Implement asset inspection and naming heuristics.
    -   [ ] Handle rate limiting.
-   [ ] Update `binst init` command to support `--source github`.

## Phase 4: File & Flags Adapters

-   [ ] Implement `fileAdapter` in `pkg/datasource/file.go` (reads `.binstaller.yml`).
-   [ ] Update `binst init` to support `--source file`.
-   [ ] Implement `flagsAdapter` logic within `binst init` for `--source cli`.
    -   [ ] Add flags like `--asset-pattern`.

## Phase 5: Embed Checksums Command (`embed`)

-   [ ] Create `internal/checksum` directory.
-   [ ] Implement checksum file parsing logic in `internal/checksum`.
-   [ ] Implement `binst embed` command:
    -   [ ] Add `--config` flag.
    -   [ ] Add `--checksum-file` flag.
    -   [ ] Add `--version` flag.
    -   [ ] Add `--algo` flag.
    -   [ ] Add `--output` flag (overwrite input if omitted).
    -   [ ] Implement logic to parse checksums, update `InstallSpec.checksums.embedded_checksums`, and write back the spec.

## Phase 6: Refinement & Documentation

-   [ ] Update script templates in `internal/shell` to support all `InstallSpec` features (variants, embedded checksums, attestation flags, unpack options). (This is covered by the detailed TODOs in Phase 2).
-   [ ] Add comprehensive unit tests for adapters, spec validation, and script generation.
-   [ ] Add integration tests in `e2etest/` covering different sources and options.
-   [ ] Update `README.md` and `docs/usage.md`.
-   [ ] Add examples for various use cases.
-   [ ] Implement `InstallSpec` validation using CUE (optional, requires integrating CUE tooling).

## Phase 7: Install Command (`install`)

-   [ ] Implement `binst install` command:
    -   [ ] Add `--config` flag.
    -   [ ] Add flags corresponding to `init` sources (e.g., `--repo`, `--tag` for GitHub).
    -   [ ] Implement logic to internally run `init` (if needed) + `gen`.
    -   [ ] Implement logic to execute the generated script (via temp file or stdin pipe to `sh`).

## Phase 8: Attestation Implementation

-   [ ] Implement attestation verification logic based on `docs/design/attestation.md` and `docs/design/attestation-implementation.md`.
-   [ ] Integrate attestation flags (`--enable-gh-attestation`, `--require-attestation`, `--gh-attestation-verify-flags`) into `binst gen` and the generated script.
-   [ ] Add tests for attestation features in `e2etest/attestation_test.go`.

## Phase 9: Reproducible Builds Implementation

-   [ ] Implement reproducible build verification logic based on `docs/design/reproducible-builds.md` and `docs/design/reproducible-builds-implementation.md`.
-   [ ] Integrate relevant flags and logic.
-   [ ] Add tests in `e2etest/reproducible_builds_test.go`.