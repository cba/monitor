# Monitor

[English](./README.md)

轻量级服务监控工具，支持多种监控类型和通知渠道，基于 YAML 配置文件，支持热更新。

## 功能特性

- **12 种监控类型**：HTTP/HTTPS、TCP 端口、ICMP Ping、SSL 证书、HTTP 关键字、MySQL、Redis、CPU 负载、内存使用率、磁盘使用率、进程存活、Docker 容器
- **3 种通知渠道**：企业微信 Webhook、企业微信应用消息、钉钉
- **每日日报**：定时汇总各监控项可用率与平均延迟，推送到已配置的通知渠道
- **SSH 远程监控**：多数类型可经 SSH 跳板执行，同一跳板共享连接并自动复用
- **热更新**：修改配置文件后自动重载，无需重启
- **全局默认值**：监控项可继承全局 interval/alert_interval 配置
- **插件架构**：易于扩展新的监控类型和通知渠道

## 安装

```bash
go install github.com/cba/monitor@latest
```

## 快速开始

### 1. 初始化配置

```bash
monitor init
```

这会在 `~/.monitor/` 目录创建默认配置文件。

### 2. 编辑配置

```bash
vim ~/.monitor/config.yaml
```

### 3. 启动服务

```bash
monitor serve
```

## 命令行

```
monitor [command]

Available Commands:
  init        生成默认配置文件
  serve       启动监控服务
  monitor     管理监控项
  notifier    管理通知渠道
```

### 全局参数

```
-c, --config string   配置文件路径 (default "config.yaml")
```

### 监控管理

```bash
# 列出所有监控
monitor monitor list

# 测试指定监控
monitor monitor test <监控名称>

# 列出支持的监控类型
monitor monitor types
```

### 通知管理

```bash
# 列出所有通知渠道
monitor notifier list

# 列出支持的通知类型
monitor notifier types
```

## 配置文件

### 配置文件位置（按优先级）

1. `-c` 参数指定的路径
2. 当前目录 `./config.yaml`
3. Home 目录 `~/.monitor/config.yaml`

### 配置示例

```yaml
defaults:
  interval: 60
  alert_interval: 300

monitors:
  - name: 官网
    type: http
    url: https://example.com

  - name: 通知群is服务
    type: redis
    host: 127.0.0.1
    port: "6379"
    interval: 30
    alert_interval: 120

notifiers:
  - name: 运维群
    type: wechat
    webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY

reporter:
  enabled: true
  cron: "30 9 * * *"   # 每天 09:30
  title: "监控日报"
```

### 全局默认值

| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| interval | int | 60 | 默认检查间隔（秒） |
| alert_interval | int | 300 | 默认报警间隔（秒） |
| enabled | bool | true | 默认启用状态 |

### 监控配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 监控名称（唯一标识） |
| type | string | 是 | 监控类型 |
| enabled | bool | 否 | 是否启用，默认 true |
| interval | int | 否 | 检查间隔（秒），继承全局默认值 |
| alert_interval | int | 否 | 报警间隔（秒），继承全局默认值 |
| url | string | HTTP/Keyword | 监控 URL |
| keyword | string | Keyword | 关键字 |
| host | string | TCP/SSL/Redis/ICMP(remote) | 主机地址 |
| port | string | TCP/SSL/Redis | 端口号 |
| target | string | ICMP | IP 地址或域名 |
| dsn | string | MySQL | MySQL DSN 字符串 |
| path | string | 磁盘 | 挂载点（默认 `/`） |
| warn | float64 | CPU/内存/磁盘 | 警告阈值 |
| crit | float64 | CPU/内存/磁盘 | 严重阈值 |
| process_name | string | 进程 | 进程名称 |
| container_name | string | 容器 | Docker 容器名称 |
| password | string | Redis | Redis 密码 |
| ssh.host | string | 远程监控 | SSH 跳板机地址（默认用 `host`） |
| ssh.user | string | 远程监控 | SSH 用户名 |
| ssh.password | string | 远程监控 | SSH 密码 |
| ssh.key_file | string | 远程监控 | SSH 私钥路径 |
| ssh.cert_file | string | 远程监控 | SSH 证书路径 |

### 通知配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 通知渠道名称 |
| type | string | 是 | 通知类型（wechat/wechat_app/dingtalk） |
| webhook | string | 否 | Webhook URL（wechat/dingtalk 必填） |
| enabled | bool | 否 | 是否启用，默认 true |
| corp_id | string | wechat_app | 企业 ID |
| agent_id | string | wechat_app | 应用 AgentId |
| secret | string | wechat_app | 应用 Secret |
| to_users | string | wechat_app | 接收人 UserID，多个用 `\|` 分隔，`@all` 发送所有人 |

### 日报配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| enabled | bool | 否 | 是否启用日报，默认 false |
| cron | string | 是 | 发送时间，仅支持 `分 时 * * *`（每天固定时刻） |
| title | string | 否 | 日报标题，默认 `每日监控日报` |
| targets | list | 否 | 预留字段，当前统计全部监控项 |

检查结果按天以 JSON 存放在进程工作目录 `data/reporter/` 下；日报复用 `notifiers` 中所有启用的渠道。

一键启用/禁用日报（热更新即时生效，无需重启）：

```yaml
# 启用
reporter:
  enabled: true
  cron: "30 9 * * *"

# 禁用（保留 cron 等配置以便随时恢复）
reporter:
  enabled: false
```

## 监控类型

### HTTP/HTTPS 网址

监控网页是否可访问。

```yaml
- name: 官网
  type: http
  url: https://example.com
```

- **url**：完整的 URL（包含 http:// 或 https://）
- **判断标准**：状态码 2xx/3xx 为 up，其他为 down

### TCP 端口

监控 TCP 端口是否可连接。

```yaml
- name: Redis服务
  type: tcp
  host: 192.168.1.100
  port: "6379"
```

- **host**：IP 地址或域名
- **port**：端口号
- **判断标准**：TCP 连接成功为 up

### ICMP Ping

监控服务器是否可达。

```yaml
# 本机
- name: 服务器存活
  type: icmp
  target: 192.168.1.1

# 远程（通过 SSH 执行 ping）
- name: 远程服务器存活
  type: icmp
  target: 192.168.1.1
  ssh:
    user: root
    host: 192.168.1.100
    key_file: /root/.ssh/id_rsa
```

- **target**：IP 地址或域名
- **ssh.host**：SSH 跳板机地址（远程监控时必填）
- **判断标准**：ping 成功为 up

### HTTPS 证书

监控 SSL 证书是否有效。

```yaml
- name: SSL证书
  type: ssl
  host: example.com
  port: "443"
  interval: 3600
  alert_interval: 86400
```

- **host**：域名
- **port**：端口号（默认 443）
- **判断标准**：
  - up：证书有效且剩余有效期 > 7 天
  - warning：证书有效但剩余有效期 ≤ 7 天
  - down：证书已过期或无证书

### HTTP 关键字

监控网页是否包含指定关键字。

```yaml
- name: 首页关键词
  type: keyword
  url: https://example.com
  keyword: Hello World
```

- **url**：监控 URL
- **keyword**：要搜索的关键字
- **判断标准**：响应体包含关键字为 up

### MySQL

监控 MySQL 数据库连接。

```yaml
# 本机
- name: MySQL数据库
  type: mysql
  dsn: "user:password@tcp(127.0.0.1:3306)/mydb"

# 远程（SSH 跳板机与 DSN 中 host 不同）
- name: 远程MySQL
  type: mysql
  dsn: "user:password@tcp(10.0.0.5:3306)/mydb"
  ssh:
    user: root
    host: 124.223.224.4
    key_file: /root/.ssh/id_rsa
```

- **dsn**：MySQL DSN 格式（`user:password@tcp(host:port)/dbname`）
- **ssh.host**：SSH 跳板机地址（与 DSN 中 host 不同时必填）
- **判断标准**：连接成功为 up

### Redis

监控 Redis 服务连接。

```yaml
# 本机
- name: Redis服务
  type: redis
  host: 127.0.0.1
  port: "6379"

# 本机（带密码）
- name: Redis服务
  type: redis
  host: 127.0.0.1
  port: "6379"
  password: your_password

# 远程（SSH 跳板机与 Redis 地址不同）
- name: 远程Redis
  type: redis
  host: 10.0.0.5
  port: "6379"
  password: your_password
  ssh:
    user: root
    host: 124.223.224.4
    key_file: /root/.ssh/id_rsa
```

- **host**：Redis 服务器地址
- **port**：端口号
- **password**：Redis 密码（无密码可省略）
- **ssh.host**：SSH 跳板机地址（与 `host` 不同时必填）
- **判断标准**：PING 命令成功为 up

### CPU 负载

监控服务器 CPU 负载。

```yaml
# 本机
- name: 服务器CPU
  type: cpu_load
  warn: 5
  crit: 10

# 远程（密码认证）
- name: 远程服务器CPU
  type: cpu_load
  host: 192.168.1.100
  warn: 5
  crit: 10
  ssh:
    user: root
    password: your_password

# 远程（密钥认证）
- name: 远程服务器CPU（密钥）
  type: cpu_load
  host: 192.168.1.100
  warn: 5
  crit: 10
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **warn**：警告阈值（默认 5.0）
- **crit**：严重阈值（默认 10.0）
- **ssh**：远程监控的 SSH 配置（可选）
- **判断标准**：1分钟 load average 低于 warning 为 up，超过为 warning，超过 critical 为 down

### 内存使用率

监控服务器内存使用率。

```yaml
# 本机
- name: 服务器内存
  type: memory
  warn: 80
  crit: 90

# 远程
- name: 远程服务器内存
  type: memory
  host: 192.168.1.100
  warn: 80
  crit: 90
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **warn**：警告阈值（默认 80%）
- **crit**：严重阈值（默认 90%）
- **判断标准**：使用率低于 warning 为 up，超过为 warning，超过 critical 为 down

### 磁盘使用通知群

监控磁盘分区使用率。

```yaml
# 本机
- name: 根分区
  type: disk
  path: /
  warn: 80
  crit: 90

# 远程
- name: 远程服务器磁盘
  type: disk
  host: 192.168.1.100
  path: /
  warn: 80
  crit: 90
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **path**：挂载点（默认 `/`）
- **warn**：警告阈值（默认 80%）
- **crit**：严重阈值（默认 90%）
- **判断标准**：使用率低于 warning 为 up，超过为 warning，超过 critical 为 down

### 进程存活

监控指定进程是否在运行。

```yaml
# 本机
- name: Nginx进程
  type: process
  process_name: nginx

# 远程
- name: 远程Nginx进程
  type: process
  host: 192.168.1.100
  process_name: nginx
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **process_name**：进程名称
- **判断标准**：进程存在为 up，不存在为 down

### Docker 容器

监控 Docker 容器是否在运行。

```yaml
# 本机
- name: Redis容器
  type: container
  container_name: redis

# 远程
- name: 远程Redis容器
  type: container
  host: 192.168.1.100
  container_name: redis
  ssh:
    user: root
    key_file: /root/.ssh/id_rsa
```

- **container_name**：Docker 容器名称
- **判断标准**：容器运行中为 up，停止或不存在为 down

## SSH 连接复用

配置了 `ssh` 的远程监控共享同一跳板机的 SSH 连接：

- 多个监控项、多个检查周期复用一条连接，避免每次检查都做完整的 TCP + 认证握手
- 连接损坏或命令出错时自动剔除并重建；新建时握手偶发被 RST（如 sshd `MaxStartups` 随机丢弃并发未认证连接）会立即重试一次，减少误报
- 公网服务器上 22 端口常有爆破流量占满未认证连接名额，若仍偶发误报，可调大其 `/etc/ssh/sshd_config` 中的 `MaxStartups`（如 `100:30:200`）并 reload sshd

## 通知渠道

### 企业微信

```yaml
- name: 运维群
  type: wechat
  webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY
```

获取 Webhook URL：
1. 在企业微信群中添加群机器人
2. 复制机器人 Webhook 地址

### 企业微信应用消息

通过企业微信应用消息 API 发送通知，支持指定接收人。

```yaml
- name: 运维通知
  type: wechat_app
  corp_id: YOUR_CORP_ID
  agent_id: YOUR_AGENT_ID
  secret: YOUR_SECRET
  to_users: "UserID1|UserID2"
```

| 字段 | 必填 | 说明 |
|------|------|------|
| corp_id | 是 | 企业 ID |
| agent_id | 是 | 应用 AgentId |
| secret | 是 | 应用 Secret |
| to_users | 否 | 接收人 UserID，多个用 `\|` 分隔，`@all` 发送所有人 |

### 钉钉

```yaml
- name: 钉钉群
  type: dingtalk
  webhook: https://oapi.dingtalk.com/robot/send?access_token=YOUR_TOKEN
```

获取 Webhook URL：
1. 在钉钉群中添加自定义机器人
2. 复制机器人 Webhook 地址

## 热更新

服务运行时修改配置文件会自动重载：

- 添加新监控：自动启动
- 删除监控：自动停止
- 修改监控配置：自动重启
- 修改通知/日报配置：即时生效
- 配置格式错误：保持旧配置，记录错误日志

```bash
# 查看重载日志
2026/08/21 22:31:46 config changed, reloading...
2026/08/21 22:31:46 [http] 新网站 started (check=60s, alert=300s)
2026/08/21 22:31:46 scheduler: 2 monitors active
```

## 报警策略

- **立即报警**：监控失败时立即发送通知
- **间隔重复**：按 `alert_interval` 设置的间隔重复报警
- **恢复通知**：监控恢复时自动发送恢复通知

## 扩展

### 添加新的监控类型

1. 创建文件 `internal/monitor/your_type.go`

2. 实现 `Monitor` 接口：

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

3. 在 `cmd/monitor/main.go` 中导入（已自动导入 `internal/monitor`）

### 添加新的通知渠道

1. 创建文件 `internal/notifier/your_type.go`

2. 实现 `Notifier` 接口：

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

3. 在 `internal/scheduler/scheduler.go` 的 `notifyAll` 方法中添加新的 case

## 后台运行

### 使用 systemd

创建 `/etc/systemd/system/monitor.service`：

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

### 使用 nohup

```bash
nohup monitor serve > /var/log/monitor.log 2>&1 &
```

## 日志

```bash
# 查看服务日志
journalctl -u monitor -f

# 或查看 nohup 输出
tail -f /var/log/monitor.log
```

## License

MIT
