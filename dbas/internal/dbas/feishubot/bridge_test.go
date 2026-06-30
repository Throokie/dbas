package feishubot

import (
	"encoding/json"
	"strings"
	"testing"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/config"
	"reasonix/dbas/internal/dbas/experienceslot"
)

// TestStripMention verifies @mention stripping.
func TestStripMention(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"@bot 你好 world", "你好 world"},
		{"@DBAS spawn test", "spawn test"},
		{"hello world", "hello world"},
		{"@bot", ""},
		{"", ""},
	}

	for _, tt := range tests {
		result := stripMention(tt.input)
		if strings.TrimSpace(result) != strings.TrimSpace(tt.expected) {
			t.Errorf("stripMention(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

// TestDispatchCommand verifies command routing with a minimal Bridge.
func TestDispatchCommand(t *testing.T) {
	slot := experienceslot.DefaultSlot()
	engine := branch.NewEngine("test-root-hash", slot)
	b := &Bridge{
		cfg:    config.FeishuBotConfig{Enabled: true},
		engine: engine,
	}

	tests := []struct {
		input    string
		contains string // substring expected in response
	}{
		{"help", "spawn"},
		{"?", "spawn"},
		{"status", "DBAS"},
		{"stats", "DBAS"},
		{"skills", "未初始化"},
		{"branches", "没有分支"},
		{"list", "没有分支"},
		{"spawn", "用法"},
		{"unknown", "未知命令"},
		{"", "请发送命令"},
	}

	for _, tt := range tests {
		result := b.dispatchCommand(tt.input)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("dispatchCommand(%q) = %q, want it to contain %q", tt.input, result, tt.contains)
		}
	}
}

// TestURLVerificationUnmarshal ensures the challenge struct parses correctly.
func TestURLVerificationUnmarshal(t *testing.T) {
	data := `{"challenge":"abc123","token":"tok","type":"url_verification"}`
	var c URLVerificationChallenge
	if err := json.Unmarshal([]byte(data), &c); err != nil {
		t.Fatal(err)
	}
	if c.Challenge != "abc123" || c.Token != "tok" || c.Type != "url_verification" {
		t.Fatalf("unexpected values: %+v", c)
	}
}

// TestFeishuEventUnmarshal verifies event parsing.
func TestFeishuEventUnmarshal(t *testing.T) {
	data := `{
		"schema": "2.0",
		"header": {
			"event_id": "evt_123",
			"event_type": "im.message.receive_v1",
			"token": "verification_token",
			"create_time": "1700000000"
		},
		"event": {
			"sender": {
				"sender_id": {
					"user_id": "user_123",
					"open_id": "open_123",
					"union_id": "union_123"
				}
			},
			"message": {
				"message_id": "msg_123",
				"chat_id": "chat_123",
				"chat_type": "p2p",
				"msg_type": "text",
				"content": "{\"text\":\"spawn test task\"}"
			}
		}
	}`

	var evt IncomingFeishuEvent
	if err := json.Unmarshal([]byte(data), &evt); err != nil {
		t.Fatal(err)
	}

	if evt.Schema != "2.0" {
		t.Errorf("schema = %q, want 2.0", evt.Schema)
	}
	if evt.Header.EventType != "im.message.receive_v1" {
		t.Errorf("event_type = %q", evt.Header.EventType)
	}

	var msgEvt MessageReceiveEvent
	if err := json.Unmarshal(evt.Event, &msgEvt); err != nil {
		t.Fatal(err)
	}
	if msgEvt.Message.MsgType != "text" {
		t.Errorf("msg_type = %q", msgEvt.Message.MsgType)
	}
	if msgEvt.Message.ChatID != "chat_123" {
		t.Errorf("chat_id = %q", msgEvt.Message.ChatID)
	}
}

// TestTextContentUnmarshal checks the nested JSON text content.
func TestTextContentUnmarshal(t *testing.T) {
	content := `{"text":"@bot spawn new branch"}`
	var tc struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(content), &tc); err != nil {
		t.Fatal(err)
	}
	if tc.Text != "@bot spawn new branch" {
		t.Errorf("text = %q", tc.Text)
	}
}
