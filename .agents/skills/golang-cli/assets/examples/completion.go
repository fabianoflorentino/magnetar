package main

import (
	"os"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(&cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate a shell completion script",
		Args:      cobra.ExactValidArgs(1),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return rootCmd.GenBashCompletionV2(os.Stdout, true)
			case "zsh":
				return rootCmd.GenZshCompletion(os.Stdout)
			case "fish":
				return rootCmd.GenFishCompletion(os.Stdout, true)
			case "powershell":
				return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	})
}

// Custom completion wiring: static flag values and dynamic argument lookup.
func customCompletionExamples() {
	deployCmd.RegisterFlagCompletionFunc("env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{
			"dev\tDevelopment environment",
			"staging\tStaging environment",
			"prod\tProduction environment",
		}, cobra.ShellCompDirectiveNoFileComp
	})

	deployCmd.ValidArgsFunction = func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		if len(args) != 0 {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		return discoverServices(), cobra.ShellCompDirectiveNoFileComp
	}
}

// discoverServices would consult a registry, file, or API at completion time.
func discoverServices() []string {
	return nil
}