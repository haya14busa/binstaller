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
