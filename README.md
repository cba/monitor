# Monitor

[中文](./README_CN.md)

A lightweight service monitoring tool with multiple monitor types and notification channels. YAML-based configuration with hot reload.

## Features

- **12 monitor types**: HTTP/HTTPS, TCP port, ICMP Ping, SSL certificate, HTTP keyword, MySQL, Redis, CPU load, Memory, Disk, Process, Docker container
- **3 notification channels**: WeChat Work Webhook, WeChat Work App API, DingTalk
- **Hot reload**: Auto-reload on config file changes, no restart needed
- **Independent config**: Per-monitor check interval and alert interval
- **Plugin architecture**: Easy to extend with new monitor types and notification channels

## Installation

```bash
go install github.com/cba/monitor@latest
```

## Quick Start

### 1. Initialize config

```bash
monitor init
```

This creates a default config file in `~/.monitor/`.

### 2. Edit config

```bash
vim ~/.monitor/config.yaml
```

### 3. Start service

```bash
monitor serve
```

## CLI

```
monitor [command]

Available Commands:
  init        Generate default config file
  serve       Start monitoring service
  monitor     Manage monitors
  notifier    Manage notifiers
```

### Global Flags

```
-c, --config string   Config file path (default "config.yaml")
```

### Monitor Management

```bash
# List all monitors
monitor monitor list

# Test a specific monitor
monitor monitor test <name>

# List supported monitor types
monitor monitor types
```

### Notifier Management

```bash
# List all notifiers
monitor notifier list

# List supported notifier types
monitor notifier types
```

## Configuration

### Config File Location (by priority)

1. Path specified by `-c` flag
2. Current directory `./config.yaml`
3. Home directory `~/.monitor/config.yaml`

### Example Config

```yaml
monitors:
  - name: example.com
    type: http
    target: https://example.com
    interval: 60
    alert_interval: 300
    enabled: true

notifiers:
  - name: ops-team
    type: wechat
    webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY
    enabled: true
```

### Monitor Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Monitor name (unique identifier) |
| type | string | Yes | Monitor type |
| target | string | Yes | Monitor target |
| interval | int | No | Check interval in seconds, default 60 |
| alert_interval | int | No | Alert interval in seconds, default 300 |
| enabled | bool | No | Enable monitor, default true |

### Notifier Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Notifier name |
| type | string | Yes | Notifier type (wechat/wechat_app/dingtalk) |
| webhook | string | No | Webhook URL (required for wechat/dingtalk) |
| enabled | bool | No | Enable notifier, default true |
| extra | map | No | Additional params (required for wechat_app, see below) |

## Monitor Types

### HTTP/HTTPS

Monitor if a webpage is accessible.

```yaml
- name: example.com
  type: http
  target: https://example.com
  interval: 60
  alert_interval: 300
```

- **target**: Full URL (must include http:// or https://)
- **Status**: 2xx/3xx = up, otherwise down

### TCP Port

Monitor if a TCP port is reachable.

```yaml
- name: Redis
  type: tcp
  target: 192.168.1.100:6379
  interval: 30
  alert_interval: 120
```

- **target**: `host:port` format
- **Status**: Connection success = up

### ICMP Ping

Monitor if a server is reachable.

```yaml
- name: Server Alive
  type: icmp
  target: 192.168.1.1
  interval: 60
  alert_interval: 300
```

- **target**: IP address or hostname
- **Status**: Ping success = up

### SSL Certificate

Monitor if an SSL certificate is valid.

```yaml
- name: SSL Cert
  type: ssl
  target: example.com:443
  interval: 3600
  alert_interval: 86400
```

- **target**: `domain:port` format, port defaults to 443
- **Status**:
  - up: Valid certificate with > 7 days remaining
  - warning: Valid certificate with <= 7 days remaining
  - down: Expired or no certificate

### HTTP Keyword

Monitor if a webpage contains a specific keyword.

```yaml
- name: Homepage Keyword
  type: keyword
  target: "https://example.com|Hello World"
  interval: 120
  alert_interval: 600
```

- **target**: `URL|keyword` format (separated by `|`)
- **Status**: Response body contains keyword = up

### MySQL

Monitor MySQL database connectivity.

```yaml
- name: MySQL
  type: mysql
  target: "user:password@tcp(127.0.0.1:3306)/mydb"
  interval: 60
  alert_interval: 300
```

- **target**: MySQL DSN format
- **Status**: Connection success = up

### Redis

Monitor Redis service connectivity.

```yaml
- name: Redis
  type: redis
  target: 127.0.0.1:6379
  interval: 30
  alert_interval: 120
```

- **target**: `host:port` format
- **Status**: PING command success = up

### CPU Load

Monitor server CPU load average.

```yaml
- name: Server CPU
  type: cpu_load
  target: cpu_load
  interval: 60
  alert_interval: 300
```

- **target**: `cpu_load` or `user:pass@host,cpu_load[,warning,critical]`
- **Status**: 1-min load average below warning = up, above = warning, above critical = down
- **Defaults**: warning=5.0, critical=10.0

### Memory Usage

Monitor server memory usage.

```yaml
- name: Server Memory
  type: memory
  target: memory
  interval: 60
  alert_interval: 300
```

- **target**: `memory` or `user:pass@host,memory[,warning,critical]`
- **Status**: Usage below warning = up, above = warning, above critical = down
- **Defaults**: warning=80%, critical=90%

### Disk Usage

Monitor disk partition usage.

```yaml
- name: Root Partition
  type: disk
  target: "disk,/,80,90"
  interval: 300
  alert_interval: 600
```

- **target**: `disk,mountpoint[,warning,critical]` or `user:pass@host,disk,mountpoint[,warning,critical]`
- **Status**: Usage below warning = up, above = warning, above critical = down
- **Defaults**: warning=80%, critical=90%

### Process Alive

Monitor if a specific process is running.

```yaml
- name: Nginx Process
  type: process
  target: "process,nginx"
  interval: 30
  alert_interval: 120
```

- **target**: `process,name` or `user:pass@host,process,name`
- **Status**: Process exists = up, not found = down

### Docker Container

Monitor if a Docker container is running.

```yaml
- name: Redis Container
  type: container
  target: "container,redis"
  interval: 60
  alert_interval: 300
```

- **target**: `container,name` or `user:pass@host,container,name`
- **Status**: Container running = up, stopped or not found = down

## Notification Channels

### WeChat Work Webhook

```yaml
- name: ops-team
  type: wechat
  webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY
```

To get the Webhook URL:
1. Add a group bot in your WeChat Work group
2. Copy the bot Webhook URL

### WeChat Work App API

Send notifications via WeChat Work application message API with recipient support.

```yaml
- name: ops-notify
  type: wechat_app
  enabled: true
  extra:
    corp_id: YOUR_CORP_ID
    agent_id: YOUR_AGENT_ID
    secret: YOUR_SECRET
    to_users: "UserID1|UserID2"
```

**extra fields:**

| Field | Required | Description |
|-------|----------|-------------|
| corp_id | Yes | Corp ID |
| agent_id | Yes | Application AgentId |
| secret | Yes | Application Secret |
| to_users | No | Recipient UserIDs, separated by `\|`, use `@all` for everyone |

### DingTalk

```yaml
- name: dingtalk-group
  type: dingtalk
  webhook: https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN
```

To get the Webhook URL:
1. Add a custom bot in your DingTalk group
2. Copy the bot Webhook URL

## Hot Reload

Config changes are automatically reloaded while the service is running:

- New monitor added: auto-started
- Monitor removed: auto-stopped
- Monitor config changed: auto-restarted
- Config format error: keeps old config, logs error

```bash
# Reload log example
2026/08/21 22:31:46 config changed, reloading...
2026/08/21 22:31:46 [http] new-site started (check=60s, alert=300s)
2026/08/21 22:31:46 scheduler: 2 monitors active
```

## Alert Strategy

- **Immediate alert**: Notification sent immediately on failure
- **Interval repeat**: Alerts repeated at `alert_interval` intervals
- **Recovery notification**: Auto-sent when monitor recovers

## Extending

### Adding a New Monitor Type

1. Create file `internal/monitor/your_type.go`

2. Implement the `Monitor` interface:

```go
package monitor

import (
    "context"
    "github.com/cba/monitor/internal/monitor"
)

type yourMonitor struct{}

func init() {
    monitor.Register(&yourMonitor{})
}

func (m *yourMonitor) Name() string { return "your_type" }

func (m *yourMonitor) Check(ctx context.Context, target string) (*monitor.Result, error) {
    return &monitor.Result{
        Status:    "up",
        Message:   "OK",
        Latency:   100 * time.Millisecond,
        Timestamp: time.Now(),
    }, nil
}
```

3. Import in `main.go` (already imports `internal/monitor`)

### Adding a New Notification Channel

1. Create file `internal/notifier/your_type.go`

2. Implement the `Notifier` interface:

```go
package notifier

import "context"

type yourNotifier struct {
    webhook string
}

func NewYourNotifier(webhook string) Notifier {
    return &yourNotifier{webhook: webhook}
}

func (n *yourNotifier) Name() string { return "your_type" }

func (n *yourNotifier) Send(ctx context.Context, title, content string) error {
    return nil
}
```

3. Add a new case in `notifyAll` method in `internal/scheduler/scheduler.go`

## Running as Background Service

### Using systemd

Create `/etc/systemd/system/monitor.service`:

```ini
[Unit]
Description=Monitor
After=network.target

[Service]
Type=simple
ExecStart=/usr/local/bin/monitor serve -c /etc/monitor/config.yaml
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable monitor
sudo systemctl start monitor
```

### Using nohup

```bash
nohup monitor serve > /var/log/monitor.log 2>&1 &
```

## Logs

```bash
# View service logs
journalctl -u monitor -f

# Or check nohup output
tail -f /var/log/monitor.log
```

## License

MIT
