package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Persistent and local flags, plus the enforcement helpers. Bind every
// configurable flag to Viper so viper.Get* honors flag > env > file > default.
func flagExamples() {
	// Persistent: inherited by every subcommand.
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file path")

	// Local: scoped to one command.
	serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))

	// Required flag.
	serveCmd.Flags().String("host", "", "hostname to bind to")
	serveCmd.MarkFlagRequired("host")

	// Mutually exclusive and at-least-one groups.
	rootCmd.MarkFlagsMutuallyExclusive("json", "yaml")
	rootCmd.MarkFlagsOneRequired("output-file", "stdout")

	// Inline suggestions for --env.
	serveCmd.Flags().String("env", "dev", "environment (dev, staging, prod)")
	serveCmd.RegisterFlagCompletionFunc("env", func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
		return []string{"dev", "staging", "prod"}, cobra.ShellCompDirectiveNoFileComp
	})
}