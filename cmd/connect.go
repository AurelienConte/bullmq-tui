package cmd

import (
	"github.com/spf13/cobra"
)

var connectCmd = &cobra.Command{
	Use:   "connect <connection-name>",
	Short: "Connect to a saved Redis connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connection = args[0]
		return runTUI(cmd, args)
	},
}
