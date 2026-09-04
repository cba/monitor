package monitor

import (
	"context"
	"net"
	"strings"
	"testing"
	"time"
)

// 服务端接受连接后立即 RST：验证 sshd 偶发重置时会重试拨号而不是直接判定宕机
func TestDialSSHRetryOnHandshakeReset(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	attempts := make(chan struct{}, 10)
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			attempts <- struct{}{}
			if tcp, ok := conn.(*net.TCPConn); ok {
				tcp.SetLinger(0)
			}
			conn.Close()
		}
	}()

	re := remoteExec{User: "root", Pass: "x", Host: ln.Addr().String()}
	if _, err := sshExec(context.Background(), re, "true"); err == nil || !strings.Contains(err.Error(), "ssh dial") {
		t.Fatalf("want ssh dial error, got %v", err)
	}

	deadline := time.After(time.Second)
	n := 0
	for n < 2 {
		select {
		case <-attempts:
			n++
		case <-deadline:
			t.Fatalf("dial attempts = %d, want 2 (retry missing?)", n)
		}
	}
}
