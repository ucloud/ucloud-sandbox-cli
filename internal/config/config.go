package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ucloud/ucloud-sandbox-sdk-go/pkg/client"
)

const (
	defaultDomain  = "cn-wlcb.sandbox.ucloudai.com"
	domainTemplate = "%s.sandbox.ucloudai.com"
	configDir      = ".ucloud-sandbox-cli"
	configFile     = "config.json"

	envAPIKey       = "UCLOUD_SANDBOX_API_KEY"
	envRegion       = "UCLOUD_SANDBOX_REGION"
	envDomain       = "UCLOUD_SANDBOX_DOMAIN"
	envInsecureHTTP = "UCLOUD_SANDBOX_INSECURE_HTTP"

	envRegistries = "UCLOUD_SANDBOX_REGISTRIES"
)

// RegistryAuth is the credentials for one container registry.
type RegistryAuth struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// Config holds the CLI configuration.
//
// No field is omitempty: the file is meant to be opened and edited by hand, so
// it always lists every setting there is, and showing the configuration shows
// what is unset as well as what is set.
type Config struct {
	APIKey       string `json:"api_key"`
	Region       string `json:"region"`
	Domain       string `json:"domain"`
	InsecureHTTP bool   `json:"insecure_http"`

	// Registries holds the credentials for pulling base images, keyed by
	// registry domain ("docker.io", "uhub.service.ucloud.cn"). A template
	// build looks up the registry of the image it starts from.
	Registries map[string]RegistryAuth `json:"registries"`
}

// RegistryAuth returns the credentials configured for a registry domain.
func (c *Config) RegistryAuth(domain string) (RegistryAuth, bool) {
	auth, ok := c.Registries[strings.ToLower(domain)]
	return auth, ok
}

// Path returns the path to the config file.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, configDir, configFile), nil
}

// LoadFile reads the config file on its own, without the environment overrides
// Load applies. It is what a command that writes the file back reads first, so
// a value that only came from the environment is not persisted into the file.
//
// A missing file is not an error: it reads as an empty config.
func LoadFile() (*Config, error) {
	path, err := Path()
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

	return cfg, nil
}

// Load reads the config file and overrides values with environment variables.
func Load() (*Config, error) {
	cfg, err := LoadFile()
	if err != nil {
		return nil, err
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
	if v, ok := os.LookupEnv(envInsecureHTTP); ok && v != "" {
		insecureHTTP, err := strconv.ParseBool(v)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", envInsecureHTTP, err)
		}
		cfg.InsecureHTTP = insecureHTTP
	}
	if v := os.Getenv(envRegistries); v != "" {
		registries := map[string]RegistryAuth{}
		if err := json.Unmarshal([]byte(v), &registries); err != nil {
			return nil, fmt.Errorf("parse %s: %w", envRegistries, err)
		}
		cfg.Registries = registries
	}

	return cfg, nil
}

// Save writes the config to ~/.ucloud-sandbox-cli/config.json.
func Save(cfg *Config) error {
	path, err := Path()
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
func NewClient(cfg *Config) (*client.Client, error) {
	if cfg.APIKey == "" {
		return nil, errors.New("API key is required; set it in config or via UCLOUD_SANDBOX_API_KEY")
	}
	domain := resolveDomain(cfg)
	return client.New(client.Options{
		APIKey:       cfg.APIKey,
		Domain:       domain,
		InsecureHTTP: cfg.InsecureHTTP,
	})
}
