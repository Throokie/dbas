# DBAS Extension Examples

## myextension
Python linting extension. Adds ruff and mypy verifiers.
Demonstrates: Extension interface, VerifierRegistry, init() registration.

## Creating Your Own
1. Create `dbas/extensions/your-extension/`
2. Implement `plugin.Extension` interface
3. Register: `plugin.Global().RegisterExtension(&Yours{})`
4. Import in runner/runner.go
5. Verify at `/api/dbas/extensions`
