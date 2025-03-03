package cmd

import (
	"fmt"

	"basal/db"

	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all basal rate records",
	Long:  `Display a list of all basal rate records in the database.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	dbPath, err := getDBPath()
	if err != nil {
		return fmt.Errorf("error getting database path: %v", err)
	}
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	records, err := db.ListBasalRecords(database)
	if err != nil {
		return fmt.Errorf("error listing records: %v", err)
	}

	if len(records) == 0 {
		fmt.Println("No records found.")
		return nil
	}

	fmt.Println("\nBasal Rate Records:")
	fmt.Println("ID        Date")
	fmt.Println("------------------")
	for _, record := range records {
		fmt.Printf("%-8d  %s\n", record.ID, record.Date.Format(db.DateFormat))
	}
	return nil
}
