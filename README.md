# Monitor

[中文](./README_CN.md)

A lightweight service monitoring tool with multiple monitor types and notification channels. YAML-based configuration with hot reload.

## Features

- **12 monitor types**: HTTP/HTTPS, TCP port, ICMP Ping, SSL certificate, HTTP keyword, MySQL, Redis, CPU load, Memory, Disk, Process, Docker container
- **3 notification channels**: WeChat Work Webhook, WeChat Work App API, DingTalk
- **Daily report**: scheduled availability & latency summary pushed to configured channels
- **SSH remote monitoring**: most types can run through an SSH bastion; connections to the same bastion are shared and reused
- **Hot reload**: Auto-reload on config file changes, no restart needed
- **Global defaults**: Per-monitor intervals optional, inherit from defaults
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
defaults:
  interval: 60
  alert_interval: 300

monitors:
  - name: example.com
    type: http
    url: https://example.com

  - name: Redis
    type: redis
    host: 127.0.0.1
    port: "6379"
    interval: 30
    alert_interval: 120

notifiers:
  - name: ops-team
    type: wechat
    webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY

reporter:
  enabled: true
  cron: "30 9 * * *"   # every day at 09:30
  title: "Daily Monitor Report"
```

### Defaults

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| interval | int | 60 | Default check interval (seconds) |
| alert_interval | int | 300 | Default alert interval (seconds) |
| enabled | bool | true | Default enabled state |

### Monitor Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Monitor name (unique identifier) |
| type | string | Yes | Monitor type |
| enabled | bool | No | Enable monitor, default true |
| interval | int | No | Check interval (seconds), inherits from defaults |
| alert_interval | int | No | Alert interval (seconds), inherits from defaults |
| url | string | HTTP/Keyword | URL to monitor |
| keyword | string | Keyword | Keyword to search for |
| host | string | TCP/SSL/Redis/ICMP(remote) | Host address |
| port | string | TCP/SSL/Redis | Port number |
| target | string | ICMP | IP address or hostname |
| dsn | string | MySQL | MySQL DSN string |
| path | string | Disk | Mount point (default `/`) |
| warn | float64 | CPU/Memory/Disk | Warning threshold |
| crit | float64 | CPU/Memory/Disk | Critical threshold |
| process_name | string | Process | Process name to check |
| container_name | string | Container | Docker container name |
| password | string | Redis | Redis password |
| ssh.host | string | Remote monitors | SSH jump host (defaults to `host`) |
| ssh.user | string | Remote monitors | SSH username |
| ssh.password | string | Remote monitors | SSH password |
| ssh.key_file | string | Remote monitors | SSH private key path |
| ssh.cert_file | string | Remote monitors | SSH certificate path |

### Notifier Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| name | string | Yes | Notifier name |
| type | string | Yes | Notifier type (wechat/wechat_app/dingtalk) |
| webhook | string | Webhook URL (required for wechat/dingtalk) |
| enabled | bool | No | Enable notifier, default true |
| corp_id | string | wechat_app | Corp ID |
| agent_id | string | wechat_app | Application AgentId |
| secret | string | wechat_app | Application Secret |
| to_users | string | wechat_app | Recipient UserIDs, separated by `\|`, `@all` for everyone |

### Reporter Config

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| enabled | bool | no | Enable daily report, default false |
| cron | string | yes | Send time, only `min hour * * *` supported (fixed daily time) |
| title | string | no | Report title, defaults to `每日监控日报` |
| targets | list | no | Reserved; all monitors are included currently |

Check results are persisted per day as JSON under `data/reporter/` in the process working directory; the report is sent through all enabled notifiers.

## Monitor Types

### HTTP/HTTPS

Monitor if a webpage is accessible.

```yaml
- name: example.com
  type: http
  url: https://example.com
```

- **url**: Full URL (must include http:// or https://)
- **Status**: 2xx/3xx = up, otherwise down

### TCP Port

Monitor if a TCP port is reachable.

```yaml
- name: Redis
  type: tcp
  host: 192.168.1.100
  port: "6379"
```

- **host**: IP address or hostname
- **port**: Port number
- **Status**: Connection success = up

### ICMP Ping

Monitor if a server is reachable.

```yaml
# Local
- name: Server Alive
  type: icmp
  target: 192.168.1.1

# Remote (ping via SSH)
- name: Remote Server Alive
  type: icmp
  target: 192.168.1.1
  ssh:
    user: root
    host: 192.168.1.100
    key_file: /root/.ssh/id_rsa
```

- **target**: IP address or hostname
- **ssh.host**: SSH jump host (required for remote monitoring)
- **Status**: Ping success = up

### SSL Certificate

Monitor if an SSL certificate is valid.

```yaml
- name: SSL Cert
  type: ssl
  host: example.com
  port: "443"
  interval: 3600
  alert_interval: 86400
```

- **host**: Domain name
- **port**: Port number (default 443)
- **Status**:
  - up: Valid certificate with > 7 days remaining
  - warning: Valid certificate with <= 7 days remaining
  - down: Expired or no certificate

### HTTP Keyword

Monitor if a webpage contains a specific keyword.

```yaml
- name: Homepage Keyword
  type: keyword
  url: https://example.com
  keyword: Hello World
```

- **url**: URL to check
- **keyword**: Keyword to search for in response body
- **Status**: Response body contains keyword = up

### MySQL

Monitor MySQL database connectivity.

```yaml
# Local
- name: MySQL
  type: mysql
  dsn: "user:password@tcp(127.0.0.1:3306)/mydb"

# Remote (SSH jump host differs from DSN host)
- name: Remote MySQL
  type: mysql
  dsn: "user:password@tcp(10.0.0.5:3306)/mydb"
  ssh:
    user: root
    host: 124.223.224.4
    key_file: /root/.ssh/id_rsa
```

- **dsn**: MySQL DSN format (`user:password@tcp(host:port)/dbname`)
- **ssh.host**: SSH jump host (required when different from DSN host)
- **Status**: Connection success = up

### Redis

Monitor Redis service connectivity.

```yaml
# Local
- name: Redis
  type: redis
  host: 127.0.0.1
  port: "6379"

# Local (with password)
- name: Redis
  type: redis
  host: 127.0.0.1
  port: "6379"
  password: your_password

# Remote (SSH jump host differs from Redis host)
- name: Remote Redis
  type: redis
  host: 10.0.0.5
  port: "6379"
  password: your_password
  ssh:
    user: root
    host: 124.223.224.4
    key_file: /root/.ssh/id_rsa
```

- **host**: Redis server address
- **port**: Port number
- **password**: Redis password (omit if none)
- **ssh.host**: SSH jump host (required when different from `host`)
- **Status**: PING command success = up

### CPU Load

Monitor server CPU load average.

```yaml
# Local
- name: Server CPU
  type: cpu_load
  warn: 5
  crit: 10

# Remote (password auth)
- name: Remote CPU
  type: cpu_load
  host: 192.168.1.100
  warn: 5
  crit: 10
  ssh:
    user: root
    password: your_password

# Remote (key auth)
- name: Remote CPU (key)
  type: cpu_load
  host: 192.168.1.100
  warn: 5
  crit: 10
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **warn**: Warning threshold (default 5.0)
- **crit**: Critical threshold (default 10.0)
- **ssh**: SSH config for remote monitoring (optional)
- **Status**: 1-min load average below warning = up, above = warning, above critical = down

### Memory Usage

Monitor server memory usage.

```yaml
# Local
- name: Server Memory
  type: memory
  warn: 80
  crit: 90

# Remote
- name: Remote Memory
  type: memory
  host: 192.168.1.100
  warn: 80
  crit: 90
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **warn**: Warning threshold (default 80%)
- **crit**: Critical threshold (default 90%)
- **Status**: Usage below warning = up, above = warning, above critical = down

### Disk Usage

Monitor disk partition usage.

```yaml
# Local
- name: Root Partition
  type: disk
  path: /
  warn: 80
  crit: 90

# Remote
- name: Remote Disk
  type: disk
  host: 192.168.1.100
  path: /
  warn: 80
  crit: 90
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **path**: Mount point (default `/`)
- **warn**: Warning threshold (default 80%)
- **crit**: Critical threshold (default 90%)
- **Status**: Usage below warning = up, above = warning, above critical = down

### Process Alive

Monitor if a specific process is running.

```yaml
# Local
- name: Nginx Process
  type: process
  process_name: nginx

# Remote
- name: Remote Nginx
  type: process
  host: 192.168.1.100
  process_name: nginx
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **process_name**: Name of the process to check
- **Status**: Process exists = up, not found = down

### Docker Container

Monitor if a Docker container is running.

```yaml
# Local
- name: Redis Container
  type: container
  container_name: redis

# Remote
- name: Remote Redis Container
  type: container
  host: 192.168.1.100
  container_name: redis
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **container_name**: Docker container name
- **Status**: Container running = up, stopped or not found = down

## SSH Connection Reuse

Remote monitors sharing an `ssh` bastion share one SSH connection:

- Multiple monitors and check cycles reuse a single connection instead of doing a full TCP + auth handshake every check
- Broken connections are evicted and rebuilt; a fresh dial whose handshake gets randomly RST (e.g. sshd `MaxStartups` early-dropping concurrent unauthenticated sessions) is retried once to avoid false alarms
- If a public server still flakes alerts, raise `MaxStartups` in its `/etc/ssh/sshd_config` (e.g. `100:30:200`) and reload sshd

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
  corp_id: YOUR_CORP_ID
  agent_id: YOUR_AGENT_ID
  secret: YOUR_SECRET
  to_users: "UserID1|UserID2"
```

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
- Notifier / reporter config changed: takes effect immediately
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
    "github.com/cba/monitor/internal/config"
)

type yourMonitor struct{}

func init() {
    Register(&yourMonitor{})
}

func (m *yourMonitor) Name() string { return "your_type" }

func (m *yourMonitor) Check(ctx context.Context, cfg *config.MonitorConfig) (*Result, error) {
    return &Result{
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
