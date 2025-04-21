package datasource

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/apex/log"
	"github.com/goreleaser/goreleaser/v2/pkg/config"

	// "github.com/goreleaser/goreleaser/v2/pkg/defaults" // Removed
	"github.com/haya14busa/goinstaller/pkg/spec"
	"github.com/pkg/errors"
)

// goreleaserAdapter implements the SourceAdapter interface for GoReleaser config files.
type goreleaserAdapter struct{}

// NewGoReleaserAdapter creates a new adapter for GoReleaser sources.
func NewGoReleaserAdapter() SourceAdapter {
	return &goreleaserAdapter{}
}

// Detect generates an InstallSpec from a GoReleaser configuration file.
// It can load the configuration from a local file path or a GitHub repository.
// It uses input.Flags["name"] and input.Repo as overrides if provided.
func (a *goreleaserAdapter) Detect(ctx context.Context, input DetectInput) (*spec.InstallSpec, error) {
	log.Infof("detecting InstallSpec using goreleaserAdapter")
	log.Debugf("Input - FilePath: %s, Repo: %s, Flags: %+v", input.FilePath, input.Repo, input.Flags)

	// Use input.Repo if provided, otherwise it's empty for local file loading
	repoOverride := input.Repo
	nameOverride := ""
	if input.Flags != nil {
		nameOverride = input.Flags["name"] // Get name override from flags map
	}

	project, sourceInfo, err := loadGoReleaserConfig(repoOverride, input.FilePath, "") // Pass repoOverride
	if err != nil {
		return nil, errors.Wrap(err, "failed to load goreleaser config")
	}

	// Map goreleaser config.Project to spec.InstallSpec, passing overrides
	installSpec, err := mapToGoInstallerSpec(project, sourceInfo, nameOverride, repoOverride)
	if err != nil {
		return nil, errors.Wrap(err, "failed to map goreleaser config to InstallSpec")
	}

	// Apply InstallSpec defaults
	installSpec.SetDefaults()

	log.Infof("successfully detected InstallSpec from goreleaser source: %s", sourceInfo)
	return installSpec, nil
}

// mapToGoInstallerSpec converts a goreleaser config.Project to spec.InstallSpec.
// It applies overrides for name and repo if provided.
func mapToGoInstallerSpec(project *config.Project, sourceInfo, nameOverride, repoOverride string) (*spec.InstallSpec, error) {
	if project == nil {
		return nil, errors.New("goreleaser project config is nil")
	}

	// --- Basic Info ---
	s := &spec.InstallSpec{}

	// Determine Name: Override > project.ProjectName
	if nameOverride != "" {
		s.Name = nameOverride
		log.Debugf("Using name override: %s", s.Name)
	} else if project.ProjectName != "" {
		s.Name = project.ProjectName
	} else {
		log.Warnf("goreleaser project_name missing. Use --name flag.")
	}

	// Determine Repo: Override > release.github > release.gitea
	if repoOverride != "" {
		s.Repo = normalizeRepo(repoOverride) // Normalize the override
		log.Debugf("Using repo override: %s", s.Repo)
	} else if project.Release.GitHub.Owner != "" && project.Release.GitHub.Name != "" {
		s.Repo = fmt.Sprintf("%s/%s", project.Release.GitHub.Owner, project.Release.GitHub.Name)
	} else if project.Release.Gitea.Owner != "" && project.Release.Gitea.Name != "" {
		s.Repo = fmt.Sprintf("%s/%s", project.Release.Gitea.Owner, project.Release.Gitea.Name)
		log.Warnf("detected Gitea repo, using it for spec.Repo")
	} else {
		// TODO: Attempt to infer from git remote if sourceInfo is a local file in a git repo
		log.Warnf("could not determine repository owner/name from goreleaser config or override. Use --repo flag.")
	}

	// --- Checksums ---
	if project.Checksum.NameTemplate != "" {
		checksumTemplate, err := translateTemplate(project.Checksum.NameTemplate)
		if err != nil {
			log.WithError(err).Warnf("Failed to translate checksum template, using raw: %s", project.Checksum.NameTemplate)
			checksumTemplate = project.Checksum.NameTemplate // Fallback to raw
		}
		s.Checksums = &spec.ChecksumConfig{
			Template:  checksumTemplate,
			Algorithm: project.Checksum.Algorithm, // Defaults handled by spec.SetDefaults()
		}
	}

	// --- Archives / Assets / Unpack ---
	if len(project.Archives) > 0 {
		archive := project.Archives[0] // Focus on the first archive

		// Asset Template
		assetTemplate, err := translateTemplate(archive.NameTemplate)
		if err != nil {
			log.WithError(err).Warnf("Failed to translate asset template, using raw: %s", archive.NameTemplate)
			assetTemplate = archive.NameTemplate // Fallback to raw
		}
		s.Asset.Template = assetTemplate

		// Asset Rules (Format Overrides)
		if len(archive.FormatOverrides) > 0 {
			s.Asset.Rules = make([]spec.AssetRule, 0, len(archive.FormatOverrides))
			for _, override := range archive.FormatOverrides {
				ext := formatToExtension(override.Format)
				// Only add rule if it results in a meaningful extension override
				// or explicitly sets format to binary (empty ext)
				if ext != "" || override.Format == "binary" {
					rule := spec.AssetRule{
						When: spec.PlatformCondition{OS: override.Goos},
						Ext:  ext,
					}
					s.Asset.Rules = append(s.Asset.Rules, rule)
				} else {
					log.Warnf("Ignoring format override for os '%s' with unknown format '%s'", override.Goos, override.Format)
				}
			}
		}

		// Unpack Config
		if archive.WrapInDirectory == "true" {
			strip := 1
			s.Unpack = &spec.UnpackConfig{StripComponents: &strip}
		} else {
			// Check if goreleaser has its own strip_components field (it doesn't seem to)
			strip := 0 // Default to 0 if not wrapped
			s.Unpack = &spec.UnpackConfig{StripComponents: &strip}
		}

	} else {
		log.Warnf("no archives found in goreleaser config, asset information may be incomplete")
		s.Asset.Template = "${NAME}_${VERSION}_${OS}_${ARCH}" // A basic default
	}

	// --- Supported Platforms (from Builds) ---
	s.SupportedPlatforms = deriveSupportedPlatforms(project.Builds) // Pass the whole slice

	// TODO: Map NamingConvention based on goreleaser settings? (e.g., archive template analysis)
	// TODO: Map Aliases based on goreleaser settings? (e.g., archive template analysis)
	// TODO: Map Variants? GoReleaser doesn't have a direct concept, might need heuristics or manual spec editing.
	// TODO: Map Attestation? Not directly in goreleaser config.

	log.Infof("initial mapping from goreleaser config complete (source: %s)", sourceInfo)
	return s, nil
}

// formatToExtension converts a goreleaser archive format to a file extension.
func formatToExtension(format string) string {
	switch format {
	case "tar.gz":
		return ".tar.gz"
	case "tgz":
		return ".tgz" // Alias for tar.gz
	case "tar.xz":
		return ".tar.xz"
	case "tar":
		return ".tar"
	case "zip":
		return ".zip"
	case "gz":
		return ".gz" // Less common for archives, but possible
	case "binary":
		return "" // No extension for binary format
	default:
		log.Warnf("unknown goreleaser archive format '%s', assuming no extension", format)
		return ""
	}
}

// deriveSupportedPlatforms generates a list of platforms from goreleaser build configurations.
func deriveSupportedPlatforms(builds []config.Build) []spec.Platform { // Accept slice
	platforms := make(map[string]spec.Platform) // Use map to deduplicate

	for _, build := range builds { // Iterate through all builds
		// Create ignore map for this build
		ignore := make(map[string]bool)
		for _, ignoredBuild := range build.Ignore {
			platformKey := makePlatformKey(ignoredBuild.Goos, ignoredBuild.Goarch, ignoredBuild.Goarm)
			ignore[platformKey] = true
		}

		// Iterate through target platforms for this build
		for _, goos := range build.Goos {
			for _, goarch := range build.Goarch {
				if goarch == "arm" {
					for _, goarm := range build.Goarm {
						platformKey := makePlatformKey(goos, goarch, goarm)
						if !ignore[platformKey] {
							// Map arm version to Arch field directly for simplicity now
							// e.g., linux/arm/6 -> {OS: linux, Arch: arm6}
							// Variant field remains empty as goreleaser doesn't map directly.
							platforms[platformKey] = spec.Platform{OS: goos, Arch: goarch + goarm}
						}
					}
				} else {
					platformKey := makePlatformKey(goos, goarch, "")
					if !ignore[platformKey] {
						// TODO: Handle variants? (e.g., amd64p32) - Not directly supported by goreleaser build targets
						platforms[platformKey] = spec.Platform{OS: goos, Arch: goarch}
					}
				}
			}
		}
	} // End loop through builds

	// Convert map to slice
	result := make([]spec.Platform, 0, len(platforms))
	for _, p := range platforms {
		result = append(result, p)
	}
	// TODO: Sort the result for deterministic output?
	return result
}

// makePlatformKey creates a unique string key for a platform combination.
func makePlatformKey(goos, goarch, goarm string) string {
	key := goos + "/" + goarch
	if goarch == "arm" && goarm != "" {
		key += goarm // Directly append arm version (e.g., linux/arm6, linux/arm7)
	}
	// TODO: Include variant in the key if/when supported
	return key
}

// translateTemplate converts common GoReleaser template variables to ${VAR} format.
// This is a basic implementation and won't handle complex conditionals.
func translateTemplate(tmpl string) (string, error) {
	r := strings.NewReplacer(
		"{{ .ProjectName }}", "${NAME}",
		"{{ .Binary }}", "${NAME}", // Assume Binary maps to spec Name
		"{{ .Version }}", "${VERSION}",
		"{{ .Tag }}", "${TAG}",
		"{{ .Os }}", "${OS}",
		"{{ .Arch }}", "${ARCH}",
		"{{ .Arm }}", "${VARIANT}", // Map Arm to VARIANT (simplification)
		"{{ .Amd64 }}", "${VARIANT}", // Map Amd64 to VARIANT (simplification)
		// Handle title case used in older goreleaser versions/examples
		"{{- title .Os }}", "${OS}",
		"{{- title .Arch }}", "${ARCH}",
		// Handle common conditionals approximately
		// '{{ if eq .Arch "amd64" }}x86_64{{ else }}{{ .Arch }}{{ end }}' -> '${ARCH}' (rely on alias map)
		// '{{ if .Arm }}v{{ .Arm }}{{ end }}' -> '${VARIANT}'
	)
	result := r.Replace(tmpl)

	// Basic check for remaining Go template syntax - very rudimentary
	if strings.Contains(result, "{{") || strings.Contains(result, "}}") {
		log.Warnf("Template '%s' may still contain unhandled Go template syntax after translation: %s", tmpl, result)
		// Return the partially translated string anyway
	}
	return result, nil
}

// --- Helper functions adapted from main.go ---

// loadGoReleaserConfig loads a goreleaser project configuration.
// It tries loading from a GitHub repo first, then falls back to a local file.
// commitHash is currently unused but kept for potential future use.
func loadGoReleaserConfig(repo, file, commitHash string) (project *config.Project, sourceInfo string, err error) {
	// Try loading from GitHub first if repo is provided
	if repo != "" {
		repo = normalizeRepo(repo)
		log.Infof("attempting to load goreleaser config from github repo: %s", repo)
		// Determine the config path within the repo (use file if provided, else default search)
		configPath := file
		if configPath == "" {
			// If file is not specified for the repo, we need to check default locations
			// This requires fetching repo contents or using default names.
			// For now, assume default .goreleaser.yml if path is empty.
			// TODO: Implement proper default config file discovery for GitHub repos.
			configPath = ".goreleaser.yml"
			log.Infof("no specific config file path provided for repo, trying default: %s", configPath)
		}
		project, sourceInfo, err = loadFromGitHub(repo, configPath, commitHash)
		if err == nil {
			log.Infof("successfully loaded config from github: %s", sourceInfo)
			return project, sourceInfo, nil
		}
		log.Warnf("failed to load config from github repo %s (path: %s): %v. Falling back to local file if specified.", repo, configPath, err)
		// Fall through to try local file if repo loading failed *and* a local file is specified
		if file == "" {
			return nil, "", errors.Wrapf(err, "failed to load config from github repo %s and no local file specified", repo)
		}
	}

	// Try loading from local file if file is provided
	if file != "" {
		log.Infof("attempting to load goreleaser config from local file: %s", file)
		project, sourceInfo, err = loadFromFile(file)
		if err == nil {
			log.Infof("successfully loaded config from local file: %s", sourceInfo)
			return project, sourceInfo, nil
		}
		return nil, "", errors.Wrapf(err, "failed to load config from local file %s", file)
	}

	return nil, "", errors.New("neither repository nor file specified for goreleaser config")
}

// loadFromGitHub loads a project configuration from a GitHub repository.
// Adapted from main.go, simplified commit handling for now.
func loadFromGitHub(repo, configPath, specifiedCommitHash string) (*config.Project, string, error) {
	log.Infof("loading config for %s at path %s from github", repo, configPath)

	// TODO: Re-implement commit hash logic if needed, using default branch for now.
	commitHash := "HEAD" // Simplification: Use HEAD for now. Need default branch logic.
	// defaultBranch := getDefaultBranch(repo) // Requires API call
	// commitHash, err := getLatestCommitSHA(repo, defaultBranch) // Requires API call

	// Construct the raw URL
	// TODO: Handle cases where configPath is empty or needs discovery
	if configPath == "" {
		return nil, "", errors.New("config path within repository must be specified")
	}
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, commitHash, configPath)

	log.Infof("fetching config from URL: %s", url)
	resp, err := http.Get(url) // Basic GET, no token handling yet
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to fetch config from %s", url)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("failed to fetch config from %s: status %d", url, resp.StatusCode)
	}

	// Read the content into a buffer first to allow parsing and potential hashing later
	buf := new(bytes.Buffer)
	if _, err := io.Copy(buf, resp.Body); err != nil {
		return nil, "", errors.Wrap(err, "failed to read config content from response body")
	}
	contentBytes := buf.Bytes() // Keep bytes if needed for hashing sourceInfo

	// Parse the content using goreleaser's logic
	project, err := config.LoadReader(bytes.NewReader(contentBytes)) // Pass only the reader
	if err != nil {
		return nil, "", errors.Wrap(err, "failed to parse goreleaser config from github")
	}

	// Create the source info
	// TODO: Use actual commit hash when implemented
	sourceInfo := fmt.Sprintf("%s@%s:%s", repo, commitHash, configPath)
	log.Infof("using source info: %s", sourceInfo)
	return &project, sourceInfo, nil
}

// loadFromFile loads a project configuration from a local file.
// Adapted from main.go.
func loadFromFile(file string) (*config.Project, string, error) {
	log.Infof("loading config from file %q", file)

	// Get absolute path for better context
	absPath, err := filepath.Abs(file)
	if err != nil {
		absPath = file // Fallback
	}

	// Parse the file using goreleaser's logic
	project, err := config.Load(file) // Pass only the file path
	if err != nil {
		return nil, "", errors.Wrapf(err, "failed to parse goreleaser config from file %s", file)
	}

	// Determine sourceInfo (simplified for now)
	// TODO: Re-implement git commit/hash checking if needed for provenance
	sourceInfo := absPath
	log.Infof("using source info: %s", sourceInfo)

	return &project, sourceInfo, nil
}

// normalizeRepo cleans up a repository string.
// Adapted from main.go.
func normalizeRepo(repo string) string {
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")
	repo = strings.Trim(repo, "/")
	return repo
}

// TODO: Add functions for default branch/commit fetching if required later.
// TODO: Add function to map goreleaser fields to AssetConfig rules, naming, aliases.
