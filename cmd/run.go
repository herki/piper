package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"piper/internal/engine"
	"piper/internal/loader"
	"piper/internal/types"
)

var (
	inputJSON   string
	dryRun      bool
	secretsFile string
)

var runCmd = &cobra.Command{
	Use:   "run <flow-name>",
	Short: "Execute a flow with JSON input",
	Args:  cobra.ExactArgs(1),
	RunE:  runFlow,
}

func init() {
	runCmd.Flags().StringVar(&inputJSON, "input", "{}", "JSON input for the flow")
	runCmd.Flags().BoolVar(&dryRun, "dry-run", false, "show what would execute without running")
	runCmd.Flags().StringVar(&secretsFile, "secrets-file", "", "path to .env-style secrets file")
	rootCmd.AddCommand(runCmd)
}

func runFlow(cmd *cobra.Command, args []string) error {
	flowName := args[0]

	flows, err := loader.LoadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("loading flows: %w", err)
	}

	flow, ok := flows[flowName]
	if !ok {
		return fmt.Errorf("flow %q not found in %s", flowName, flowsDir)
	}

	var input map[string]any
	if err := json.Unmarshal([]byte(inputJSON), &input); err != nil {
		return fmt.Errorf("parsing input JSON: %w", err)
	}

	registry := defaultRegistry()
	eng := engine.NewEngine(registry)

	// Enable flow composition.
	eng.FlowLoader = func(name string) (*types.FlowDef, error) {
		f, ok := flows[name]
		if !ok {
			return nil, fmt.Errorf("flow %q not found", name)
		}
		return f, nil
	}

	if err := engine.ValidateFlow(flow, registry); err != nil {
		return err
	}

	var result any
	if dryRun {
		result, err = eng.DryRun(flow, input)
	} else {
		ctx := context.Background()

		// Load secrets if provided.
		var secrets map[string]string
		if secretsFile != "" {
			secrets, err = engine.LoadSecrets(secretsFile)
			if err != nil {
				return fmt.Errorf("loading secrets: %w", err)
			}
		}

		flowResult, runErr := eng.RunWithSecrets(ctx, flow, input, secrets)
		result = flowResult
		err = runErr
	}
	if err != nil {
		return err
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}
