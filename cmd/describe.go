package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"piper/internal/loader"
)

var describeCmd = &cobra.Command{
	Use:   "describe <flow-name>",
	Short: "Show details of a flow",
	Args:  cobra.ExactArgs(1),
	RunE:  describeFlow,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func describeFlow(cmd *cobra.Command, args []string) error {
	flowName := args[0]

	flows, err := loader.LoadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("loading flows: %w", err)
	}

	flow, ok := flows[flowName]
	if !ok {
		return fmt.Errorf("flow %q not found in %s", flowName, flowsDir)
	}

	if outputFormat == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(flow)
	}

	fmt.Printf("Name:        %s\n", flow.Name)
	fmt.Printf("Version:     %s\n", flow.Version)
	fmt.Printf("Description: %s\n", flow.Description)
	if flow.Trigger != nil {
		fmt.Printf("Trigger:     %s (%s)\n", flow.Trigger.Type, flow.Trigger.Path)
	}

	if flow.Input != nil && len(flow.Input.Properties) > 0 {
		fmt.Println("\nInput Schema:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  FIELD\tTYPE\tREQUIRED\tDESCRIPTION")
		for name, field := range flow.Input.Properties {
			fmt.Fprintf(w, "  %s\t%s\t%v\t%s\n", name, field.Type, field.Required, field.Description)
		}
		w.Flush()
	}

	if flow.Output != nil && len(flow.Output.Properties) > 0 {
		fmt.Println("\nOutput Schema:")
		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "  FIELD\tTYPE\tDESCRIPTION")
		for name, field := range flow.Output.Properties {
			fmt.Fprintf(w, "  %s\t%s\t%s\n", name, field.Type, field.Description)
		}
		w.Flush()
	}

	fmt.Println("\nSteps:")
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  #\tNAME\tCONNECTOR\tACTION\tON_ERROR")
	for i, step := range flow.Steps {
		onError := step.OnError
		if onError == "" {
			onError = "abort"
		}
		fmt.Fprintf(w, "  %d\t%s\t%s\t%s\t%s\n", i+1, step.Name, step.Connector, step.Action, onError)
	}
	return w.Flush()
}
