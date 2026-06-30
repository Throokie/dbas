package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"

	"reasonix/internal/provider"
)

const (
	RoleDecider  = "decider"
	RoleReviewer = "reviewer"
	RoleExecutor = "executor"
	RoleAuditor  = "auditor"
)

type Config struct {
	ProjectID         string           `toml:"project_id"`
	RootDir           string           `toml:"root_dir"`
	Port              int              `toml:"port"`
	KanbanPollSeconds int              `toml:"kanban_poll_seconds"`
	MaxParallel       int              `toml:"max_parallel"`
	Providers         []ProviderConfig `toml:"providers"`
	Executor          ExecutorConfig   `toml:"executor"`
	ExperienceSlot    ExperienceSlot   `toml:"experience_slot"`
	Kanban            KanbanConfig     `toml:"kanban"`
	Logging           LoggingConfig    `toml:"logging"`
	Plugins           []PluginConfig   `toml:"plugins"`
}

type ProviderConfig struct {
	Name          string `toml:"name"`
	Kind          string `toml:"kind"`
	BaseURL       string `toml:"base_url"`
	Model         string `toml:"model"`
	APIKeyEnv     string `toml:"api_key_env"`
	Role          string `toml:"role"`
	ContextWindow int    `toml:"context_window"`
}

type ExecutorConfig struct {
	Type        string            `toml:"type"`
	Command     string            `toml:"command"`
	Args        []string          `toml:"args"`
	Env         map[string]string `toml:"env"`
	Workdir     string            `toml:"workdir"`
	StdinFormat string            `toml:"stdin_format"`
}

type ExperienceSlot struct {
	EscalationThreshold    int            `toml:"escalation_threshold"`
	ForceVerifierForStates []string       `toml:"force_verifier_for_states"`
	Verifier               VerifierConfig `toml:"verifier"`
	Workflows              map[string]any `toml:"workflows"`
}

type VerifierConfig struct {
	CodeSyntax   string `toml:"code_syntax"`
	TestPassRate int    `toml:"test_pass_rate"`
}

type KanbanConfig struct {
	Enabled  bool         `toml:"enabled"`
	Notifier string       `toml:"notifier"`
	Feishu   FeishuConfig `toml:"feishu"`
}

type FeishuConfig struct {
	AppID        string `toml:"app_id"`
	AppSecretEnv string `toml:"app_secret_env"`
	WebhookURL   string `toml:"webhook_url"`
}

type LoggingConfig struct {
	Level  string `toml:"level"`
	Format string `toml:"format"`
}

// PluginConfig declares an external MCP server. Type selects the transport:
// "stdio" (default) runs Command/Args as a subprocess; other values reserved.
type PluginConfig struct {
	Name    string            `toml:"name"`
	Type    string            `toml:"type"`
	Command string            `toml:"command"`
	Args    []string          `toml:"args"`
	Env     map[string]string `toml:"env"`
	URL     string            `toml:"url"`
	Headers map[string]string `toml:"headers"`
}

func Default() Config {
	return Config{
		ProjectID:         "default-dbas-project",
		RootDir:           "./dbas-data",
		Port:              8080,
		KanbanPollSeconds: 5,
		MaxParallel:       4,
		Providers: []ProviderConfig{
			{Name: "local-decider", Kind: "openai", BaseURL: "http://localhost:8000/v1", Model: "local-decider", Role: RoleDecider, ContextWindow: 128000},
			{Name: "local-reviewer", Kind: "openai", BaseURL: "http://localhost:8000/v1", Model: "local-reviewer", Role: RoleReviewer, ContextWindow: 128000},
			{Name: "local-executor", Kind: "openai", BaseURL: "http://localhost:8000/v1", Model: "local-executor", Role: RoleExecutor, ContextWindow: 128000},
		},
		Executor: ExecutorConfig{Type: "reasonix-native"},
		Kanban:   KanbanConfig{Enabled: true, Notifier: "console", Feishu: FeishuConfig{AppSecretEnv: "FEISHU_BOT_APP_SECRET"}},
		Logging:  LoggingConfig{Level: "info", Format: "json"},
	}
}

func Load(path string) (Config, string, error) {
	cfg := Default()
	resolved := strings.TrimSpace(path)
	if resolved == "" {
		if _, err := os.Stat("dbas.toml"); err == nil {
			resolved = "dbas.toml"
		} else if !os.IsNotExist(err) {
			return cfg, "", err
		}
	}
	if resolved == "" {
		return cfg, "", cfg.Validate()
	}
	if _, err := toml.DecodeFile(resolved, &cfg); err != nil {
		return cfg, resolved, err
	}
	return cfg, resolved, cfg.Validate()
}

func (c Config) Validate() error {
	if c.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}
	if c.KanbanPollSeconds <= 0 {
		return fmt.Errorf("kanban_poll_seconds must be positive")
	}
	if c.MaxParallel <= 0 {
		return fmt.Errorf("max_parallel must be positive")
	}
	if c.Executor.Type == "" {
		return fmt.Errorf("executor.type is required")
	}
	switch c.Executor.Type {
	case "reasonix-native", "command":
	default:
		return fmt.Errorf("unsupported executor.type %q", c.Executor.Type)
	}
	if c.Executor.Type == "command" && strings.TrimSpace(c.Executor.Command) == "" {
		return fmt.Errorf("executor.command is required when executor.type is command")
	}
	if c.Executor.StdinFormat != "" {
		switch c.Executor.StdinFormat {
		case "text", "json":
		default:
			return fmt.Errorf("executor.stdin_format must be text or json")
		}
	}
	for _, role := range []string{RoleDecider, RoleReviewer, RoleExecutor} {
		if _, ok := c.ProviderForRole(role); !ok {
			return fmt.Errorf("provider role %q is required", role)
		}
	}
	for _, p := range c.Providers {
		if err := p.validateAuth(); err != nil {
			return err
		}
	}
	return nil
}

func (c Config) ProviderForRole(role string) (ProviderConfig, bool) {
	for _, p := range c.Providers {
		if strings.EqualFold(p.Role, role) {
			return p, true
		}
	}
	return ProviderConfig{}, false
}

func (c Config) BuildProvider(role string) (provider.Provider, ProviderConfig, error) {
	p, ok := c.ProviderForRole(role)
	if !ok {
		return nil, ProviderConfig{}, fmt.Errorf("provider role %q not configured", role)
	}
	apiKey := ""
	if p.APIKeyEnv != "" {
		apiKey = os.Getenv(p.APIKeyEnv)
	}
	prov, err := provider.New(p.Kind, provider.Config{
		Name:    p.Name,
		BaseURL: p.BaseURL,
		Model:   p.Model,
		APIKey:  apiKey,
		Extra: map[string]any{
			"api_key_env":    p.APIKeyEnv,
			"api_key_source": "DBAS config",
		},
	})
	if err != nil {
		return nil, p, err
	}
	return prov, p, nil
}

func (p ProviderConfig) validateAuth() error {
	if strings.TrimSpace(p.Name) == "" || strings.TrimSpace(p.Kind) == "" || strings.TrimSpace(p.BaseURL) == "" || strings.TrimSpace(p.Model) == "" || strings.TrimSpace(p.Role) == "" {
		return fmt.Errorf("provider entries require name, kind, base_url, model, and role")
	}
	if isPrivateEndpoint(p.BaseURL) {
		return nil
	}
	if p.APIKeyEnv == "" {
		return fmt.Errorf("provider %q uses a non-local endpoint and must set api_key_env", p.Name)
	}
	if os.Getenv(p.APIKeyEnv) == "" {
		return fmt.Errorf("provider %q requires API key env %s", p.Name, p.APIKeyEnv)
	}
	return nil
}

func isPrivateEndpoint(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return false
	}
	host := u.Hostname()
	if host == "" {
		return false
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	if strings.HasSuffix(strings.ToLower(host), ".local") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback() || ip.IsPrivate()
}

// PluginSpecs converts Plugins to Reasonix plugin.Spec values.
// It imports from reasonix/internal/plugin at the call site.
func (c Config) PluginSpecs() []any {
	env := func(m map[string]string) map[string]string {
		if len(m) == 0 {
			return nil
		}
		out := make(map[string]string, len(m))
		for k, v := range m {
			out[k] = v
		}
		return out
	}
	out := make([]any, 0, len(c.Plugins))
	for _, p := range c.Plugins {
		spec := map[string]any{
			"name":    p.Name,
			"type":    p.Type,
			"command": p.Command,
			"args":    p.Args,
			"env":     env(p.Env),
			"url":     p.URL,
			"headers": p.Headers,
		}
		if spec["type"] == "" || spec["type"] == "stdio" {
			spec["type"] = "stdio"
		}
		out = append(out, spec)
	}
	return out
}

func ResolvePath(baseFile, path string) string {
	if path == "" || filepath.IsAbs(path) || baseFile == "" {
		return path
	}
	return filepath.Join(filepath.Dir(baseFile), path)
}
