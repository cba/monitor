package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
)

type remoteExec struct {
	User     string
	Pass     string
	Host     string
	KeyPath  string
	CertPath string
}

// buildRemoteExec creates remoteExec from MonitorConfig.SSH + Host.
// ssh.host overrides host when set.
func buildRemoteExec(cfg *config.MonitorConfig) remoteExec {
	if cfg.SSH == nil {
		return remoteExec{}
	}
	sshHost := cfg.Host
	if cfg.SSH.Host != "" {
		sshHost = cfg.SSH.Host
	}
	return remoteExec{
		User:     cfg.SSH.User,
		Pass:     cfg.SSH.Password,
		Host:     sshHost,
		KeyPath:  cfg.SSH.KeyFile,
		CertPath: cfg.SSH.CertFile,
	}
}

// execCommand executes a command locally or via SSH.
func execCommand(ctx context.Context, re remoteExec, cmd string) (string, error) {
	if re.User == "" {
		return localExec(ctx, cmd)
	}
	return sshExec(ctx, re, cmd)
}

func localExec(ctx context.Context, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("local exec: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
