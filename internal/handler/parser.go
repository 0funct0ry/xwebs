package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadConfig reads a YAML file from the given path and returns a parsed Config.
func LoadConfig(path string, mode RegistryMode) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading handler config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing handler config YAML: %w", err)
	}

	// Set base directory for relative path resolution (e.g. for JSON Schema)
	// Ensure it is an absolute path to satisfy gojsonschema's canonical requirement
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolving absolute path for %s: %w", path, err)
	}
	cfg.BaseDir = filepath.Dir(absPath)
	for i := range cfg.Handlers {
		cfg.Handlers[i].BaseDir = cfg.BaseDir
	}

	if err := cfg.Validate(mode); err != nil {
		// Provide a more user-friendly error message for validation failures
		return nil, fmt.Errorf("invalid handler configuration in %s: %w", path, err)
	}

	return &cfg, nil
}
