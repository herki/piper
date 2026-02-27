package engine

import (
	"strings"
	"testing"

	"piper/internal/plugin"
	"piper/internal/plugin/builtin"
	"piper/internal/types"
)

func testRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	r.Register(builtin.NewHTTPConnector())
	r.Register(builtin.NewShellConnector())
	r.Register(builtin.NewLogConnector())
	return r
}

func TestValidateFlowValid(t *testing.T) {
	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{Name: "step1", Connector: "http", Action: "request", Input: map[string]any{"url": "http://example.com"}},
			{Name: "step2", Connector: "log", Action: "print", Input: map[string]any{"message": "${{ steps.step1.output.body }}"}},
		},
	}
	if err := ValidateFlow(flow, testRegistry()); err != nil {
		t.Errorf("expected valid flow, got: %v", err)
	}
}

func TestValidateFlowMissingName(t *testing.T) {
	flow := &types.FlowDef{
		Steps: []types.StepDef{
			{Name: "step1", Connector: "http", Action: "request"},
		},
	}
	err := ValidateFlow(flow, testRegistry())
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !strings.Contains(err.Error(), "'name' is required") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlowUnknownConnector(t *testing.T) {
	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{Name: "step1", Connector: "nonexistent", Action: "do"},
		},
	}
	err := ValidateFlow(flow, testRegistry())
	if err == nil {
		t.Fatal("expected error for unknown connector")
	}
	if !strings.Contains(err.Error(), "not found in registry") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateFlowForwardStepReference(t *testing.T) {
	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{Name: "step1", Connector: "log", Action: "print", Input: map[string]any{
				"message": "${{ steps.step2.output.body }}",
			}},
			{Name: "step2", Connector: "http", Action: "request", Input: map[string]any{"url": "http://example.com"}},
		},
	}
	err := ValidateFlow(flow, testRegistry())
	if err == nil {
		t.Fatal("expected error for forward step reference")
	}
	if !strings.Contains(err.Error(), "step2") {
		t.Errorf("expected error about step2, got: %v", err)
	}
}

func TestValidateFlowInvalidAction(t *testing.T) {
	flow := &types.FlowDef{
		Name: "test",
		Steps: []types.StepDef{
			{Name: "step1", Connector: "http", Action: "nonexistent"},
		},
	}
	err := ValidateFlow(flow, testRegistry())
	if err == nil {
		t.Fatal("expected error for invalid action")
	}
	if !strings.Contains(err.Error(), "does not support action") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateInputRequired(t *testing.T) {
	flow := &types.FlowDef{
		Name: "test",
		Input: &types.SchemaDef{
			Properties: map[string]types.FieldDef{
				"name":  {Type: "string", Required: true},
				"email": {Type: "string", Required: true},
			},
		},
		Steps: []types.StepDef{{Name: "s", Connector: "log", Action: "print"}},
	}

	err := ValidateInput(flow, map[string]any{"name": "test"})
	if err == nil {
		t.Fatal("expected error for missing required field")
	}
	if !strings.Contains(err.Error(), "email") {
		t.Errorf("expected error about 'email', got: %v", err)
	}

	err = ValidateInput(flow, map[string]any{"name": "test", "email": "a@b.com"})
	if err != nil {
		t.Errorf("expected no error, got: %v", err)
	}
}
