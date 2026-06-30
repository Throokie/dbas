package mcpplugin

// PluginConfig represents a single MCP plugin configuration.
type PluginConfig struct {
	Name string `toml:"name"`
	Args []string `toml:"args"`
}

// Config holds all MCP plugin configurations.
type Config struct {
	Plugins []PluginConfig `toml:"plugins"`
}
