package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Positional-argument validation through Cobra's built-in validators —
// NoArgs, ExactArgs(n), MinimumNArgs(n), MaximumNArgs(n), RangeArgs(min, max),
// ExactValidArgs(n).
var deployCmd = &cobra.Command{
	Use:   "deploy [environment]",
	Short: "Deploy to an environment",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return deploy(args[0])
	},
}

// A custom validator falls back to plain Args funcs for rules Cobra
// does not express.
var deployCheckedCmd = &cobra.Command{
	Use:   "deploy-checked [environment]",
	Short: "Deploy to an environment with a curated allowlist",
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			return fmt.Errorf("expected exactly 1 argument, got %d", len(args))
		}
		allowed := map[string]bool{"dev": true, "staging": true, "prod": true}
		if !allowed[args[0]] {
			return fmt.Errorf("invalid environment %q: pick dev, staging, or prod", args[0])
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return deploy(args[0])
	},
}

func deploy(environment string) error {
	_ = environment
	return nil
}