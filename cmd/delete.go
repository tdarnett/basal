package cmd

import (
	"fmt"
	"strconv"

	"basal/db"

	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [id]",
	Short: "Delete a basal rate record",
	Long:  `Delete a basal rate record by its ID.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	id, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid ID: %v", err)
	}

	database, err := db.InitDB(getDBPath())
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	if err := db.DeleteBasalRecord(database, id); err != nil {
		return fmt.Errorf("error deleting record: %v", err)
	}

	fmt.Printf("Record %d deleted successfully!\n", id)
	return nil
}
