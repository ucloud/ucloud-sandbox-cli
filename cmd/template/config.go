package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// LocalConfig represents the local template configuration file.
type LocalConfig struct {
	TemplateName string `json:"template_name,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	CPUCount     int    `json:"cpu_count,omitempty"`
	MemoryMB     int    `json:"memory_mb,omitempty"`
	Dockerfile   string `json:"dockerfile,omitempty"`
}

const configFileName = "ucloud-template.json"

// configPath returns the full path to the config file.
func configPath(root string) string {
	return filepath.Join(root, configFileName)
}

// loadConfig reads the local template config from the specified directory.
func loadConfig(root string) (*LocalConfig, error) {
	path := configPath(root)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg LocalConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	return &cfg, nil
}

// saveConfig writes the local template config to the specified directory.
func saveConfig(root string, cfg *LocalConfig) error {
	path := configPath(root)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// deleteConfig removes the local template config file.
func deleteConfig(root string) error {
	path := configPath(root)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete config: %w", err)
	}
	return nil
}
