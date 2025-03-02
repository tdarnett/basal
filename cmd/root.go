package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "basal",
	Short: "A CLI application that helps track and manage insulin basal rates over time.",
	Long: `A CLI application that helps track and manage insulin basal rates over time.
It stores data in SQLite and provides various commands for updating and querying your basal rates.`,
}

func init() {
	// Disable the built-in help command since we have our own
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:    "no-help",
		Hidden: true,
	})
}

func Execute() error {
	return rootCmd.Execute()
}

func getDBPath() string {
	// Check if path is already configured
	configDir, err := os.UserConfigDir()
	if err != nil {
		fmt.Println("Error getting config directory:", err)
		os.Exit(1)
	}

	configFile := filepath.Join(configDir, "basal", "config.txt")

	// Try to read existing path
	if content, err := os.ReadFile(configFile); err == nil {
		return string(content)
	}

	// Get path from user
	fmt.Print("Enter the full path where you want to store the database: ")
	var dbPath string
	fmt.Scanln(&dbPath)

	// Ensure directory exists
	dbDir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		fmt.Println("Error creating database directory:", err)
		os.Exit(1)
	}

	// Save path for future use
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		fmt.Println("Error creating config directory:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(configFile, []byte(dbPath), 0644); err != nil {
		fmt.Println("Error saving database path:", err)
		os.Exit(1)
	}

	return dbPath
}
