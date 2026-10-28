package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestValidateReadOnlySQL(t *testing.T) {
	t.Parallel()

	allow := []string{
		"SELECT 1",
		"select market_id, price from price_data",
		"SELECT * FROM price_data WHERE market_id = 'BTC-USD'",
		"SELECT market_id, COUNT(*) FROM price_data GROUP BY 1 ORDER BY 1",
		"WITH t AS (SELECT * FROM price_data LIMIT 100) SELECT exchange_id, AVG(price) FROM t GROUP BY 1",
		"with latest as (select * from price_data) select * from latest",
		"SELECT 1;", // single trailing semicolon allowed
		"  SELECT 1  ",
	}

	for _, sqlText := range allow {
		if err := validateReadOnlySQL(sqlText); err != nil {
			t.Errorf("expected allow %q, got error: %v", sqlText, err)
		}
	}

	reject := []struct {
		sql     string
		contain string
	}{
		{"", "empty"},
		{"   ", "empty"},
		{"SELECT 1; DROP TABLE price_data", "multiple"},
		{"SELECT 1; SELECT 2", "multiple"},
		{"INSERT INTO price_data VALUES (NOW(), 'a', 'b', 1)", "SELECT"},
		{"UPDATE price_data SET price = 0", "SELECT"},
		{"DELETE FROM price_data", "SELECT"},
		{"DROP TABLE price_data", "SELECT"},
		{"ALTER TABLE price_data ADD COLUMN x INT", "SELECT"},
		{"CREATE TABLE x (id INT)", "SELECT"},
		{"TRUNCATE price_data", "SELECT"},
		{"GRANT SELECT ON price_data TO public", "SELECT"},
		{"REVOKE SELECT ON price_data FROM public", "SELECT"},
		{"COPY price_data TO STDOUT", "SELECT"},
		{"SELECT * FROM price_data; DELETE FROM price_data", "multiple"},
		// Leading comments are rejected in v1 (must start with SELECT/WITH)
		{"-- comment\nSELECT 1", "SELECT"},
		{"/* comment */ SELECT 1", "SELECT"},
		// Forbidden keyword even inside an otherwise-valid SELECT (coarse filter)
		{"SELECT * FROM price_data WHERE note = 'ok'; DROP TABLE x; --'", "multiple"},
		{"SELECT DROP FROM price_data", "forbidden"},
	}

	for _, tc := range reject {
		err := validateReadOnlySQL(tc.sql)
		if err == nil {
			t.Errorf("expected reject %q", tc.sql)
			continue
		}
		if tc.contain != "" && !strings.Contains(strings.ToLower(err.Error()), strings.ToLower(tc.contain)) {
			t.Errorf("reject %q: error %q does not contain %q", tc.sql, err.Error(), tc.contain)
		}
	}
}

func TestQueryHandlerAuthAndValidation(t *testing.T) {
	t.Setenv("API_PASSWORD", "test-key")

	db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		QueryTimeoutSeconds: 5,
		QueryMaxRows:        10,
	}
	handler := authMiddleware(queryHandler(db, cfg))

	t.Run("missing auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"sql":"SELECT 1"}`))
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		handler(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d want 401 body=%s", rr.Code, rr.Body.String())
		}
	})

	t.Run("reject delete", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"sql":"DELETE FROM price_data"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-API-Key", "test-key")
		rr := httptest.NewRecorder()
		handler(rr, req)
		if rr.Code != http.StatusBadRequest {
			t.Fatalf("status=%d want 400 body=%s", rr.Code, rr.Body.String())
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unexpected DB use: %v", err)
		}
	})

	t.Run("method not allowed", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/query", nil)
		req.Header.Set("X-API-Key", "test-key")
		rr := httptest.NewRecorder()
		handler(rr, req)
		if rr.Code != http.StatusMethodNotAllowed {
			t.Fatalf("status=%d want 405", rr.Code)
		}
	})
}

func TestQueryHandlerSuccessAndTruncate(t *testing.T) {
	t.Setenv("API_PASSWORD", "test-key")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := Config{
		QueryTimeoutSeconds: 5,
		QueryMaxRows:        2,
	}
	handler := authMiddleware(queryHandler(db, cfg))

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"market_id", "n"}).
		AddRow("BTC-USD", 1).
		AddRow("ETH-USD", 2).
		AddRow("SOL-USD", 3)
	mock.ExpectQuery("SELECT market_id, COUNT").WillReturnRows(rows)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/api/query",
		bytes.NewBufferString(`{"sql":"SELECT market_id, COUNT(*) AS n FROM price_data GROUP BY 1"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var resp QueryAPIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("json: %v", err)
	}
	if !resp.Success {
		t.Fatalf("success=false error=%s", resp.Error)
	}
	if resp.Count != 2 {
		t.Fatalf("count=%d want 2", resp.Count)
	}
	if !resp.Truncated {
		t.Fatalf("expected truncated=true")
	}
	if len(resp.Columns) != 2 || resp.Columns[0] != "market_id" {
		t.Fatalf("columns=%v", resp.Columns)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}

func TestQueryHandlerCSV(t *testing.T) {
	t.Setenv("API_PASSWORD", "test-key")

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
	defer db.Close()

	cfg := Config{QueryTimeoutSeconds: 5, QueryMaxRows: 100}
	handler := authMiddleware(queryHandler(db, cfg))

	mock.ExpectBegin()
	rows := sqlmock.NewRows([]string{"n"}).AddRow(42)
	mock.ExpectQuery("SELECT 42").WillReturnRows(rows)
	mock.ExpectRollback()

	req := httptest.NewRequest(http.MethodPost, "/api/query",
		bytes.NewBufferString(`{"sql":"SELECT 42 AS n","format":"csv"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	ct := rr.Header().Get("Content-Type")
	if !strings.Contains(ct, "text/csv") {
		t.Fatalf("content-type=%s", ct)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "n") || !strings.Contains(body, "42") {
		t.Fatalf("csv body=%q", body)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("sqlmock: %v", err)
	}
}
