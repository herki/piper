package builtin

import (
	"context"
	"fmt"

	"piper/internal/plugin"
	"piper/internal/types"
)

// WebhookConnector is a placeholder for webhook trigger handling.
// Actual webhook serving is done by the server package; this connector
// exists so that flows referencing "webhook" as a trigger type can be validated.
type WebhookConnector struct{}

func NewWebhookConnector() *WebhookConnector { return &WebhookConnector{} }

func (w *WebhookConnector) Name() string { return "webhook" }

func (w *WebhookConnector) Actions() []plugin.ActionDef {
	return []plugin.ActionDef{
		{
			Name:        "trigger",
			Description: "Webhook trigger (handled by the server)",
		},
	}
}

func (w *WebhookConnector) Execute(_ context.Context, action string, _ map[string]any) (*types.StepResult, error) {
	return nil, fmt.Errorf("webhook connector: cannot be executed directly; it is a trigger-only connector")
}

func (w *WebhookConnector) Validate() error { return nil }
