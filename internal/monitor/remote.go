package monitor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type remoteExec struct {
	User string
	Pass string
	Host string
}

// parseTarget 解析 target 字符串。
// 格式: "type,params..." 或 "user:pass@host,type,params..."
func parseTarget(target string) (remoteExec, string) {
	atIdx := strings.LastIndex(target, "@")
	if atIdx < 0 {
		return remoteExec{}, target
	}

	prefix := target[:atIdx]
	commaIdx := strings.Index(target[atIdx:], ",")
	if commaIdx < 0 {
		return remoteExec{}, target
	}

	var user, pass string
	colonIdx := strings.Index(prefix, ":")
	if colonIdx < 0 {
		user = prefix
	} else {
		user = prefix[:colonIdx]
		pass = prefix[colonIdx+1:]
	}

	hostAndRest := target[atIdx+1:]
	nextComma := strings.Index(hostAndRest, ",")
	if nextComma < 0 {
		return remoteExec{User: user, Pass: pass, Host: hostAndRest}, ""
	}

	host := hostAndRest[:nextComma]
	rest := hostAndRest[nextComma+1:]
	return remoteExec{User: user, Pass: pass, Host: host}, rest
}

// execCommand 执行命令，本机走 exec.Command，远程走 SSH
func execCommand(ctx context.Context, re remoteExec, cmd string) (string, error) {
	if re.User == "" {
		return localExec(ctx, cmd)
	}
	return sshExec(ctx, re.User, re.Pass, re.Host, cmd)
}

func localExec(ctx context.Context, cmd string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	out, err := exec.CommandContext(ctx, "sh", "-c", cmd).CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("local exec: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return string(out), nil
}
