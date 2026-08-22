package main

import (
	// Import monitor plugins to trigger registration
	_ "github.com/cba/monitor/internal/monitor"
	// Import notifier plugins to trigger registration
	_ "github.com/cba/monitor/internal/notifier"

	"github.com/cba/monitor/internal/cli"
)

func main() {
	cli.Execute()
}
