package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"piper/internal/plugin"
	"piper/internal/plugin/builtin"
)

var pluginsDir string

func defaultRegistry() *plugin.Registry {
	r := plugin.NewRegistry()
	r.Register(builtin.NewHTTPConnector())
	r.Register(builtin.NewShellConnector())
	r.Register(builtin.NewLogConnector())
	r.Register(builtin.NewWebhookConnector())

	// Load external plugins if directory exists.
	dir := pluginsDir
	if dir == "" {
		dir = filepath.Join(".", "plugins")
	}
	if _, err := os.Stat(dir); err == nil {
		plugins, err := plugin.LoadExternalPlugins(dir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: loading external plugins: %v\n", err)
		}
		for _, p := range plugins {
			if err := r.Register(p); err != nil {
				fmt.Fprintf(os.Stderr, "warning: registering plugin %s: %v\n", p.Name(), err)
			}
		}
	}

	return r
}
