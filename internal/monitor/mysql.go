package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/cba/monitor/internal/config"
	mysqldriver "github.com/go-sql-driver/mysql"
)

type mysqlMonitor struct{}

func init() {
	Register(&mysqlMonitor{})
}

func (m *mysqlMonitor) Name() string { return "mysql" }

func (m *mysqlMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
	dsn := cfg.DSN
	re := buildRemoteExec(cfg)

	if re.User != "" {
		sshClient, err := sshClientFor(ctx, re)
		if err != nil {
			return nil, fmt.Errorf("mysql ssh: %w", err)
		}

		host, port := parseMySQLHostPort(dsn)
		if host == "" || port == "" {
			return nil, fmt.Errorf("mysql: cannot parse host:port from DSN")
		}

		dbCfg, err := mysqldriver.ParseDSN(dsn)
		if err != nil {
			return nil, fmt.Errorf("mysql: parse DSN: %w", err)
		}

		dbCfg.DialFunc = func(ctx context.Context, network, addr string) (net.Conn, error) {
			conn, err := sshDial(sshClient, host, port)
			if err != nil {
				markSSHBad(re, sshClient)
			}
			return conn, err
		}

		connector, err := mysqldriver.NewConnector(dbCfg)
		if err != nil {
			return nil, fmt.Errorf("mysql: create connector: %w", err)
		}

		db := sql.OpenDB(connector)
		defer db.Close()

		start := time.Now()
		err = db.PingContext(ctx)
		latency := time.Since(start)
		if err != nil {
			return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
		}
		return &Result{Status: "up", Message: "MySQL connect OK", Latency: latency, Timestamp: time.Now()}, nil
	}

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: 0, Timestamp: time.Now()}, nil
	}
	defer db.Close()

	start := time.Now()
	err = db.PingContext(ctx)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: "MySQL connect OK", Latency: latency, Timestamp: time.Now()}, nil
}

// parseMySQLHostPort extracts host and port from DSN: user:pass@tcp(host:port)/db
func parseMySQLHostPort(dsn string) (host, port string) {
	start := strings.Index(dsn, "tcp(")
	if start < 0 {
		return "", ""
	}
	start += 4
	end := strings.Index(dsn[start:], ")")
	if end < 0 {
		return "", ""
	}
	hp := dsn[start : start+end]
	parts := strings.SplitN(hp, ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
