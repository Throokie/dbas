package feishu

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"reasonix/dbas/internal/dbas/kanban"
)

// WebhookNotifier 通过飞书 Webhook 发送看板卡片。
// 这是最简单的飞书集成方式，不需要复用 Reasonix 的完整 Bot Gateway。
type WebhookNotifier struct {
	webhookURL string
	appSecret  string
	client     *http.Client
}

// NewWebhookNotifier 初始化飞书 webhook 通知器。
func NewWebhookNotifier(webhookURL, appSecret string) *WebhookNotifier {
	return &WebhookNotifier{
		webhookURL: webhookURL,
		appSecret:  appSecret,
		client:     &http.Client{Timeout: 10 * time.Second},
	}
}

// Notify 发送看板卡片到飞书。
func (n *WebhookNotifier) Notify(card *kanban.KanbanCard) error {
	if n.webhookURL == "" {
		return fmt.Errorf("webhook URL not configured")
	}

	formatter := kanban.FeishuCardFormatter{}
	cardJSON := formatter.Format(card)

	payload := map[string]interface{}{
		"msg_type": "interactive",
		"card":     json.RawMessage(cardJSON),
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload: %w", err)
	}

	req, err := http.NewRequest("POST", n.webhookURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := n.client.Do(req)
	if err != nil {
		return fmt.Errorf("send webhook: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

// BotGatewayNotifier 复用 Reasonix 的 Bot Gateway 发送消息。
// 需要外部传入适配器实例。
type BotGatewayNotifier struct {
	adapter BotAdapter
}

// BotAdapter 是对 Reasonix bot.Adapter 的简化接口。
type BotAdapter interface {
	Platform() string
	Send(ctx context.Context, msg OutboundMessage) (string, error)
}

// OutboundMessage 对应 Reasonix 的 bot.OutboundMessage。
type OutboundMessage struct {
	ConnectionID string
	Domain       string
	ChatID       string
	ChatType     string
	Text         string
	ReplyToMsgID string
	Card         *InteractiveCard
}

// InteractiveCard 对应飞书交互式卡片。
type InteractiveCard struct {
	Header   string
	Elements []InteractiveCardElement
}

// InteractiveCardElement 是卡片元素。
type InteractiveCardElement struct {
	Tag     string
	Content string
	Extra   map[string]any
}

// NewBotGatewayNotifier 初始化复用 Reasonix Bot Gateway 的通知器。
func NewBotGatewayNotifier(adapter BotAdapter) *BotGatewayNotifier {
	return &BotGatewayNotifier{adapter: adapter}
}

// Notify 通过 Bot Gateway 发送看板卡片。
func (n *BotGatewayNotifier) Notify(card *kanban.KanbanCard) error {
	if n.adapter == nil {
		return fmt.Errorf("bot adapter not configured")
	}

	formatter := kanban.FeishuCardFormatter{}
	cardJSON := formatter.Format(card)

	msg := OutboundMessage{
		Text: fmt.Sprintf("DBAS Kanban Update | %s", card.ProjectID),
		Card: &InteractiveCard{
			Header: "DBAS 看板更新",
			Elements: []InteractiveCardElement{
				{Tag: "markdown", Content: cardJSON},
			},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_, err := n.adapter.Send(ctx, msg)
	return err
}
