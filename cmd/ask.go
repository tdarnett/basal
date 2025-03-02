package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"basal/db"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [natural language question]",
	Short: "Ask questions about your basal rates",
	Long: `Ask questions about your basal rates in natural language.
Uses a local LLM via Ollama to convert natural language to SQL queries.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

type BasalOllamaRequest struct {
	Model    string               `json:"model"`
	Messages []BasalOllamaMessage `json:"messages"`
}

type BasalOllamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type BasalOllamaResponse struct {
	Message BasalOllamaMessage `json:"message"`
}

func init() {
	rootCmd.AddCommand(askCmd)
}

func runAsk(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Get database schema
	schema := `
	CREATE TABLE basal_records (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		date DATE NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE basal_intervals (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		basal_record_id INTEGER,
		start_time TEXT NOT NULL,
		end_time TEXT NOT NULL,
		units_per_hour REAL NOT NULL,
		FOREIGN KEY (basal_record_id) REFERENCES basal_records(id)
	);`

	prompt := fmt.Sprintf(`You are an SQL expert. Convert the following natural language question into a SQL query that will work with SQLite.
Here's the database schema:
%s

The query should return meaningful information about basal rates based on the user's question.
Include appropriate JOINs between tables and format dates properly.
Only return the SQL query without any explanation.

Natural language question: %s`, schema, query)

	// Call Ollama API
	ollamaReq := BasalOllamaRequest{
		Model: "mistral",
		Messages: []BasalOllamaMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
	}

	jsonData, err := json.Marshal(ollamaReq)
	if err != nil {
		return fmt.Errorf("error marshaling request: %v", err)
	}

	resp, err := http.Post("http://localhost:11434/api/chat", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error calling Ollama API: %v", err)
	}
	defer resp.Body.Close()

	var ollamaResp BasalOllamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return fmt.Errorf("error decoding response: %v", err)
	}

	sqlQuery := strings.TrimSpace(ollamaResp.Message.Content)

	// Execute the SQL query
	database, err := db.InitDB(getDBPath())
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	rows, err := database.Query(sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %v", err)
	}

	// Create table
	table := tablewriter.NewWriter(cmd.OutOrStdout())
	table.SetHeader(columns)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")

	// Prepare values holder
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Add rows
	for rows.Next() {
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return fmt.Errorf("error scanning row: %v", err)
		}

		// Convert values to strings
		stringValues := make([]string, len(columns))
		for i, val := range values {
			if val == nil {
				stringValues[i] = "NULL"
			} else {
				switch v := val.(type) {
				case []byte:
					stringValues[i] = string(v)
				default:
					stringValues[i] = fmt.Sprintf("%v", v)
				}
			}
		}

		table.Append(stringValues)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	// Print table
	table.Render()
	return nil
}
