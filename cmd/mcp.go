package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"piper/internal/engine"
	"piper/internal/loader"
	"piper/internal/server"
	"piper/internal/types"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Start MCP (Model Context Protocol) server on stdin/stdout",
	Long:  "Exposes all flows as MCP tools. AI agents can discover and call flows via the MCP protocol.",
	Args:  cobra.NoArgs,
	RunE:  serveMCP,
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

func serveMCP(cmd *cobra.Command, args []string) error {
	flows, err := loader.LoadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("loading flows: %w", err)
	}

	registry := defaultRegistry()
	eng := engine.NewEngine(registry)
	eng.FlowLoader = func(name string) (*types.FlowDef, error) {
		f, ok := flows[name]
		if !ok {
			return nil, fmt.Errorf("flow %q not found", name)
		}
		return f, nil
	}

	srv := server.NewMCPServer(eng, flows)
	return srv.ServeStdio()
}
