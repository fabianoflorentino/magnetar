package main

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the HTTP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		port := viper.GetInt("port")
		fmt.Fprintf(cmd.OutOrStdout(), "listening on :%d\n", port)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)

	// Local flag: affects only this command, but still bound to Viper so
	// MYAPP_PORT and the config file participate in the precedence order.
	serveCmd.Flags().IntP("port", "p", 8080, "port to listen on")
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
}