package engine

import (
	"context"
	"testing"

	"piper/internal/plugin"
	"piper/internal/plugin/builtin"
	"piper/internal/types"
)

func TestEngineRunSuccess(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "log-hello",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "hello ${{ input.name }}"},
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{"name": "world"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want success", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Fatalf("expected 1 step result, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "success" {
		t.Errorf("step status = %q, want success", result.Steps[0].Status)
	}
}

func TestEngineRunOnErrorAbort(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewShellConnector())
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "fail",
				Connector: "shell",
				Action:    "run",
				Input:     map[string]any{"command": "exit 1"},
				OnError:   "abort",
			},
			{
				Name:      "should-not-run",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "this should not run"},
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "failed" {
		t.Errorf("status = %q, want failed", result.Status)
	}
	if len(result.Steps) != 1 {
		t.Errorf("expected 1 step (aborted), got %d", len(result.Steps))
	}
}

func TestEngineRunOnErrorContinue(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewShellConnector())
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "fail",
				Connector: "shell",
				Action:    "run",
				Input:     map[string]any{"command": "exit 1"},
				OnError:   "continue",
			},
			{
				Name:      "log-after",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "still running"},
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "partial" {
		t.Errorf("status = %q, want partial", result.Status)
	}
	if len(result.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(result.Steps))
	}
}

func TestEngineDryRun(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewHTTPConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "fetch",
				Connector: "http",
				Action:    "request",
				Input:     map[string]any{"url": "${{ input.url }}", "method": "GET"},
			},
		},
	}

	result, err := eng.DryRun(flow, map[string]any{"url": "https://example.com"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "dry_run" {
		t.Errorf("status = %q, want dry_run", result.Status)
	}
	if result.Steps[0].Output["url"] != "https://example.com" {
		t.Errorf("resolved url = %v, want https://example.com", result.Steps[0].Output["url"])
	}
}

func TestEngineValidateInputFails(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Input: &types.SchemaDef{
			Properties: map[string]types.FieldDef{
				"required_field": {Type: "string", Required: true},
			},
		},
		Steps: []types.StepDef{
			{Name: "s", Connector: "log", Action: "print", Input: map[string]any{"message": "test"}},
		},
	}

	_, err := eng.Run(context.Background(), flow, map[string]any{})
	if err == nil {
		t.Fatal("expected error for missing required input")
	}
}
