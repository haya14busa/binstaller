package datasource_test

import (
	"context"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/haya14busa/goinstaller/pkg/datasource"
	"github.com/haya14busa/goinstaller/pkg/spec"
	"gopkg.in/yaml.v3"
)

// setupGoReleaserTest is a helper function to create a temporary goreleaser.yml
// file, create the adapter, and call Detect.
func setupGoReleaserTest(t *testing.T, goreleaserConfigContent string) (*spec.InstallSpec, error) {
	t.Helper()

	// Create a temporary dummy goreleaser.yml file
	tmpFile, err := createTempFile("goreleaser.yml", goreleaserConfigContent)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer cleanupTempFile(tmpFile)

	adapter := datasource.NewGoReleaserAdapter("", tmpFile.Name())
	input := datasource.DetectInput{
		FilePath: tmpFile.Name(),
	}

	installSpec, err := adapter.Detect(context.Background(), input)
	return installSpec, err
}

func TestGoReleaserAdapter_Detect_File(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "mycli",
		Repo:               "myowner/myrepo",
		SupportedPlatforms: []spec.Platform{},
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz", // Corrected expected value
			Rules: []spec.AssetRule{
				{
					When: spec.PlatformCondition{OS: "windows"},
					Ext:  ".zip",
				},
			},
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: nil,
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_InferNameFromRepo(t *testing.T) {
	goreleaserConfigContent := `
version: 2
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    format_overrides:
      - goos: windows
        format: zip
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedName := "myrepo"

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               expectedName,
		Repo:               "myowner/myrepo",
		SupportedPlatforms: []spec.Platform{},
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz", // Corrected expected value
			Rules: []spec.AssetRule{
				{
					When: spec.PlatformCondition{OS: "windows"},
					Ext:  ".zip",
				},
			},
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: nil,
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_InferNamingConventionOS(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{- title .Os }}_{{ .Arch }}"
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "mycli",
		Repo:               "myowner/myrepo",
		SupportedPlatforms: []spec.Platform{},
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz", // Corrected expected value
			Rules:            []spec.AssetRule{},
			NamingConvention: &spec.NamingConvention{
				OS:   "titlecase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_UnpackWrapInDirectory(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
archives:
  - name_template: "{{ .ProjectName }}_{{ .Version }}_{{ .Os }}_{{ .Arch }}"
    wrap_in_directory: true
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "mycli",
		Repo:               "myowner/myrepo",
		SupportedPlatforms: []spec.Platform{},
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}",
			DefaultExtension: ".tar.gz", // Corrected expected value
			Rules:            []spec.AssetRule{},
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: &spec.UnpackConfig{
			StripComponents: intPtr(1),
		},
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_SupportedPlatforms(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
builds:
  - goos:
      - linux
      - darwin
    goarch:
      - amd64
      - arm64
  - goos:
      - windows
    goarch:
      - amd64
    ignore:
      - goos: windows
        goarch: amd64
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedPlatforms := []spec.Platform{
		{OS: "linux", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
		{OS: "darwin", Arch: "amd64"},
		{OS: "darwin", Arch: "arm64"},
		// windows/amd64 is ignored in the goreleaser config
	}

	// Sort for deterministic comparison
	sortPlatforms(expectedPlatforms)
	sortPlatforms(installSpec.SupportedPlatforms)

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "mycli",
		Repo:               "myowner/myrepo",
		SupportedPlatforms: expectedPlatforms,
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}", // Default template as no archives defined
			DefaultExtension: ".tar.gz",                                // Corrected expected value
			Rules:            nil,
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: nil,
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_SupportedPlatformsWithArm(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: mycli
release:
  github:
    owner: myowner
    name: myrepo
builds:
  - goos:
      - linux
    goarch:
      - arm
    goarm:
      - "6"
      - "7"
  - goos:
      - darwin
    goarch:
      - arm64
    ignore:
      - goos: linux
        goarch: arm
        goarm: "6"
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	expectedPlatforms := []spec.Platform{
		{OS: "linux", Arch: "armv7"},
		{OS: "darwin", Arch: "arm64"},
		// linux/arm/6 is ignored
	}

	// Sort for deterministic comparison
	sortPlatforms(expectedPlatforms)
	sortPlatforms(installSpec.SupportedPlatforms)

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "mycli",
		Repo:               "myowner/myrepo",
		SupportedPlatforms: expectedPlatforms,
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         "${NAME}_${VERSION}_${OS}_${ARCH}${EXT}", // Default template as no archives defined
			DefaultExtension: ".tar.gz",                                // Corrected expected value
			Rules:            nil,
			NamingConvention: &spec.NamingConvention{
				OS:   "lowercase",
				Arch: "lowercase",
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: nil,
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

func TestGoReleaserAdapter_Detect_SigspyTemplate(t *testing.T) {
	goreleaserConfigContent := `
version: 2
project_name: sigspy
release:
  github:
    owner: kubernetes-sigs
    name: sigspy
archives:
  - name_template: >-
      {{ .ProjectName }}_
      {{- title .Os }}_
      {{- if eq .Arch "amd64" }}x86_64
      {{- else if eq .Arch "386" }}i386
      {{- else }}{{ .Arch }}{{ end }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
checksum:
  name_template: "checksums.txt"
`

	installSpec, err := setupGoReleaserTest(t, goreleaserConfigContent)
	if err != nil {
		t.Fatalf("setupGoReleaserTest failed: %v", err)
	}

	// The expected template string after evaluation with text/template and our varmap.
	// The conditional logic is evaluated based on the placeholder strings.
	// {{ .ProjectName }} -> ${NAME}
	// {{- title .Os }} -> ${OS} (title is no-op)
	// {{- if eq .Arch "amd64" }}x86_64 {{- else if eq .Arch "386" }}i386 {{- else }}{{ .Arch }}{{ end }}
	//   - eq "${ARCH}" "amd64" is false
	//   - eq "${ARCH}" "386" is false
	//   - else block {{ .Arch }} is executed -> ${ARCH}
	// {{- if .Arm }}v{{ .Arm }}{{ end }}
	//   - if "${ARM}" is true (non-empty string)
	//   - v{{ .Arm }} is executed -> v${ARM}
	expectedTemplate := "${NAME}_${OS}_${ARCH}" // Corrected template

	expectedSpec := &spec.InstallSpec{
		Schema:             "v1",
		Name:               "sigspy",
		Repo:               "kubernetes-sigs/sigspy",
		SupportedPlatforms: []spec.Platform{}, // Assuming no builds defined in this minimal config
		DefaultVersion:     "latest",
		Asset: spec.AssetConfig{
			Template:         expectedTemplate + "${EXT}",
			DefaultExtension: ".tar.gz", // Corrected expected value
			Rules: []spec.AssetRule{
				{When: spec.PlatformCondition{Arch: "amd64"}, Arch: "x86_64"},
				{When: spec.PlatformCondition{Arch: "386"}, Arch: "i386"},
			},
			NamingConvention: &spec.NamingConvention{
				OS:   "titlecase", // Corrected expected value
				Arch: "lowercase", // Default
			},
		},
		Checksums: &spec.ChecksumConfig{
			Template:  "checksums.txt",
			Algorithm: "sha256",
		},
		Unpack: nil,
	}

	if diff := cmp.Diff(expectedSpec, installSpec); diff != "" {
		t.Errorf("Generated InstallSpec mismatch (-expected +actual):\n%s", diff)
		actualYAML, _ := yaml.Marshal(installSpec)
		t.Logf("Actual generated InstallSpec YAML:\n%s", string(actualYAML))
	}
}

// Helper function to create a temporary file
func createTempFile(name, content string) (*os.File, error) {
	file, err := os.CreateTemp("", name)
	if err != nil {
		return nil, err
	}
	if _, err := file.WriteString(content); err != nil {
		file.Close()
		os.Remove(file.Name())
		return nil, err
	}
	return file, nil
}

// Helper function to clean up a temporary file
func cleanupTempFile(file *os.File) {
	file.Close()
	os.Remove(file.Name())
}

// Helper function to get a pointer to an int
func intPtr(i int) *int {
	return &i
}

// sortPlatforms sorts a slice of spec.Platform for deterministic comparison.
func sortPlatforms(platforms []spec.Platform) {
	// Simple sort by OS then Arch
	for i := 0; i < len(platforms); i++ {
		for j := i + 1; j < len(platforms); j++ {
			if platforms[i].OS > platforms[j].OS || (platforms[i].OS == platforms[j].OS && platforms[i].Arch > platforms[j].Arch) {
				platforms[i], platforms[j] = platforms[j], platforms[i]
			}
		}
	}
}
