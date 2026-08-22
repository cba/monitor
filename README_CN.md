# Monitor

[English](./README.md)

轻量级服务监控工具，支持多种监控类型和通知渠道，基于 YAML 配置文件，支持热更新。

## 功能特性

- **12 种监控类型**：HTTP/HTTPS、TCP 端口、ICMP Ping、SSL 证书、HTTP 关键字、MySQL、Redis、CPU 负载、内存使用率、磁盘使用率、进程存活、Docker 容器
- **3 种通知渠道**：企业微信 Webhook、企业微信应用消息、钉钉
- **热更新**：修改配置文件后自动重载，无需重启
- **独立配置**：每个监控项独立设置检查间隔和报警间隔
- **插件架构**：易于扩展新的监控类型和通知渠道

## 安装

### 编译安装

```bash
git clone https://github.com/cba/monitor.git
cd monitor
go build -o monitor .
```

### 安装到系统路径

```bash
go install .
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
monitors:
  - name: 官网
    type: http
    target: https://example.com
    interval: 60
    alert_interval: 300
    enabled: true

notifiers:
  - name: 运维群
    type: wechat
    webhook: https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=YOUR_KEY
    enabled: true
```

### 监控配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 监控名称（唯一标识） |
| type | string | 是 | 监控类型 |
| target | string | 是 | 监控目标 |
| interval | int | 否 | 检查间隔（秒），默认 60 |
| alert_interval | int | 否 | 报警间隔（秒），默认 300 |
| enabled | bool | 否 | 是否启用，默认 true |

### 通知配置

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 是 | 通知渠道名称 |
| type | string | 是 | 通知类型（wechat/dingtalk） |
| webhook | string | 否 | Webhook URL（wechat/dingtalk 必填） |
| enabled | bool | 否 | 是否启用，默认 true |
| extra | map | 否 | 附加参数（wechat_app 必填，见下方说明） |

## 监控类型

### HTTP/HTTPS 网址

监控网页是否可访问。

```yaml
- name: 官网
  type: http
  target: https://example.com
  interval: 60
  alert_interval: 300
```

- **target**：完整的 URL（包含 http:// 或 https://）
- **判断标准**：状态码 2xx/3xx 为 up，其他为 down

### TCP 端口

监控 TCP 端口是否可连接。

```yaml
- name: Redis服务
  type: tcp
  target: 192.168.1.100:6379
  interval: 30
  alert_interval: 120
```

- **target**：`host:port` 格式
- **判断标准**：TCP 连接成功为 up

### ICMP Ping

监控服务器是否可达。

```yaml
- name: 服务器存活
  type: icmp
  target: 192.168.1.1
  interval: 60
  alert_interval: 300
```

- **target**：IP 地址或域名
- **判断标准**：ping 成功为 up

### HTTPS 证书

监控 SSL 证书是否有效。

```yaml
- name: SSL证书
  type: ssl
  target: example.com:443
  interval: 3600
  alert_interval: 86400
```

- **target**：`域名:端口` 格式，端口默认 443
- **判断标准**：
  - up：证书有效且剩余有效期 > 7 天
  - warning：证书有效但剩余有效期 ≤ 7 天
  - down：证书已过期或无证书

### HTTP 关键字

监控网页是否包含指定关键字。

```yaml
- name: 首页关键词
  type: keyword
  target: "https://example.com|Hello World"
  interval: 120
  alert_interval: 600
```

- **target**：`URL|关键字` 格式（用 `|` 分隔）
- **判断标准**：响应体包含关键字为 up

### MySQL

监控 MySQL 数据库连接。

```yaml
- name: MySQL数据库
  type: mysql
  target: "user:password@tcp(127.0.0.1:3306)/mydb"
  interval: 60
  alert_interval: 300
```

- **target**：MySQL DSN 格式
- **判断标准**：连接成功为 up

### Redis

监控 Redis 服务连接。

```yaml
- name: Redis服务
  type: redis
  target: 127.0.0.1:6379
  interval: 30
  alert_interval: 120
```

- **target**：`host:port` 格式
- **判断标准**：PING 命令成功为 up

### CPU 负载

监控服务器 CPU 负载。

```yaml
- name: 服务器CPU
  type: cpu_load
  target: cpu_load
  interval: 60
  alert_interval: 300
```

- **target**：`cpu_load` 或 `user:pass@host,cpu_load[,warning,critical]`
- **判断标准**：1分钟 load average 低于 warning 为 up，超过为 warning，超过 critical 为 down
- **默认阈值**：warning=5.0，critical=10.0

### 内存使用率

监控服务器内存使用率。

```yaml
- name: 服务器内存
  type: memory
  target: memory
  interval: 60
  alert_interval: 300
```

- **target**：`memory` 或 `user:pass@host,memory[,warning,critical]`
- **判断标准**：使用率低于 warning 为 up，超过为 warning，超过 critical 为 down
- **默认阈值**：warning=80%，critical=90%

### 磁盘使用率

监控磁盘分区使用率。

```yaml
- name: 根分区
  type: disk
  target: "disk,/,80,90"
  interval: 300
  alert_interval: 600
```

- **target**：`disk,mountpoint[,warning,critical]` 或 `user:pass@host,disk,mountpoint[,warning,critical]`
- **判断标准**：使用率低于 warning 为 up，超过为 warning，超过 critical 为 down
- **默认阈值**：warning=80%，critical=90%

### 进程存活

监控指定进程是否在运行。

```yaml
- name: Nginx进程
  type: process
  target: "process,nginx"
  interval: 30
  alert_interval: 120
```

- **target**：`process,name` 或 `user:pass@host,process,name`
- **判断标准**：进程存在为 up，不存在为 down

### Docker 容器

监控 Docker 容器是否在运行。

```yaml
- name: Redis容器
  type: container
  target: "container,redis"
  interval: 60
  alert_interval: 300
```

- **target**：`container,name` 或 `user:pass@host,container,name`
- **判断标准**：容器运行中为 up，停止或不存在为 down

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
  enabled: true
  extra:
    corp_id: YOUR_CORP_ID
    agent_id: YOUR_AGENT_ID
    secret: YOUR_SECRET
    to_users: "UserID1|UserID2"
```

**extra 字段说明：**

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
    "github.com/cba/monitor/internal/monitor"
)

type yourMonitor struct{}

func init() {
    monitor.Register(&yourMonitor{})
}

func (m *yourMonitor) Name() string { return "your_type" }

func (m *yourMonitor) Check(ctx context.Context, target string) (*monitor.Result, error) {
    // 实现检查逻辑
    return &monitor.Result{
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
    // 实现发送逻辑
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
