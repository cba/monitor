package cli

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/cba/monitor/internal/config"
	"github.com/cba/monitor/internal/scheduler"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the monitoring service",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := resolveConfigFile()

		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
		defer cancel()

		// Create scheduler
		s := scheduler.New()

		// Create config watcher for hot reload
		watcher := config.NewWatcher(path, cfg, func(newCfg *config.Config) {
			log.Println("config changed, reloading...")
			s.Reload(newCfg)
		})
		if err := watcher.Start(); err != nil {
			log.Printf("failed to start config watcher: %v", err)
		}
		defer watcher.Stop()

		// Start scheduler and wait for context to be ready
		ready := make(chan struct{})
		go func() {
			s.StartWithContext(ctx, ready)
		}()
		<-ready // Wait until context is set

		// Now safe to reload
		s.Reload(cfg)

		log.Println("monitor started")
		<-ctx.Done()
		s.Stop()
		log.Println("monitor stopped")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
