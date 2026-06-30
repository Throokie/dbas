// Package feishubot provides a self-contained bidirectional Feishu bot for DBAS.
// It uses raw HTTP to the Feishu Open API (no larksuite SDK dependency).
// Supports webhook mode: receives events via HTTP, sends messages via API.
package feishubot

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"reasonix/dbas/internal/dbas/branch"
	"reasonix/dbas/internal/dbas/config"
	"reasonix/dbas/internal/dbas/skill"
)

// Bridge connects Feishu to DBAS with bidirectional chat.
type Bridge struct {
	cfg     config.FeishuBotConfig
	engine  *branch.BranchEngine
	skills  *skill.Loader
	logger  *slog.Logger
	cancel  context.CancelFunc
	server  *http.Server
	token   tenantToken
	tokenMu sync.Mutex
}

type tenantToken struct {
	Token     string
	ExpiresAt time.Time
}

// IncomingFeishuEvent represents a Feishu event callback payload.
type IncomingFeishuEvent struct {
	Schema string            `json:"schema"`
	Header FeishuEventHeader `json:"header"`
	Event  json.RawMessage   `json:"event"`
}

// FeishuEventHeader is the header of every Feishu event.
type FeishuEventHeader struct {
	EventID    string `json:"event_id"`
	EventType  string `json:"event_type"`
	Token      string `json:"token"`
	CreateTime string `json:"create_time"`
}

// URLVerificationChallenge is sent by Feishu for webhook verification.
type URLVerificationChallenge struct {
	Challenge string `json:"challenge"`
	Token     string `json:"token"`
	Type      string `json:"type"`
}

// MessageReceiveEvent is the im.message.receive_v1 event payload.
type MessageReceiveEvent struct {
	Sender  MessageSender  `json:"sender"`
	Message MessageContent `json:"message"`
}

type MessageSender struct {
	SenderID struct {
		UserID  string `json:"user_id"`
		OpenID  string `json:"open_id"`
		UnionID string `json:"union_id"`
	} `json:"sender_id"`
}

type MessageContent struct {
	MessageID string `json:"message_id"`
	RootID    string `json:"root_id"`
	ParentID  string `json:"parent_id"`
	ChatID    string `json:"chat_id"`
	ChatType  string `json:"chat_type"`
	MsgType   string `json:"msg_type"`
	Content   string `json:"content"`
}

// New creates a DBAS Feishu bot bridge.
func New(cfg config.FeishuBotConfig, engine *branch.BranchEngine, skills *skill.Loader, logger *slog.Logger) *Bridge {
	return &Bridge{
		cfg:    cfg,
		engine: engine,
		skills: skills,
		logger: logger.With("bot", "feishu"),
	}
}

// Start launches the Feishu bot webhook server.
func (b *Bridge) Start(ctx context.Context) error {
	if !b.cfg.Enabled {
		b.logger.Info("feishu bot disabled, skipping")
		return nil
	}

	ctx, b.cancel = context.WithCancel(ctx)

	port := b.cfg.WebhookPort
	if port == 0 {
		port = 8080
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/feishu/webhook", b.handleWebhook)

	b.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: mux,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		b.server.Shutdown(shutdownCtx)
	}()

	go func() {
		b.logger.Info("feishu bot webhook listening", "port", port, "path", "/api/feishu/webhook")
		if err := b.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			b.logger.Error("feishu webhook server error", "err", err)
		}
	}()

	return nil
}

// Stop shuts down the Feishu bot.
func (b *Bridge) Stop() {
	if b.cancel != nil {
		b.cancel()
	}
}

// handleWebhook receives Feishu event callbacks.
func (b *Bridge) handleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024*1024))
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Check for URL verification challenge
	var challenge URLVerificationChallenge
	if err := json.Unmarshal(body, &challenge); err == nil && challenge.Type == "url_verification" {
		if !b.verifyToken(challenge.Token) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"challenge": challenge.Challenge})
		return
	}

	// Parse event
	var evt IncomingFeishuEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		b.logger.Error("failed to parse feishu event", "err", err)
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	// Verify token
	if !b.verifyToken(evt.Header.Token) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	// Handle message event
	if evt.Header.EventType == "im.message.receive_v1" {
		b.handleMessageEvent(evt.Event)
	}

	w.WriteHeader(http.StatusOK)
}

// verifyToken checks the verification token from Feishu.
func (b *Bridge) verifyToken(token string) bool {
	if b.cfg.VerificationToken == "" {
		return true // no token configured = accept all (not recommended)
	}
	return token == b.cfg.VerificationToken
}

// handleMessageEvent processes an incoming message event.
func (b *Bridge) handleMessageEvent(raw json.RawMessage) {
	var msgEvt MessageReceiveEvent
	if err := json.Unmarshal(raw, &msgEvt); err != nil {
		b.logger.Error("failed to parse message event", "err", err)
		return
	}

	// Only handle text messages
	if msgEvt.Message.MsgType != "text" {
		return
	}

	// Parse text content (Feishu sends text as JSON string)
	var textContent struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal([]byte(msgEvt.Message.Content), &textContent); err != nil {
		b.logger.Error("failed to parse text content", "err", err)
		return
	}

	text := strings.TrimSpace(textContent.Text)

	// If require_mention is enabled, strip @bot mention
	if b.cfg.RequireMention {
		text = stripMention(text)
	}

	text = strings.TrimSpace(text)
	if text == "" {
		return
	}

	chatID := msgEvt.Message.ChatID
	messageID := msgEvt.Message.MessageID

	b.logger.Info("feishu message received",
		"user", msgEvt.Sender.SenderID.OpenID, "chat", chatID, "text", text)

	// Dispatch command and send reply
	reply := b.dispatchCommand(text)
	if reply != "" {
		if err := b.sendMessage(chatID, messageID, reply); err != nil {
			b.logger.Error("failed to send reply", "err", err)
		}
	}
}

// stripMention removes @bot mention text from the message.
func stripMention(text string) string {
	parts := strings.SplitN(text, " ", 2)
	if strings.HasPrefix(parts[0], "@") {
		if len(parts) > 1 {
			return parts[1]
		}
		return ""
	}
	return text
}
func (b *Bridge) sendMessage(chatID, replyToMsgID, text string) error {
	token, err := b.getAccessToken()
	if err != nil {
		return fmt.Errorf("get access token: %w", err)
	}

	// Feishu text content must be JSON-escaped text
	contentJSON, _ := json.Marshal(map[string]string{"text": text})

	body := map[string]interface{}{
		"receive_id": chatID,
		"msg_type":   "text",
		"content":    string(contentJSON),
	}
	bodyJSON, _ := json.Marshal(body)

	req, err := http.NewRequest("POST",
		"https://open.feishu.cn/open-apis/im/v1/messages?receive_id_type=chat_id",
		bytes.NewReader(bodyJSON))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("feishu api returned %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}

// getAccessToken gets a Feishu tenant_access_token, caching it until expiry.
func (b *Bridge) getAccessToken() (string, error) {
	b.tokenMu.Lock()
	defer b.tokenMu.Unlock()

	if b.token.Token != "" && time.Now().Before(b.token.ExpiresAt) {
		return b.token.Token, nil
	}

	appID := b.cfg.AppID
	appSecretEnv := b.cfg.AppSecretEnv

	if appID == "" {
		return "", fmt.Errorf("feishu app_id not configured")
	}

	appSecret := os.Getenv(appSecretEnv)
	if appSecret == "" {
		return "", fmt.Errorf("feishu app_secret env %s is empty", appSecretEnv)
	}

	authBody := map[string]string{
		"app_id":     appID,
		"app_secret": appSecret,
	}
	authJSON, _ := json.Marshal(authBody)

	resp, err := http.Post(
		"https://open.feishu.cn/open-apis/auth/v3/tenant_access_token/internal",
		"application/json; charset=utf-8",
		bytes.NewReader(authJSON))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		Code              int    `json:"code"`
		Msg               string `json:"msg"`
		TenantAccessToken string `json:"tenant_access_token"`
		Expire            int    `json:"expire"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.Code != 0 {
		return "", fmt.Errorf("feishu auth error: %s (code %d)", result.Msg, result.Code)
	}

	b.token = tenantToken{
		Token:     result.TenantAccessToken,
		ExpiresAt: time.Now().Add(time.Duration(result.Expire) * time.Second).Add(-5 * time.Minute),
	}
	return b.token.Token, nil
}

// dispatchCommand parses text and returns a DBAS command response.
func (b *Bridge) dispatchCommand(text string) string {
	parts := strings.Fields(text)
	if len(parts) == 0 {
		return "请发送命令: spawn, status, skills, branches, help"
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "spawn", "new", "create":
		return b.handleSpawn(parts[1:])
	case "status", "stats":
		return b.handleStatus()
	case "skills":
		return b.handleSkills()
	case "branches", "list":
		return b.handleBranches()
	case "help", "?":
		return b.handleHelp()
	default:
		return fmt.Sprintf("未知命令: %s\n\n支持的命令: spawn, status, skills, branches, help", cmd)
	}
}

// handleSpawn creates a new branch. The task name becomes part of the branch ID.
func (b *Bridge) handleSpawn(args []string) string {
	if len(args) == 0 {
		return "用法: spawn <任务名称>\n示例: spawn 修复登录页面的CSS样式"
	}

	taskName := strings.Join(args, " ")
	br, err := b.engine.CreateBranch(taskName, nil)
	if err != nil {
		return fmt.Sprintf("创建分支失败: %s", err)
	}

	return fmt.Sprintf("✅ 分支已创建\n• ID: %s\n• 状态: %s\n• 时间: %s",
		br.BranchID, br.CurrentState, br.CreatedAt.Format(time.RFC3339))
}

// handleStatus returns DBAS overall status.
func (b *Bridge) handleStatus() string {
	allBranches := b.engine.GetAllBranches()
	total := len(allBranches)
	active := 0
	passed := 0
	failed := 0

	for _, br := range allBranches {
		switch string(br.CurrentState) {
		case "PASS", "ARCHIVE":
			passed++
		case "FAIL", "ERROR":
			failed++
		default:
			active++
		}
	}

	skillsCount := 0
	if b.skills != nil {
		if loaded, err := b.skills.LoadAll(); err == nil {
			skillsCount = len(loaded)
		}
	}

	return fmt.Sprintf("📊 DBAS 状态\n• 总分叉: %d\n• 活跃中: %d\n• 已完成: %d\n• 已失败: %d\n• 技能数: %d\n• 根哈希: %s",
		total, active, passed, failed, skillsCount, b.engine.RootHash())
}

// handleSkills lists loaded skills.
func (b *Bridge) handleSkills() string {
	if b.skills == nil {
		return "技能加载器未初始化"
	}

	loaded, err := b.skills.LoadAll()
	if err != nil {
		return fmt.Sprintf("加载技能失败: %s", err)
	}

	if len(loaded) == 0 {
		return "没有已加载的技能"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📚 已加载 %d 个技能:\n\n", len(loaded)))
	for _, sk := range loaded {
		sb.WriteString(fmt.Sprintf("• %s", sk.Name))
		if sk.Description != "" {
			sb.WriteString(fmt.Sprintf(" — %s", sk.Description))
		}
		sb.WriteString("\n")
	}
	return sb.String()
}

// handleBranches lists all branches.
func (b *Bridge) handleBranches() string {
	allBranches := b.engine.GetAllBranches()
	if len(allBranches) == 0 {
		return "当前没有分支"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("🌿 共 %d 个分支:\n\n", len(allBranches)))
	for _, br := range allBranches {
		sb.WriteString(fmt.Sprintf("• %s\n  ID: %s | 状态: %s | 时间: %s\n",
			br.BranchID, br.BranchID, br.CurrentState, br.CreatedAt.Format("15:04:05")))
	}
	return sb.String()
}

// handleHelp returns help text.
func (b *Bridge) handleHelp() string {
	return "🤖 DBAS 飞书 Bot 帮助\n\n命令列表:\n\nspawn <任务名>   — 创建新分支执行任务\nstatus / stats   — 查看 DBAS 整体状态\nskills           — 列出已加载的技能\nbranches / list  — 查看所有分支\nhelp / ?         — 显示此帮助\n\n示例:\n  spawn 修复登录页面样式\n  status\n  skills\n  branches"
}
