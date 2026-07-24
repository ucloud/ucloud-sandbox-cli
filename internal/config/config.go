package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	sandbox "github.com/ucloud/ucloud-sandbox-sdk-go"
)

const (
	defaultDomain  = "cn-wlcb.sandbox.ucloudai.com"
	domainTemplate = "%s.sandbox.ucloudai.com"
	configDir      = ".ucloud-sandbox-cli"
	configFile     = "config.json"

	envAPIKey = "UCLOUD_SANDBOX_API_KEY"
	envRegion = "UCLOUD_SANDBOX_REGION"
	envDomain = "UCLOUD_SANDBOX_DOMAIN"
	envInsure = "UCLOUD_SANDBOX_INSURE"
)

// Config holds the CLI configuration.
type Config struct {
	APIKey string `json:"api_key,omitempty"`
	Region string `json:"region,omitempty"`
	Domain string `json:"domain,omitempty"`
	Insure bool   `json:"insure,omitempty"`
}

// configPath returns the path to the config file.
func configPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// Load reads the config file and overrides values with environment variables.
func Load() (*Config, error) {
	path, err := configPath()
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read config: %w", err)
	}
	if len(data) > 0 {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("parse config: %w", err)
		}
	}

	// Environment variables take precedence over the config file.
	if v := os.Getenv(envAPIKey); v != "" {
		cfg.APIKey = v
	}
	if v := os.Getenv(envRegion); v != "" {
		cfg.Region = v
	}
	if v := os.Getenv(envDomain); v != "" {
		cfg.Domain = v
	}
	if v, ok := os.LookupEnv(envInsure); ok && v != "" {
		insure, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", envInsure, err)
		}
		cfg.Insure = insure
	}

	return cfg, nil
}

// Save writes the config to ~/.ucloud-sandbox-cli/config.json.
func Save(cfg *Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

// resolveDomain returns the API domain based on the config.
func resolveDomain(cfg *Config) string {
	if cfg.Domain != "" {
		return cfg.Domain
	}
	if cfg.Region != "" {
		return fmt.Sprintf(domainTemplate, cfg.Region)
	}
	return defaultDomain
}

// NewClient validates the config and creates a sandbox client.
func NewClient(cfg *Config) (*sandbox.Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required; set it in config or via UCLOUD_SANDBOX_API_KEY")
	}
	domain := resolveDomain(cfg)
	return sandbox.NewClient(domain, cfg.APIKey, sandbox.WithInsecureHTTP(cfg.Insure)), nil
}
