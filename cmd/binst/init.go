package main

import (
	"context"
	"fmt"
	"os"

	"github.com/apex/log"
	"github.com/haya14busa/goinstaller/pkg/datasource"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	// Flags for init command
	initSource       string
	initSourceFile   string
	initRepo         string // Repo for GitHub source OR explicit override
	initName         string // Explicit override for binary name
	initTag          string
	initCommitSHA    string
	initAssetPattern string
	initOutputFile   string
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate an InstallSpec config file from various sources",
	Long: `Initializes a binstaller configuration file (.binstaller.yml) by detecting
settings from a source like a GoReleaser config file or a GitHub repository.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Infof("Running init command...")

		var adapter datasource.SourceAdapter
		var input datasource.DetectInput

		switch initSource {
		case "goreleaser":
			adapter = datasource.NewGoReleaserAdapter(initCommitSHA, initSourceFile)
			input = datasource.DetectInput{
				FilePath: initSourceFile,          // Path to .goreleaser.yml
				Repo:     initRepo,                // Pass repo flag value
				Flags:    make(map[string]string), // Initialize flags map
				// Tag is not directly used by goreleaser adapter's Load currently, but keep for consistency
			}
			if initName != "" {
				input.Flags["name"] = initName // Pass name override via Flags map
			}
			// TODO: Add validation: require --file or --repo for goreleaser?
		case "github":
			// TODO: Implement githubProbeAdapter
			log.Errorf("source 'github' not yet implemented")
			return fmt.Errorf("source 'github' not yet implemented")
		case "cli":
			// TODO: Implement flagsAdapter logic
			log.Errorf("source 'cli' not yet implemented")
			return fmt.Errorf("source 'cli' not yet implemented")
		case "file":
			// TODO: Implement fileAdapter (for reading existing .binstaller.yml)
			log.Errorf("source 'file' not yet implemented")
			return fmt.Errorf("source 'file' not yet implemented")
		default:
			err := fmt.Errorf("unknown source specified: %s. Valid sources are: goreleaser, github, cli, file", initSource)
			log.WithError(err).Error("invalid source")
			return err
		}

		// Create context
		ctx := context.Background() // TODO: Add timeout from global flags?

		// Detect the InstallSpec
		log.Infof("Detecting InstallSpec using source: %s", initSource)
		installSpec, err := adapter.Detect(ctx, input)
		if err != nil {
			log.WithError(err).Error("Failed to detect install spec")
			return fmt.Errorf("failed to detect install spec: %w", err)
		}
		log.Info("Successfully detected InstallSpec")

		// Marshal the spec to YAML
		log.Debug("Marshalling InstallSpec to YAML")
		yamlData, err := yaml.Marshal(installSpec)
		if err != nil {
			log.WithError(err).Error("Failed to marshal InstallSpec to YAML")
			return fmt.Errorf("failed to marshal install spec to YAML: %w", err)
		}

		// Write the output
		if initOutputFile == "" || initOutputFile == "-" {
			// Write to stdout
			log.Debug("Writing InstallSpec YAML to stdout")
			fmt.Println(string(yamlData))
			log.Info("InstallSpec YAML written to stdout")
		} else {
			// Write to file
			log.Infof("Writing InstallSpec YAML to file: %s", initOutputFile)
			err = os.WriteFile(initOutputFile, yamlData, 0644) // Use standard file permissions
			if err != nil {
				log.WithError(err).Errorf("Failed to write InstallSpec to file: %s", initOutputFile)
				return fmt.Errorf("failed to write install spec to file %s: %w", initOutputFile, err)
			}
			log.Infof("InstallSpec successfully written to %s", initOutputFile)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Required flags
	initCmd.Flags().StringVar(&initSource, "source", "", "Source type to detect spec from (required: goreleaser, github, cli, file)")
	_ = initCmd.MarkFlagRequired("source")

	// Optional flags (depending on source)
	initCmd.Flags().StringVar(&initSourceFile, "file", "", "Path to source file (e.g., .goreleaser.yml)")
	initCmd.Flags().StringVar(&initRepo, "repo", "", "GitHub repository (owner/repo) for source 'goreleaser'/'github', or explicit override")
	initCmd.Flags().StringVar(&initName, "name", "", "Explicit binary name override")
	initCmd.Flags().StringVar(&initTag, "tag", "", "Release tag/ref to inspect (for source 'github')")
	initCmd.Flags().StringVar(&initCommitSHA, "sha", "", "Commit SHA for source 'goreleaser'")
	initCmd.Flags().StringVar(&initAssetPattern, "asset-pattern", "", "Template for asset file names (for source 'cli')") // TODO: Implement usage
	initCmd.Flags().StringVarP(&initOutputFile, "output", "o", ".binstaller.yml", "Write spec to file instead of stdout (use '-' for stdout)")

	// TODO: Add dependencies between flags (e.g., --file required if --source goreleaser and no --repo)
}
