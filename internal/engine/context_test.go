package engine

import (
	"testing"

	"piper/internal/types"
)

func TestResolveInputVariables(t *testing.T) {
	ctx := NewStepContext(map[string]any{
		"name":  "Acme Corp",
		"email": "cto@acme.com",
	})

	tests := []struct {
		input    string
		expected string
	}{
		{"${{ input.name }}", "Acme Corp"},
		{"Hello ${{ input.name }}!", "Hello Acme Corp!"},
		{"${{ input.email }}", "cto@acme.com"},
	}

	for _, tt := range tests {
		result, err := ctx.resolveString(tt.input)
		if err != nil {
			t.Errorf("resolveString(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("resolveString(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestResolveStepOutputVariables(t *testing.T) {
	ctx := NewStepContext(map[string]any{"name": "Test"})
	ctx.AddStepResult("step1", &types.StepResult{
		Status: "success",
		Output: map[string]any{
			"repo_url": "https://github.com/test",
			"count":    42,
		},
	})

	val, err := ctx.resolveString("${{ steps.step1.output.repo_url }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "https://github.com/test" {
		t.Errorf("got %v, want https://github.com/test", val)
	}

	val2, err := ctx.resolveString("${{ steps.step1.output.count }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val2 != 42 {
		t.Errorf("got %v (%T), want 42 (int)", val2, val2)
	}
}

func TestResolvePipeFunctions(t *testing.T) {
	ctx := NewStepContext(map[string]any{"name": "Acme Corp"})

	tests := []struct {
		input    string
		expected string
	}{
		{"${{ input.name | slugify }}", "acme-corp"},
		{"${{ input.name | upper }}", "ACME CORP"},
		{"${{ input.name | lower }}", "acme corp"},
	}

	for _, tt := range tests {
		result, err := ctx.resolveString(tt.input)
		if err != nil {
			t.Errorf("resolveString(%q) error: %v", tt.input, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("resolveString(%q) = %v, want %v", tt.input, result, tt.expected)
		}
	}
}

func TestResolveEnvVariables(t *testing.T) {
	ctx := NewStepContext(map[string]any{})
	val, err := ctx.resolveString("${{ env.HOME }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val == "" {
		t.Error("expected non-empty HOME env var")
	}
}

func TestResolveSecrets(t *testing.T) {
	ctx := NewStepContext(map[string]any{})
	ctx.Secrets["API_KEY"] = "secret123"

	val, err := ctx.resolveString("${{ secret.API_KEY }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != "secret123" {
		t.Errorf("got %v, want secret123", val)
	}

	// Missing secret returns empty string.
	val2, err := ctx.resolveString("${{ secret.MISSING }}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val2 != "" {
		t.Errorf("got %v, want empty string", val2)
	}
}

func TestResolveMap(t *testing.T) {
	ctx := NewStepContext(map[string]any{"name": "Test"})

	input := map[string]any{
		"title":   "Hello ${{ input.name }}",
		"literal": "no variables here",
		"number":  42,
	}

	result, err := ctx.ResolveMap(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result["title"] != "Hello Test" {
		t.Errorf("title = %v, want Hello Test", result["title"])
	}
	if result["literal"] != "no variables here" {
		t.Errorf("literal changed unexpectedly: %v", result["literal"])
	}
	if result["number"] != 42 {
		t.Errorf("number = %v, want 42", result["number"])
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"Acme Corp", "acme-corp"},
		{"Hello   World", "hello-world"},
		{"test-already-slug", "test-already-slug"},
		{"  leading trailing  ", "leading-trailing"},
		{"Special!@#Chars", "specialchars"},
	}

	for _, tt := range tests {
		result := slugify(tt.input)
		if result != tt.expected {
			t.Errorf("slugify(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestEvaluateConditionEquals(t *testing.T) {
	ctx := NewStepContext(map[string]any{"status": "active"})
	ctx.AddStepResult("step1", &types.StepResult{Status: "success"})

	tests := []struct {
		when     string
		expected bool
	}{
		{`${{ steps.step1.status == "success" }}`, true},
		{`${{ steps.step1.status == "failed" }}`, false},
		{`${{ steps.step1.status != "failed" }}`, true},
		{`${{ input.status == "active" }}`, true},
		{`${{ input.status == "inactive" }}`, false},
	}

	for _, tt := range tests {
		result, err := ctx.EvaluateCondition(tt.when)
		if err != nil {
			t.Errorf("EvaluateCondition(%q) error: %v", tt.when, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.when, result, tt.expected)
		}
	}
}

func TestEvaluateConditionTruthy(t *testing.T) {
	ctx := NewStepContext(map[string]any{"flag": "true", "empty": ""})

	tests := []struct {
		when     string
		expected bool
	}{
		{`${{ input.flag }}`, true},
		{`${{ input.empty }}`, false},
		{`${{ true }}`, true},
		{`${{ false }}`, false},
		{"", true}, // empty when = always run
	}

	for _, tt := range tests {
		result, err := ctx.EvaluateCondition(tt.when)
		if err != nil {
			t.Errorf("EvaluateCondition(%q) error: %v", tt.when, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.when, result, tt.expected)
		}
	}
}

func TestEvaluateConditionNumeric(t *testing.T) {
	ctx := NewStepContext(map[string]any{})
	ctx.AddStepResult("api", &types.StepResult{
		Status: "success",
		Output: map[string]any{"status_code": 200},
	})

	tests := []struct {
		when     string
		expected bool
	}{
		{`${{ steps.api.output.status_code == "200" }}`, true},
		{`${{ steps.api.output.status_code >= "200" }}`, true},
		{`${{ steps.api.output.status_code < "300" }}`, true},
		{`${{ steps.api.output.status_code > "300" }}`, false},
	}

	for _, tt := range tests {
		result, err := ctx.EvaluateCondition(tt.when)
		if err != nil {
			t.Errorf("EvaluateCondition(%q) error: %v", tt.when, err)
			continue
		}
		if result != tt.expected {
			t.Errorf("EvaluateCondition(%q) = %v, want %v", tt.when, result, tt.expected)
		}
	}
}
