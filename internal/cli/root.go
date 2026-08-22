package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var configFile string

var rootCmd = &cobra.Command{
	Use:   "monitor",
	Short: "A lightweight monitoring service",
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "", "Configuration file path")
}

// resolveConfigFile finds the config file in priority order:
// 1. User-specified via -c flag
// 2. Current directory: ./config.yaml
// 3. Home directory: ~/.monitor/config.yaml
func resolveConfigFile() string {
	// If user specified a config file, use it
	if configFile != "" {
		return configFile
	}

	// Check current directory
	if _, err := os.Stat("config.yaml"); err == nil {
		return "config.yaml"
	}

	// Check home directory
	if home, err := os.UserHomeDir(); err == nil {
		homeConfig := filepath.Join(home, ".monitor", "config.yaml")
		if _, err := os.Stat(homeConfig); err == nil {
			return homeConfig
		}
	}

	// Default to current directory (will fail with clear error)
	return "config.yaml"
}
