// Package runner starts the DBAS server.
// Shared between the standalone binary and the Reasonix CLI.
package runner

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"reasonix/dbas/internal/dbas/api"
	"reasonix/dbas/internal/dbas/brain"
	"reasonix/dbas/internal/dbas/branch"
	dbasconfig "reasonix/dbas/internal/dbas/config"
	"reasonix/dbas/internal/dbas/demo"
	"reasonix/dbas/internal/dbas/executor"
	"reasonix/dbas/internal/dbas/experienceslot"
	"reasonix/dbas/internal/dbas/feishu"
	"reasonix/dbas/internal/dbas/kanban"
	"reasonix/dbas/internal/dbas/multibranch"
	"reasonix/dbas/internal/dbas/persistence"
	"reasonix/dbas/internal/dbas/plugin"
	"reasonix/dbas/internal/dbas/skill"
	"reasonix/dbas/internal/dbas/verifier"
	_ "reasonix/dbas/internal/dbas/extend"
	_ "reasonix/internal/provider/openai"
	"reasonix/internal/tool"
	_ "reasonix/internal/tool/builtin"
)

const Version = "0.1.0-alpha"

// RunConfig holds startup parameters.
type RunConfig struct {
	DemoMode   bool
	Port       int
	ConfigPath string
}

// RunDefaults returns default RunConfig.
func RunDefaults() RunConfig {
	return RunConfig{
		Port: 8080,
	}
}

// Run starts the DBAS server and blocks until exit.
// Returns exit code (0 = success, 1 = error).
func Run(cfg RunConfig) int {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))
	slog.SetDefault(logger)

	configPath := cfg.ConfigPath
	if configPath == "" {
		// Check for dbas.toml in current directory
		if _, err := os.Stat("dbas.toml"); err == nil {
			configPath = "dbas.toml"
		}
	}

	loadedCfg, loadedConfig, err := dbasconfig.Load(configPath)
	if err != nil {
		slog.Error("config load failed", "err", err)
		return 1
	}
	if cfg.Port != 8080 {
		loadedCfg.Port = cfg.Port
	}
	if loadedConfig != "" {
		loadedCfg.RootDir = dbasconfig.ResolvePath(loadedConfig, loadedCfg.RootDir)
		loadedCfg.Executor.Workdir = dbasconfig.ResolvePath(loadedConfig, loadedCfg.Executor.Workdir)
	}
	if err := loadedCfg.Validate(); err != nil {
		slog.Error("config validation failed", "err", err)
		return 1
	}

	slog.Info("DBAS starting", "version", Version, "port", loadedCfg.Port, "root", loadedCfg.RootDir, "executor", loadedCfg.Executor.Type)

	slot := experienceslot.DefaultSlot()

	deciderProv, _, err := loadedCfg.BuildProvider(dbasconfig.RoleDecider)
	if err != nil { slog.Error("decider init failed", "err", err); return 1 }
	reviewerProv, _, err := loadedCfg.BuildProvider(dbasconfig.RoleReviewer)
	if err != nil { slog.Error("reviewer init failed", "err", err); return 1 }
	executorProv, execProvCfg, err := loadedCfg.BuildProvider(dbasconfig.RoleExecutor)
	if err != nil { slog.Error("executor init failed", "err", err); return 1 }
	auditorProv := reviewerProv
	if _, ok := loadedCfg.ProviderForRole(dbasconfig.RoleAuditor); ok {
		auditorProv, _, err = loadedCfg.BuildProvider(dbasconfig.RoleAuditor)
		if err != nil { slog.Error("auditor init failed", "err", err); return 1 }
	}

	decider := brain.NewDecider(deciderProv)
	reviewer := brain.NewReviewer(reviewerProv)

	registry := tool.NewRegistry()
	for _, t := range tool.Builtins() { registry.Add(t) }

	executorRunner, err := executor.New(loadedCfg.Executor, executorProv, registry, brain.ExecutorHandoffPrompt, execProvCfg.ContextWindow)
	if err != nil { slog.Error("executor init failed", "err", err); return 1 }

	store := persistence.NewBranchStore(loadedCfg.RootDir)
	if err := os.MkdirAll(loadedCfg.RootDir, 0o755); err != nil { slog.Error("create root dir", "err", err); return 1 }

	toolSchemaJSON, err := json.Marshal(registry.Schemas())
	if err != nil { slog.Error("marshal tool schemas", "err", err); return 1 }
	rootHash := branch.ComputeRootHash(
		brain.DeciderSystemPrompt+"\n"+brain.ReviewerSystemPrompt+"\n"+brain.ExecutorHandoffPrompt,
		string(toolSchemaJSON),
	)

	engine := branch.NewEngine(rootHash, slot)
	verifierTeam := verifier.NewVerifier()
	llmVerifier := verifier.NewLLMVerifier(auditorProv)

	coordFactory := &coordFactory{
		decider: decider, reviewer: reviewer,
		executor: executorRunner, verifier: verifierTeam,
		llmVerifier: llmVerifier, store: store,
	}

	scheduler := multibranch.NewScheduler(engine, store, coordFactory, loadedCfg.MaxParallel)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var kanbanDaemon *kanban.Daemon
	if loadedCfg.Kanban.Enabled {
		notifier := buildNotifier(loadedCfg)
		kanbanDaemon = kanban.NewDaemon(engine, notifier, time.Duration(loadedCfg.KanbanPollSeconds)*time.Second)
		go kanbanDaemon.Start(ctx)
	}

	skillLoader := skill.NewLoader("./skills", "./dbas/skills")
	loadedSkills, _ := skillLoader.LoadAll()
	slog.Info("DBAS skills loaded", "count", len(loadedSkills))
	for _, sk := range loadedSkills {
		slog.Debug("skill loaded", "name", sk.Name)
	}

	extRegistry := plugin.Global()
	extHandler := api.NewExtHandler(skillLoader, extRegistry)

	frontendDir := resolveFrontendDir()
	server := api.NewServer(engine, store, coordFactory, scheduler, loadedCfg.Port, frontendDir)
	extHandler.RegisterRoutes(server)

	handler := api.ChainMiddleware(server,
		api.RecoveryMiddleware,
		api.LoggingMiddleware,
		api.CORSMiddleware,
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		slog.Info("shutting down DBAS")
		cancel()
		if kanbanDaemon != nil { kanbanDaemon.Stop() }
		os.Exit(0)
	}()

	addr := fmt.Sprintf(":%d", loadedCfg.Port)
	slog.Info("DBAS ready", "api", fmt.Sprintf("http://localhost:%d", loadedCfg.Port), "tools", registry.Names(), "root_hash", rootHash)

	if cfg.DemoMode {
		go demo.RunDemo(engine, store, loadedCfg.Port)
	}

	if err := http.ListenAndServe(addr, handler); err != nil {
		slog.Error("server error", "err", err)
		return 1
	}
	return 0
}

func resolveFrontendDir() string {
	candidates := []string{"./dbas/frontend/dist", "./frontend/dist"}
	for _, d := range candidates {
		if info, err := os.Stat(d); err == nil && info.IsDir() { return d }
	}
	return ""
}

type coordFactory struct {
	decider     *brain.Decider
	reviewer    *brain.Reviewer
	executor    interface{ Run(ctx context.Context, input string) error }
	verifier    *verifier.Verifier
	llmVerifier *verifier.LLMVerifier
	store       *persistence.BranchStore
}

func (f *coordFactory) Create(b *branch.Branch) (multibranch.Runner, error) {
	return brain.NewDeciderCoordinator(
		f.decider, f.reviewer, f.executor, f.verifier, f.llmVerifier, f.store, b,
	), nil
}

type multiNotifier struct{ notifiers []kanban.Notifier }
func (m *multiNotifier) Notify(card *kanban.KanbanCard) error {
	for _, n := range m.notifiers { _ = n.Notify(card) }
	return nil
}

func buildNotifier(cfg dbasconfig.Config) kanban.Notifier {
	n := kanban.Notifier(&kanban.ConsoleNotifier{})
	if cfg.Kanban.Feishu.WebhookURL == "" { return n }
	return &multiNotifier{notifiers: []kanban.Notifier{n, feishu.NewWebhookNotifier(cfg.Kanban.Feishu.WebhookURL, "")}}
}



