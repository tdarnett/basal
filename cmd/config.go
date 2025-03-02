package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Show or update configuration",
	Long:  `Configure database location and LLM settings.`,
}

var configDBCmd = &cobra.Command{
	Use:   "db [path]",
	Short: "Configure database location",
	Long: `Show current database location or update it to a new path.
If a path is provided as an argument, it will be used as the new location.
Otherwise, an interactive prompt will be shown.`,
	RunE: runConfigDB,
}

var configLLMCmd = &cobra.Command{
	Use:   "llm",
	Short: "Configure LLM settings",
	Long:  `Configure the LLM endpoint and model settings for natural language queries.`,
	RunE:  runConfigLLM,
}

func init() {
	rootCmd.AddCommand(configCmd)
	configCmd.AddCommand(configDBCmd)
	configCmd.AddCommand(configLLMCmd)
}

func cleanPath(path string) (string, error) {
	// Remove any quotes
	path = strings.Trim(path, `"'`)

	// Remove any backslashes used for escaping
	path = strings.ReplaceAll(path, `\`, "")

	// Expand home directory if path starts with ~
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("error expanding home directory: %v", err)
		}
		path = filepath.Join(home, path[2:])
	}

	return path, nil
}

func validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("directory does not exist: %s", dir)
	}
	return nil
}

func runConfigDB(cmd *cobra.Command, args []string) error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return fmt.Errorf("error getting config directory: %v", err)
	}

	configFile := filepath.Join(configDir, "basal", "config.txt")
	currentPath := ""

	// Try to read existing path
	if content, err := os.ReadFile(configFile); err == nil {
		currentPath = string(content)
		fmt.Printf("Current database location: %s\n", currentPath)
	} else {
		fmt.Println("No database location configured yet.")
	}

	var newPath string

	if len(args) > 0 {
		// Use path from command line argument
		newPath = strings.Join(args, " ") // Join all args to handle paths with spaces
		newPath, err = cleanPath(newPath)
		if err != nil {
			return err
		}
		if err := validatePath(newPath); err != nil {
			return err
		}
	} else {
		// Interactive mode
		updatePrompt := promptui.Prompt{
			Label:     "Would you like to update the database location",
			IsConfirm: true,
		}

		if _, err := updatePrompt.Run(); err != nil {
			return nil // User chose not to update
		}

		pathPrompt := promptui.Prompt{
			Label: "Enter new database path",
			Validate: func(input string) error {
				cleanInput, err := cleanPath(input)
				if err != nil {
					return err
				}
				return validatePath(cleanInput)
			},
		}

		if currentPath != "" {
			pathPrompt.Default = currentPath
		}

		newPath, err = pathPrompt.Run()
		if err != nil {
			return fmt.Errorf("path prompt failed: %v", err)
		}

		newPath, err = cleanPath(newPath)
		if err != nil {
			return err
		}
	}

	// Ensure directory exists
	dbDir := filepath.Dir(newPath)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return fmt.Errorf("error creating database directory: %v", err)
	}

	// Save path
	if err := os.MkdirAll(filepath.Dir(configFile), 0755); err != nil {
		return fmt.Errorf("error creating config directory: %v", err)
	}

	if err := os.WriteFile(configFile, []byte(newPath), 0644); err != nil {
		return fmt.Errorf("error saving database path: %v", err)
	}

	// If there was an existing database, offer to copy it
	if currentPath != "" && currentPath != newPath {
		copyPrompt := promptui.Prompt{
			Label:     "Would you like to copy existing database to the new location",
			IsConfirm: true,
		}

		if _, err := copyPrompt.Run(); err == nil {
			// User wants to copy
			if err := copyFile(currentPath, newPath); err != nil {
				return fmt.Errorf("error copying database: %v", err)
			}
			fmt.Println("Database copied successfully!")
		}
	}

	fmt.Printf("Database location updated to: %s\n", newPath)
	return nil
}

func runConfigLLM(cmd *cobra.Command, args []string) error {
	// Get endpoint
	endpointPrompt := promptui.Prompt{
		Label:   "LLM API Endpoint",
		Default: "http://localhost:11434",
	}

	endpoint, err := endpointPrompt.Run()
	if err != nil {
		return fmt.Errorf("endpoint prompt failed: %v", err)
	}

	// Get model
	modelPrompt := promptui.Prompt{
		Label:   "LLM Model",
		Default: "mistral",
	}

	model, err := modelPrompt.Run()
	if err != nil {
		return fmt.Errorf("model prompt failed: %v", err)
	}

	// Save configuration
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "basal")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("error creating config directory: %v", err)
	}

	configFile := filepath.Join(configDir, "llm_config")
	f, err := os.Create(configFile)
	if err != nil {
		return fmt.Errorf("error creating config file: %v", err)
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "endpoint=%s\nmodel=%s\n", endpoint, model)
	if err != nil {
		return fmt.Errorf("error writing config: %v", err)
	}

	fmt.Println("LLM configuration updated successfully!")
	return nil
}

func copyFile(src, dst string) error {
	// Read the source file
	content, err := os.ReadFile(src)
	if err != nil {
		if os.IsNotExist(err) {
			// Source file doesn't exist yet, which is fine
			return nil
		}
		return err
	}

	// Write to destination
	return os.WriteFile(dst, content, 0644)
}
