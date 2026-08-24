package monitor

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"
)

func sshExec(ctx context.Context, re remoteExec, cmd string) (string, error) {
	host := re.Host
	if !strings.Contains(host, ":") {
		host += ":22"
	}

	authMethods, err := buildAuthMethods(re)
	if err != nil {
		return "", err
	}

	config := &ssh.ClientConfig{
		User:            re.User,
		Auth:            authMethods,
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

// newSSHClient creates an SSH client from remoteExec config.
func newSSHClient(ctx context.Context, re remoteExec) (*ssh.Client, error) {
	host := re.Host
	if !strings.Contains(host, ":") {
		host += ":22"
	}

	authMethods, err := buildAuthMethods(re)
	if err != nil {
		return nil, err
	}

	sshCfg := &ssh.ClientConfig{
		User:            re.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	return ssh.Dial("tcp", host, sshCfg)
}

// sshDial dials targetHost:targetPort through an SSH client.
func sshDial(client *ssh.Client, targetHost, targetPort string) (net.Conn, error) {
	conn, err := client.Dial("tcp", net.JoinHostPort(targetHost, targetPort))
	if err != nil {
		return nil, err
	}
	return &sshConn{Conn: conn}, nil
}

// sshConn wraps an SSH channel to satisfy net.Conn.
// SSH channels don't support SetDeadline; this makes them no-ops.
type sshConn struct {
	net.Conn
}

func (c *sshConn) SetDeadline(t time.Time) error      { return nil }
func (c *sshConn) SetReadDeadline(t time.Time) error  { return nil }
func (c *sshConn) SetWriteDeadline(t time.Time) error { return nil }

func buildAuthMethods(re remoteExec) ([]ssh.AuthMethod, error) {
	if re.CertPath != "" && re.KeyPath != "" {
		return certAuth(re.CertPath, re.KeyPath)
	}
	if re.KeyPath != "" {
		return keyAuth(re.KeyPath)
	}
	if re.Pass != "" {
		return []ssh.AuthMethod{ssh.Password(re.Pass)}, nil
	}
	return nil, fmt.Errorf("ssh: no auth method configured for %s@%s", re.User, re.Host)
}

func certAuth(certPath, keyPath string) ([]ssh.AuthMethod, error) {
	cert, err := os.ReadFile(certPath)
	if err != nil {
		return nil, fmt.Errorf("ssh: read cert %s: %w", certPath, err)
	}
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ssh: read key %s: %w", keyPath, err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("ssh: parse key: %w", err)
	}

	pubKey, _, _, _, err := ssh.ParseAuthorizedKey(cert)
	if err != nil {
		return nil, fmt.Errorf("ssh: parse cert: %w", err)
	}
	sshCert, ok := pubKey.(*ssh.Certificate)
	if !ok {
		return nil, fmt.Errorf("ssh: cert is not an SSH certificate")
	}

	signed, err := ssh.NewCertSigner(sshCert, signer)
	if err != nil {
		return nil, fmt.Errorf("ssh: cert signer: %w", err)
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signed)}, nil
}

func keyAuth(keyPath string) ([]ssh.AuthMethod, error) {
	key, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("ssh: read key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("ssh: parse key: %w", err)
	}
	return []ssh.AuthMethod{ssh.PublicKeys(signer)}, nil
}
