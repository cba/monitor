package cli

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/monitor"

	"github.com/spf13/cobra"
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "Manage monitors",
}

var monitorListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all monitors",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(resolveConfigFile())
		if err != nil {
			return err
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "NAME\tTYPE\tTARGET\tINTERVAL\tALERT\tENABLED")
		for _, m := range cfg.Monitors {
			target := monitorTarget(&m)
			enabled := m.Enabled != nil && *m.Enabled
			fmt.Fprintf(w, "%s\t%s\t%s\t%ds\t%ds\t%v\n", m.Name, m.Type, target, m.Interval, m.AlertInterval, enabled)
		}
		w.Flush()
		return nil
	},
}

var monitorTestCmd = &cobra.Command{
	Use:   "test [name]",
	Short: "Test a monitor by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(resolveConfigFile())
		if err != nil {
			return err
		}

		name := args[0]
		for _, m := range cfg.Monitors {
			if m.Name == name {
				mon, err := monitor.Get(m.Type)
				if err != nil {
					return err
				}

				result, err := mon.Check(cmd.Context(), &m)
				if err != nil {
					return err
				}
				fmt.Printf("Status: %s\nMessage: %s\nLatency: %v\n", result.Status, result.Message, result.Latency)
				return nil
			}
		}
		return fmt.Errorf("monitor not found: %s", name)
	},
}

var monitorTypesCmd = &cobra.Command{
	Use:   "types",
	Short: "List available monitor types",
	RunE: func(cmd *cobra.Command, args []string) error {
		types := monitor.List()
		for _, t := range types {
			fmt.Println(t)
		}
		return nil
	},
}

func init() {
	monitorCmd.AddCommand(monitorListCmd)
	monitorCmd.AddCommand(monitorTestCmd)
	monitorCmd.AddCommand(monitorTypesCmd)
	rootCmd.AddCommand(monitorCmd)
}

func monitorTarget(m *config.MonitorConfig) string {
	switch m.Type {
	case "http", "keyword":
		return m.URL
	case "tcp", "redis":
		if m.Port != "" {
			return m.Host + ":" + m.Port
		}
		return m.Host
	case "ssl":
		if m.Port != "" && m.Port != "443" {
			return m.Host + ":" + m.Port
		}
		return m.Host
	case "mysql":
		return m.DSN
	case "icmp":
		return m.Target
	case "disk":
		return m.Path
	case "cpu_load", "memory":
		if m.SSH != nil {
			return m.Host
		}
		return "local"
	case "process":
		if m.SSH != nil {
			return m.Host + "/" + m.ProcessName
		}
		return m.ProcessName
	case "container":
		if m.SSH != nil {
			return m.Host + "/" + m.ContainerName
		}
		return m.ContainerName
	default:
		return m.Target
	}
}
