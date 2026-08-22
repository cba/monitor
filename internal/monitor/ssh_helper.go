package monitor

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func sshExec(ctx context.Context, user, pass, host, cmd string) (string, error) {
	if !strings.Contains(host, ":") {
		host += ":22"
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	client, err := ssh.Dial("tcp", host, config)
	if err != nil {
		return "", fmt.Errorf("ssh dial: %w", err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh session: %w", err)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	if err := session.Run(cmd); err != nil {
		return "", fmt.Errorf("ssh exec: %s: %w", strings.TrimSpace(stderr.String()), err)
	}
	return stdout.String(), nil
}
