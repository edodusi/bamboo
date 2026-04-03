package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("BAMBOO_API_KEY", "test-key")
	t.Setenv("BAMBOO_COMPANY", "test-co")
	t.Setenv("BAMBOO_EMPLOYEE_ID", "42")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", cfg.APIKey, "test-key")
	}
	if cfg.Company != "test-co" {
		t.Errorf("Company = %q, want %q", cfg.Company, "test-co")
	}
	if cfg.EmployeeID != "42" {
		t.Errorf("EmployeeID = %q, want %q", cfg.EmployeeID, "42")
	}
}

func TestLoadConfig_MissingKey(t *testing.T) {
	t.Setenv("BAMBOO_API_KEY", "")
	t.Setenv("BAMBOO_COMPANY", "test-co")
	t.Setenv("BAMBOO_EMPLOYEE_ID", "42")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing API key")
	}
}

func TestLoadConfig_MissingCompany(t *testing.T) {
	t.Setenv("BAMBOO_API_KEY", "test-key")
	t.Setenv("BAMBOO_COMPANY", "")
	t.Setenv("BAMBOO_EMPLOYEE_ID", "42")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing company")
	}
}

func TestLoadConfig_MissingEmployeeID(t *testing.T) {
	t.Setenv("BAMBOO_API_KEY", "test-key")
	t.Setenv("BAMBOO_COMPANY", "test-co")
	t.Setenv("BAMBOO_EMPLOYEE_ID", "")

	_, err := LoadConfig()
	if err == nil {
		t.Fatal("expected error for missing employee ID")
	}
}

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("TEST_BAMBOO_VAR=hello\n# comment\n\nTEST_BAMBOO_OTHER=world\n"), 0600)

	t.Setenv("TEST_BAMBOO_VAR", "")
	t.Setenv("TEST_BAMBOO_OTHER", "")

	loadEnvFile(envFile)

	if got := os.Getenv("TEST_BAMBOO_VAR"); got != "hello" {
		t.Errorf("TEST_BAMBOO_VAR = %q, want %q", got, "hello")
	}
	if got := os.Getenv("TEST_BAMBOO_OTHER"); got != "world" {
		t.Errorf("TEST_BAMBOO_OTHER = %q, want %q", got, "world")
	}
}

func TestLoadEnvFile_DoesNotOverrideExisting(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")
	os.WriteFile(envFile, []byte("TEST_BAMBOO_EXISTING=from-file\n"), 0600)

	t.Setenv("TEST_BAMBOO_EXISTING", "from-env")

	loadEnvFile(envFile)

	if got := os.Getenv("TEST_BAMBOO_EXISTING"); got != "from-env" {
		t.Errorf("should not override: got %q, want %q", got, "from-env")
	}
}
