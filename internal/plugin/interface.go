package plugin

import (
	"context"

	"piper/internal/types"
)

// Connector defines the interface that all flow connectors must implement.
type Connector interface {
	// Name returns the connector identifier (e.g., "http", "shell", "log").
	Name() string

	// Actions returns available actions with their input/output schemas.
	Actions() []ActionDef

	// Execute runs a specific action with the given input.
	Execute(ctx context.Context, action string, input map[string]any) (*types.StepResult, error)

	// Validate checks if the connector is properly configured.
	Validate() error
}

// ActionDef describes an action a connector supports.
type ActionDef struct {
	Name        string
	Description string
	Input       map[string]types.FieldDef
	Output      map[string]types.FieldDef
}
