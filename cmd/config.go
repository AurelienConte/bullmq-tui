package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/AurelienConte/bullmq-tui/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage configuration and connections",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Create a default configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.GetConfigPath()

		// Check if config already exists
		if _, err := os.Stat(path); err == nil {
			fmt.Printf("Config file already exists at: %s\n", path)
			fmt.Print("Overwrite? (y/N): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Println("Aborted.")
				return nil
			}
		}

		_, err := config.CreateDefault(path)
		if err != nil {
			return fmt.Errorf("failed to create config: %w", err)
		}

		fmt.Printf("Created default config at: %s\n", path)
		return nil
	},
}

var configAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Add a new Redis connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connID := args[0]

		// Load existing config
		path := config.GetConfigPath()
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		// Check if connection already exists
		if _, ok := cfg.Connections[connID]; ok {
			return fmt.Errorf("connection '%s' already exists", connID)
		}

		// Get values from flags or interactive prompts
		host, _ := cmd.Flags().GetString("host")
		port, _ := cmd.Flags().GetInt("port")
		password, _ := cmd.Flags().GetString("password")
		db, _ := cmd.Flags().GetInt("db")
		tls, _ := cmd.Flags().GetBool("tls")
		prefix, _ := cmd.Flags().GetString("prefix")
		name, _ := cmd.Flags().GetString("name")

		// Interactive mode if name not provided
		reader := bufio.NewReader(os.Stdin)
		if name == "" {
			fmt.Printf("Connection name [%s]: ", connID)
			input, _ := reader.ReadString('\n')
			name = strings.TrimSpace(input)
			if name == "" {
				name = connID
			}
		}

		if !cmd.Flags().Changed("host") {
			fmt.Printf("Redis host [localhost]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				host = input
			}
		}

		if !cmd.Flags().Changed("port") {
			fmt.Printf("Redis port [6379]: ")
			input, _ := reader.ReadString('\n')
			input = strings.TrimSpace(input)
			if input != "" {
				p, err := strconv.Atoi(input)
				if err == nil {
					port = p
				}
			}
		}

		if !cmd.Flags().Changed("password") {
			fmt.Print("Redis password (leave empty for none): ")
			input, _ := reader.ReadString('\n')
			password = strings.TrimSpace(input)
		}

		// Create connection
		conn := &config.Connection{
			Name:     name,
			Host:     host,
			Port:     port,
			Password: password,
			DB:       db,
			TLS:      tls,
			Prefix:   prefix,
		}

		cfg.AddConnection(connID, conn)

		// Save config
		if err := config.Save(cfg, path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Added connection '%s' to config.\n", connID)
		return nil
	},
}

var configRemoveCmd = &cobra.Command{
	Use:   "remove <name>",
	Short: "Remove a saved connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connID := args[0]

		path := config.GetConfigPath()
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		if err := cfg.RemoveConnection(connID); err != nil {
			return err
		}

		if err := config.Save(cfg, path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Removed connection '%s'.\n", connID)
		if cfg.DefaultConnection == "" {
			fmt.Println("Warning: No connections remaining. Add a new connection or run 'config init'.")
		}

		return nil
	},
}

var configListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all saved connections",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.GetConfigPath()
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		if len(cfg.Connections) == 0 {
			fmt.Println("No connections configured.")
			return nil
		}

		fmt.Println("Available connections:")
		fmt.Println()

		for id, conn := range cfg.Connections {
			marker := "  "
			if id == cfg.DefaultConnection {
				marker = "* "
			}

			tlsStr := ""
			if conn.TLS {
				tlsStr = " (TLS)"
			}

			fmt.Printf("%s%-15s  %s  %s:%d%s\n",
				marker, id, conn.Name, conn.Host, conn.Port, tlsStr)
		}

		fmt.Println()
		fmt.Printf("* = default connection\n")

		return nil
	},
}

var configSetDefaultCmd = &cobra.Command{
	Use:   "set-default <name>",
	Short: "Set the default connection",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		connID := args[0]

		path := config.GetConfigPath()
		cfg, err := config.Load(path)
		if err != nil {
			return err
		}

		if err := cfg.SetDefault(connID); err != nil {
			return err
		}

		if err := config.Save(cfg, path); err != nil {
			return fmt.Errorf("failed to save config: %w", err)
		}

		fmt.Printf("Set '%s' as default connection.\n", connID)
		return nil
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open config file in your editor",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := config.GetConfigPath()

		// Check if config exists
		if _, err := os.Stat(path); os.IsNotExist(err) {
			return fmt.Errorf("config file does not exist at %s (run 'bullmq-tui config init' first)", path)
		}

		// Get editor from environment or use default
		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vim"
		}

		// Open editor
		editorCmd := exec.Command(editor, path)
		editorCmd.Stdin = os.Stdin
		editorCmd.Stdout = os.Stdout
		editorCmd.Stderr = os.Stderr

		return editorCmd.Run()
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the config file path",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(config.GetConfigPath())
	},
}

func init() {
	// Add subcommands to config command
	configCmd.AddCommand(configInitCmd)
	configCmd.AddCommand(configAddCmd)
	configCmd.AddCommand(configRemoveCmd)
	configCmd.AddCommand(configListCmd)
	configCmd.AddCommand(configSetDefaultCmd)
	configCmd.AddCommand(configEditCmd)
	configCmd.AddCommand(configPathCmd)

	// Flags for config add
	configAddCmd.Flags().String("name", "", "Display name for connection")
	configAddCmd.Flags().String("host", "localhost", "Redis host")
	configAddCmd.Flags().Int("port", 6379, "Redis port")
	configAddCmd.Flags().String("password", "", "Redis password")
	configAddCmd.Flags().Int("db", 0, "Redis database number")
	configAddCmd.Flags().Bool("tls", false, "Enable TLS")
	configAddCmd.Flags().String("prefix", "bull", "BullMQ key prefix")
}
