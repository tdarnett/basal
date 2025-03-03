package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"basal/db"

	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var basalAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new basal rate record",
	Long:  `Add a new basal rate record with time intervals and rates for a specific date.`,
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(basalAddCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	dbPath, err := getDBPath()
	if err != nil {
		return fmt.Errorf("error getting database path: %v", err)
	}
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	// Get date
	datePrompt := promptui.Prompt{
		Label:   "Date (YYYY-MM-DD, press enter for today)",
		Default: time.Now().Format("2006-01-02"),
		Validate: func(input string) error {
			if input == "" {
				return nil
			}
			_, err := time.Parse("2006-01-02", input)
			if err != nil {
				return fmt.Errorf("invalid date format")
			}
			return nil
		},
	}

	dateStr, err := datePrompt.Run()
	if err != nil {
		return fmt.Errorf("date prompt failed: %v", err)
	}

	var date time.Time
	if dateStr == "" {
		date = time.Now()
	} else {
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return fmt.Errorf("invalid date: %v", err)
		}
	}

	fmt.Println("\nEnter basal rate intervals for the entire day (00:00 to 00:00)")
	fmt.Println("Each interval must start where the previous one ended.")
	fmt.Println("The last interval must end at 00:00 to complete the day.")

	var intervals []db.BasalInterval
	for {
		// Get start time
		startPrompt := promptui.Prompt{
			Label: "Start time (HH:MM, H:MM, or HHMM format)",
			Validate: func(input string) error {
				normalized, err := parseTimeFormat(input)
				if err != nil {
					return err
				}
				if len(intervals) == 0 && normalized != "00:00" {
					return fmt.Errorf("first interval must start at 00:00")
				}
				if len(intervals) > 0 {
					lastInterval := intervals[len(intervals)-1]
					if normalized != lastInterval.EndTime {
						return fmt.Errorf("start time must match previous end time (%s)", lastInterval.EndTime)
					}
				}
				return nil
			},
		}

		startTime, err := startPrompt.Run()
		if err != nil {
			return fmt.Errorf("start time prompt failed: %v", err)
		}

		startTime, err = parseTimeFormat(startTime)
		if err != nil {
			return fmt.Errorf("invalid start time: %v", err)
		}

		// Get end time
		endPrompt := promptui.Prompt{
			Label: "End time (HH:MM, H:MM, or HHMM format)",
			Validate: func(input string) error {
				normalized, err := parseTimeFormat(input)
				if err != nil {
					return err
				}

				// For other times, ensure end time is after start time
				startMinutes := convertTimeToMinutes(startTime)
				endMinutes := convertTimeToMinutes(normalized)
				if endMinutes <= startMinutes {
					return fmt.Errorf("end time must be after start time (%s)", startTime)
				}
				return nil
			},
		}

		endTime, err := endPrompt.Run()
		if err != nil {
			return fmt.Errorf("end time prompt failed: %v", err)
		}

		endTime, err = parseTimeFormat(endTime)
		if err != nil {
			return fmt.Errorf("invalid end time: %v", err)
		}

		// Get units per hour
		unitsPrompt := promptui.Prompt{
			Label: "Units per hour",
			Validate: func(input string) error {
				var units float64
				_, err := fmt.Sscanf(input, "%f", &units)
				if err != nil || units < 0 {
					return fmt.Errorf("invalid units value")
				}
				return nil
			},
		}

		unitsStr, err := unitsPrompt.Run()
		if err != nil {
			return fmt.Errorf("units prompt failed: %v", err)
		}

		var units float64
		fmt.Sscanf(unitsStr, "%f", &units)

		intervals = append(intervals, db.BasalInterval{
			StartTime:    startTime,
			EndTime:      endTime,
			UnitsPerHour: units,
		})

		// If we've completed the day (ended at 00:00), break
		if endTime == "00:00" {
			break
		}
	}

	// Calculate daily total
	dailyTotal := db.CalculateDailyBasal(intervals)

	// Show summary and get confirmation
	fmt.Printf("\nDaily Summary for %s:\n", date.Format("2006-01-02"))
	fmt.Println("Time Interval    Units/hr")
	fmt.Println("------------------------")
	for _, interval := range intervals {
		fmt.Printf("%s - %s    %.2f\n", interval.StartTime, interval.EndTime, interval.UnitsPerHour)
	}
	fmt.Printf("\nTotal daily basal: %.2f units\n", dailyTotal)

	confirmPrompt := promptui.Prompt{
		Label:     "Save this record",
		IsConfirm: true,
	}

	if _, err := confirmPrompt.Run(); err != nil {
		fmt.Println("Record not saved.")
		return nil
	}

	// Create the record
	err = db.CreateBasalRecord(database, date, intervals)
	if err != nil {
		return fmt.Errorf("error creating basal record: %v", err)
	}

	fmt.Println("Basal rates added successfully!")
	return nil
}

// parseTimeFormat converts various time formats to a standardized HH:MM format
func parseTimeFormat(input string) (string, error) {
	// Remove any spaces
	input = strings.TrimSpace(input)

	// Try different formats
	var hour, min int
	var err error

	// Try H:MM format
	if n, _ := fmt.Sscanf(input, "%d:%d", &hour, &min); n == 2 {
		if hour < 0 || hour > 23 || min < 0 || min > 59 {
			return "", fmt.Errorf("invalid time: hours must be 0-23, minutes must be 0-59")
		}
		return fmt.Sprintf("%02d:%02d", hour, min), nil
	}

	// Try HMM format (e.g., "230" for 2:30)
	if len(input) == 3 {
		if hour, err = strconv.Atoi(input[:1]); err == nil {
			if min, err = strconv.Atoi(input[1:]); err == nil {
				if hour >= 0 && hour <= 23 && min >= 0 && min <= 59 {
					return fmt.Sprintf("%02d:%02d", hour, min), nil
				}
			}
		}
	}

	// Try HHMM format (e.g., "0230" for 02:30)
	if len(input) == 4 {
		if hour, err = strconv.Atoi(input[:2]); err == nil {
			if min, err = strconv.Atoi(input[2:]); err == nil {
				if hour >= 0 && hour <= 23 && min >= 0 && min <= 59 {
					return fmt.Sprintf("%02d:%02d", hour, min), nil
				}
			}
		}
	}

	return "", fmt.Errorf("invalid time format: use HH:MM, H:MM, HMM, or HHMM")
}

// convertTimeToMinutes converts HH:MM format to minutes since midnight
func convertTimeToMinutes(timeStr string) int {
	var hour, min int
	fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
	return hour*60 + min
}

// hasTimeOverlap is no longer needed since we enforce continuous intervals
