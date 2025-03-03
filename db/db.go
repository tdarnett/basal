// Package db provides database operations for managing insulin basal rates.
// It uses SQLite as the underlying storage and provides CRUD operations
// for basal records and their associated time intervals.
package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// BasalRecord represents a daily basal insulin rate record.
type BasalRecord struct {
	ID         int64
	Date       time.Time
	TotalUnits float64
	CreatedAt  time.Time
}

// BasalInterval represents a time interval within a day with a specific basal rate.
type BasalInterval struct {
	ID            int64
	BasalRecordID int64
	StartTime     string // Format: HH:MM
	EndTime       string // Format: HH:MM
	UnitsPerHour  float64
}

var ErrNoRecords = fmt.Errorf("no basal records found")

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// Create tables if they don't exist
	if _, err = db.Exec(GetSchema()); err != nil {
		return nil, fmt.Errorf("creating tables: %w", err)
	}

	return db, nil
}

// CreateBasalRecord adds a new basal record with its intervals to the database.
// It uses a transaction to ensure all operations succeed or fail together.
func CreateBasalRecord(db *sql.DB, date time.Time, intervals []BasalInterval) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("beginning transaction: %w", err)
	}
	defer tx.Rollback()

	// Calculate total units for the day
	totalUnits := CalculateDailyBasal(intervals)

	result, err := tx.Exec(
		"INSERT INTO basal_records (date, total_units) VALUES (?, ?)",
		date.Format("2006-01-02"),
		totalUnits,
	)
	if err != nil {
		return fmt.Errorf("inserting basal record: %w", err)
	}

	recordID, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("getting last insert ID: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO basal_intervals (
			basal_record_id, start_time, end_time, units_per_hour
		) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("preparing interval statement: %w", err)
	}
	defer stmt.Close()

	for i, interval := range intervals {
		_, err = stmt.Exec(
			recordID,
			interval.StartTime,
			interval.EndTime,
			interval.UnitsPerHour,
		)
		if err != nil {
			return fmt.Errorf("inserting interval %d: %w", i+1, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("committing transaction: %w", err)
	}

	return nil
}

func GetBasalRecordByDate(db *sql.DB, date time.Time) (*BasalRecord, []BasalInterval, error) {
	// First try to get exact match
	query := `
	SELECT br.id, strftime('%Y-%m-%d', br.date) as date, br.total_units, br.created_at,
		   bi.id, bi.start_time, bi.end_time, bi.units_per_hour
	FROM basal_records br
	LEFT JOIN basal_intervals bi ON br.id = bi.basal_record_id
	WHERE date(br.date) = date(?)
	ORDER BY bi.start_time`

	rows, err := db.Query(query, date.Format("2006-01-02"))
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		// If no exact match, get the closest previous record's ID first
		idQuery := `
		SELECT id, strftime('%Y-%m-%d', date) as date
		FROM basal_records
		WHERE date(date) <= date(?)
		ORDER BY date DESC
		LIMIT 1`

		var recordID int64
		var dateStr string
		err = db.QueryRow(idQuery, date.Format("2006-01-02")).Scan(&recordID, &dateStr)
		if err != nil {
			if err == sql.ErrNoRows {
				// If no previous record, get the earliest record's ID
				err = db.QueryRow(`
					SELECT id, strftime('%Y-%m-%d', date) as date
					FROM basal_records
					ORDER BY date ASC
					LIMIT 1`).Scan(&recordID, &dateStr)
				if err != nil {
					if err == sql.ErrNoRows {
						return nil, nil, fmt.Errorf("no basal records found")
					}
					return nil, nil, err
				}
			} else {
				return nil, nil, err
			}
		}

		// Now get all intervals for this record
		query = `
		SELECT br.id, strftime('%Y-%m-%d', br.date) as date, br.total_units, br.created_at,
			   bi.id, bi.start_time, bi.end_time, bi.units_per_hour
		FROM basal_records br
		LEFT JOIN basal_intervals bi ON br.id = bi.basal_record_id
		WHERE br.id = ?
		ORDER BY bi.start_time`

		rows, err = db.Query(query, recordID)
		if err != nil {
			return nil, nil, err
		}
		defer rows.Close()

		if !rows.Next() {
			return nil, nil, fmt.Errorf("no intervals found for record %d", recordID)
		}
	}

	var record BasalRecord
	var intervals []BasalInterval

	for {
		var interval BasalInterval
		var dateStr string
		err := rows.Scan(
			&record.ID,
			&dateStr,
			&record.TotalUnits,
			&record.CreatedAt,
			&interval.ID,
			&interval.StartTime,
			&interval.EndTime,
			&interval.UnitsPerHour,
		)
		if err != nil {
			return nil, nil, err
		}

		record.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, nil, err
		}

		interval.BasalRecordID = record.ID
		intervals = append(intervals, interval)

		if !rows.Next() {
			break
		}
	}

	return &record, intervals, nil
}

func ListBasalRecords(db *sql.DB) ([]BasalRecord, error) {
	rows, err := db.Query(`
		SELECT id, strftime('%Y-%m-%d', date) as date, created_at
		FROM basal_records
		ORDER BY date DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []BasalRecord
	for rows.Next() {
		var record BasalRecord
		var dateStr string
		err := rows.Scan(&record.ID, &dateStr, &record.CreatedAt)
		if err != nil {
			return nil, err
		}

		record.Date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			return nil, err
		}

		records = append(records, record)
	}

	return records, nil
}

func DeleteBasalRecord(db *sql.DB, id int64) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Delete intervals first due to foreign key constraint
	_, err = tx.Exec("DELETE FROM basal_intervals WHERE basal_record_id = ?", id)
	if err != nil {
		return err
	}

	result, err := tx.Exec("DELETE FROM basal_records WHERE id = ?", id)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("record with ID %d not found", id)
	}

	return tx.Commit()
}

func CalculateDailyBasal(intervals []BasalInterval) float64 {
	var total float64
	for _, interval := range intervals {
		startParts := splitTime(interval.StartTime)
		endParts := splitTime(interval.EndTime)

		startMinutes := startParts[0]*60 + startParts[1]
		endMinutes := endParts[0]*60 + endParts[1]

		// Handle cases where the interval crosses midnight
		if endMinutes <= startMinutes {
			endMinutes += 24 * 60
		}

		hours := float64(endMinutes-startMinutes) / 60.0
		total += hours * interval.UnitsPerHour
	}
	return total
}

func splitTime(timeStr string) [2]int {
	var hour, min int
	fmt.Sscanf(timeStr, "%d:%d", &hour, &min)
	return [2]int{hour, min}
}

// GetSchema returns the SQLite schema for the basal database.
func GetSchema() string {
	return `
	CREATE TABLE IF NOT EXISTS basal_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for each basal record
		date DATE NOT NULL,                   -- Date for which this basal profile applies
		total_units REAL NOT NULL,            -- Total daily insulin units for this profile
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP -- When this record was created
	);

	CREATE TABLE IF NOT EXISTS basal_intervals (
		id INTEGER PRIMARY KEY AUTOINCREMENT, -- Unique identifier for each interval
		basal_record_id INTEGER,              -- Foreign key to the parent basal record
		start_time TEXT NOT NULL,             -- Start time of interval in HH:MM format
		end_time TEXT NOT NULL,               -- End time of interval in HH:MM format
		units_per_hour REAL NOT NULL,         -- Insulin units per hour during this interval
		FOREIGN KEY (basal_record_id) REFERENCES basal_records(id),
		CHECK (start_time >= '00:00' AND start_time <= '23:59'), -- Validate time format
		CHECK (end_time >= '00:00' AND end_time <= '23:59')      -- Validate time format
	);`
}
