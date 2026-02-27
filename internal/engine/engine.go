package engine

import (
	"context"
	"fmt"
	"time"

	"piper/internal/plugin"
	"piper/internal/types"
)

// Engine executes flow definitions.
type Engine struct {
	Registry *plugin.Registry
}

// NewEngine creates a new flow execution engine.
func NewEngine(registry *plugin.Registry) *Engine {
	return &Engine{Registry: registry}
}

// Run executes a flow with the given input.
func (e *Engine) Run(ctx context.Context, flow *types.FlowDef, input map[string]any) (*types.FlowResult, error) {
	if err := ValidateInput(flow, input); err != nil {
		return nil, err
	}

	result := &types.FlowResult{
		Flow:      flow.Name,
		Status:    "success",
		StartedAt: time.Now().UTC(),
		Input:     input,
		Steps:     make([]types.StepResult, 0, len(flow.Steps)),
	}

	sctx := NewStepContext(input)

	for _, step := range flow.Steps {
		sr := e.executeStep(ctx, step, sctx)
		result.Steps = append(result.Steps, sr)
		sctx.AddStepResult(step.Name, &sr)

		if sr.Status == "failed" || sr.Status == "error" {
			onError := step.OnError
			if onError == "" {
				onError = "abort"
			}

			switch onError {
			case "abort":
				result.Status = "failed"
				result.Error = fmt.Sprintf("step %q failed: %s", step.Name, sr.Error)
				result.CompletedAt = time.Now().UTC()
				return result, nil
			case "continue":
				result.Status = "partial"
			case "skip":
				// Just skip, don't affect overall status.
			}
		}
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

// DryRun validates and resolves variables without actually executing steps.
func (e *Engine) DryRun(flow *types.FlowDef, input map[string]any) (*types.FlowResult, error) {
	if err := ValidateFlow(flow, e.Registry); err != nil {
		return nil, err
	}
	if err := ValidateInput(flow, input); err != nil {
		return nil, err
	}

	result := &types.FlowResult{
		Flow:      flow.Name,
		Status:    "dry_run",
		StartedAt: time.Now().UTC(),
		Input:     input,
		Steps:     make([]types.StepResult, 0, len(flow.Steps)),
	}

	sctx := NewStepContext(input)

	for _, step := range flow.Steps {
		resolvedInput, err := sctx.ResolveMap(step.Input)

		sr := types.StepResult{
			Name:      step.Name,
			Connector: step.Connector,
			Action:    step.Action,
			Status:    "dry_run",
		}

		if err != nil {
			sr.Status = "resolve_error"
			sr.Error = err.Error()
		} else {
			sr.Output = resolvedInput // Show what would be sent.
		}

		result.Steps = append(result.Steps, sr)
		// For dry-run, add a synthetic step result so later steps can reference it.
		sctx.AddStepResult(step.Name, &types.StepResult{
			Status: "dry_run",
			Output: map[string]any{"_dry_run": true},
		})
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

func (e *Engine) executeStep(ctx context.Context, step types.StepDef, sctx *StepContext) types.StepResult {
	sr := types.StepResult{
		Name:      step.Name,
		Connector: step.Connector,
		Action:    step.Action,
	}

	start := time.Now()
	defer func() {
		sr.DurationMs = time.Since(start).Milliseconds()
	}()

	conn, ok := e.Registry.Get(step.Connector)
	if !ok {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("connector %q not found", step.Connector)
		return sr
	}

	resolvedInput, err := sctx.ResolveMap(step.Input)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("resolving input: %v", err)
		return sr
	}

	stepResult, err := conn.Execute(ctx, step.Action, resolvedInput)
	if err != nil {
		sr.Status = "error"
		sr.Error = err.Error()
		return sr
	}

	sr.Status = stepResult.Status
	sr.Output = stepResult.Output
	sr.Error = stepResult.Error
	return sr
}
