package builtin

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"piper/internal/plugin"
	"piper/internal/types"
)

// ShellConnector executes shell commands.
type ShellConnector struct{}

func NewShellConnector() *ShellConnector { return &ShellConnector{} }

func (s *ShellConnector) Name() string { return "shell" }

func (s *ShellConnector) Actions() []plugin.ActionDef {
	return []plugin.ActionDef{
		{
			Name:        "run",
			Description: "Execute a shell command",
			Input: map[string]types.FieldDef{
				"command": {Type: "string", Description: "Command to execute", Required: true},
				"dir":     {Type: "string", Description: "Working directory", Required: false},
			},
			Output: map[string]types.FieldDef{
				"stdout":    {Type: "string", Description: "Standard output"},
				"stderr":    {Type: "string", Description: "Standard error"},
				"exit_code": {Type: "integer", Description: "Exit code"},
			},
		},
	}
}

func (s *ShellConnector) Execute(ctx context.Context, action string, input map[string]any) (*types.StepResult, error) {
	if action != "run" {
		return nil, fmt.Errorf("shell connector: unknown action %q", action)
	}

	command, _ := input["command"].(string)
	if command == "" {
		return nil, fmt.Errorf("shell connector: 'command' is required")
	}

	cmd := exec.CommandContext(ctx, "sh", "-c", command)

	if dir, ok := input["dir"].(string); ok && dir != "" {
		cmd.Dir = dir
	}

	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return nil, fmt.Errorf("shell connector: %w", err)
		}
	}

	status := "success"
	if exitCode != 0 {
		status = "failed"
	}

	return &types.StepResult{
		Status: status,
		Output: map[string]any{
			"stdout":    strings.TrimRight(stdout.String(), "\n"),
			"stderr":    strings.TrimRight(stderr.String(), "\n"),
			"exit_code": exitCode,
		},
	}, nil
}

func (s *ShellConnector) Validate() error { return nil }
