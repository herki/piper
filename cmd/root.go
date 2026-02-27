package cmd

import (
	"github.com/spf13/cobra"
)

var (
	flowsDir     string
	outputFormat string
)

var rootCmd = &cobra.Command{
	Use:   "flow",
	Short: "Flow — CLI workflow engine for AI agents",
	Long:  "A CLI-first workflow orchestration tool where AI agents define multi-step automation flows in YAML.",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&flowsDir, "flows-dir", "./flows", "directory containing flow YAML files")
	rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table or json")
	rootCmd.PersistentFlags().StringVar(&pluginsDir, "plugins-dir", "./plugins", "directory containing external plugin executables")
}

func Execute() error {
	return rootCmd.Execute()
}
