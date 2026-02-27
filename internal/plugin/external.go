package plugin

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"piper/internal/types"
)

// ExternalConnector wraps an external executable that speaks JSON over stdin/stdout.
// Protocol:
//
//	Request:  {"action": "<action>", "input": {...}}
//	Response: {"status": "success|failed", "output": {...}, "error": "..."}
//	Metadata: run with --describe to get {"name": "...", "actions": [...]}
type ExternalConnector struct {
	name    string
	path    string
	actions []ActionDef
}

type externalDescribe struct {
	Name    string      `json:"name"`
	Actions []ActionDef `json:"actions"`
}

type externalRequest struct {
	Action string         `json:"action"`
	Input  map[string]any `json:"input"`
}

type externalResponse struct {
	Status string         `json:"status"`
	Output map[string]any `json:"output"`
	Error  string         `json:"error"`
}

// LoadExternalPlugin loads an external plugin from an executable path.
func LoadExternalPlugin(path string) (*ExternalConnector, error) {
	// Run with --describe to get metadata.
	cmd := exec.Command(path, "--describe")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("running %s --describe: %w", path, err)
	}

	var desc externalDescribe
	if err := json.Unmarshal(out, &desc); err != nil {
		return nil, fmt.Errorf("parsing describe output from %s: %w", path, err)
	}

	if desc.Name == "" {
		desc.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	return &ExternalConnector{
		name:    desc.Name,
		path:    path,
		actions: desc.Actions,
	}, nil
}

func (ec *ExternalConnector) Name() string         { return ec.name }
func (ec *ExternalConnector) Actions() []ActionDef { return ec.actions }

func (ec *ExternalConnector) Execute(ctx context.Context, action string, input map[string]any) (*types.StepResult, error) {
	req := externalRequest{
		Action: action,
		Input:  input,
	}

	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	cmd := exec.CommandContext(ctx, ec.path)
	cmd.Stdin = strings.NewReader(string(reqJSON))

	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return &types.StepResult{
				Status: "error",
				Error:  fmt.Sprintf("plugin exited with code %d: %s", exitErr.ExitCode(), string(exitErr.Stderr)),
			}, nil
		}
		return nil, fmt.Errorf("running plugin: %w", err)
	}

	var resp externalResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("parsing plugin response: %w", err)
	}

	return &types.StepResult{
		Status: resp.Status,
		Output: resp.Output,
		Error:  resp.Error,
	}, nil
}

func (ec *ExternalConnector) Validate() error {
	_, err := os.Stat(ec.path)
	return err
}

// LoadExternalPlugins discovers and loads all plugins from a directory.
// Plugins are executable files in the directory.
func LoadExternalPlugins(dir string) ([]*ExternalConnector, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading plugins directory: %w", err)
	}

	var plugins []*ExternalConnector
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		path := filepath.Join(dir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}
		// Check if executable.
		if info.Mode()&0111 == 0 {
			continue
		}

		plugin, err := LoadExternalPlugin(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to load plugin %s: %v\n", path, err)
			continue
		}
		plugins = append(plugins, plugin)
	}

	return plugins, nil
}
