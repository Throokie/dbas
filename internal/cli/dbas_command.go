package cli

import (
	"flag"
	"fmt"
	"os"

	"reasonix/dbas/runner"
)

func dbasCommand(args []string) int {
	fs := flag.NewFlagSet("dbas", flag.ExitOnError)
	demo := fs.Bool("demo", false, "Offline demo mode (no API keys)")
	port := fs.Int("port", 8080, "HTTP API server port")
	config := fs.String("config", "", "Path to DBAS TOML config")
	showVer := fs.Bool("version", false, "Show version")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: reasonix dbas [options]

DBAS governance-first control plane for AI coding agents.

Options:
  --demo         Run offline demo
  --port N       HTTP API port (default: 8080)
  --config path  TOML config path
  --version      Show version
")
	}
	if err := fs.Parse(args); err != nil {
		return 1
	}
	if *showVer {
		fmt.Println("DBAS", runner.Version)
		return 0
	}
	return runner.Run(runner.RunConfig{DemoMode: *demo, Port: *port, ConfigPath: *config})
}
