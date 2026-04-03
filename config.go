package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	APIKey     string
	Company    string
	EmployeeID string
}

// envFiles returns the list of .env paths to try, in priority order.
var envFiles = func() []string {
	paths := []string{".env"}
	if home, err := os.UserHomeDir(); err == nil {
		paths = append(paths, filepath.Join(home, ".config", "bamboo", ".env"))
	}
	return paths
}

func LoadConfig() (*Config, error) {
	for _, path := range envFiles() {
		loadEnvFile(path)
	}

	cfg := &Config{
		APIKey:     os.Getenv("BAMBOO_API_KEY"),
		Company:    os.Getenv("BAMBOO_COMPANY"),
		EmployeeID: os.Getenv("BAMBOO_EMPLOYEE_ID"),
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("BAMBOO_API_KEY is not set")
	}
	if cfg.Company == "" {
		return nil, fmt.Errorf("BAMBOO_COMPANY is not set")
	}
	if cfg.EmployeeID == "" {
		return nil, fmt.Errorf("BAMBOO_EMPLOYEE_ID is not set")
	}

	return cfg, nil
}

func loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}
