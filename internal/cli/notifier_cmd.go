package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cba/monitor/internal/config"

	"github.com/spf13/cobra"
)

var notifierCmd = &cobra.Command{
	Use:   "notifier",
	Short: "Manage notifiers",
}

var notifierListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all notifiers",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(resolveConfigFile())
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tENABLED")
		for _, n := range cfg.Notifiers {
			fmt.Fprintf(w, "%s\t%s\t%v\n", n.Name, n.Type, n.Enabled)
		}
		w.Flush()
		return nil
	},
}

var notifierTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List available notifier types",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("wechat")
		fmt.Println("wechat_app")
		fmt.Println("dingtalk")
		return nil
	},
}

func init() {
	notifierCmd.AddCommand(notifierListCmd)
	notifierCmd.AddCommand(notifierTypesCmd)
	rootCmd.AddCommand(notifierCmd)
}
