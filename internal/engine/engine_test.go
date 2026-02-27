package engine

import (
	"context"
	"fmt"
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

func TestEngineConditionalStepSkip(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "always-run",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "hello"},
			},
			{
				Name:      "conditional-skip",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "should not run"},
				When:      `${{ input.run_this == "yes" }}`,
			},
			{
				Name:      "conditional-run",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "should run"},
				When:      `${{ input.mode == "active" }}`,
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{"mode": "active"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want success", result.Status)
	}
	if len(result.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(result.Steps))
	}
	if result.Steps[0].Status != "success" {
		t.Errorf("step 0 status = %q, want success", result.Steps[0].Status)
	}
	if result.Steps[1].Status != "skipped" {
		t.Errorf("step 1 status = %q, want skipped", result.Steps[1].Status)
	}
	if result.Steps[2].Status != "success" {
		t.Errorf("step 2 status = %q, want success", result.Steps[2].Status)
	}
}

func TestEngineConditionalOnStepStatus(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewShellConnector())
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "step1",
				Connector: "shell",
				Action:    "run",
				Input:     map[string]any{"command": "echo ok"},
			},
			{
				Name:      "only-on-success",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "step1 succeeded"},
				When:      `${{ steps.step1.status == "success" }}`,
			},
			{
				Name:      "only-on-failure",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "step1 failed"},
				When:      `${{ steps.step1.status == "failed" }}`,
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Steps[1].Status != "success" {
		t.Errorf("only-on-success should have run, got %q", result.Steps[1].Status)
	}
	if result.Steps[2].Status != "skipped" {
		t.Errorf("only-on-failure should be skipped, got %q", result.Steps[2].Status)
	}
}

func TestEngineParallelExecution(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewShellConnector())
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name: "parallel-group",
				Parallel: []types.StepDef{
					{
						Name:      "p1",
						Connector: "shell",
						Action:    "run",
						Input:     map[string]any{"command": "echo one"},
					},
					{
						Name:      "p2",
						Connector: "shell",
						Action:    "run",
						Input:     map[string]any{"command": "echo two"},
					},
					{
						Name:      "p3",
						Connector: "log",
						Action:    "print",
						Input:     map[string]any{"message": "three"},
					},
				},
			},
			{
				Name:      "after-parallel",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "p1=${{ steps.p1.output.stdout }}, p2=${{ steps.p2.output.stdout }}"},
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want success", result.Status)
	}
	// 3 parallel + 1 sequential = 4 steps.
	if len(result.Steps) != 4 {
		t.Fatalf("expected 4 steps, got %d", len(result.Steps))
	}
	for _, sr := range result.Steps {
		if sr.Status != "success" {
			t.Errorf("step %q status = %q, want success", sr.Name, sr.Status)
		}
	}
}

func TestEngineRetry(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewShellConnector())
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	// This will always fail, testing that retry is attempted.
	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "flaky",
				Connector: "shell",
				Action:    "run",
				Input:     map[string]any{"command": "exit 1"},
				OnError:   "retry",
				Retry:     &types.RetryConfig{MaxRetries: 2, BackoffSeconds: 0.01},
			},
			{
				Name:      "after",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "after retry"},
			},
		},
	}

	result, err := eng.Run(context.Background(), flow, map[string]any{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// After retries exhausted, on_error=retry falls through to abort behavior.
	if result.Steps[0].Retries != 2 {
		t.Errorf("retries = %d, want 2", result.Steps[0].Retries)
	}
}

func TestEngineFlowComposition(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	childFlow := &types.FlowDef{
		Name: "child",
		Steps: []types.StepDef{
			{
				Name:      "child-step",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "hello from child: ${{ input.name }}"},
			},
		},
	}

	eng.FlowLoader = func(name string) (*types.FlowDef, error) {
		if name == "child" {
			return childFlow, nil
		}
		return nil, fmt.Errorf("not found")
	}

	parentFlow := &types.FlowDef{
		Name: "parent",
		Steps: []types.StepDef{
			{
				Name:      "call-child",
				Connector: "flow",
				Flow:      "child",
				Input:     map[string]any{"name": "${{ input.name }}"},
			},
		},
	}

	result, err := eng.Run(context.Background(), parentFlow, map[string]any{"name": "World"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want success", result.Status)
	}
	if result.Steps[0].Connector != "flow" {
		t.Errorf("connector = %q, want flow", result.Steps[0].Connector)
	}
	if result.Steps[0].Output["flow_status"] != "success" {
		t.Errorf("flow_status = %v, want success", result.Steps[0].Output["flow_status"])
	}
}

func TestEngineSecretsIntegration(t *testing.T) {
	registry := plugin.NewRegistry()
	registry.Register(builtin.NewLogConnector())

	eng := NewEngine(registry)

	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{
				Name:      "use-secret",
				Connector: "log",
				Action:    "print",
				Input:     map[string]any{"message": "key=${{ secret.API_KEY }}"},
			},
		},
	}

	secrets := map[string]string{"API_KEY": "sk-test-123"}
	result, err := eng.RunWithSecrets(context.Background(), flow, map[string]any{}, secrets)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Status != "success" {
		t.Errorf("status = %q, want success", result.Status)
	}
}
