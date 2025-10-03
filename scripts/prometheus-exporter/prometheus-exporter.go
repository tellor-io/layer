package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// PrometheusResponse represents the structure of the Prometheus API response
type PrometheusResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric struct {
				Name       string `json:"__name__"`
				ChainID    string `json:"chain_id"`
				ExchangeID string `json:"exchange_id"`
				Instance   string `json:"instance"`
				Job        string `json:"job"`
				MarketID   string `json:"market_id"`
			} `json:"metric"`
			Values [][]interface{} `json:"values"`
		} `json:"result"`
	} `json:"data"`
}

// PriceData represents a single price data point
type PriceData struct {
	Timestamp  time.Time
	MarketID   string
	ExchangeID string
	Price      float64
}

// Config holds application configuration
type Config struct {
	PrometheusURL string
	DBHost        string
	DBPort        string
	DBUser        string
	DBPassword    string
	DBName        string
	APIPassword   string
	APIPort       string
}

// APIResponse represents a standard API response
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Count   int         `json:"count,omitempty"`
}

func main() {
	// Check if we should run in API server mode
	if getEnv("API_MODE", "false") == "true" {
		log.Println("Starting API server mode")
		runAPIServer()
	} else if getEnv("SCHEDULER_MODE", "false") == "true" {
		log.Println("Starting in scheduler mode - will run daily at midnight")
		runScheduler()
	} else {
		log.Println("Running one-time data collection")
		runDataCollection()
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func initDB(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		config.DBHost, config.DBPort, config.DBUser, config.DBPassword, config.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}

func createTable(db *sql.DB) error {
	query := `
	CREATE TABLE IF NOT EXISTS price_data (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP NOT NULL,
		market_id VARCHAR(255) NOT NULL,
		exchange_id VARCHAR(255) NOT NULL,
		price DECIMAL(20,8) NOT NULL,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE INDEX IF NOT EXISTS idx_price_data_timestamp ON price_data(timestamp);
	CREATE INDEX IF NOT EXISTS idx_price_data_market_id ON price_data(market_id);
	CREATE INDEX IF NOT EXISTS idx_price_data_exchange_id ON price_data(exchange_id);
	`

	_, err := db.Exec(query)
	return err
}

func collectAndStoreData(config Config, db *sql.DB) error {
	// Calculate yesterday's date range
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	// Query Prometheus API
	data, err := queryPrometheus(config.PrometheusURL, start, end)
	if err != nil {
		return fmt.Errorf("failed to query Prometheus: %v", err)
	}

	// Parse and store data
	if err := storePriceData(db, data); err != nil {
		return fmt.Errorf("failed to store price data: %v", err)
	}

	return nil
}

func queryPrometheus(url string, start, end time.Time) (*PrometheusResponse, error) {
	// Format dates for Prometheus API
	startStr := start.Format("2006-01-02T15:04:05Z")
	endStr := end.Format("2006-01-02T15:04:05Z")

	// Build query URL
	queryURL := fmt.Sprintf("%s/api/v1/query_range?query=pricefeed_daemon_price_encoder_update_price&start=%s&end=%s&step=60s",
		url, startStr, endStr)

	log.Printf("Querying Prometheus: %s", queryURL)

	// Make HTTP request
	resp, err := http.Get(queryURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	// Parse JSON response
	var prometheusResp PrometheusResponse
	if err := json.NewDecoder(resp.Body).Decode(&prometheusResp); err != nil {
		return nil, err
	}

	return &prometheusResp, nil
}

func storePriceData(db *sql.DB, data *PrometheusResponse) error {
	// Prepare insert statement
	stmt, err := db.Prepare(`
		INSERT INTO price_data (timestamp, market_id, exchange_id, price) 
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	// Process each result
	for _, result := range data.Data.Result {
		marketID := result.Metric.MarketID
		exchangeID := result.Metric.ExchangeID

		// Process each value in the time series
		for _, value := range result.Values {
			if len(value) != 2 {
				continue
			}

			// Parse timestamp (Unix timestamp)
			timestampInt, ok := value[0].(int64)
			if !ok {
				continue
			}
			timestamp := time.Unix(timestampInt, 0)

			// Parse price
			priceStr, ok := value[1].(string)
			if !ok {
				continue
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				log.Printf("Failed to parse price %s: %v", priceStr, err)
				continue
			}

			// Insert into database
			_, err = stmt.Exec(timestamp, marketID, exchangeID, price)
			if err != nil {
				log.Printf("Failed to insert data: %v", err)
				continue
			}
		}
	}

	log.Printf("Successfully stored price data")
	return nil
}

// runAPIServer starts the HTTP API server
func runAPIServer() {
	config := Config{
		PrometheusURL: getEnv("PROMETHEUS_URL", "http://54.160.217.166:9090"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "pricefeed"),
		APIPassword:   getEnv("API_PASSWORD", "admin123"),
		APIPort:       getEnv("API_PORT", "8080"),
	}

	// Initialize database connection
	db, err := initDB(config)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create table if it doesn't exist
	if err := createTable(db); err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	// Setup routes
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/prices", authMiddleware(getPricesHandler(db)))
	http.HandleFunc("/api/prices/latest", authMiddleware(getLatestPricesHandler(db)))
	http.HandleFunc("/api/prices/market/", authMiddleware(getPricesByMarketHandler(db)))
	http.HandleFunc("/api/prices/exchange/", authMiddleware(getPricesByExchangeHandler(db)))
	http.HandleFunc("/api/prices/range", authMiddleware(getPricesByRangeHandler(db)))

	log.Printf("API server starting on port %s", config.APIPort)
	log.Fatal(http.ListenAndServe(":"+config.APIPort, nil))
}

// authMiddleware provides password-based authentication
func authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		password := r.Header.Get("X-API-Key")
		expectedPassword := getEnv("API_PASSWORD", "admin123")

		if password != expectedPassword {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid API key",
			})
			return
		}

		next(w, r)
	}
}

// healthHandler provides a simple health check endpoint
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    map[string]string{"status": "healthy"},
	})
}

// getPricesHandler returns all price data with optional filtering
func getPricesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Parse query parameters
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")
		marketID := r.URL.Query().Get("market_id")
		exchangeID := r.URL.Query().Get("exchange_id")

		// Build query
		query := "SELECT timestamp, market_id, exchange_id, price FROM price_data WHERE 1=1"
		args := []interface{}{}
		argIndex := 1

		if marketID != "" {
			query += fmt.Sprintf(" AND market_id = $%d", argIndex)
			args = append(args, marketID)
			argIndex++
		}

		if exchangeID != "" {
			query += fmt.Sprintf(" AND exchange_id = $%d", argIndex)
			args = append(args, exchangeID)
			argIndex++
		}

		query += " ORDER BY timestamp DESC"

		if limit != "" {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, limit)
			argIndex++
		}

		if offset != "" {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Database error: %v", err),
			})
			return
		}
		defer rows.Close()

		var prices []PriceData
		for rows.Next() {
			var p PriceData
			err := rows.Scan(&p.Timestamp, &p.MarketID, &p.ExchangeID, &p.Price)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			prices = append(prices, p)
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    prices,
			Count:   len(prices),
		})
	}
}

// getLatestPricesHandler returns the latest price for each market/exchange combination
func getLatestPricesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		query := `
			SELECT DISTINCT ON (market_id, exchange_id) 
				timestamp, market_id, exchange_id, price 
			FROM price_data 
			ORDER BY market_id, exchange_id, timestamp DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Database error: %v", err),
			})
			return
		}
		defer rows.Close()

		var prices []PriceData
		for rows.Next() {
			var p PriceData
			err := rows.Scan(&p.Timestamp, &p.MarketID, &p.ExchangeID, &p.Price)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			prices = append(prices, p)
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    prices,
			Count:   len(prices),
		})
	}
}

// getPricesByMarketHandler returns prices for a specific market
func getPricesByMarketHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract market ID from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/prices/market/")
		if path == "" {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Market ID is required",
			})
			return
		}

		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")

		query := "SELECT timestamp, market_id, exchange_id, price FROM price_data WHERE market_id = $1 ORDER BY timestamp DESC"
		args := []interface{}{path}
		argIndex := 2

		if limit != "" {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, limit)
			argIndex++
		}

		if offset != "" {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Database error: %v", err),
			})
			return
		}
		defer rows.Close()

		var prices []PriceData
		for rows.Next() {
			var p PriceData
			err := rows.Scan(&p.Timestamp, &p.MarketID, &p.ExchangeID, &p.Price)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			prices = append(prices, p)
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    prices,
			Count:   len(prices),
		})
	}
}

// getPricesByExchangeHandler returns prices for a specific exchange
func getPricesByExchangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Extract exchange ID from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/prices/exchange/")
		if path == "" {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Exchange ID is required",
			})
			return
		}

		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")

		query := "SELECT timestamp, market_id, exchange_id, price FROM price_data WHERE exchange_id = $1 ORDER BY timestamp DESC"
		args := []interface{}{path}
		argIndex := 2

		if limit != "" {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, limit)
			argIndex++
		}

		if offset != "" {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Database error: %v", err),
			})
			return
		}
		defer rows.Close()

		var prices []PriceData
		for rows.Next() {
			var p PriceData
			err := rows.Scan(&p.Timestamp, &p.MarketID, &p.ExchangeID, &p.Price)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			prices = append(prices, p)
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    prices,
			Count:   len(prices),
		})
	}
}

// getPricesByRangeHandler returns prices within a date range
func getPricesByRangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		startDate := r.URL.Query().Get("start")
		endDate := r.URL.Query().Get("end")
		marketID := r.URL.Query().Get("market_id")
		exchangeID := r.URL.Query().Get("exchange_id")
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")

		if startDate == "" || endDate == "" {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "start and end date parameters are required (format: YYYY-MM-DD)",
			})
			return
		}

		// Parse dates
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid start date format. Use YYYY-MM-DD",
			})
			return
		}

		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   "Invalid end date format. Use YYYY-MM-DD",
			})
			return
		}

		// Build query
		query := "SELECT timestamp, market_id, exchange_id, price FROM price_data WHERE timestamp >= $1 AND timestamp <= $2"
		args := []interface{}{start, end}
		argIndex := 3

		if marketID != "" {
			query += fmt.Sprintf(" AND market_id = $%d", argIndex)
			args = append(args, marketID)
			argIndex++
		}

		if exchangeID != "" {
			query += fmt.Sprintf(" AND exchange_id = $%d", argIndex)
			args = append(args, exchangeID)
			argIndex++
		}

		query += " ORDER BY timestamp DESC"

		if limit != "" {
			query += fmt.Sprintf(" LIMIT $%d", argIndex)
			args = append(args, limit)
			argIndex++
		}

		if offset != "" {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}

		rows, err := db.Query(query, args...)
		if err != nil {
			json.NewEncoder(w).Encode(APIResponse{
				Success: false,
				Error:   fmt.Sprintf("Database error: %v", err),
			})
			return
		}
		defer rows.Close()

		var prices []PriceData
		for rows.Next() {
			var p PriceData
			err := rows.Scan(&p.Timestamp, &p.MarketID, &p.ExchangeID, &p.Price)
			if err != nil {
				log.Printf("Error scanning row: %v", err)
				continue
			}
			prices = append(prices, p)
		}

		json.NewEncoder(w).Encode(APIResponse{
			Success: true,
			Data:    prices,
			Count:   len(prices),
		})
	}
}
