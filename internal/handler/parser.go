package handler

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads a YAML file from the given path and returns a parsed Config.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading handler config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing handler config YAML: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		// Provide a more user-friendly error message for validation failures
		return nil, fmt.Errorf("invalid handler configuration in %s: %w", path, err)
	}

	return &cfg, nil
}
