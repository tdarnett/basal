package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"basal/db"

	"github.com/guptarohit/asciigraph"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show [date]",
	Short: "Show basal rates for a specific date",
	Long: `Display the basal rates for a specific date.
If no date is provided, shows today's rates.
If no exact record exists for the date, it will show the closest previous record.
If no previous record exists, it will show the earliest record available.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runShow,
}

func init() {
	rootCmd.AddCommand(showCmd)
}

// timeToMinutes converts HH:MM format to minutes since midnight
func timeToMinutes(timeStr string) int {
	parts := strings.Split(timeStr, ":")
	hours, _ := strconv.Atoi(parts[0])
	minutes, _ := strconv.Atoi(parts[1])
	return hours*60 + minutes
}

// minutesToTime converts minutes since midnight to HH:MM format
func minutesToTime(minutes int) string {
	hours := minutes / 60
	mins := minutes % 60
	return fmt.Sprintf("%02d:%02d", hours, mins)
}

// generateGraphData creates data points for the line chart
func generateGraphData(intervals []db.BasalInterval) []float64 {
	// Create a slice for all 24 hours (1440 minutes)
	data := make([]float64, 1440)

	// Fill in the basal rates for each interval
	for _, interval := range intervals {
		start := timeToMinutes(interval.StartTime)
		end := timeToMinutes(interval.EndTime)
		if end == 0 {
			end = 1440 // Handle midnight wrap-around
		}

		// Fill in the rate for each minute in the interval
		for i := start; i < end; i++ {
			data[i] = interval.UnitsPerHour
		}
	}

	return data
}

func runShow(cmd *cobra.Command, args []string) error {
	var date time.Time
	var err error

	if len(args) > 0 {
		date, err = time.Parse(db.DateFormat, args[0])
		if err != nil {
			return fmt.Errorf("invalid date format: %v", err)
		}
	} else {
		date = time.Now()
	}

	dbPath, err := getDBPath()
	if err != nil {
		return fmt.Errorf("error getting database path: %v", err)
	}
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	record, intervals, err := db.GetBasalRecordByDate(database, date)
	if err != nil {
		return fmt.Errorf("error retrieving basal record: %v", err)
	}

	// Print date and whether it's an exact match
	if record.Date.Format(db.DateFormat) != date.Format(db.DateFormat) {
		fmt.Printf("\nShowing closest record from: %s\n", record.Date.Format(db.DateFormat))
	} else {
		fmt.Printf("\n%s\n", date.Format(db.DateFormat))
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

	// Print table
	table.Render()

	// Print daily total from stored value
	fmt.Printf("\nDaily basal: %.2f units\n", record.TotalUnits)

	// Generate and display the graph
	graphData := generateGraphData(intervals)

	// Find min and max values for better y-axis scaling
	minVal, maxVal := 0.0, 0.0
	for _, v := range graphData {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	// Add some padding to the y-axis
	minVal = max(0, minVal-0.1)
	maxVal = maxVal + 0.1

	graph := asciigraph.Plot(graphData,
		asciigraph.Height(8),
		asciigraph.Caption("Basal Rate (U/hr)"),
		asciigraph.Precision(1),
		asciigraph.Offset(3),
		asciigraph.Width(60),
		asciigraph.SeriesColors(asciigraph.Blue),
	)
	fmt.Println("\n" + graph)

	return nil
}

// max returns the larger of x or y
func max(x, y float64) float64 {
	if x > y {
		return x
	}
	return y
}
