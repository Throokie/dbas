package kanban

import (
	"fmt"
	"strings"

)

// FeishuCardFormatter 生成飞书/ Lark 交互式卡片 JSON。
// 复用 Reasonix 的 Bot Gateway 发送。
type FeishuCardFormatter struct{}

// Format 将 KanbanCard 转换为飞书卡片 JSON 字符串。
func (f *FeishuCardFormatter) Format(card *KanbanCard) string {
	var b strings.Builder

	b.WriteString(`{"config": {"wide_screen_mode": true}, "header": {"title": {"tag": "plain_text", "content": "`)
	b.WriteString(fmt.Sprintf("DBAS 看板 | %s", card.ProjectID))
	b.WriteString(`"}, "template": "blue"}, "elements": [`)

	// 统计行
	b.WriteString(fmt.Sprintf(`{"tag": "div", "text": {"tag": "lark_md", "content": "**活跃分支: %d** | **已归档: %d** | **升级阈值: %d 次**"}},`,
		len(card.ActiveBranches), card.TerminalCount, card.Slot.Rules.EscalationThreshold))

	// 每个分支一个 section
	for _, branch := range card.ActiveBranches {

		b.WriteString(fmt.Sprintf(`
{"tag": "div", "fields": [
  {"is_short": true, "text": {"tag": "lark_md", "content": "**%s**\n`+"`%s`"+`"}},
  {"is_short": true, "text": {"tag": "lark_md", "content": "**状态: %s**\n**模型: %s**\n**重试: %d**"}}
]},`,
			branch.Name, branch.ID, branch.State, branch.Model, branch.Retries))

		b.WriteString(fmt.Sprintf(`
{"tag": "actions", "actions": [
  {"tag": "button", "text": {"tag": "plain_text", "content": "重置分支"}, "type": "default", "value": {"action": "reset", "branch_id": "%s"}},
  {"tag": "button", "text": {"tag": "plain_text", "content": "查看日志"}, "type": "primary", "value": {"action": "logs", "branch_id": "%s"}}
]},`, branch.ID, branch.ID))
	}

	b.WriteString(`]}`)
	return b.String()
}

// MarkdownFormatter 生成 Markdown 格式的看板文本（用于 CLI/日志）。
type MarkdownFormatter struct{}

// Format 生成 Markdown 看板。
func (m *MarkdownFormatter) Format(card *KanbanCard) string {
	var b strings.Builder

	fmt.Fprintf(&b, "## DBAS Kanban | Project: %s\n\n", card.ProjectID)
	fmt.Fprintf(&b, "- **Active Branches:** %d\n", len(card.ActiveBranches))
	fmt.Fprintf(&b, "- **Terminal:** %d\n", card.TerminalCount)
	fmt.Fprintf(&b, "- **Escalation Threshold:** %d fails\n\n", card.Slot.Rules.EscalationThreshold)

	b.WriteString("| 分支 | 状态 | 模型 | 重试 | 日志 |\n")
	b.WriteString("|------|------|------|------|------|\n")
	for _, branch := range card.ActiveBranches {
		icon := "🟡"
		if branch.IsTerminal {
			icon = "🟢"
		}
		if branch.State == "ERROR" || branch.State == "FAIL_RETREAT" {
			icon = "🔴"
		}
		fmt.Fprintf(&b, "| %s %s | `%s` | `%s` | %d | %d |\n",
			icon, branch.Name, branch.State, branch.Model, branch.Retries, branch.LogCount)
	}
	return b.String()
}

// SimpleTextFormatter 生成纯文本看板（最简格式）。
type SimpleTextFormatter struct{}

// Format 生成纯文本看板。
func (s *SimpleTextFormatter) Format(card *KanbanCard) string {
	var b strings.Builder
	fmt.Fprintf(&b, "DBAS Kanban | %s | %d active, %d terminal\n",
		card.ProjectID, len(card.ActiveBranches), card.TerminalCount)
	for _, branch := range card.ActiveBranches {
		fmt.Fprintf(&b, "  [%s] %-20s model=%s retries=%d logs=%d\n",
			branch.State, branch.Name, branch.Model, branch.Retries, branch.LogCount)
	}
	return b.String()
}

// Formatter 接口。
type Formatter interface {
	Format(card *KanbanCard) string
}

// FormatWith 使用指定格式化器生成看板文本。
func FormatWith(card *KanbanCard, f Formatter) string {
	return f.Format(card)
}

// DefaultFormatter 返回默认的 Markdown 格式化器。
func DefaultFormatter() Formatter {
	return &MarkdownFormatter{}
}

// FeishuFormatter 返回飞书卡片格式化器。
func FeishuFormatter() Formatter {
	return &FeishuCardFormatter{}
}
