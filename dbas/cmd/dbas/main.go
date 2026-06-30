package main

import (
	"flag"
	"fmt"
	"os"

	"reasonix/dbas/runner"
)

const version = runner.Version

func main() {
	var (
		configPath = flag.String("config", "", "Path to DBAS TOML config")
		port       = flag.Int("port", 8080, "HTTP API server port")
		demoMode   = flag.Bool("demo", false, "Offline demo: no API keys needed")
		showVer    = flag.Bool("version", false, "Show version")
	)
	flag.Parse()

	if *showVer {
		fmt.Printf("DBAS v%s\n", version)
		os.Exit(0)
	}

	cfg := runner.RunConfig{
		DemoMode:   *demoMode,
		Port:       *port,
		ConfigPath: *configPath,
	}
	os.Exit(runner.Run(cfg))
}

