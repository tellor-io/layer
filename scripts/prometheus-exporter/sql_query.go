package main

import (
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var (
	forbiddenSQLKeyword = regexp.MustCompile(`(?i)\b(INSERT|UPDATE|DELETE|DROP|ALTER|CREATE|TRUNCATE|GRANT|REVOKE|COPY|CALL|EXECUTE|DO)\b`)
)

type queryRequest struct {
	SQL    string `json:"sql"`
	Format string `json:"format"`
}

// validateReadOnlySQL allows a single SELECT / WITH ... SELECT statement.
// The read-only DB role is the authoritative safety control; this is defense-in-depth.
func validateReadOnlySQL(sqlText string) error {
	s := strings.TrimSpace(sqlText)
	if s == "" {
		return fmt.Errorf("sql is empty")
	}

	// Allow one optional trailing semicolon, but reject additional statements.
	s = strings.TrimSpace(strings.TrimSuffix(s, ";"))
	if s == "" {
		return fmt.Errorf("sql is empty")
	}
	if strings.Contains(s, ";") {
		return fmt.Errorf("multiple statements are not allowed")
	}

	upper := strings.ToUpper(s)
	if !strings.HasPrefix(upper, "SELECT") && !strings.HasPrefix(upper, "WITH") {
		return fmt.Errorf("only SELECT or WITH ... SELECT queries are allowed")
	}

	if forbiddenSQLKeyword.MatchString(s) {
		return fmt.Errorf("query contains a forbidden keyword")
	}

	return nil
}

func queryHandler(db *sql.DB, config Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Method != http.MethodPost {
			writeQueryError(w, http.StatusMethodNotAllowed, "POST required", "")
			return
		}

		body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20)) // 1 MiB
		if err != nil {
			writeQueryError(w, http.StatusBadRequest, "failed to read request body", "")
			return
		}
		defer r.Body.Close()

		var req queryRequest
		if err := json.Unmarshal(body, &req); err != nil {
			writeQueryError(w, http.StatusBadRequest, "invalid JSON body; expected {\"sql\":\"...\"}", "")
			return
		}

		format := strings.ToLower(strings.TrimSpace(req.Format))
		if format == "" {
			format = strings.ToLower(r.URL.Query().Get("format"))
		}

		if err := validateReadOnlySQL(req.SQL); err != nil {
			log.Printf("query rejected: %v sql=%q", err, req.SQL)
			writeQueryError(w, http.StatusBadRequest, err.Error(), format)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), time.Duration(config.QueryTimeoutSeconds)*time.Second)
		defer cancel()

		tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: true})
		if err != nil {
			log.Printf("query begin failed: %v sql=%q", err, req.SQL)
			writeQueryError(w, http.StatusInternalServerError, fmt.Sprintf("failed to begin transaction: %v", err), format)
			return
		}
		defer func() { _ = tx.Rollback() }()

		rows, err := tx.QueryContext(ctx, req.SQL)
		if err != nil {
			log.Printf("query failed duration=%s err=%v sql=%q", time.Since(start), err, req.SQL)
			writeQueryError(w, http.StatusBadRequest, fmt.Sprintf("query failed: %v", err), format)
			return
		}
		defer rows.Close()

		columns, err := rows.Columns()
		if err != nil {
			writeQueryError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read columns: %v", err), format)
			return
		}

		data := make([]map[string]interface{}, 0)
		truncated := false

		for rows.Next() {
			if len(data) >= config.QueryMaxRows {
				truncated = true
				break
			}

			rawVals := make([]interface{}, len(columns))
			ptrs := make([]interface{}, len(columns))
			for i := range rawVals {
				ptrs[i] = &rawVals[i]
			}
			if err := rows.Scan(ptrs...); err != nil {
				log.Printf("query scan failed: %v sql=%q", err, req.SQL)
				writeQueryError(w, http.StatusInternalServerError, fmt.Sprintf("failed to scan row: %v", err), format)
				return
			}

			row := make(map[string]interface{}, len(columns))
			for i, col := range columns {
				row[col] = normalizeSQLValue(rawVals[i])
			}
			data = append(data, row)
		}
		if err := rows.Err(); err != nil {
			log.Printf("query rows error duration=%s err=%v sql=%q", time.Since(start), err, req.SQL)
			writeQueryError(w, http.StatusInternalServerError, fmt.Sprintf("row iteration error: %v", err), format)
			return
		}

		log.Printf("query ok duration=%s rows=%d truncated=%v sql=%q",
			time.Since(start), len(data), truncated, req.SQL)

		if format == CSVFormat {
			writeDynamicCSVResponse(w, columns, data)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(QueryAPIResponse{
			Success:   true,
			Columns:   columns,
			Data:      data,
			Count:     len(data),
			Truncated: truncated,
		}); err != nil {
			log.Printf("Error encoding query API response: %v", err)
		}
	}
}

func normalizeSQLValue(v interface{}) interface{} {
	switch t := v.(type) {
	case nil:
		return nil
	case []byte:
		return string(t)
	default:
		return t
	}
}

func writeQueryError(w http.ResponseWriter, status int, message, format string) {
	if format == CSVFormat {
		w.Header().Set("Content-Type", "text/csv")
		w.WriteHeader(status)
		_, _ = w.Write([]byte("error," + message + "\n"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(QueryAPIResponse{
		Success: false,
		Error:   message,
	})
}

func writeDynamicCSVResponse(w http.ResponseWriter, columns []string, data []map[string]interface{}) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=query_result.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	if err := writer.Write(columns); err != nil {
		log.Printf("Error writing CSV header: %v", err)
		return
	}

	for _, row := range data {
		record := make([]string, len(columns))
		for i, col := range columns {
			if row[col] == nil {
				record[i] = ""
			} else {
				record[i] = fmt.Sprint(row[col])
			}
		}
		if err := writer.Write(record); err != nil {
			log.Printf("Error writing CSV row: %v", err)
			return
		}
	}
}
