package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"piper/internal/loader"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available flows",
	Args:  cobra.NoArgs,
	RunE:  listFlows,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func listFlows(cmd *cobra.Command, args []string) error {
	flows, err := loader.LoadFlows(flowsDir)
	if err != nil {
		return fmt.Errorf("loading flows: %w", err)
	}

	if outputFormat == "json" {
		type flowSummary struct {
			Name        string `json:"name"`
			Version     string `json:"version"`
			Description string `json:"description"`
			Trigger     string `json:"trigger,omitempty"`
			Steps       int    `json:"steps"`
		}
		summaries := make([]flowSummary, 0, len(flows))
		for _, f := range flows {
			s := flowSummary{
				Name:        f.Name,
				Version:     f.Version,
				Description: f.Description,
				Steps:       len(f.Steps),
			}
			if f.Trigger != nil {
				s.Trigger = f.Trigger.Type
			}
			summaries = append(summaries, s)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(summaries)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVERSION\tDESCRIPTION\tSTEPS\tTRIGGER")
	for _, f := range flows {
		trigger := "-"
		if f.Trigger != nil {
			trigger = f.Trigger.Type
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\n", f.Name, f.Version, f.Description, len(f.Steps), trigger)
	}
	return w.Flush()
}
