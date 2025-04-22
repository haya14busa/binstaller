package shell

import (
	"bytes"

	// "errors" // Use pkg/errors instead for wrapping
	"fmt"

	// "runtime" // No longer needed, OS/Arch detection happens in shell
	"strings"
	"text/template"
	"time"

	"github.com/haya14busa/goinstaller/pkg/spec"
	"github.com/pkg/errors" // Use pkg/errors for wrapping
)

// templateData holds the data passed to the shell script template execution.
// It only includes static data from the spec.
type templateData struct {
	*spec.InstallSpec        // Embed the original spec for access to fields like Name, Repo, Asset, Checksums, etc.
	BinstallerVersion string // Version of the binstaller tool generating the script
	SourceInfo        string // Information about the source of the spec (e.g., file path, git commit)
	ShellFunctions    string // The content of the shell function library
	HashFunctions     string
	UntarFunction     string
}

// Generate creates the installer shell script content based on the InstallSpec.
// The generated script will dynamically determine OS, Arch, and Version at runtime.
func Generate(installSpec *spec.InstallSpec) ([]byte, error) {
	if installSpec == nil {
		return nil, errors.New("install spec cannot be nil")
	}
	// Apply spec defaults first - this is still useful for the spec structure itself
	installSpec.SetDefaults()

	hashFunc := hashSHA256
	if installSpec.Checksums != nil && installSpec.Checksums.Algorithm == "sha1" {
		hashFunc = hashSHA1
	}

	// --- Prepare Template Data ---
	// Only pass static data known at generation time, plus the shell functions
	data := templateData{
		InstallSpec:       installSpec,
		BinstallerVersion: "dev",             // TODO: Get actual version
		SourceInfo:        "binstaller spec", // TODO: Pass source info down if available from adapter
		ShellFunctions:    shellFunctions,
		HashFunctions:     hashFunc,
		UntarFunction:     untar,
	}

	// --- Prepare Template ---
	// The template now needs to contain the logic for runtime detection and asset resolution
	// It will include {{ .ShellFunctions }} explicitly.
	funcMap := createFuncMap() // Keep helper funcs like default, tolower etc.

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
	}
}
