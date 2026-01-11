package cmd

import (
	"fmt"
	"os"

	"github.com/AurelienConte/bullmq-tui/internal/config"
	"github.com/AurelienConte/bullmq-tui/internal/ui"
	"github.com/spf13/cobra"
)

var (
	cfgFile    string
	connection string
	cfg        *config.Config
)

var rootCmd = &cobra.Command{
	Use:   "bullmq-tui",
	Short: "A TUI for monitoring and managing BullMQ queues",
	Long: `BullMQ TUI is a terminal-based user interface for monitoring
and managing BullMQ job queues backed by Redis.

Launch the TUI:
  bullmq-tui                    # Use default connection
  bullmq-tui connect production # Use named connection
  bullmq-tui -c myconn          # Short form`,
	RunE: runTUI,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "",
		"config file (default is $HOME/.config/bullmq-tui/config.yaml)")
	rootCmd.Flags().StringVarP(&connection, "connection", "c", "",
		"connection name to use")

	// Add subcommands
	rootCmd.AddCommand(connectCmd)
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(versionCmd)
}

func initConfig() {
	var err error

	// Load config
	cfg, err = config.Load(cfgFile)
	if err != nil {
		// Only fail if not running config init
		if rootCmd.Use != "init" && os.Args[len(os.Args)-1] != "init" {
			fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
			fmt.Fprintf(os.Stderr, "Run 'bullmq-tui config init' to create a default configuration.\n")
			os.Exit(1)
		}
	}
}

func runTUI(cmd *cobra.Command, args []string) error {
	// Validate config
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Get connection
	conn, err := cfg.GetConnection(connection)
	if err != nil {
		return err
	}

	// Launch Bubbletea app
	return ui.Run(conn, cfg)
}
