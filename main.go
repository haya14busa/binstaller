package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/client9/codegen/shell"
	"github.com/goreleaser/goreleaser/v2/pkg/config"
	"github.com/goreleaser/goreleaser/v2/pkg/context"
	"github.com/goreleaser/goreleaser/v2/pkg/defaults"
	"github.com/pkg/errors"
	kingpin "gopkg.in/alecthomas/kingpin.v2"
)

// nolint: gochecknoglobals
var (
	version = "dev"
	commit  = "none"
	datestr = "unknown"
)

// given a template, and a config, generate shell script.
// TemplateContext extends the config.Project with attestation options and source information
type TemplateContext struct {
	*config.Project
	EnableGHAttestation      bool
	RequireAttestation       bool
	GHAttestationVerifyFlags string
	SourceInfo               string // Information about the source used to generate the script
}

func makeShell(tplsrc string, ctx TemplateContext) ([]byte, error) {
	// if we want to add a timestamp in the templates this
	//  function will generate it
	funcMap := template.FuncMap{
		"join":             strings.Join,
		"platformBinaries": makePlatformBinaries,
		// Keep the timestamp function for backward compatibility
		"timestamp": func() string {
			return time.Now().UTC().Format(time.RFC3339)
		},
		// Add the sourceInfo function that returns the source information
		"sourceInfo": func() string {
			return ctx.SourceInfo
		},
		// Add the version function that returns the goinstaller version
		"version": func() string {
			return version
		},
		"replace": strings.ReplaceAll,
		"time": func(s string) string {
			return time.Now().UTC().Format(s)
		},
		"tolower": strings.ToLower,
		"toupper": strings.ToUpper,
		"trim":    strings.TrimSpace,
		"title": func(s string) string {
			// We intentionally don't use strings.Title or cases.Title here
			// because it can cause issues with template variables like ${OS}.
			// If we transform "OS" to "Os", the shell script will break.
			return s
		},
		"contains": strings.Contains,
		"evaluateNameTemplate": func(nameTemplate string) string {
			result, _ := makeName("", nameTemplate)
			return "NAME=" + result
		},
	}

	out := bytes.Buffer{}
	t, err := template.New("shell").Funcs(funcMap).Parse(tplsrc)
	if err != nil {
		return nil, err
	}
	err = t.Execute(&out, ctx)
	return out.Bytes(), err
}

// makePlatform returns a platform string combining goos, goarch, and goarm.
func makePlatform(goos, goarch, goarm string) string {
	platform := goos + "/" + goarch
	if goarch == "arm" && goarm != "" {
		platform += "v" + goarm
	}
	return platform
}

// makePlatformBinaries returns a map from platforms to a slice of binaries
// built for that platform.
func makePlatformBinaries(ctx TemplateContext) map[string][]string {
	cfg := ctx.Project

	platformBinaries := make(map[string][]string)
	for _, build := range cfg.Builds {
		ignore := make(map[string]bool)
		for _, ignoredBuild := range build.Ignore {
			platform := makePlatform(ignoredBuild.Goos, ignoredBuild.Goarch, ignoredBuild.Goarm)
			ignore[platform] = true
		}
		for _, goos := range build.Goos {
			for _, goarch := range build.Goarch {
				switch goarch {
				case "arm":
					for _, goarm := range build.Goarm {
						platform := makePlatform(goos, goarch, goarm)
						if !ignore[platform] {
							platformBinaries[platform] = append(platformBinaries[platform], build.Binary)
						}
					}
				default:
					platform := makePlatform(goos, goarch, "")
					if !ignore[platform] {
						platformBinaries[platform] = append(platformBinaries[platform], build.Binary)
					}
				}
			}
		}
	}
	return platformBinaries
}

// converts the given name template to it's equivalent in shell
// except for the default goreleaser templates, templates with
// conditionals will return an error
//
// {{ .Binary }} --->  [prefix]${BINARY}, etc.
func makeName(prefix, target string) (string, error) {
	// armv6 is the default in the shell script
	// so do not need special template condition for ARM
	armversion := "{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
	target = strings.ReplaceAll(target, armversion, "{{ .Arch }}")

	// hack for https://github.com/goreleaser/godownloader/issues/70
	armversion = "{{ .Arch }}{{ if .Arm }}{{ .Arm }}{{ end }}"
	target = strings.ReplaceAll(target, armversion, "{{ .Arch }}")

	target = strings.ReplaceAll(target, "{{.Arm}}", "{{ .Arch }}")
	target = strings.ReplaceAll(target, "{{ .Arm }}", "{{ .Arch }}")

	// We used to check for conditionals here and return an error if found,
	// but that prevented templates with conditionals from working.
	// By removing this check, we allow the template engine to try to process
	// conditionals. This might not always work correctly, but it's better
	// than failing outright. We provide empty defaults for Arm, Mips, and Amd64
	// in the varmap to avoid "<no value>" errors when these fields are used
	// in conditionals.

	varmap := map[string]string{
		"Os":          "${OS}",
		"OS":          "${OS}",
		"Arch":        "${ARCH}",
		"Version":     "${VERSION}",
		"Tag":         "${TAG}",
		"Binary":      "${BINARY}",
		"ProjectName": "${PROJECT_NAME}",
		"Arm":         "",
		"Mips":        "",
		"Amd64":       "",
	}

	out := bytes.Buffer{}
	if _, err := out.WriteString(prefix); err != nil {
		return "", err
	}

	// Create a function map for the name template
	funcMap := template.FuncMap{
		"title": func(s string) string {
			// We intentionally don't use strings.Title or cases.Title here
			// because it can cause issues with template variables like ${OS}.
			// If we transform "OS" to "Os", the shell script will break.
			return s
		},
		"tolower": strings.ToLower,
		"toupper": strings.ToUpper,
		"trim":    strings.TrimSpace,
	}

	t, err := template.New("name").Funcs(funcMap).Parse(target)
	if err != nil {
		return "", err
	}
	err = t.Execute(&out, varmap)
	return out.String(), err
}

// returns the owner/name repo from input
//
// see https://github.com/goreleaser/godownloader/issues/55
func normalizeRepo(repo string) string {
	// handle full or partial URLs
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimPrefix(repo, "http://github.com/")
	repo = strings.TrimPrefix(repo, "github.com/")

	// hande /name/repo or name/repo/ cases
	repo = strings.Trim(repo, "/")

	return repo
}

// getDefaultBranch gets the default branch for a GitHub repository.
// If it fails, it returns "master" as a fallback.
func getDefaultBranch(repo string) string {
	url := fmt.Sprintf("https://api.github.com/repos/%s", repo)
	log.Infof("getting default branch for %s", repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Warnf("failed to create request for %s: %v", repo, err)
		return "master"
	}

	// Use GITHUB_TOKEN if available to avoid rate limiting
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
		log.Infof("using GITHUB_TOKEN for API request")
	}

	// nolint: gosec
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warnf("failed to get default branch for %s: %v", repo, err)
		return "master"
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("failed to get default branch for %s: %d %s", repo, resp.StatusCode, http.StatusText(resp.StatusCode))
		return "master"
	}

	var repoInfo struct {
		DefaultBranch string `json:"default_branch"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		log.Warnf("failed to decode response for %s: %v", repo, err)
		return "master"
	}

	if repoInfo.DefaultBranch == "" {
		log.Warnf("default branch for %s is empty, using master", repo)
		return "master"
	}

	log.Infof("default branch for %s is %s", repo, repoInfo.DefaultBranch)
	return repoInfo.DefaultBranch
}

// getLatestCommitSHA gets the SHA of the latest commit on the specified branch
func getLatestCommitSHA(repo, branch string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, branch)
	log.Infof("getting latest commit SHA for %s on branch %s", repo, branch)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Warnf("failed to create request for %s: %v", repo, err)
		return "", err
	}

	// Use GITHUB_TOKEN if available to avoid rate limiting
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	// nolint: gosec
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Warnf("failed to get latest commit SHA for %s: %v", repo, err)
		return "", err
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Warnf("failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		log.Warnf("failed to get latest commit SHA for %s: %d %s", repo, resp.StatusCode, http.StatusText(resp.StatusCode))
		return "", fmt.Errorf("failed to get commit SHA: %d %s", resp.StatusCode, http.StatusText(resp.StatusCode))
	}

	var commit struct {
		SHA string `json:"sha"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&commit); err != nil {
		log.Warnf("failed to decode response for %s: %v", repo, err)
		return "", err
	}

	return commit.SHA, nil
}

// getGitCommitHashForFile returns the commit hash of the last commit that modified the file
func getGitCommitHashForFile(file string) (string, error) {
	// Get the absolute path of the file
	absPath, err := filepath.Abs(file)
	if err != nil {
		log.Warnf("failed to get absolute path for %s: %v", file, err)
		return "", err
	}

	// Check if the file is in a git repository
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = filepath.Dir(absPath)
	if err := cmd.Run(); err != nil {
		log.Warnf("file %s is not in a git repository: %v", file, err)
		return "", fmt.Errorf("file is not in a git repository: %v", err)
	}

	// Get the commit hash of the last commit that modified the file
	cmd = exec.Command("git", "log", "-n", "1", "--pretty=format:%H", "--", absPath)
	cmd.Dir = filepath.Dir(absPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		log.Warnf("failed to get git commit hash for %s: %v", file, err)
		return "", fmt.Errorf("failed to get git commit hash: %v", err)
	}

	return strings.TrimSpace(out.String()), nil
}

// isFileModifiedInGit checks if the file has uncommitted changes in git
func isFileModifiedInGit(file string) (bool, error) {
	// Get the absolute path of the file
	absPath, err := filepath.Abs(file)
	if err != nil {
		log.Warnf("failed to get absolute path for %s: %v", file, err)
		return false, err
	}

	// Check if the file has uncommitted changes
	cmd := exec.Command("git", "diff", "--quiet", "--", absPath)
	cmd.Dir = filepath.Dir(absPath)
	err = cmd.Run()

	// If the command exits with a non-zero status, the file has uncommitted changes
	if err != nil {
		log.Infof("file %s has uncommitted changes", file)
		return true, nil
	}

	// Also check if the file is staged but not committed
	cmd = exec.Command("git", "diff", "--cached", "--quiet", "--", absPath)
	cmd.Dir = filepath.Dir(absPath)
	err = cmd.Run()

	// If the command exits with a non-zero status, the file has staged changes
	if err != nil {
		log.Infof("file %s has staged changes", file)
		return true, nil
	}

	return false, nil
}

// calculateFileHash calculates the SHA-256 hash of a file
func calculateFileHash(file string) (string, error) {
	f, err := os.Open(file)
	if err != nil {
		log.Warnf("failed to open file %s: %v", file, err)
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		log.Warnf("failed to calculate hash for %s: %v", file, err)
		return "", err
	}

	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// getVersion returns the version of goinstaller
func getVersion() string {
	return version
}

// loadFromGitHub loads a project configuration from a GitHub repository
// and returns the project and source information.
func loadFromGitHub(repo, configPath, version string) (*config.Project, string, error) {
	repo = normalizeRepo(repo)
	log.Infof("reading repo %q on github", repo)

	// Get the default branch
	defaultBranch := getDefaultBranch(repo)

	// Try to get the commit hash for the default branch
	commitHash, err := getLatestCommitSHA(repo, defaultBranch)
	if err != nil {
		log.Warnf("failed to get commit hash for %s: %v", repo, err)
		commitHash = defaultBranch // Fallback to using the branch name
	}

	// Determine the actual config file path that was used
	var actualConfigPath string
	for _, file := range []string{configPath, "goreleaser.yml", ".goreleaser.yml", "goreleaser.yaml", ".goreleaser.yaml"} {
		if file == "" {
			continue
		}
		url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s", repo, commitHash, file)
		resp, err := http.Head(url)
		if err == nil && resp.StatusCode == http.StatusOK {
			actualConfigPath = file
			resp.Body.Close()
			break
		}
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
	}

	if actualConfigPath == "" {
		actualConfigPath = ".goreleaser.yml" // Default fallback
	}

	// Load the project configuration using the commit hash
	project, err := loadURLs(
		fmt.Sprintf("https://raw.githubusercontent.com/%s/%s", repo, commitHash),
		configPath,
	)
	if err != nil {
		return nil, "", err
	}

	// Create the source info
	sourceInfo := fmt.Sprintf("%s@%s", repo, commitHash)
	log.Infof("using source info with commit hash: %s", sourceInfo)
	return project, sourceInfo, nil
}

// loadFromFile loads a project configuration from a local file
// and returns the project and source information.
func loadFromFile(file, version string) (*config.Project, string, error) {
	log.Infof("reading file %q", file)

	// Load the project configuration
	project, err := loadFile(file)
	if err != nil {
		return nil, "", err
	}

	// Get absolute path for better context
	absPath, err := filepath.Abs(file)
	if err != nil {
		absPath = file // Fallback to the original path
	}

	// Try to get the git repository information
	repoRoot, relPath, repoErr := getGitRepoRootAndRelPath(file)
	if repoErr == nil {
		// Get the repository owner/name
		repoOwner, repoName, err := getGitRepoOwnerAndName(repoRoot)
		if err != nil {
			// If we can't get the owner/name, use the directory name
			repoName = filepath.Base(repoRoot)
			repoOwner = repoName
		}
		repoFullName := fmt.Sprintf("%s/%s", repoOwner, repoName)

		// Get the current commit hash
		gitCommitHash, err := getGitRepoHeadCommitHash(repoRoot)
		if err == nil && gitCommitHash != "" {
			// Check if the file has uncommitted changes
			isModified, err := isFileModifiedInGit(file)
			if err == nil && !isModified {
				// File is in a git repository and has no uncommitted changes
				sourceInfo := fmt.Sprintf("%s@%s", repoFullName, gitCommitHash)
				log.Infof("using source info with git commit hash: %s", sourceInfo)
				return project, sourceInfo, nil
			}

			// File has uncommitted changes
			hash, err := calculateFileHash(file)
			if err == nil {
				sourceInfo := fmt.Sprintf("%s:%s(uncommitted sha256:%s)", repoFullName, relPath, hash)
				log.Infof("using source info with uncommitted changes: %s", sourceInfo)
				return project, sourceInfo, nil
			}
		}
	}

	// Fallback to using the absolute path and content hash
	hash, err := calculateFileHash(file)
	if err == nil {
		sourceInfo := fmt.Sprintf("%s@sha256:%s", absPath, hash)
		log.Infof("using source info with absolute path and content hash: %s", sourceInfo)
		return project, sourceInfo, nil
	}

	// Final fallback to just using the file path
	sourceInfo := fmt.Sprintf("%s", absPath)
	log.Infof("using fallback source info: %s", sourceInfo)
	return project, sourceInfo, nil
}

// getGitRepoOwnerAndName tries to determine the owner and name of a git repository
// by looking at the remote URL
func getGitRepoOwnerAndName(repoRoot string) (string, string, error) {
	// Run git remote -v to get the remote URL
	cmd := exec.Command("git", "remote", "-v")
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", "", err
	}

	// Parse the output to find the origin URL
	remotes := strings.Split(out.String(), "\n")
	var originURL string
	for _, remote := range remotes {
		if strings.HasPrefix(remote, "origin") {
			parts := strings.Fields(remote)
			if len(parts) >= 2 {
				originURL = parts[1]
				break
			}
		}
	}

	if originURL == "" {
		return "", "", fmt.Errorf("no origin remote found")
	}

	// Extract owner/name from the URL
	// Handle different URL formats:
	// - https://github.com/owner/name.git
	// - git@github.com:owner/name.git
	var ownerName string
	if strings.Contains(originURL, "github.com") {
		if strings.HasPrefix(originURL, "https://") {
			// https URL format
			parts := strings.Split(originURL, "/")
			if len(parts) >= 5 {
				ownerName = fmt.Sprintf("%s/%s", parts[3], strings.TrimSuffix(parts[4], ".git"))
			}
		} else if strings.HasPrefix(originURL, "git@") {
			// SSH URL format
			parts := strings.Split(originURL, ":")
			if len(parts) >= 2 {
				ownerName = strings.TrimSuffix(parts[1], ".git")
			}
		}
	}

	if ownerName == "" {
		return "", "", fmt.Errorf("could not parse owner/name from URL: %s", originURL)
	}

	parts := strings.Split(ownerName, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid owner/name format: %s", ownerName)
	}

	return parts[0], parts[1], nil
}

// getGitRepoHeadCommitHash returns the commit hash of the HEAD of the repository
func getGitRepoHeadCommitHash(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = repoRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return strings.TrimSpace(out.String()), nil
}

// getGitRepoRootAndRelPath returns the git repository root and the relative path of the file
func getGitRepoRootAndRelPath(file string) (string, string, error) {
	// Get the absolute path of the file
	absPath, err := filepath.Abs(file)
	if err != nil {
		return "", "", err
	}

	// Get the git repository root
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = filepath.Dir(absPath)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", "", err
	}
	repoRoot := strings.TrimSpace(out.String())

	// Get the relative path from the repository root
	relPath, err := filepath.Rel(repoRoot, absPath)
	if err != nil {
		return "", "", err
	}

	return repoRoot, relPath, nil
}

func loadURLs(path, configPath string) (*config.Project, error) {
	for _, file := range []string{configPath, "goreleaser.yml", ".goreleaser.yml", "goreleaser.yaml", ".goreleaser.yaml"} {
		if file == "" {
			continue
		}
		url := fmt.Sprintf("%s/%s", path, file)
		log.Infof("reading %s", url)
		project, err := loadURL(url)
		if err != nil {
			return nil, err
		}
		if project != nil {
			return project, nil
		}
	}
	return nil, fmt.Errorf("could not fetch a goreleaser configuration file")
}

func loadURL(file string) (*config.Project, error) {
	// nolint: gosec
	resp, err := http.Get(file)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Errorf("reading %s returned %d %s\n", file, resp.StatusCode, http.StatusText(resp.StatusCode))
		return nil, nil
	}
	p, err := config.LoadReader(resp.Body)

	// to make errcheck happy
	errc := resp.Body.Close()
	if errc != nil {
		return nil, errc
	}
	return &p, err
}

func loadFile(file string) (*config.Project, error) {
	p, err := config.Load(file)
	return &p, err
}

// Load project configuration from a given repo name or filepath/url.
// Returns the project configuration and source information.
func Load(repo, configPath, file string) (project *config.Project, sourceInfo string, err error) {
	if repo == "" && file == "" {
		return nil, "", fmt.Errorf("repo or file not specified")
	}

	// Get the goinstaller version to include in the source info
	ver := getVersion()

	// Load the project configuration
	if file == "" {
		// GitHub repository
		project, sourceInfo, err = loadFromGitHub(repo, configPath, ver)
	} else {
		// Local file
		project, sourceInfo, err = loadFromFile(file, ver)
	}
	if err != nil {
		return nil, "", err
	}

	// if not specified add in GitHub owner/repo info
	if project.Release.GitHub.Owner == "" {
		if repo == "" {
			return nil, "", fmt.Errorf("owner/name repo not specified")
		}
		project.Release.GitHub.Owner = path.Dir(repo)
		project.Release.GitHub.Name = path.Base(repo)
	}

	// avoid errors in docker defaulter
	for i := range project.Dockers {
		project.Dockers[i].Files = []string{}
	}

	var ctx = context.New(*project)
	for _, defaulter := range defaults.Defaulters {
		log.Infof("setting defaults for %s", defaulter)
		if err := defaulter.Default(ctx); err != nil {
			return nil, "", errors.Wrap(err, "failed to set defaults")
		}
	}
	project = &ctx.Config

	// set default binary name
	if len(project.Builds) == 0 {
		project.Builds = []config.Build{
			{Binary: path.Base(repo)},
		}
	}
	if project.Builds[0].Binary == "" {
		project.Builds[0].Binary = path.Base(repo)
	}

	return project, sourceInfo, err
}

func main() {
	log.SetHandler(cli.Default)

	var (
		repo                     = kingpin.Flag("repo", "owner/name or URL of GitHub repository").Short('r').String()
		output                   = kingpin.Flag("output", "output file, default stdout").Short('o').String()
		force                    = kingpin.Flag("force", "force writing of output").Short('f').Bool()
		enableGHAttestation      = kingpin.Flag("enable-gh-attestation", "enable GitHub attestation verification").Bool()
		requireAttestation       = kingpin.Flag("require-attestation", "require attestation verification").Bool()
		ghAttestationVerifyFlags = kingpin.Flag("gh-attestation-verify-flags", "additional flags to pass to gh attestation verify").String()
		file                     = kingpin.Arg("file", "godownloader.yaml file or URL").String()
	)

	kingpin.CommandLine.Version(fmt.Sprintf("%v, commit %v, built at %v", version, commit, datestr))
	kingpin.CommandLine.VersionFlag.Short('v')
	kingpin.CommandLine.HelpFlag.Short('h')
	kingpin.Parse()

	// Validate attestation options
	if *requireAttestation && !*enableGHAttestation {
		log.Error("cannot specify --require-attestation without --enable-gh-attestation")
		os.Exit(1)
	}

	// Create attestation options
	attestationOpts := AttestationOptions{
		EnableGHAttestation:      *enableGHAttestation,
		RequireAttestation:       *requireAttestation,
		GHAttestationVerifyFlags: *ghAttestationVerifyFlags,
	}

	// Process the source
	out, err := processSource("godownloader", *repo, "", *file, attestationOpts)

	if err != nil {
		log.WithError(err).Error("failed")
		os.Exit(1)
	}

	// stdout case
	if *output == "" {
		if _, err = os.Stdout.Write(out); err != nil {
			log.WithError(err).Error("unable to write")
			os.Exit(1)
		}
		return
	}

	// only write out if forced to, OR if output is effectively different
	// than what the file has.
	if *force || shell.ShouldWriteFile(*output, out) {
		if err = os.WriteFile(*output, out, 0666); err != nil { //nolint: gosec
			log.WithError(err).Errorf("unable to write to %s", *output)
			os.Exit(1)
		}
		return
	}

	// output is effectively the same as new content
	// (comments and most whitespace doesn't matter)
	// nothing to do
}
