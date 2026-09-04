package monitor

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"golang.org/x/crypto/ssh"
)

type fakeSSHServer struct {
	ln         net.Listener
	handshakes atomic.Int64
	sessions   atomic.Int64

	mu       sync.Mutex
	rawConns []net.Conn
}

func newFakeSSHServer(t *testing.T) *fakeSSHServer {
	t.Helper()
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	signer, err := ssh.NewSignerFromKey(priv)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &ssh.ServerConfig{
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			if c.User() == "root" && string(pass) == "x" {
				return nil, nil
			}
			return nil, errors.New("bad credentials")
		},
	}
	cfg.AddHostKey(signer)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeSSHServer{ln: ln}
	go s.accept(cfg)
	t.Cleanup(func() {
		s.killAll()
		ln.Close()
	})
	return s
}

func (s *fakeSSHServer) accept(cfg *ssh.ServerConfig) {
	for {
		ncc, err := s.ln.Accept()
		if err != nil {
			return
		}
		s.mu.Lock()
		s.rawConns = append(s.rawConns, ncc)
		s.mu.Unlock()
		go s.handle(ncc, cfg)
	}
}

func (s *fakeSSHServer) handle(ncc net.Conn, cfg *ssh.ServerConfig) {
	_, chans, reqs, err := ssh.NewServerConn(ncc, cfg)
	if err != nil {
		ncc.Close()
		return
	}
	s.handshakes.Add(1)
	go ssh.DiscardRequests(reqs)
	for nc := range chans {
		if nc.ChannelType() != "session" {
			nc.Reject(ssh.UnknownChannelType, "unsupported")
			continue
		}
		ch, requests, err := nc.Accept()
		if err != nil {
			continue
		}
		s.sessions.Add(1)
		go func(ch ssh.Channel, requests <-chan *ssh.Request) {
			defer ch.Close()
			for req := range requests {
				switch req.Type {
				case "exec":
					req.Reply(true, nil)
					ch.Write([]byte("ok\n"))
					ch.SendRequest("exit-status", false, ssh.Marshal(struct{ Status uint32 }{}))
					return
				default:
					if req.WantReply {
						req.Reply(false, nil)
					}
				}
			}
		}(ch, requests)
	}
}

// killAll 模拟服务器重启：RST 掉所有已建立连接，但监听继续接受新拨号。
func (s *fakeSSHServer) killAll() {
	s.mu.Lock()
	conns := s.rawConns
	s.rawConns = nil
	s.mu.Unlock()
	for _, c := range conns {
		if tcp, ok := c.(*net.TCPConn); ok {
			tcp.SetLinger(0)
		}
		c.Close()
	}
}

func TestSSHPoolReuseAndEvict(t *testing.T) {
	s := newFakeSSHServer(t)
	re := remoteExec{User: "root", Pass: "x", Host: s.ln.Addr().String()}
	ctx := context.Background()

	check := func(wantHS int64) {
		t.Helper()
		out, err := sshExec(ctx, re, "true")
		if err != nil || out != "ok\n" {
			t.Fatalf("sshExec: out=%q err=%v", out, err)
		}
		if n := s.handshakes.Load(); n != wantHS {
			t.Fatalf("handshakes = %d, want %d", n, wantHS)
		}
	}

	// 多次检测只握手一次
	check(1)
	check(1)
	check(1)
	if n := s.sessions.Load(); n != 3 {
		t.Fatalf("sessions = %d, want 3", n)
	}

	// 服务器重启（连接被 RST）后，下一次检测探活失败并自动重连
	s.killAll()
	check(2)

	// markSSHBad 踢出后，下一次检测重新拨号
	client, err := sshClientFor(ctx, re)
	if err != nil {
		t.Fatal(err)
	}
	markSSHBad(re, client)
	check(3)

	// 池中应只有一条存活连接：再复用不增加握手
	check(3)
}
