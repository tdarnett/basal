package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var helpCmd = &cobra.Command{
	Use:   "help",
	Short: "Show help for basal commands",
	Long:  `Display detailed help information for all available basal commands.`,
	RunE:  runHelp,
}

func init() {
	rootCmd.AddCommand(helpCmd)
}

func runHelp(cmd *cobra.Command, args []string) error {
	fmt.Println(`
Basal - Insulin Basal Rate Tracker

Available Commands:
  add                  Add a new basal rate record
    Usage: basal add
    Interactively add a new basal rate record with time intervals.

  list                 List all basal rate records
    Usage: basal list
    Shows all basal rate records with their IDs and dates.

  delete [id]          Delete a basal rate record
    Usage: basal delete 123
    Deletes the basal rate record with the specified ID.

  show [YYYY-MM-DD]    Show basal rates for a specific date
    Usage: basal show 2024-03-15
    Shows the basal rate schedule for the given date.
    If no exact match exists, shows the closest previous record.

  ask [text]           Ask questions about your basal rates
    Usage: basal ask "what was my basal rate on Dec 2, 2023"
    Converts natural language to SQL and queries the database.
    Requires Ollama to be running locally.

  config               Configuration commands
    Usage: basal config [command]
    Available subcommands:
      db [path]        Configure database location
                      Can provide path directly or use interactive prompt
      llm              Configure LLM settings

  help                 Show this help message
    Usage: basal help`)
	return nil
}
