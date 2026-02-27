package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"piper/internal/engine"
	"piper/internal/loader"
	"piper/internal/server"
)

var servePort int

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start webhook server",
	Args:  cobra.NoArgs,
	RunE:  serveWebhook,
}

func init() {
	serveCmd.Flags().IntVar(&servePort, "port", 8080, "port to listen on")
	rootCmd.AddCommand(serveCmd)
}

func serveWebhook(cmd *cobra.Command, args []string) error {
	flows, err := loader.LoadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("loading flows: %w", err)
	}

	registry := defaultRegistry()
	eng := engine.NewEngine(registry)

	srv := server.NewWebhookServer(eng, flows)
	addr := fmt.Sprintf(":%d", servePort)
	fmt.Printf("Starting webhook server on %s\n", addr)
	fmt.Printf("Loaded %d flow(s)\n", len(flows))
	for _, f := range flows {
		if f.Trigger != nil && f.Trigger.Type == "webhook" {
			fmt.Printf("  POST %s -> %s\n", f.Trigger.Path, f.Name)
		}
	}
	return srv.ListenAndServe(addr)
}
