package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// configFileName is the per-template file that records what `build` should use
// when the command line says nothing.
const configFileName = "ucloud-template.json"

// LocalConfig is the local template configuration file.
type LocalConfig struct {
	TemplateName string `json:"template_name,omitempty"`
	TemplateID   string `json:"template_id,omitempty"`
	CPUCount     int32  `json:"cpu_count,omitempty"`
	MemoryMB     int32  `json:"memory_mb,omitempty"`
	Dockerfile   string `json:"dockerfile,omitempty"`
}

// configPath returns the config file's path inside a template directory.
func configPath(root string) string {
	return filepath.Join(root, configFileName)
}

// loadConfig reads the local template config out of a directory.
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

// saveConfig writes the local template config into a directory.
func saveConfig(root string, cfg *LocalConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}

	if err := os.WriteFile(configPath(root), data, 0644); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	return nil
}

// deleteConfig removes the local template config file. A directory that never
// had one is not an error.
func deleteConfig(root string) error {
	if err := os.Remove(configPath(root)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("delete config: %w", err)
	}

	return nil
}
