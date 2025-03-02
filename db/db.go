package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type BasalRecord struct {
	ID        int64
	Date      time.Time
	CreatedAt time.Time
}

type BasalInterval struct {
	ID            int64
	BasalRecordID int64
	StartTime     string // Format: HH:MM
	EndTime       string // Format: HH:MM
	UnitsPerHour  float64
}

func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("error opening database: %v", err)
	}

	// Create tables if they don't exist
	createTables := `
	CREATE TABLE IF NOT EXISTS basal_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS basal_intervals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		basal_record_id INTEGER,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		units_per_hour REAL NOT NULL,
		FOREIGN KEY (basal_record_id) REFERENCES basal_records(id),
		CHECK (start_time >= '00:00' AND start_time <= '23:59'),
		CHECK (end_time >= '00:00' AND end_time <= '23:59')
	);`

	_, err = db.Exec(createTables)
	if err != nil {
		return nil, fmt.Errorf("error creating tables: %v", err)
	}

	return db, nil
}

func CreateBasalRecord(db *sql.DB, date time.Time, intervals []BasalInterval) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Insert basal record
	result, err := tx.Exec("INSERT INTO basal_records (date) VALUES (?)", date.Format("2006-01-02"))
	if err != nil {
		return err
	}

	recordID, err := result.LastInsertId()
	if err != nil {
		return err
	}

	// Insert intervals
	stmt, err := tx.Prepare("INSERT INTO basal_intervals (basal_record_id, start_time, end_time, units_per_hour) VALUES (?, ?, ?, ?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, interval := range intervals {
		_, err = stmt.Exec(recordID, interval.StartTime, interval.EndTime, interval.UnitsPerHour)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func GetBasalRecordByDate(db *sql.DB, date time.Time) (*BasalRecord, []BasalInterval, error) {
	// First try to get exact match
	query := `
	SELECT br.id, strftime('%Y-%m-%d', br.date) as date, br.created_at,
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
		// If no exact match, get the closest previous record
		query = `
		SELECT br.id, strftime('%Y-%m-%d', br.date) as date, br.created_at,
			   bi.id, bi.start_time, bi.end_time, bi.units_per_hour
		FROM basal_records br
		LEFT JOIN basal_intervals bi ON br.id = bi.basal_record_id
		WHERE date(br.date) <= date(?)
		ORDER BY br.date DESC, bi.start_time
		LIMIT 1`

		rows, err = db.Query(query, date.Format("2006-01-02"))
		if err != nil {
			return nil, nil, err
		}
		defer rows.Close()

		if !rows.Next() {
			// If still no record, get the earliest record
			query = `
			SELECT br.id, strftime('%Y-%m-%d', br.date) as date, br.created_at,
				   bi.id, bi.start_time, bi.end_time, bi.units_per_hour
			FROM basal_records br
			LEFT JOIN basal_intervals bi ON br.id = bi.basal_record_id
			ORDER BY br.date ASC, bi.start_time
			LIMIT 1`

			rows, err = db.Query(query)
			if err != nil {
				return nil, nil, err
			}
			defer rows.Close()

			if !rows.Next() {
				return nil, nil, fmt.Errorf("no basal records found")
			}
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
