package feishu

import (
	"testing"
)

func TestNewWebhookNotifier(t *testing.T) {
	n := NewWebhookNotifier("https://open.feishu.cn/open-apis/bot/v2/hook/test123", "test-app")
	if n == nil {
		t.Fatal("NewWebhookNotifier returned nil")
	}
}

func TestNewWebhookNotifierEmptyURL(t *testing.T) {
	n := NewWebhookNotifier("", "")
	if n == nil {
		t.Fatal("NewWebhookNotifier returned nil even with empty URL")
	}
}
