package plugin

import (
	"testing"
)

type testExtension struct {
	name        string
	description string
	version     string
}

func (e *testExtension) Name() string                  { return e.name }
func (e *testExtension) Description() string           { return e.description }
func (e *testExtension) Version() string                { return e.version }
func (e *testExtension) RegisterExecutors(r ExecutorRegistry) {
	r.RegisterExecutor(e.name+"-exec", func(cfg map[string]interface{}) (interface{}, error) {
		return "mock-executor", nil
	})
}
func (e *testExtension) RegisterVerifiers(r VerifierRegistry) {
	r.RegisterVerifier(VerifierTool{Name: e.name + "-verifier", Command: "echo ok"})
}
func (e *testExtension) RegisterNotifiers(r NotifierRegistry) {
	r.RegisterNotifier(e.name+"-notifier", &testNotifier{name: e.name + "-notifier"})
}

type testNotifier struct{ name string }
func (n *testNotifier) Name() string { return n.name }
func (n *testNotifier) Notify(title, message string) error { return nil }

func TestNewRegistry(t *testing.T) {
	r := NewRegistry()
	if r == nil {
		t.Fatal("NewRegistry returned nil")
	}
}

func TestRegisterExtension(t *testing.T) {
	r := NewRegistry()
	ext := &testExtension{name: "test-ext", description: "A test", version: "1.0.0"}
	r.RegisterExtension(ext)

	exts := r.ListExtensions()
	if len(exts) != 1 {
		t.Fatalf("expected 1 extension, got %d", len(exts))
	}
	if exts[0].Name != "test-ext" {
		t.Errorf("expected name 'test-ext', got %q", exts[0].Name)
	}
}

func TestListExecutors(t *testing.T) {
	r := NewRegistry()
	r.RegisterExtension(&testExtension{name: "ext-a"})
	r.RegisterExtension(&testExtension{name: "ext-b"})

	execs := r.ListExecutors()
	if len(execs) != 2 {
		t.Fatalf("expected 2 executors, got %d: %v", len(execs), execs)
	}
}

func TestListVerifiers(t *testing.T) {
	r := NewRegistry()
	r.RegisterExtension(&testExtension{name: "ext-a"})

	vers := r.ListVerifiers()
	if len(vers) != 1 {
		t.Fatalf("expected 1 verifier, got %d", len(vers))
	}
	if vers[0].Name != "ext-a-verifier" {
		t.Errorf("expected verifier name 'ext-a-verifier', got %q", vers[0].Name)
	}
}

func TestListNotifiers(t *testing.T) {
	r := NewRegistry()
	r.RegisterExtension(&testExtension{name: "ext-a"})

	notifs := r.ListNotifiers()
	if len(notifs) != 1 {
		t.Fatalf("expected 1 notifier, got %d", len(notifs))
	}
	if notifs[0] != "ext-a-notifier" {
		t.Errorf("expected notifier 'ext-a-notifier', got %q", notifs[0])
	}
}

func TestDisableExtension(t *testing.T) {
	r := NewRegistry()
	ext := &testExtension{name: "disable-me"}
	r.RegisterExtension(ext)

	if r.IsDisabled("disable-me") {
		t.Error("extension should not be disabled initially")
	}

	r.Disable("disable-me")
	if !r.IsDisabled("disable-me") {
		t.Error("extension should be disabled")
	}

	r.Enable("disable-me")
	if r.IsDisabled("disable-me") {
		t.Error("extension should be re-enabled")
	}
}

func TestDisabledExtensionNotRegistered(t *testing.T) {
	r := NewRegistry()
	r.Disable("no-register")
	r.RegisterExtension(&testExtension{name: "no-register"})

	exts := r.ListExtensions()
	for _, e := range exts {
		if e.Name == "no-register" {
			t.Error("disabled extension should not be registered")
		}
	}
}

func TestGetExecutor(t *testing.T) {
	r := NewRegistry()
	r.RegisterExtension(&testExtension{name: "finder"})

	f, err := r.GetExecutor("finder-exec")
	if err != nil {
		t.Fatalf("GetExecutor failed: %v", err)
	}
	if f == nil {
		t.Fatal("GetExecutor returned nil")
	}

	// Test getting nonexistent executor
	_, err = r.GetExecutor("nonexistent-exec")
	if err == nil {
		t.Fatal("expected error for nonexistent executor")
	}
}

func TestGetNotifier(t *testing.T) {
	r := NewRegistry()
	r.RegisterExtension(&testExtension{name: "getter"})

	n, err := r.GetNotifier("getter-notifier")
	if err != nil {
		t.Fatalf("GetNotifier failed: %v", err)
	}
	if n == nil {
		t.Fatal("GetNotifier returned nil")
	}

	_, err = r.GetNotifier("nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent notifier")
	}
}

func TestGlobalRegistry(t *testing.T) {
	r := Global()
	if r == nil {
		t.Fatal("Global() returned nil")
	}
	// Calling twice should return the same instance
	r2 := Global()
	if r != r2 {
		t.Error("Global() should return the same registry")
	}
}
