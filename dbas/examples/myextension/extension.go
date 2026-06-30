package myextension

import (
	"reasonix/dbas/internal/dbas/plugin"
)

func init() {
	plugin.Global().RegisterExtension(&PythonLintExtension{})
}

type PythonLintExtension struct{}

func (e *PythonLintExtension) Name() string        { return "python-lint" }
func (e *PythonLintExtension) Description() string  { return "Python linting extension using ruff and mypy" }
func (e *PythonLintExtension) Version() string      { return "1.0.0" }

func (e *PythonLintExtension) RegisterExecutors(r plugin.ExecutorRegistry)    {}
func (e *PythonLintExtension) RegisterVerifiers(r plugin.VerifierRegistry) {
	r.RegisterVerifier(plugin.VerifierTool{Name: "ruff-check", Command: "ruff check ."})
	r.RegisterVerifier(plugin.VerifierTool{Name: "mypy-check", Command: "mypy ."})
}
func (e *PythonLintExtension) RegisterNotifiers(r plugin.NotifierRegistry) {}
