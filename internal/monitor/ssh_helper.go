package monitor

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/ssh"
)

func sshExec(ctx context.Context, re remoteExec, cmd string) (string, error) {
	client, err := sshClientFor(ctx, re)
	if err != nil {
		return "", fmt.Errorf("ssh dial: %w", err)
	}

	session, err := client.NewSession()
	if err != nil {
		markSSHBad(re, client)
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

	return dialSSH(host, sshCfg)
}

// ---- 连接池 ----
// 同一 SSH 跳板的多个监控项/多个周期共享一条连接，
// 避免每次检测都做完整 TCP+认证握手（会触发 sshd MaxStartups 限流）。

const (
	sshIdleTTL = 10 * time.Minute // 空闲超过此时长的连接被回收
)

type sshPoolEntry struct {
	client   *ssh.Client
	lastUsed time.Time
}

var (
	sshPoolMu   sync.Mutex
	sshPool     = map[string]*sshPoolEntry{}
	sshDialMu   = map[string]*sync.Mutex{} // 串行化同 key 拨号，防并发握手风暴
	sshSweepOne sync.Once
)

func sshPoolKey(re remoteExec) string {
	host := re.Host
	if !strings.Contains(host, ":") {
		host += ":22"
	}
	return re.User + "@" + host
}

// sshClientFor 返回共享的 SSH client：池中存活则复用，否则拨号入池。
func sshClientFor(ctx context.Context, re remoteExec) (*ssh.Client, error) {
	key := sshPoolKey(re)

	sshPoolMu.Lock()
	e := sshPool[key]
	sshPoolMu.Unlock()

	if e != nil {
		// ponytail: 半开连接（对端消失无 RST）时此探活会阻塞到 OS TCP 超时；
		// x/crypto 不暴露底层 conn，如需收紧可用 watchdog goroutine 包一层。
		if _, _, err := e.client.SendRequest("keepalive@openssh.com", true, nil); err == nil {
			sshPoolMu.Lock()
			if cur := sshPool[key]; cur == e {
				e.lastUsed = time.Now()
			}
			sshPoolMu.Unlock()
			return e.client, nil
		}
		markSSHBad(re, e.client)
	}

	mu := dialLock(key)
	mu.Lock()
	defer mu.Unlock()

	sshPoolMu.Lock() // double-check：等锁期间其他 goroutine 可能已拨好
	if e := sshPool[key]; e != nil {
		c := e.client
		e.lastUsed = time.Now()
		sshPoolMu.Unlock()
		return c, nil
	}
	sshPoolMu.Unlock()

	client, err := newSSHClient(ctx, re)
	if err != nil {
		return nil, err
	}
	sshPoolMu.Lock()
	sshPool[key] = &sshPoolEntry{client: client, lastUsed: time.Now()}
	sshPoolMu.Unlock()
	startSSHSweeper()
	return client, nil
}

// markSSHBad 将出错（或探活失败）的 client 逐出并关闭；幂等，凭指针比对防止误杀新连接。
func markSSHBad(re remoteExec, client *ssh.Client) {
	key := sshPoolKey(re)
	sshPoolMu.Lock()
	e, ok := sshPool[key]
	if ok && e.client == client {
		delete(sshPool, key)
		sshPoolMu.Unlock()
		client.Close()
		return
	}
	sshPoolMu.Unlock()
}

func dialLock(key string) *sync.Mutex {
	sshPoolMu.Lock()
	defer sshPoolMu.Unlock()
	mu, ok := sshDialMu[key]
	if !ok {
		mu = &sync.Mutex{}
		sshDialMu[key] = mu // ponytail: 每 key 一把锁，数量=配置里的主机数，不回收
	}
	return mu
}

func startSSHSweeper() {
	sshSweepOne.Do(func() {
		go func() {
			for range time.Tick(time.Minute) {
				sshPoolMu.Lock()
				now := time.Now()
				for k, e := range sshPool {
					if now.Sub(e.lastUsed) > sshIdleTTL {
						delete(sshPool, k)
						delete(sshDialMu, k)
						go e.client.Close()
					}
				}
				sshPoolMu.Unlock()
			}
		}()
	})
}

// dialSSH 建立 SSH 连接。握手偶发被 RST 时（sshd 的 MaxStartups 会随机丢弃
// 并发未认证连接，公网 22 端口的爆破流量会占满这些名额）重试一次，
// 避免瞬时抖动被当成服务器宕机报警。
func dialSSH(host string, config *ssh.ClientConfig) (*ssh.Client, error) {
	client, err := ssh.Dial("tcp", host, config)
	if err == nil {
		return client, nil
	}
	time.Sleep(time.Second)
	return ssh.Dial("tcp", host, config)
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
