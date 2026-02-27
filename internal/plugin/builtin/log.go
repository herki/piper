package builtin

import (
	"context"
	"fmt"

	"piper/internal/plugin"
	"piper/internal/types"
)

// LogConnector prints messages for debugging flows.
type LogConnector struct{}

func NewLogConnector() *LogConnector { return &LogConnector{} }

func (l *LogConnector) Name() string { return "log" }

func (l *LogConnector) Actions() []plugin.ActionDef {
	return []plugin.ActionDef{
		{
			Name:        "print",
			Description: "Print a message to stdout",
			Input: map[string]types.FieldDef{
				"message": {Type: "string", Description: "Message to print", Required: true},
			},
			Output: map[string]types.FieldDef{
				"message": {Type: "string", Description: "The printed message"},
			},
		},
	}
}

func (l *LogConnector) Execute(_ context.Context, action string, input map[string]any) (*types.StepResult, error) {
	if action != "print" {
		return nil, fmt.Errorf("log connector: unknown action %q", action)
	}

	message := fmt.Sprintf("%v", input["message"])
	fmt.Println("[log]", message)

	return &types.StepResult{
		Status: "success",
		Output: map[string]any{
			"message": message,
		},
	}, nil
}

func (l *LogConnector) Validate() error { return nil }
