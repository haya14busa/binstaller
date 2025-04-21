package shell

import (
	"bytes"
	"encoding/json"

	// "errors" // Use pkg/errors instead for wrapping
	"fmt"
	"io"
	"net/http"

	// "runtime" // No longer needed, OS/Arch detection happens in shell
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/haya14busa/goinstaller/pkg/spec"
	"github.com/pkg/errors" // Use pkg/errors for wrapping
)

// templateData holds the data passed to the shell script template execution.
// It only includes static data from the spec.
type templateData struct {
	*spec.InstallSpec              // Embed the original spec for access to fields like Name, Repo, Asset, Checksums, etc.
	BinstallerVersion       string // Version of the binstaller tool generating the script
	SourceInfo              string // Information about the source of the spec (e.g., file path, git commit)
	ShellFunctions          string // The content of the shell function library
	EscapedAssetTemplate    string // Asset template with dollar signs escaped for shell
	EscapedChecksumTemplate string // Checksum template with dollar signs escaped for shell
}

// Generate creates the installer shell script content based on the InstallSpec.
// The generated script will dynamically determine OS, Arch, and Version at runtime.
func Generate(installSpec *spec.InstallSpec) ([]byte, error) {
	if installSpec == nil {
		return nil, errors.New("install spec cannot be nil")
	}
	// Apply spec defaults first - this is still useful for the spec structure itself
	installSpec.SetDefaults()

	// --- Prepare Template Data ---
	// Only pass static data known at generation time, plus the shell functions and escaped templates
	data := templateData{
		InstallSpec:          installSpec,
		BinstallerVersion:    "dev",                                                      // TODO: Get actual version
		SourceInfo:           "binstaller spec",                                          // TODO: Pass source info down if available from adapter
		ShellFunctions:       shellFunctions,                                             // Pass the shell functions as data (from functions.go)
		EscapedAssetTemplate: strings.ReplaceAll(installSpec.Asset.Template, "$", "\\$"), // Escape dollar signs
		EscapedChecksumTemplate: func() string { // Escape checksum template if it exists
			if installSpec.Checksums != nil {
				return strings.ReplaceAll(installSpec.Checksums.Template, "$", "\\$")
			}
			return ""
		}(), // Immediately invoke the helper function
	}

	// --- Prepare Template ---
	// The template now needs to contain the logic for runtime detection and asset resolution
	// It will include {{ .ShellFunctions }} explicitly.
	funcMap := createFuncMap() // Keep helper funcs like default, tolower etc.
	// Remove the raw template helper functions from the funcMap as they are no longer needed
	delete(funcMap, "rawAssetTemplate")
	delete(funcMap, "rawChecksumTemplate")

	tmpl, err := template.New("installer").Funcs(funcMap).Parse(mainScriptTemplate) // Parse only the main template
	if err != nil {
		return nil, errors.Wrap(err, "failed to parse installer template")
	}

	// --- Execute Template ---
	var buf bytes.Buffer
	// Execute the template with the data struct.
	err = tmpl.Execute(&buf, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to execute installer template")
	}

	return buf.Bytes(), nil
}

// assetResolutionResult is no longer needed as resolution happens in shell
/*
type assetResolutionResult struct {
	AssetFilename    string
	AssetURL         string
	ChecksumFilename string // Optional
	ChecksumURL      string // Optional
	ChecksumHash     string // Optional (from embedded)
	StripComponents  int
}
*/

// resolveAssetDetails is no longer needed as resolution happens in shell
/*
func resolveAssetDetails(spec *spec.InstallSpec, osIn, archIn, variantIn, version, tag string) (*assetResolutionResult, error) {
	// ... implementation removed ...
	return nil, errors.New("asset resolution now happens in shell script")
}
*/

// --- Helper Functions ---

// resolveLatestTag fetches the latest release tag name from the GitHub API.
func resolveLatestTag(repo string) (string, error) {
	if repo == "" {
		return "", errors.New("repository cannot be empty to resolve latest tag") // Use errors.New from pkg/errors
	}
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	log.Debugf("Fetching latest release tag from: %s", apiURL)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to create request for GitHub API") // Use errors.Wrap from pkg/errors
	}
	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	// TODO: Add User-Agent?
	// TODO: Handle GITHUB_TOKEN for rate limiting?

	client := &http.Client{Timeout: 30 * time.Second} // Add a reasonable timeout
	resp, err := client.Do(req)
	if err != nil {
		return "", errors.Wrapf(err, "failed to call GitHub API: %s", apiURL) // Use errors.Wrapf from pkg/errors
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try to read body for error message
		bodyBytes, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get latest release from GitHub API (%s): status %d, body: %s", apiURL, resp.StatusCode, string(bodyBytes))
	}

	var releaseInfo struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releaseInfo); err != nil {
		return "", errors.Wrap(err, "failed to decode GitHub API response") // Use errors.Wrap from pkg/errors
	}

	if releaseInfo.TagName == "" {
		return "", fmt.Errorf("no releases found or tag_name missing in GitHub API response for %s", repo)
	}

	return releaseInfo.TagName, nil
}

// applyPlatformMapping applies aliases and naming conventions.
func applyPlatformMapping(assetCfg spec.AssetConfig, osIn, archIn string) (osOut, archOut string) {
	osOut, archOut = osIn, archIn

	// Apply aliases first
	if alias, ok := assetCfg.OSAlias[osOut]; ok {
		osOut = alias
	}
	if alias, ok := assetCfg.ArchAlias[archOut]; ok {
		archOut = alias
	}

	// Apply naming convention
	if assetCfg.NamingConvention != nil {
		switch assetCfg.NamingConvention.OS {
		case "lowercase":
			osOut = strings.ToLower(osOut)
		case "titlecase":
			// Simple title casing, might need refinement for specific cases
			if len(osOut) > 0 {
				osOut = strings.ToUpper(string(osOut[0])) + strings.ToLower(osOut[1:])
			}
		}
		// Arch is always lowercase according to spec v1
		archOut = strings.ToLower(archOut)
	} else {
		// Default is lowercase
		osOut = strings.ToLower(osOut)
		archOut = strings.ToLower(archOut)
	}

	return osOut, archOut
}

// ruleMatches checks if a platform condition matches the target platform.
func ruleMatches(cond spec.PlatformCondition, targetOS, targetArch, targetVariant string) bool {
	if cond.OS != "" && cond.OS != targetOS {
		return false
	}
	if cond.Arch != "" && cond.Arch != targetArch {
		return false
	}
	if cond.Variant != "" && cond.Variant != targetVariant {
		return false
	}
	// If we reached here, all specified conditions matched (or were empty)
	return true
}

// substitutePlaceholders replaces ${VAR} style placeholders in a template string.
func substitutePlaceholders(template string, placeholders map[string]string) (string, error) {
	result := template
	for key, value := range placeholders {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	// Check if any placeholders remain unsubstituted
	if strings.Contains(result, "${") {
		// Find the first remaining placeholder for a better error message
		start := strings.Index(result, "${")
		if start != -1 {
			end := strings.Index(result[start:], "}")
			if end != -1 {
				unsub := result[start : start+end+1]
				return "", fmt.Errorf("unsubstituted placeholder found: %s in template '%s'", unsub, template)
			}
		}
		return "", fmt.Errorf("unsubstituted placeholder found in template '%s'", template)
	}
	return result, nil
}

// removeKnownExtensions tries to remove common archive extensions.
func removeKnownExtensions(filename string) string {
	extensions := []string{".tar.gz", ".tgz", ".tar.xz", ".tar", ".zip", ".gz"}
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return strings.TrimSuffix(filename, ext)
		}
	}
	return filename
}

// inferExtension attempts to guess the file extension.
func inferExtension(filename string) string {
	extensions := []string{".tar.gz", ".tgz", ".tar.xz", ".tar", ".zip", ".gz"}
	for _, ext := range extensions {
		if strings.HasSuffix(filename, ext) {
			return ext
		}
	}
	// Check for simple extensions
	if dotIndex := strings.LastIndex(filename, "."); dotIndex > 0 && dotIndex < len(filename)-1 {
		// Avoid matching dotfiles like .checksums
		if !strings.Contains(filename[dotIndex+1:], "/") {
			return filename[dotIndex:]
		}
	}
	return "" // No extension found or binary
}

// getInstallBinName determines the filename for the installed binary (e.g., adding .exe on windows).
func getInstallBinName(name, targetOS string) string {
	if targetOS == "windows" && !strings.HasSuffix(name, ".exe") {
		return name + ".exe"
	}
	return name
}

// createFuncMap defines the functions available to the Go template.
func createFuncMap() template.FuncMap {
	return template.FuncMap{
		"join":    strings.Join,
		"replace": strings.ReplaceAll,
		"time": func(s string) string {
			return time.Now().UTC().Format(s)
		},
		"tolower": strings.ToLower,
		"toupper": strings.ToUpper,
		"trim":    strings.TrimSpace,
		"version": func() string { // Renamed from binstallerVersion for template clarity
			// TODO: Get binstaller version properly
			return "dev"
		},
		"sourceInfo": func() string {
			// TODO: How to represent source info in the new model? Maybe pass it down?
			return "binstaller spec" // Placeholder
		},
		"default": func(def, val interface{}) interface{} {
			sVal := fmt.Sprintf("%v", val)
			if sVal == "" || sVal == "0" || sVal == "<nil>" || sVal == "false" {
				return def
			}
			return val
		},
		// rawAssetTemplate and rawChecksumTemplate functions are removed as they are no longer needed
	}
}

// mainScriptTemplate is the main body of the installer script.
// It performs runtime detection and resolution.
const mainScriptTemplate = `#!/bin/sh
set -e
# Code generated by binstaller ({{ .BinstallerVersion }}) from {{ .SourceInfo }}. DO NOT EDIT.
#

# --- Configuration from Spec (Embedded via Go Template) ---
SPEC_NAME="{{ .Name }}"
SPEC_REPO="{{ .Repo }}"
SPEC_DEFAULT_VERSION="{{ .DefaultVersion | default "latest" }}"
SPEC_ASSET_TEMPLATE="{{ .EscapedAssetTemplate }}" # Use the escaped template directly
# TODO: Embed Asset Rules, Aliases, Naming Convention as shell variables/functions
# Example (requires more complex Go templating):
# SPEC_OS_ALIAS_darwin="macOS"
# SPEC_ARCH_ALIAS_amd64="x86_64"
SPEC_NAMING_CONVENTION_OS="{{ .Asset.NamingConvention.OS | default "lowercase" }}" # Embed naming convention OS
# SPEC_RULE_1_WHEN_OS="windows"
# SPEC_RULE_1_EXT=".zip"

SPEC_CHECKSUM_TEMPLATE="{{ .EscapedChecksumTemplate }}" # Use the escaped template directly
SPEC_CHECKSUM_ALGO="{{ if .Checksums }}{{ .Checksums.Algorithm | default "sha256" }}{{ else }}sha256{{ end }}"
# TODO: Embed Checksums Map (requires complex Go templating or JSON embedding + jq)
# Example: EMBEDDED_CHECKSUMS_v1_2_3_foo_linux_amd64_tar_gz="hash..."

SPEC_STRIP_COMPONENTS={{ if .Unpack }}{{ .Unpack.StripComponents | default 0 }}{{ else }}0{{ end }}
# TODO: Embed Attestation settings

# --- Shell Function Library ---
{{ .ShellFunctions }}
# --- End Shell Function Library ---


usage() {
  this=$1
  cat <<EOF
$this: download ${SPEC_NAME} from ${SPEC_REPO}

Usage: $this [-b bindir] [-d] [-t tag]
  -b sets bindir or installation directory, Defaults to ./bin
  -d turns on debug logging
  -t specific tag to install (defaults to '${SPEC_DEFAULT_VERSION}')
  # TODO: Add flags for variant selection?
  # TODO: Add flags for attestation?

 Generated by binstaller ({{ .BinstallerVersion }})
  https://github.com/haya14busa/binstaller # TODO: Update URL?

EOF
  exit 2
}

parse_args() {
  # Default values
  BINDIR="./bin"
  TAG="${SPEC_DEFAULT_VERSION}" # Default to spec default version
  # TODO: Set default attestation flags based on spec

  while getopts "b:dt:h?x" arg; do
    case "$arg" in
      b) BINDIR="$OPTARG" ;;
      d) log_set_priority 7 ;; # Use 7 for debug in shlib
      t) TAG="$OPTARG" ;;      # Allow tag override via flag
      h | \?) usage "$0" ;;
      x) set -x ;;
      # TODO: Add attestation/variant flags
    esac
  done
  shift $((OPTIND - 1))
}

# --- Helper functions for runtime resolution ---

# substitute_placeholders replaces ${VAR} style placeholders
# $1: template string
# $2: NAME
# $3: VERSION
# $4: TAG
# $5: OS
# $6: ARCH
# $7: VARIANT
# $8: EXT
substitute_placeholders() {
  _tmpl=$1
  _name=$2
  _version=$3
  _tag=$4
  _os=$5
  _arch=$6
  _variant=$7
  _ext=$8

  log_debug "Substituting template: ${_tmpl}"
  log_debug "  NAME=${_name}"
  log_debug "  VERSION=${_version}"
  log_debug "  TAG=${_tag}"
  log_debug "  OS=${_os}"
  log_debug "  ARCH=${_arch}"
  log_debug "  VARIANT=${_variant}"
  log_debug "  EXT=${_ext}"

  # Use printf and chained sed for substitution. Handle potential slashes in values.
  # Using a separator other than / for sed, e.g., #
  _result=$(printf "%s" "${_tmpl}" | sed \
    -e "s#\${NAME}#${_name}#g" \
    -e "s#\${VERSION}#${_version}#g" \
    -e "s#\${TAG}#${_tag}#g" \
    -e "s#\${OS}#${_os}#g" \
    -e "s#\${ARCH}#${_arch}#g" \
    -e "s#\${VARIANT}#${_variant}#g" \
    -e "s#\${EXT}#${_ext}#g" )

  log_debug "Substitution result: ${_result}"

  # Check for unsubstituted placeholders (using POSIX ERE)
  if printf "%s\n" "${_result}" | grep -q -E '\$\{([A-Za-z0-9_]+)\}'; then
    log_err "Warning: Potential unsubstituted placeholder found in result: ${_result}"
  fi
  printf "%s\n" "${_result}" # Use printf for safer output
}

# apply_platform_mapping applies aliases and naming conventions
# $1: OS (input)
# $2: Arch (input)
# Outputs space-separated OS Arch
apply_platform_mapping() {
  _os_in=$1
  _arch_in=$2
  _os_out=$_os_in
  _arch_out=$_arch_in

  # TODO: Embed Alias maps from spec and apply them here using shell variables
  # Example:
  # _alias_var="SPEC_OS_ALIAS_${_os_out}"
  # if [ -n "${!_alias_var}" ]; then _os_out="${!_alias_var}"; fi
  # _alias_var="SPEC_ARCH_ALIAS_${_arch_out}"
  # if [ -n "${!_alias_var}" ]; then _arch_out="${!_alias_var}"; fi

  # Apply NamingConvention from spec
  case "$SPEC_NAMING_CONVENTION_OS" in
      "lowercase")
          _os_out=$(echo "$_os_out" | tr '[:upper:]' '[:lower:]')
          ;;
      "titlecase")
          # Basic title casing in shell
          _os_out=$(echo "$_os_out" | awk '{print toupper(substr($0,1,1)) tolower(substr($0,2))}')
          ;;
      *)
          # Default to lowercase if convention is unknown or not set
          _os_out=$(echo "$_os_out" | tr '[:upper:]' '[:lower:]')
          ;;
  esac
  # Arch is always lowercase according to spec v1
  _arch_out=$(echo "$_arch_out" | tr '[:upper:]' '[:lower:]')

  echo "$_os_out $_arch_out"
}

# resolve_asset resolves asset details at runtime
# $1: OS
# $2: Arch
# $3: Variant
# $4: Version
# $5: Tag
# Outputs space-separated: AssetFilename AssetExt ChecksumFilename StripComponents
resolve_asset() {
  _os=$1
  _arch=$2
  _variant=$3
  _version=$4
  _tag=$5

  _asset_template="${SPEC_ASSET_TEMPLATE}"
  _checksum_template="${SPEC_CHECKSUM_TEMPLATE}"
  _asset_ext="" # Default
  _strip_components=${SPEC_STRIP_COMPONENTS}

  # TODO: Embed Asset Rules from spec and implement matching logic here
  # Example (requires embedding rules):
  # if [ "$_os" = "$SPEC_RULE_1_WHEN_OS" ]; then _asset_ext="$SPEC_RULE_1_EXT"; fi

  # Determine final extension
  _final_ext=$_asset_ext
  if [ -z "$_final_ext" ]; then
     # Infer from template - basic inference
     case "$_asset_template" in
        *.tar.gz) _final_ext=".tar.gz" ;; *.tgz) _final_ext=".tgz" ;;
        *.zip) _final_ext=".zip" ;; *.tar) _final_ext=".tar" ;;
        *.tar.xz) _final_ext=".tar.xz" ;; *.gz) _final_ext=".gz" ;;
     esac
  fi

  # Substitute placeholders using the resolved values
  _asset_filename=$(substitute_placeholders \
      "${_asset_template}" \
      "${SPEC_NAME}" \
      "${_version}" \
      "${_tag}" \
      "${_os}" \
      "${_arch}" \
      "${_variant}" \
      "${_final_ext}")

  # Ensure final extension is correct
  # TODO: Improve extension handling logic
  _current_ext=""
  case "$_asset_filename" in
     *.tar.gz|*.tgz|*.zip|*.tar) ;; # Already has known extension
     *) # Add extension if needed and determined
        if [ -n "$_final_ext" ]; then
           _asset_filename="${_asset_filename}${_final_ext}"
        fi
     ;;
  esac


  _checksum_filename=""
  if [ -n "$_checksum_template" ]; then
     log_debug "Calling substitute_placeholders for checksum filename"
     log_debug "  Template: ${_checksum_template}"
     _checksum_filename=$(substitute_placeholders \
         "${_checksum_template}" \
         "${SPEC_NAME}" \
         "${_version}" \
         "${_tag}" \
         "${_os}" \
         "${_arch}" \
         "${_variant}" \
         "${_final_ext}") # Use final_ext here too? Or does checksum have its own? Assume final_ext for now.
     log_debug "Resulting checksum filename: ${_checksum_filename}"
  fi

  echo "$_asset_filename $_final_ext $_checksum_filename $_strip_components"
}

# find_embedded_checksum looks for a checksum in embedded data
# $1: Version
# $2: Asset Filename
# Outputs the hash if found, otherwise empty string
find_embedded_checksum() {
  _version=$1
  _asset_filename=$2
  _hash=""
  # TODO: Embed checksum map from spec and implement lookup logic here
  # Example (requires embedding data as shell variables):
  # _key_v="EMBEDDED_CHECKSUMS_v${_version}_${_asset_filename}" # Need to sanitize filename for var name
  # _key="EMBEDDED_CHECKSUMS_${_version}_${_asset_filename}"
  # if [ -n "${!_key_v}" ]; then _hash="${!_key_v}";
  # elif [ -n "${!_key}" ]; then _hash="${!_key}"; fi
  echo "$_hash"
}


# TODO: Adapt verify_attestation function if needed

# --- Main Execution ---
execute() {
  # --- Determine target platform ---
  OS=$(uname_os)
  ARCH=$(uname_arch)
  # TODO: Handle VARIANT detection/selection

  # --- Validate platform ---
  uname_os_check "$OS"
  uname_arch_check "$ARCH"
  # TODO: Check if determined OS/ARCH/VARIANT is in .SupportedPlatforms if provided

  log_info "Detected Platform: ${OS}/${ARCH}" # TODO: Add VARIANT

  # --- Apply Platform Mapping ---
  # TODO: Implement apply_platform_mapping based on embedded spec data
  mapped_platform=$(apply_platform_mapping "$OS" "$ARCH")
  OS_MAPPED=$(echo "$mapped_platform" | cut -d' ' -f1)
  ARCH_MAPPED=$(echo "$mapped_platform" | cut -d' ' -f2)
  log_debug "Mapped Platform: ${OS_MAPPED}/${ARCH_MAPPED}"

  # --- Determine Version ---
  log_info "Selected tag: ${TAG}"
  if [ "$TAG" = "latest" ]; then
    REALTAG=$(github_release "${SPEC_REPO}" "latest") || exit 1
    test -n "$REALTAG" || { log_crit "Could not determine latest tag for ${SPEC_REPO}"; exit 1; }
  else
    # Assume TAG is a valid tag/version string
    REALTAG="$TAG"
  fi
  VERSION=${REALTAG#v} # Strip leading 'v'
  TAG="$REALTAG"       # Use the resolved tag
  log_info "Resolved version: ${VERSION} (tag: ${TAG})"

  # --- Resolve Asset Details ---
  # TODO: Pass variant to resolve_asset when implemented
  resolved_details=$(resolve_asset "$OS_MAPPED" "$ARCH_MAPPED" "" "$VERSION" "$TAG")
  ASSET_FILENAME=$(echo "$resolved_details" | cut -d' ' -f1)
  # ASSET_EXT=$(echo "$resolved_details" | cut -d' ' -f2) # Not currently used directly
  CHECKSUM_FILENAME=$(echo "$resolved_details" | cut -d' ' -f3)
  STRIP_COMPONENTS=$(echo "$resolved_details" | cut -d' ' -f4)

  # --- Construct URLs ---
  # TODO: Make base URL configurable?
  GITHUB_DOWNLOAD="https://github.com/${SPEC_REPO}/releases/download"
  ASSET_URL="${GITHUB_DOWNLOAD}/${TAG}/${ASSET_FILENAME}"
  CHECKSUM_URL=""
  if [ -n "$CHECKSUM_FILENAME" ]; then
    CHECKSUM_URL="${GITHUB_DOWNLOAD}/${TAG}/${CHECKSUM_FILENAME}"
  fi

  log_info "Asset URL: ${ASSET_URL}"
  if [ -n "$CHECKSUM_URL" ]; then
    log_info "Checksum URL: ${CHECKSUM_URL}"
  fi

  # --- Download and Verify ---
  tmpdir=$(mktemp -d)
  log_debug "Downloading files into ${tmpdir}"
  log_info "Downloading ${ASSET_FILENAME} from ${ASSET_URL}"
  http_download "${tmpdir}/${ASSET_FILENAME}" "${ASSET_URL}" || { rm -rf "${tmpdir}"; exit 1; }

  # Checksum Verification
  CHECKSUM_HASH=$(find_embedded_checksum "$VERSION" "$ASSET_FILENAME")
  CHECKSUM_ALGO="${SPEC_CHECKSUM_ALGO}"

  if [ -n "$CHECKSUM_HASH" ]; then
    # Embedded checksum available
    log_info "Verifying embedded checksum (${CHECKSUM_ALGO})..."
    echo "${CHECKSUM_HASH}  ${ASSET_FILENAME}" > "${tmpdir}/embedded_checksum.txt"
    hash_sha256_verify "${tmpdir}/${ASSET_FILENAME}" "${tmpdir}/embedded_checksum.txt" || { rm -rf "${tmpdir}"; exit 1; } # TODO: Use CHECKSUM_ALGO
    rm "${tmpdir}/embedded_checksum.txt"
  elif [ -n "$CHECKSUM_URL" ]; then
    # Download checksum file
    log_info "Downloading checksums from ${CHECKSUM_URL}"
    http_download "${tmpdir}/${CHECKSUM_FILENAME}" "${CHECKSUM_URL}" || { rm -rf "${tmpdir}"; exit 1; }
    log_info "Verifying checksum (${CHECKSUM_ALGO})..."
    hash_sha256_verify "${tmpdir}/${ASSET_FILENAME}" "${tmpdir}/${CHECKSUM_FILENAME}" || { rm -rf "${tmpdir}"; exit 1; } # TODO: Use CHECKSUM_ALGO
  else
    log_info "No checksum URL or embedded hash found, skipping verification."
  fi

  # TODO: Attestation Verification

  # --- Extract and Install ---
  log_info "Extracting ${ASSET_FILENAME}..."
  (cd "${tmpdir}" && untar "${ASSET_FILENAME}" "${STRIP_COMPONENTS}") || { rm -rf "${tmpdir}"; exit 1; }

  # Determine binary name based on spec
  BINARY_NAME="${SPEC_NAME}"
  INSTALL_BIN_NAME="$BINARY_NAME"
  if [ "$OS" = "windows" ]; then
     case "$INSTALL_BIN_NAME" in *.exe) ;; *) INSTALL_BIN_NAME="${INSTALL_BIN_NAME}.exe" ;; esac
  fi

  # Find the binary
  extracted_binary_path=""
  if [ -f "${tmpdir}/${BINARY_NAME}" ]; then
     extracted_binary_path="${tmpdir}/${BINARY_NAME}"
  elif [ "$OS" = "windows" ] && [ -f "${tmpdir}/${BINARY_NAME}.exe" ]; then
     extracted_binary_path="${tmpdir}/${BINARY_NAME}.exe"
  else
     log_debug "Searching for ${BINARY_NAME} (or .exe) in subdirectories..."
     found_path=$(find "${tmpdir}" -name "${BINARY_NAME}" -type f -print -quit)
     if [ -z "$found_path" ] && [ "$OS" = "windows" ]; then
        found_path=$(find "${tmpdir}" -name "${BINARY_NAME}.exe" -type f -print -quit)
     fi
     if [ -n "$found_path" ]; then extracted_binary_path="$found_path"; fi
  fi

  if [ -z "$extracted_binary_path" ]; then
      log_crit "Could not find binary '${BINARY_NAME}' after extraction in ${tmpdir}"
      rm -rf "${tmpdir}"; exit 1
  fi
  log_debug "Found binary at: ${extracted_binary_path}"

  # Install the binary
  install_path="${BINDIR}/${INSTALL_BIN_NAME}"
  log_info "Installing binary to ${install_path}"
  test ! -d "${BINDIR}" && install -d "${BINDIR}"
  install "${extracted_binary_path}" "${install_path}"
  log_info "${SPEC_NAME} installation complete!"

  # --- Cleanup ---
  rm -rf "${tmpdir}"
}

# --- Main Script Logic ---
# Override log prefix
log_prefix() {
	echo "binst"
}

parse_args "$@"
execute
`
