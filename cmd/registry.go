package cmd

import (
	"piper/internal/plugin"
	"piper/internal/plugin/builtin"
)

func defaultRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	r.Register(builtin.NewHTTPConnector())
	r.Register(builtin.NewShellConnector())
	r.Register(builtin.NewLogConnector())
	r.Register(builtin.NewWebhookConnector())
	return r
}
