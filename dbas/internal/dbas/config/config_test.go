package config

import "testing"

func TestValidateAllowsPrivateProviderWithoutAPIKey(t *testing.T) {
	cfg := Default()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestValidateRequiresEnvForPublicProvider(t *testing.T) {
	t.Setenv("DBAS_TEST_KEY", "")
	cfg := Default()
	cfg.Providers[0].BaseURL = "https://api.example.com/v1"
	cfg.Providers[0].APIKeyEnv = "DBAS_TEST_KEY"

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() error = nil, want missing env error")
	}
}

func TestValidateAcceptsPublicProviderWithEnv(t *testing.T) {
	t.Setenv("DBAS_TEST_KEY", "secret")
	cfg := Default()
	cfg.Providers[0].BaseURL = "https://api.example.com/v1"
	cfg.Providers[0].APIKeyEnv = "DBAS_TEST_KEY"

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
