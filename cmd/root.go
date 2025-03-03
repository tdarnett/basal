// Package cmd implements the command-line interface for the basal tracker.
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
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

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func getConfigDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("getting user config directory: %w", err)
	}
	return filepath.Join(configDir, "basal"), nil
}

func getDBPath() (string, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return "", err
	}

	configFile := filepath.Join(configDir, "config.txt")

	// Try to read existing path
	if content, err := os.ReadFile(configFile); err == nil {
		return string(content), nil
	}

	// Get path from user
	fmt.Print("Enter the full path where you want to store the database: ")
	var dbPath string
	fmt.Scanln(&dbPath)

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		return "", fmt.Errorf("creating database directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return "", fmt.Errorf("creating config directory: %w", err)
	}

	if err := os.WriteFile(configFile, []byte(dbPath), 0644); err != nil {
		return "", fmt.Errorf("saving database path: %w", err)
	}

	return dbPath, nil
}
