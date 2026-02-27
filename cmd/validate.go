package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"piper/internal/engine"
	"piper/internal/loader"
)

var validateCmd = &cobra.Command{
	Use:   "validate <flow-file>",
	Short: "Validate a YAML flow file",
	Args:  cobra.ExactArgs(1),
	RunE:  validateFlow,
}

func init() {
	rootCmd.AddCommand(validateCmd)
}

func validateFlow(cmd *cobra.Command, args []string) error {
	path := args[0]

	flow, err := loader.LoadFlow(path)
	if err != nil {
		return err
	}

	registry := defaultRegistry()

	if err := engine.ValidateFlow(flow, registry); err != nil {
		return err
	}

	fmt.Printf("Flow %q is valid.\n", flow.Name)
	return nil
}
