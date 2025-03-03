package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"basal/cmd/ollama"
	"basal/db"

	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

var askCmd = &cobra.Command{
	Use:   "ask [natural language question]",
	Short: "Ask questions about your basal rates",
	Long: `Ask questions about your basal rates in natural language.
Uses a local LLM via Ollama to convert natural language to SQL queries and interpret results.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAsk,
}

// tableOutputWriter captures table output while also writing to stdout
type tableOutputWriter struct {
	output io.Writer
	buf    strings.Builder
}

func (w *tableOutputWriter) Write(p []byte) (n int, err error) {
	w.buf.Write(p)
	return w.output.Write(p)
}

func init() {
	rootCmd.AddCommand(askCmd)
}

// validateSQLQuery performs basic validation of the SQL query
func validateSQLQuery(query string) error {
	query = strings.TrimSpace(strings.ToLower(query))
	if !strings.HasPrefix(query, "select") {
		return fmt.Errorf("invalid query: only SELECT queries are allowed")
	}
	if strings.Contains(query, "drop") || strings.Contains(query, "delete") || strings.Contains(query, "update") || strings.Contains(query, "insert") {
		return fmt.Errorf("invalid query: only SELECT queries are allowed")
	}
	return nil
}

// callOllama makes a request to the Ollama API and returns the response
func callOllama(llmConfig *LLMConfig, prompt string) (string, error) {
	req := ollama.Request{
		Model: llmConfig.Model,
		Messages: []ollama.Message{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: false,
	}

	jsonData, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("error marshaling request: %v", err)
	}

	apiURL := fmt.Sprintf("%s/api/chat", llmConfig.Endpoint)
	resp, err := http.Post(apiURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("error calling Ollama API: %v", err)
	}
	defer resp.Body.Close()

	var ollamaResp ollama.Response
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return "", fmt.Errorf("error decoding response: %v", err)
	}

	return ollamaResp.Message.Content, nil
}

// getLLMInterpretation asks the LLM to interpret the query results
func getLLMInterpretation(llmConfig *LLMConfig, originalQuestion string, tableOutput string) (string, error) {
	prompt := fmt.Sprintf(`You are a helpful assistant that interprets SQL query results in natural language.
The user asked: %s

Here are the query results:
%s

Please provide a clear, concise answer to the user's question based on these results.
Keep your response brief and focused on answering the specific question asked.`, originalQuestion, tableOutput)

	return callOllama(llmConfig, prompt)
}

// renderTable creates and renders a table with the given columns and rows
func renderTable(writer io.Writer, columns []string, rows [][]string) {
	table := tablewriter.NewWriter(writer)
	table.SetHeader(columns)
	table.SetBorder(false)
	table.SetColumnSeparator("  ")

	for _, row := range rows {
		table.Append(row)
	}

	table.Render()
}

func runAsk(cmd *cobra.Command, args []string) error {
	query := strings.Join(args, " ")

	// Get LLM configuration
	llmConfig, err := getLLMConfig()
	if err != nil {
		return fmt.Errorf("error getting LLM configuration: %v", err)
	}

	// Get database schema
	schema := db.GetSchema()

	prompt := fmt.Sprintf(`You are an SQL expert. Convert the following natural language question into a SQL query that will work with SQLite.
Here's the database schema:
%s

The query should return meaningful information about basal rates based on the user's question.
Keep queries as simple as possible - don't add unnecessary complexity.

IMPORTANT:
1. Return ONLY the SQL query, no explanations or markdown formatting
2. Start with SELECT
3. Use SQLite-specific syntax:
   - Use strftime('%Y-%m-%d', date) for date formatting
   - Use time('now') instead of now() for current time
   - Use date('now') instead of now() for current date
   - Use proper time format (HH:MM) for time comparisons
   - Use proper date format (YYYY-MM-DD) for date comparisons

Examples:
- For "which day has the greatest total_units": 
  SELECT date, total_units FROM basal_records ORDER BY total_units DESC LIMIT 1
- For "what was my basal rate on Dec 2, 2023":
  SELECT * FROM basal_records WHERE date = '2023-12-02'

Natural language question: %s`, schema, query)

	sqlQuery, err := callOllama(llmConfig, prompt)
	if err != nil {
		return err
	}

	sqlQuery = strings.TrimSpace(sqlQuery)
	fmt.Printf("\nGenerated SQL query:\n%s\n\n", sqlQuery)

	// Validate SQL query before database operations
	if err := validateSQLQuery(sqlQuery); err != nil {
		return err
	}

	// Execute the SQL query
	dbPath, err := getDBPath()
	if err != nil {
		return fmt.Errorf("error getting database path: %v", err)
	}
	database, err := db.InitDB(dbPath)
	if err != nil {
		return fmt.Errorf("error initializing database: %v", err)
	}
	defer database.Close()

	rows, err := database.Query(sqlQuery)
	if err != nil {
		return fmt.Errorf("error executing query: %v\nQuery: %s", err, sqlQuery)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return fmt.Errorf("error getting columns: %v", err)
	}

	// Create a custom writer to capture table output
	outputWriter := &tableOutputWriter{
		output: cmd.OutOrStdout(),
	}

	// Prepare values holder
	values := make([]interface{}, len(columns))
	valuePtrs := make([]interface{}, len(columns))
	for i := range values {
		valuePtrs[i] = &values[i]
	}

	// Collect all rows
	var tableRows [][]string
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

		tableRows = append(tableRows, stringValues)
	}

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %v", err)
	}

	// Render the table
	renderTable(outputWriter, columns, tableRows)

	// Get LLM interpretation of the results
	interpretation, err := getLLMInterpretation(llmConfig, query, outputWriter.buf.String())
	if err != nil {
		return fmt.Errorf("error getting interpretation: %v", err)
	}

	fmt.Printf("\nAnswer: %s\n", interpretation)
	return nil
}
