package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// Integration tests require a live Postgres with the read-only role configured.
// Run with:
//
//	DB_READONLY_PASSWORD=... go test -count=1 -run Integration ./...
//
// Optional overrides: DB_READONLY_HOST, DB_READONLY_PORT, DB_READONLY_USER, DB_READONLY_NAME
func integrationReadonlyDSN(t *testing.T) string {
	t.Helper()
	password := os.Getenv("DB_READONLY_PASSWORD")
	if password == "" {
		t.Skip("DB_READONLY_PASSWORD not set; skipping integration tests")
	}
	host := getEnv("DB_READONLY_HOST", getEnv("DB_HOST", "localhost"))
	port := getEnv("DB_READONLY_PORT", getEnv("DB_PORT", "5432"))
	user := getEnv("DB_READONLY_USER", "readonly_user")
	name := getEnv("DB_READONLY_NAME", getEnv("DB_NAME", "pricefeed"))
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, name)
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("postgres", integrationReadonlyDSN(t))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

func TestIntegrationReadonlyCanSelectJoinUnion(t *testing.T) {
	db := openIntegrationDB(t)

	queries := []string{
		`SELECT COUNT(*) FROM price_data`,
		`SELECT a.market_id FROM price_data a
		 INNER JOIN price_data b ON a.id = b.id
		 LIMIT 5`,
		`SELECT market_id FROM price_data WHERE false
		 UNION
		 SELECT market_id FROM price_data WHERE false`,
		`WITH t AS (SELECT market_id, price FROM price_data LIMIT 10)
		 SELECT market_id, AVG(price) FROM t GROUP BY 1`,
	}

	for _, q := range queries {
		rows, err := db.Query(q)
		if err != nil {
			t.Fatalf("query failed %q: %v", q, err)
		}
		for rows.Next() {
		}
		if err := rows.Err(); err != nil {
			t.Fatalf("rows err %q: %v", q, err)
		}
		_ = rows.Close()
	}
}

func TestIntegrationReadonlyCannotWrite(t *testing.T) {
	db := openIntegrationDB(t)

	writes := []string{
		`INSERT INTO price_data (timestamp, market_id, exchange_id, price) VALUES (NOW(), 'TEST', 'TEST', 1)`,
		`UPDATE price_data SET price = price WHERE false`,
		`DELETE FROM price_data WHERE false`,
		`DROP TABLE IF EXISTS price_data`,
	}

	for _, q := range writes {
		_, err := db.Exec(q)
		if err == nil {
			t.Fatalf("expected permission error for %q", q)
		}
	}
}

func TestIntegrationQueryHandlerTimeout(t *testing.T) {
	db := openIntegrationDB(t)
	t.Setenv("API_PASSWORD", "test-key")

	cfg := Config{
		QueryTimeoutSeconds: 1,
		QueryMaxRows:        100,
	}
	handler := authMiddleware(queryHandler(db, cfg))

	// pg_sleep requires ability to call it; if denied, skip.
	req := httptest.NewRequest(http.MethodPost, "/api/query",
		bytes.NewBufferString(`{"sql":"SELECT pg_sleep(3)"}`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "test-key")
	rr := httptest.NewRecorder()

	start := time.Now()
	handler(rr, req)
	elapsed := time.Since(start)

	if rr.Code == http.StatusOK {
		t.Fatalf("expected timeout/error, got 200 body=%s", rr.Body.String())
	}
	if elapsed > 5*time.Second {
		t.Fatalf("handler took too long: %s", elapsed)
	}
	var resp QueryAPIResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if resp.Success {
		t.Fatalf("expected success=false body=%s", rr.Body.String())
	}
}

func TestIntegrationQueryHandlerAuthMissing(t *testing.T) {
	db := openIntegrationDB(t)
	t.Setenv("API_PASSWORD", "test-key")

	handler := authMiddleware(queryHandler(db, Config{QueryTimeoutSeconds: 5, QueryMaxRows: 10}))
	req := httptest.NewRequest(http.MethodPost, "/api/query", bytes.NewBufferString(`{"sql":"SELECT 1"}`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d want 401", rr.Code)
	}
}
