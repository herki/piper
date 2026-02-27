package engine

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"piper/internal/plugin"
	"piper/internal/types"
)

// Engine executes flow definitions.
type Engine struct {
	Registry *plugin.Registry
	// FlowLoader is set when flow composition is enabled (avoids import cycle).
	FlowLoader func(name string) (*types.FlowDef, error)
}

// NewEngine creates a new flow execution engine.
func NewEngine(registry *plugin.Registry) *Engine {
	return &Engine{Registry: registry}
}

// RunWithSecrets executes a flow with the given input and secrets.
func (e *Engine) RunWithSecrets(ctx context.Context, flow *types.FlowDef, input map[string]any, secrets map[string]string) (*types.FlowResult, error) {
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
	if secrets != nil {
		sctx.Secrets = secrets
	}

	return e.runWithContext(ctx, flow, result, sctx)
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

	return e.runWithContext(ctx, flow, result, sctx)
}

func (e *Engine) runWithContext(ctx context.Context, flow *types.FlowDef, result *types.FlowResult, sctx *StepContext) (*types.FlowResult, error) {

	for _, step := range flow.Steps {
		// Handle parallel step groups.
		if len(step.Parallel) > 0 {
			results := e.executeParallel(ctx, step.Parallel, sctx)
			for _, sr := range results {
				result.Steps = append(result.Steps, sr)
				sctx.AddStepResult(sr.Name, &sr)
				if failed := e.handleStepError(&sr, step.OnError, result); failed {
					result.CompletedAt = time.Now().UTC()
					return result, nil
				}
			}
			continue
		}

		// Evaluate conditional.
		if step.When != "" {
			shouldRun, err := sctx.EvaluateCondition(step.When)
			if err != nil {
				sr := types.StepResult{
					Name:      step.Name,
					Connector: step.Connector,
					Action:    step.Action,
					Status:    "error",
					Error:     fmt.Sprintf("evaluating condition: %v", err),
				}
				result.Steps = append(result.Steps, sr)
				sctx.AddStepResult(step.Name, &sr)
				if failed := e.handleStepError(&sr, step.OnError, result); failed {
					result.CompletedAt = time.Now().UTC()
					return result, nil
				}
				continue
			}
			if !shouldRun {
				sr := types.StepResult{
					Name:      step.Name,
					Connector: step.Connector,
					Action:    step.Action,
					Status:    "skipped",
				}
				result.Steps = append(result.Steps, sr)
				sctx.AddStepResult(step.Name, &sr)
				continue
			}
		}

		sr := e.executeStepWithRetry(ctx, step, sctx)
		result.Steps = append(result.Steps, sr)
		sctx.AddStepResult(step.Name, &sr)

		if failed := e.handleStepError(&sr, step.OnError, result); failed {
			result.CompletedAt = time.Now().UTC()
			return result, nil
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
		// Flatten parallel groups for dry-run.
		steps := []types.StepDef{step}
		if len(step.Parallel) > 0 {
			steps = step.Parallel
		}

		for _, s := range steps {
			resolvedInput, err := sctx.ResolveMap(s.Input)

			sr := types.StepResult{
				Name:      s.Name,
				Connector: s.Connector,
				Action:    s.Action,
				Status:    "dry_run",
			}

			if s.When != "" {
				sr.Output = map[string]any{"_when": s.When}
			}

			if err != nil {
				sr.Status = "resolve_error"
				sr.Error = err.Error()
			} else {
				if sr.Output == nil {
					sr.Output = resolvedInput
				} else {
					for k, v := range resolvedInput {
						sr.Output[k] = v
					}
				}
			}

			result.Steps = append(result.Steps, sr)
			sctx.AddStepResult(s.Name, &types.StepResult{
				Status: "dry_run",
				Output: map[string]any{"_dry_run": true},
			})
		}
	}

	result.CompletedAt = time.Now().UTC()
	return result, nil
}

// handleStepError processes a step failure based on its on_error policy.
// Returns true if the flow should abort.
func (e *Engine) handleStepError(sr *types.StepResult, onError string, result *types.FlowResult) bool {
	if sr.Status != "failed" && sr.Status != "error" {
		return false
	}

	if onError == "" {
		onError = "abort"
	}

	switch onError {
	case "abort":
		result.Status = "failed"
		result.Error = fmt.Sprintf("step %q failed: %s", sr.Name, sr.Error)
		return true
	case "continue":
		result.Status = "partial"
	case "skip":
		// Don't affect overall status.
	}
	return false
}

// executeStepWithRetry executes a step, retrying on failure if configured.
func (e *Engine) executeStepWithRetry(ctx context.Context, step types.StepDef, sctx *StepContext) types.StepResult {
	sr := e.executeStep(ctx, step, sctx)

	if step.Retry == nil || step.OnError != "retry" {
		return sr
	}

	maxRetries := step.Retry.MaxRetries
	if maxRetries <= 0 {
		maxRetries = 3
	}
	backoff := step.Retry.BackoffSeconds
	if backoff <= 0 {
		backoff = 1.0
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if sr.Status != "failed" && sr.Status != "error" {
			break
		}

		// Exponential backoff.
		sleepDuration := time.Duration(backoff*math.Pow(2, float64(attempt-1))) * time.Second
		select {
		case <-ctx.Done():
			sr.Status = "error"
			sr.Error = "context cancelled during retry"
			sr.Retries = attempt
			return sr
		case <-time.After(sleepDuration):
		}

		sr = e.executeStep(ctx, step, sctx)
		sr.Retries = attempt
	}

	return sr
}

// executeParallel runs multiple steps concurrently and collects results.
func (e *Engine) executeParallel(ctx context.Context, steps []types.StepDef, sctx *StepContext) []types.StepResult {
	results := make([]types.StepResult, len(steps))
	var wg sync.WaitGroup

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, s types.StepDef) {
			defer wg.Done()

			// Evaluate conditional.
			if s.When != "" {
				shouldRun, err := sctx.EvaluateCondition(s.When)
				if err != nil {
					results[idx] = types.StepResult{
						Name:      s.Name,
						Connector: s.Connector,
						Action:    s.Action,
						Status:    "error",
						Error:     fmt.Sprintf("evaluating condition: %v", err),
					}
					return
				}
				if !shouldRun {
					results[idx] = types.StepResult{
						Name:      s.Name,
						Connector: s.Connector,
						Action:    s.Action,
						Status:    "skipped",
					}
					return
				}
			}

			results[idx] = e.executeStepWithRetry(ctx, s, sctx)
		}(i, step)
	}

	wg.Wait()
	return results
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

	// Handle flow composition — connector "flow" calls another flow.
	if step.Connector == "flow" {
		return e.executeFlowStep(ctx, step, sctx)
	}

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

// executeFlowStep runs another flow as a step (flow composition).
func (e *Engine) executeFlowStep(ctx context.Context, step types.StepDef, sctx *StepContext) types.StepResult {
	sr := types.StepResult{
		Name:      step.Name,
		Connector: "flow",
		Action:    "run",
	}

	start := time.Now()
	defer func() {
		sr.DurationMs = time.Since(start).Milliseconds()
	}()

	if e.FlowLoader == nil {
		sr.Status = "error"
		sr.Error = "flow composition not configured (no FlowLoader set)"
		return sr
	}

	flowName := step.Flow
	if flowName == "" {
		if name, ok := step.Input["flow"].(string); ok {
			flowName = name
		}
	}
	if flowName == "" {
		sr.Status = "error"
		sr.Error = "flow step requires 'flow' field or input.flow"
		return sr
	}

	childFlow, err := e.FlowLoader(flowName)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("loading flow %q: %v", flowName, err)
		return sr
	}

	// Resolve input for the child flow.
	childInput, err := sctx.ResolveMap(step.Input)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("resolving child flow input: %v", err)
		return sr
	}
	// Remove the "flow" key from input — it's not an input field.
	delete(childInput, "flow")

	childResult, err := e.Run(ctx, childFlow, childInput)
	if err != nil {
		sr.Status = "error"
		sr.Error = fmt.Sprintf("running flow %q: %v", flowName, err)
		return sr
	}

	sr.Status = childResult.Status
	sr.Output = map[string]any{
		"flow_status": childResult.Status,
		"steps":       len(childResult.Steps),
	}
	// Merge child flow step outputs into parent output.
	for _, childStep := range childResult.Steps {
		if childStep.Output != nil {
			for k, v := range childStep.Output {
				sr.Output[k] = v
			}
		}
	}
	if childResult.Error != "" {
		sr.Error = childResult.Error
	}

	return sr
}
