package engine

import (
	"fmt"
	"strings"

	"piper/internal/plugin"
	"piper/internal/types"
)

// ValidationError collects multiple validation issues.
type ValidationError struct {
	Errors []string
}

func (ve *ValidationError) Error() string {
	return fmt.Sprintf("validation failed:\n  - %s", strings.Join(ve.Errors, "\n  - "))
}

func (ve *ValidationError) Add(msg string) {
	ve.Errors = append(ve.Errors, msg)
}

func (ve *ValidationError) HasErrors() bool {
	return len(ve.Errors) > 0
}

// ValidateFlow validates a flow definition against the registry and internal consistency.
func ValidateFlow(flow *types.FlowDef, registry *plugin.Registry) error {
	ve := &ValidationError{}

	if flow.Name == "" {
		ve.Add("flow 'name' is required")
	}
	if len(flow.Steps) == 0 {
		ve.Add("flow must have at least one step")
	}

	stepNames := make(map[string]int)
	for i, step := range flow.Steps {
		if step.Name == "" {
			ve.Add(fmt.Sprintf("step %d: 'name' is required", i+1))
			continue
		}
		if prev, exists := stepNames[step.Name]; exists {
			ve.Add(fmt.Sprintf("step %d: duplicate step name %q (first at step %d)", i+1, step.Name, prev+1))
		}
		stepNames[step.Name] = i

		if step.Connector == "" {
			ve.Add(fmt.Sprintf("step %q: 'connector' is required", step.Name))
		} else if !registry.Has(step.Connector) {
			ve.Add(fmt.Sprintf("step %q: connector %q not found in registry", step.Name, step.Connector))
		} else {
			conn, _ := registry.Get(step.Connector)
			if step.Action != "" {
				found := false
				for _, a := range conn.Actions() {
					if a.Name == step.Action {
						found = true
						break
					}
				}
				if !found {
					ve.Add(fmt.Sprintf("step %q: connector %q does not support action %q", step.Name, step.Connector, step.Action))
				}
			}
		}

		switch step.OnError {
		case "", "abort", "continue", "skip":
			// valid
		default:
			ve.Add(fmt.Sprintf("step %q: invalid on_error value %q (must be abort, continue, or skip)", step.Name, step.OnError))
		}

		// Validate step references point to previous steps.
		if step.Input != nil {
			validateStepRefs(step.Input, stepNames, step.Name, i, ve)
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

// ValidateInput checks that required input fields are present.
func ValidateInput(flow *types.FlowDef, input map[string]any) error {
	if flow.Input == nil {
		return nil
	}

	ve := &ValidationError{}
	for name, field := range flow.Input.Properties {
		if field.Required {
			if _, ok := input[name]; !ok {
				ve.Add(fmt.Sprintf("required input field %q is missing", name))
			}
		}
	}

	if ve.HasErrors() {
		return ve
	}
	return nil
}

func validateStepRefs(input map[string]any, stepNames map[string]int, currentStep string, currentIndex int, ve *ValidationError) {
	for _, v := range input {
		switch val := v.(type) {
		case string:
			checkStringRefs(val, stepNames, currentStep, currentIndex, ve)
		case map[string]any:
			validateStepRefs(val, stepNames, currentStep, currentIndex, ve)
		case []any:
			for _, item := range val {
				if s, ok := item.(string); ok {
					checkStringRefs(s, stepNames, currentStep, currentIndex, ve)
				}
			}
		}
	}
}

func checkStringRefs(s string, stepNames map[string]int, currentStep string, currentIndex int, ve *ValidationError) {
	matches := exprRegex.FindAllStringSubmatch(s, -1)
	for _, match := range matches {
		expr := strings.TrimSpace(match[1])
		path := strings.SplitN(expr, "|", 2)[0]
		path = strings.TrimSpace(path)

		if !strings.HasPrefix(path, "steps.") {
			continue
		}

		// Extract step name from "steps.step-name.output.field"
		rest := strings.TrimPrefix(path, "steps.")
		parts := strings.SplitN(rest, ".", 2)
		refName := parts[0]

		idx, exists := stepNames[refName]
		if !exists {
			ve.Add(fmt.Sprintf("step %q: references unknown step %q", currentStep, refName))
		} else if idx >= currentIndex {
			ve.Add(fmt.Sprintf("step %q: references step %q which has not executed yet", currentStep, refName))
		}
	}
}
