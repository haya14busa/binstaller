package datasource

import (
	"bytes"
	"strings"
	"text/template"
)

// ConvertAquaTemplateToInstallSpec evaluates the Aqua asset template as a Go template,
// replacing variables with InstallSpec placeholders (e.g., .OS → ${OS}, .Version → ${TAG}, .SemVer → ${VERSION}).
// Accepts an optional extraVars map for additional template variables (e.g., AssetWithoutExt).
func ConvertAquaTemplateToInstallSpec(tmpl string, extraVars map[string]string) (string, error) {
	// Map Aqua template variables to InstallSpec placeholders
	varMap := map[string]string{
		"SemVer":  "${VERSION}",
		"Version": "${TAG}",
		"OS":      "${OS}",
		"Arch":    "${ARCH}",
		"Format":  "${EXT}",
		"Asset":   "${ASSET_FILENAME}",
	}
	// Merge extraVars
	for k, v := range extraVars {
		varMap[k] = v
	}

	// Define a function map that ignores any function and just returns the variable placeholder
	funcMap := template.FuncMap{
		"trimV": func(s string) string { return "${VERSION}" },
		"title": func(s string) string { return s },
	}

	// Use a custom template to replace variables with placeholders
	tmplObj, err := template.New("aqua").Option("missingkey=error").Funcs(funcMap).Parse(tmpl)
	if err != nil {
		return "", err
	}

	// The data map provides the placeholder for each variable
	data := make(map[string]string)
	for k, v := range varMap {
		data[k] = v
	}

	var buf bytes.Buffer
	if err := tmplObj.Execute(&buf, data); err != nil {
		return "", err
	}
	result := strings.TrimSpace(buf.String())
	result = strings.ReplaceAll(result, ".${EXT}", "${EXT}")
	return result, nil
}
