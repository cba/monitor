package notifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

// wechatAppNotifier sends messages via WeChat Work application message API.
type wechatAppNotifier struct {
	corpID  string
	agentID string
	secret  string
	toUsers string

	mu          sync.Mutex
	accessToken string
	tokenExpiry time.Time
}

// NewWeChatAppNotifier creates a WeChat Work application API notifier.
func NewWeChatAppNotifier(corpID, agentID, secret, toUsers string) Notifier {
	return &wechatAppNotifier{
		corpID:  corpID,
		agentID: agentID,
		secret:  secret,
		toUsers: toUsers,
	}
}

func (n *wechatAppNotifier) Name() string { return "wechat_app" }

func (n *wechatAppNotifier) getAccessToken() (string, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.accessToken != "" && time.Now().Before(n.tokenExpiry) {
		return n.accessToken, nil
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/gettoken?corpid=%s&corpsecret=%s", n.corpID, n.secret)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", fmt.Errorf("get token: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("decode token response: %w", err)
	}

	if code, ok := result["errcode"].(float64); ok && code != 0 {
		return "", fmt.Errorf("get token failed: %v", result["errmsg"])
	}

	token := result["access_token"].(string)
	n.accessToken = token
	n.tokenExpiry = time.Now().Add(90 * time.Minute)
	return token, nil
}

func (n *wechatAppNotifier) sendMsg(msgtype string, content interface{}) error {
	token, err := n.getAccessToken()
	if err != nil {
		return err
	}

	payload := map[string]interface{}{
		"touser":  n.toUsers,
		"msgtype": msgtype,
		"agentid": n.agentID,
		msgtype:   content,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/message/send?access_token=%s", token)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if code, ok := result["errcode"].(float64); ok && code != 0 {
		return fmt.Errorf("send failed: %v", result["errmsg"])
	}
	return nil
}

func (n *wechatAppNotifier) Send(ctx context.Context, title, content string) error {
	return n.sendMsg("text", map[string]string{
		"content": content,
	})
}
