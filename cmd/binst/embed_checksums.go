package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/goccy/go-yaml"
	"github.com/goccy/go-yaml/parser"
	"github.com/haya14busa/goinstaller/pkg/checksums"
	"github.com/haya14busa/goinstaller/pkg/spec"
	"github.com/spf13/cobra"
)

var (
	// Flags for embed-checksums command
	embedVersion      string
	embedOutput       string
	embedMode         string
	embedFile         string
	embedAllPlatforms bool
)

// embedChecksumsCmd represents the embed-checksums command
var embedChecksumsCmd = &cobra.Command{
	Use:   "embed-checksums",
	Short: "Embed checksums for release assets into a binstaller configuration",
	Long: `Reads an InstallSpec configuration file and embeds checksums for the assets.
This command supports three modes of operation:
- download: Fetches the checksum file from GitHub releases
- checksum-file: Uses a local checksum file
- calculate: Downloads the assets and calculates checksums directly`,
	RunE: func(cmd *cobra.Command, args []string) error {
		log.Info("Running embed-checksums command...")

		// Determine config file path
		cfgFile := configFile // Use the global flag value
		if cfgFile == "" {
			// Default detection logic if global flag is not set
			defaultPath := ".binstaller.yml"
			if _, err := os.Stat(defaultPath); err == nil {
				cfgFile = defaultPath
				log.Infof("Using default config file: %s", cfgFile)
			} else {
				// Try .binstaller.yaml as fallback
				defaultPathYaml := ".binstaller.yaml"
				if _, errYaml := os.Stat(defaultPathYaml); errYaml == nil {
					cfgFile = defaultPathYaml
					log.Infof("Using default config file: %s", cfgFile)
				} else {
					err := fmt.Errorf("config file not specified via --config and default (.binstaller.yml or .binstaller.yaml) not found")
					log.WithError(err).Error("Config file detection failed")
					return err
				}
			}
		}
		log.Debugf("Using config file: %s", cfgFile)

		// Read the InstallSpec YAML file
		log.Debugf("Reading InstallSpec from: %s", cfgFile)

		ast, err := parser.ParseFile(cfgFile, parser.ParseComments)
		if err != nil {
			return err
		}

		yamlData, err := os.ReadFile(cfgFile)
		if err != nil {
			log.WithError(err).Errorf("Failed to read install spec file: %s", cfgFile)
			return fmt.Errorf("failed to read install spec file %s: %w", cfgFile, err)
		}

		// Unmarshal YAML into InstallSpec struct
		log.Debug("Unmarshalling InstallSpec YAML")
		var installSpec spec.InstallSpec
		err = yaml.UnmarshalWithOptions(yamlData, &installSpec, yaml.UseOrderedMap())
		if err != nil {
			log.WithError(err).Errorf("Failed to unmarshal install spec YAML from: %s", cfgFile)
			return fmt.Errorf("failed to unmarshal install spec YAML from %s: %w", cfgFile, err)
		}

		// Create the embedder
		var mode checksums.EmbedMode
		switch embedMode {
		case "download":
			mode = checksums.EmbedModeDownload
		case "checksum-file":
			mode = checksums.EmbedModeChecksumFile
		case "calculate":
			mode = checksums.EmbedModeCalculate
		default:
			return fmt.Errorf("invalid mode: %s. Must be one of: download, checksum-file, calculate", embedMode)
		}

		// Validate checksum-file mode has a file
		if mode == checksums.EmbedModeChecksumFile && embedFile == "" {
			log.Error("--file flag is required for checksum-file mode")
			return fmt.Errorf("--file flag is required for checksum-file mode")
		}

		embedder := &checksums.Embedder{
			Mode:         mode,
			Version:      embedVersion,
			Spec:         &installSpec,
			SpecAST:      ast,
			ChecksumFile: embedFile,
			AllPlatforms: embedAllPlatforms,
		}

		// Embed the checksums
		log.Infof("Embedding checksums using %s mode for version: %s", mode, embedVersion)
		if err := embedder.Embed(); err != nil {
			log.WithError(err).Error("Failed to embed checksums")
			return fmt.Errorf("failed to embed checksums: %w", err)
		}

		// Determine output file
		outputFile := embedOutput
		if outputFile == "" {
			outputFile = cfgFile
			log.Infof("No output specified, overwriting input file: %s", outputFile)
		}

		// Write the updated InstallSpec back to the output file
		log.Infof("Writing updated InstallSpec to file: %s", outputFile)

		// Ensure the output directory exists
		outputDir := filepath.Dir(outputFile)
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			log.WithError(err).Errorf("Failed to create output directory: %s", outputDir)
			return fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
		}

		// Write the YAML to the output file
		if err := os.WriteFile(outputFile, []byte(ast.String()), 0644); err != nil {
			log.WithError(err).Errorf("Failed to write InstallSpec to file: %s", outputFile)
			return fmt.Errorf("failed to write InstallSpec to file %s: %w", outputFile, err)
		}
		log.Infof("InstallSpec successfully updated with embedded checksums")

		return nil
	},
}

func init() {
	rootCmd.AddCommand(embedChecksumsCmd)

	// Flags specific to embed-checksums command
	embedChecksumsCmd.Flags().StringVarP(&embedVersion, "version", "v", "", "Version to embed checksums for (default: latest)")
	embedChecksumsCmd.Flags().StringVarP(&embedOutput, "output", "o", "", "Output path for the updated InstallSpec (default: overwrite input file)")
	embedChecksumsCmd.Flags().StringVarP(&embedMode, "mode", "m", "download", "Checksums acquisition mode (download, checksum-file, calculate)")
	embedChecksumsCmd.Flags().StringVarP(&embedFile, "file", "f", "", "Path to checksum file (required for checksum-file mode)")
	embedChecksumsCmd.Flags().BoolVar(&embedAllPlatforms, "all-platforms", false, "Generate checksums for all supported platforms (for calculate mode)")

	// Mark required flags
	embedChecksumsCmd.MarkFlagRequired("mode")
}
