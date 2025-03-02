package cmd

import (
	"fmt"
	"time"

	"basal/db"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [date]",
	Short: "Show basal rates for a specific date",
	Long: `Display the basal rates for a specific date.
If no exact record exists for the date, it will show the closest previous record.
If no previous record exists, it will show the earliest record available.`,
	Args: cobra.ExactArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

func runShow(cmd *cobra.Command, args []string) error {
	date, err := time.Parse("2006-01-02", args[0])
	if err != nil {
		return fmt.Errorf("invalid date format. Please use YYYY-MM-DD")
	}

	database, err := db.InitDB(getDBPath())
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	record, intervals, err := db.GetBasalRecordByDate(database, date)
	if err != nil {
		return fmt.Errorf("error retrieving basal record: %v", err)
	}

	// Create table
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader([]string{"Time Interval", "Units/hr"})
	table.SetBorder(false)
	table.SetColumnSeparator("  ")

	// Add rows
	for _, interval := range intervals {
		table.Append([]string{
			fmt.Sprintf("%s - %s", interval.StartTime, interval.EndTime),
			fmt.Sprintf("%.2f", interval.UnitsPerHour),
		})
	}

	// Print date and whether it's an exact match
	if record.Date.Format("2006-01-02") != date.Format("2006-01-02") {
		fmt.Printf("\nShowing closest record from: %s\n", record.Date.Format("2006-01-02"))
	} else {
		fmt.Printf("\n%s\n", date.Format("2006-01-02"))
	}

	// Print table
	table.Render()

	// Calculate and print daily total
	dailyTotal := db.CalculateDailyBasal(intervals)
	fmt.Printf("\nDaily basal: %.2f units\n", dailyTotal)

	return nil
}
