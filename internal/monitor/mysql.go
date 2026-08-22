package monitor

import (
	"context"
	"database/sql"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type mysqlMonitor struct{}

func init() {
	Register(&mysqlMonitor{})
}

func (m *mysqlMonitor) Name() string { return "mysql" }

func (m *mysqlMonitor) Check(ctx context.Context, target string) (*Result, error) {
	db, err := sql.Open("mysql", target)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: 0, Timestamp: time.Now()}, nil
	}
	defer db.Close()

	db.SetConnMaxLifetime(time.Second * 3)

	start := time.Now()
	err = db.PingContext(ctx)
	latency := time.Since(start)
	if err != nil {
		return &Result{Status: "down", Message: err.Error(), Latency: latency, Timestamp: time.Now()}, nil
	}
	return &Result{Status: "up", Message: "Connection successful", Latency: latency, Timestamp: time.Now()}, nil
}
