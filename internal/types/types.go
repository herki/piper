package types

import "time"

// FlowDef represents a parsed YAML flow definition.
type FlowDef struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Description string            `yaml:"description" json:"description"`
	Input       *SchemaDef        `yaml:"input,omitempty" json:"input,omitempty"`
	Output      *SchemaDef        `yaml:"output,omitempty" json:"output,omitempty"`
	Trigger     *TriggerDef       `yaml:"trigger,omitempty" json:"trigger,omitempty"`
	Steps       []StepDef         `yaml:"steps" json:"steps"`
	Metadata    map[string]string `yaml:"metadata,omitempty" json:"metadata,omitempty"`
}

// SchemaDef describes the input or output schema of a flow.
type SchemaDef struct {
	Properties map[string]FieldDef `yaml:"properties" json:"properties"`
}

// FieldDef describes a single field in a schema.
type FieldDef struct {
	Type        string `yaml:"type" json:"type"`
	Description string `yaml:"description" json:"description"`
	Required    bool   `yaml:"required" json:"required"`
}

// TriggerDef describes how a flow is triggered.
type TriggerDef struct {
	Type string `yaml:"type" json:"type"`
	Path string `yaml:"path" json:"path"`
}

// StepDef represents a single step in a flow.
type StepDef struct {
	Name      string         `yaml:"name" json:"name"`
	Connector string         `yaml:"connector" json:"connector"`
	Action    string         `yaml:"action" json:"action"`
	Input     map[string]any `yaml:"input" json:"input"`
	OnError   string         `yaml:"on_error" json:"on_error"`
}

// StepResult holds the result of executing a single step.
type StepResult struct {
	Name       string         `json:"name"`
	Connector  string         `json:"connector"`
	Action     string         `json:"action"`
	Status     string         `json:"status"`
	Output     map[string]any `json:"output,omitempty"`
	Error      string         `json:"error,omitempty"`
	DurationMs int64          `json:"duration_ms"`
}

// FlowResult holds the result of an entire flow execution.
type FlowResult struct {
	Flow        string         `json:"flow"`
	Status      string         `json:"status"`
	StartedAt   time.Time      `json:"started_at"`
	CompletedAt time.Time      `json:"completed_at"`
	Input       map[string]any `json:"input"`
	Steps       []StepResult   `json:"steps"`
	Output      map[string]any `json:"output,omitempty"`
	Error       string         `json:"error,omitempty"`
}
