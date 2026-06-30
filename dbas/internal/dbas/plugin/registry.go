package plugin

import (
	"fmt"
	"sort"
	"sync"
)

// ExtensionType categorizes what an extension provides.
type ExtensionType string

const (
	ExtExecutor ExtensionType = "executor"
	ExtVerifier ExtensionType = "verifier"
	ExtNotifier ExtensionType = "notifier"
	ExtTool     ExtensionType = "tool"
)

// ExtensionMeta describes a registered extension.
type ExtensionMeta struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version,omitempty"`
	Types       []ExtensionType   `json:"types,omitempty"`
	Config      map[string]string `json:"config,omitempty"`
}

// ExecutorFactory creates an executor runner from config.
type ExecutorFactory func(cfg map[string]interface{}) (interface{}, error)

// VerifierTool defines an external verification command.
type VerifierTool struct {
	Name    string `json:"name"`
	Command string `json:"command"`
}

// Notifier sends notifications for kanban events.
type Notifier interface {
	Name() string
	Notify(title, message string) error
}

// Extension is a DBAS extension that can register capabilities.
type Extension interface {
	Name() string
	Description() string
	Version() string
	RegisterExecutors(r ExecutorRegistry)
	RegisterVerifiers(r VerifierRegistry)
	RegisterNotifiers(r NotifierRegistry)
}

// ExecutorRegistry allows extensions to register executor factories.
type ExecutorRegistry interface {
	RegisterExecutor(name string, factory ExecutorFactory)
}

// VerifierRegistry allows extensions to register verifier tools.
type VerifierRegistry interface {
	RegisterVerifier(tool VerifierTool)
}

// NotifierRegistry allows extensions to register notification channels.
type NotifierRegistry interface {
	RegisterNotifier(name string, n Notifier)
}

// Registry collects all registered extensions with lifecycle management.
type Registry struct {
	mu         sync.RWMutex
	executors  map[string]ExecutorFactory
	verifiers  []VerifierTool
	notifiers  map[string]Notifier
	extensions []ExtensionMeta
	disabled   map[string]bool
}

var globalRegistry *Registry

// Global returns the global extension registry, creating it if needed.
func Global() *Registry {
	if globalRegistry == nil {
		globalRegistry = NewRegistry()
	}
	return globalRegistry
}

// NewRegistry creates an empty extension registry.
func NewRegistry() *Registry {
	return &Registry{
		executors:  make(map[string]ExecutorFactory),
		verifiers:  make([]VerifierTool, 0),
		notifiers:  make(map[string]Notifier),
		extensions: make([]ExtensionMeta, 0),
		disabled:   make(map[string]bool),
	}
}

// RegisterExtension adds a single extension's capabilities.
func (r *Registry) RegisterExtension(ext Extension) {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := ext.Name()
	if r.disabled[name] {
		return
	}

	ext.RegisterExecutors(r)
	ext.RegisterVerifiers(r)
	ext.RegisterNotifiers(r)

	r.extensions = append(r.extensions, ExtensionMeta{
		Name:        name,
		Description: ext.Description(),
		Version:     ext.Version(),
	})
}

// RegisterExecutor implements ExecutorRegistry.
func (r *Registry) RegisterExecutor(name string, factory ExecutorFactory) {
	r.executors[name] = factory
}

// RegisterVerifier implements VerifierRegistry.
func (r *Registry) RegisterVerifier(tool VerifierTool) {
	r.verifiers = append(r.verifiers, tool)
}

// RegisterNotifier implements NotifierRegistry.
func (r *Registry) RegisterNotifier(name string, n Notifier) {
	r.notifiers[name] = n
}

// ListExtensions returns metadata for all registered extensions.
func (r *Registry) ListExtensions() []ExtensionMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]ExtensionMeta, len(r.extensions))
	copy(out, r.extensions)
	return out
}

// ListExecutors returns all registered executor names.
func (r *Registry) ListExecutors() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.executors))
	for name := range r.executors {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// ListVerifiers returns all registered verifier tools.
func (r *Registry) ListVerifiers() []VerifierTool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]VerifierTool, len(r.verifiers))
	copy(out, r.verifiers)
	return out
}

// ListNotifiers returns all registered notifier names.
func (r *Registry) ListNotifiers() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.notifiers))
	for name := range r.notifiers {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// GetExecutor returns a registered executor factory by name.
func (r *Registry) GetExecutor(name string) (ExecutorFactory, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.executors[name]
	if !ok {
		return nil, fmt.Errorf("executor %q not registered (available: %v)", name, r.executorNames())
	}
	return f, nil
}

// GetNotifier returns a registered notifier by name.
func (r *Registry) GetNotifier(name string) (Notifier, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.notifiers[name]
	if !ok {
		return nil, fmt.Errorf("notifier %q not registered", name)
	}
	return n, nil
}

// Disable disables an extension by name.
func (r *Registry) Disable(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.disabled[name] = true
}

// Enable re-enables a disabled extension.
func (r *Registry) Enable(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.disabled, name)
}

// IsDisabled checks if an extension is disabled.
func (r *Registry) IsDisabled(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.disabled[name]
}

func (r *Registry) executorNames() []string {
	names := make([]string, 0, len(r.executors))
	for n := range r.executors {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
