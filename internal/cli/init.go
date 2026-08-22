package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/cba/monitor/internal/config"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a default config.yaml",
	RunE: func(cmd *cobra.Command, args []string) error {
		// For init, use home directory if no config specified
		target := configFile
		if target == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("cannot get home directory: %w", err)
			}
			dir := filepath.Join(home, ".monitor")
			if err := os.MkdirAll(dir, 0755); err != nil {
				return fmt.Errorf("create directory: %w", err)
			}
			target = filepath.Join(dir, "config.yaml")
		}

		cfg := config.DefaultConfig()
		if err := config.Save(target, cfg); err != nil {
			return err
		}
		fmt.Printf("Config created: %s\n", target)
		fmt.Println("Edit the file and run: monitor serve")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}
