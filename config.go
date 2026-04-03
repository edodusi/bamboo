package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

type Config struct {
	APIKey     string
	Company    string
	EmployeeID string
}

func LoadConfig() (*Config, error) {
	loadEnvFile(".env")

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
