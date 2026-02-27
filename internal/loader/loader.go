package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"piper/internal/types"
)

// LoadFlow reads and parses a single YAML flow file.
func LoadFlow(path string) (*types.FlowDef, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading flow file %s: %w", path, err)
	}

	var flow types.FlowDef
	if err := yaml.Unmarshal(data, &flow); err != nil {
		return nil, fmt.Errorf("parsing flow file %s: %w", path, err)
	}

	if flow.Name == "" {
		return nil, fmt.Errorf("flow file %s: missing required field 'name'", path)
	}
	if len(flow.Steps) == 0 {
		return nil, fmt.Errorf("flow file %s: must have at least one step", path)
	}

	return &flow, nil
}

// LoadFlows reads all YAML flow files from a directory.
func LoadFlows(dir string) (map[string]*types.FlowDef, error) {
	flows := make(map[string]*types.FlowDef)

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading flows directory %s: %w", dir, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".yaml" && ext != ".yml" {
			continue
		}

		path := filepath.Join(dir, entry.Name())
		flow, err := LoadFlow(path)
		if err != nil {
			return nil, err
		}

		if _, exists := flows[flow.Name]; exists {
			return nil, fmt.Errorf("duplicate flow name %q in %s", flow.Name, path)
		}
		flows[flow.Name] = flow
	}

	return flows, nil
}
