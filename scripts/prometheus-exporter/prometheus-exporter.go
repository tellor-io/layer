package main

import (
	"database/sql"
	"encoding/csv"
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
			Values [][]json.RawMessage `json:"values"`
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

// writeCSVResponse writes price data as CSV to the response writer
func writeCSVResponse(w http.ResponseWriter, prices []PriceData) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=price_data.csv")

	writer := csv.NewWriter(w)
	defer writer.Flush()

	// Write header
	writer.Write([]string{"timestamp", "market_id", "exchange_id", "price"})

	// Write data rows
	for _, price := range prices {
		writer.Write([]string{
			price.Timestamp.Format("2006-01-02 15:04:05"),
			price.MarketID,
			price.ExchangeID,
			fmt.Sprintf("%.8f", price.Price),
		})
	}
}

func main() {
	// Check if we should run in API server mode
	if getEnv("API_MODE", "false") == "true" {
		log.Println("Starting API server mode")
		runAPIServer()
	} else if getEnv("COMBINED_MODE", "false") == "true" {
		log.Println("Starting in combined mode - API server + data collection")
		runCombinedMode()
	} else if getEnv("RESTART_MODE", "false") == "true" {
		log.Println("Starting in restart mode - API server + data collection (skipping initial collection)")
		runCombinedMode()
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

	log.Printf("Connecting to database: host=%s port=%s user=%s dbname=%s",
		config.DBHost, config.DBPort, config.DBUser, config.DBName)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	log.Println("Database connection successful")
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

	log.Println("Creating table and indexes...")
	_, err := db.Exec(query)
	if err != nil {
		return err
	}
	log.Println("Table and indexes created successfully")
	return nil
}

// runDataCollection runs a one-time data collection
func runDataCollection() {
	config := Config{
		PrometheusURL: getEnv("PROMETHEUS_URL", "http://54.160.217.166:9090"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "pricefeed"),
	}

	log.Printf("Configuration: PrometheusURL=%s, DBHost=%s, DBPort=%s, DBUser=%s, DBName=%s",
		config.PrometheusURL, config.DBHost, config.DBPort, config.DBUser, config.DBName)

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

	// Run data collection
	log.Println("Starting data collection...")
	if err := collectAndStoreData(config, db); err != nil {
		log.Fatalf("Data collection failed: %v", err)
	}
	log.Println("Data collection completed successfully")
}

// runScheduler runs the data collection on a schedule
func runScheduler() {
	config := Config{
		PrometheusURL: getEnv("PROMETHEUS_URL", "http://54.160.217.166:9090"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "pricefeed"),
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

	// Run initial data collection
	log.Println("Running initial data collection...")
	if err := collectAndStoreData(config, db); err != nil {
		log.Printf("Initial data collection failed: %v", err)
	} else {
		log.Println("Initial data collection completed successfully")
	}

	// Start scheduler
	ticker := time.NewTicker(24 * time.Hour) // Run daily
	defer ticker.Stop()

	log.Println("Scheduler started - will run daily at midnight")
	for range ticker.C {
		log.Println("Running scheduled data collection...")
		if err := collectAndStoreData(config, db); err != nil {
			log.Printf("Scheduled data collection failed: %v", err)
		} else {
			log.Println("Scheduled data collection completed successfully")
		}
	}
}

func collectAndStoreData(config Config, db *sql.DB) error {
	// Calculate yesterday's date range
	now := time.Now()
	yesterday := now.AddDate(0, 0, -1)
	start := time.Date(yesterday.Year(), yesterday.Month(), yesterday.Day(), 0, 0, 0, 0, time.UTC)
	end := start.Add(24 * time.Hour)

	log.Printf("Collecting data for date range: %s to %s", start.Format("2006-01-02 15:04:05"), end.Format("2006-01-02 15:04:05"))

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

	log.Printf("Prometheus API response status: %s", prometheusResp.Status)
	log.Printf("Number of result series: %d", len(prometheusResp.Data.Result))

	// Log detailed information about the response
	for i, result := range prometheusResp.Data.Result {
		log.Printf("Result %d: MarketID=%s, ExchangeID=%s, ChainID=%s, Job=%s, Values count=%d",
			i+1, result.Metric.MarketID, result.Metric.ExchangeID, result.Metric.ChainID, result.Metric.Job, len(result.Values))

		// Log first few values for debugging
		for j, value := range result.Values {
			if j < 3 { // Only log first 3 values to avoid spam
				if len(value) == 2 {
					// Parse timestamp from JSON
					var timestampInt int64
					if err := json.Unmarshal(value[0], &timestampInt); err != nil {
						// Try as float64 if int64 fails
						var timestampFloat float64
						if err := json.Unmarshal(value[0], &timestampFloat); err != nil {
							log.Printf("  Value %d: Failed to parse timestamp: %s", j+1, string(value[0]))
							continue
						}
						timestampInt = int64(timestampFloat)
					}

					// Parse price from JSON
					var priceStr string
					if err := json.Unmarshal(value[1], &priceStr); err != nil {
						log.Printf("  Value %d: Failed to parse price: %s", j+1, string(value[1]))
						continue
					}

					log.Printf("  Value %d: timestamp=%d (%s), price=%s", j+1, timestampInt, time.Unix(timestampInt, 0).Format("2006-01-02 15:04:05"), priceStr)
				}
			}
		}
	}

	return &prometheusResp, nil
}

func storePriceData(db *sql.DB, data *PrometheusResponse) error {
	log.Printf("Starting to store price data...")

	// Prepare insert statement
	stmt, err := db.Prepare(`
		INSERT INTO price_data (timestamp, market_id, exchange_id, price) 
		VALUES ($1, $2, $3, $4)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	totalInserted := 0
	totalSkipped := 0

	// Process each result
	for _, result := range data.Data.Result {
		marketID := result.Metric.MarketID
		exchangeID := result.Metric.ExchangeID

		// Skip entries with empty market_id or exchange_id
		if marketID == "" || exchangeID == "" {
			log.Printf("Skipping series with empty market_id or exchange_id: MarketID='%s', ExchangeID='%s'", marketID, exchangeID)
			continue
		}

		log.Printf("Processing series: MarketID=%s, ExchangeID=%s, Values=%d", marketID, exchangeID, len(result.Values))

		// Process each value in the time series
		for _, value := range result.Values {
			if len(value) != 2 {
				totalSkipped++
				continue
			}

			// Parse timestamp from JSON
			var timestampInt int64
			if err := json.Unmarshal(value[0], &timestampInt); err != nil {
				// Try as float64 if int64 fails
				var timestampFloat float64
				if err := json.Unmarshal(value[0], &timestampFloat); err != nil {
					log.Printf("Failed to parse timestamp: %s, error: %v", string(value[0]), err)
					totalSkipped++
					continue
				}
				timestampInt = int64(timestampFloat)
			}
			timestamp := time.Unix(timestampInt, 0)

			// Parse price from JSON
			var price float64
			if err := json.Unmarshal(value[1], &price); err != nil {
				// Try as string if float64 fails
				var priceStr string
				if err := json.Unmarshal(value[1], &priceStr); err != nil {
					log.Printf("Failed to parse price: %s, error: %v", string(value[1]), err)
					totalSkipped++
					continue
				}
				var err2 error
				price, err2 = strconv.ParseFloat(priceStr, 64)
				if err2 != nil {
					log.Printf("Failed to parse price string %s: %v", priceStr, err2)
					totalSkipped++
					continue
				}
			}

			// Insert into database
			_, err = stmt.Exec(timestamp, marketID, exchangeID, price)
			if err != nil {
				log.Printf("Failed to insert data: timestamp=%s, market=%s, exchange=%s, price=%f, error=%v",
					timestamp.Format("2006-01-02 15:04:05"), marketID, exchangeID, price, err)
				totalSkipped++
				continue
			}
			totalInserted++
		}
	}

	log.Printf("Data storage completed: %d records inserted, %d records skipped", totalInserted, totalSkipped)
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

// runCombinedMode starts both API server and data collection
func runCombinedMode() {
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

	// Run initial data collection (skip if RESTART_MODE is enabled)
	if getEnv("RESTART_MODE", "false") == "true" {
		log.Println("Skipping initial data collection (restart mode)")
	} else {
		log.Println("Running initial data collection...")
		if err := collectAndStoreData(config, db); err != nil {
			log.Printf("Initial data collection failed: %v", err)
		} else {
			log.Println("Initial data collection completed successfully")
		}
	}

	// Setup API routes
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/prices", authMiddleware(getPricesHandler(db)))
	http.HandleFunc("/api/prices/latest", authMiddleware(getLatestPricesHandler(db)))
	http.HandleFunc("/api/prices/market/", authMiddleware(getPricesByMarketHandler(db)))
	http.HandleFunc("/api/prices/exchange/", authMiddleware(getPricesByExchangeHandler(db)))
	http.HandleFunc("/api/prices/range", authMiddleware(getPricesByRangeHandler(db)))

	// Start data collection in a goroutine
	go func() {
		ticker := time.NewTicker(24 * time.Hour) // Run daily
		defer ticker.Stop()

		for range ticker.C {
			log.Println("Running scheduled data collection...")
			if err := collectAndStoreData(config, db); err != nil {
				log.Printf("Scheduled data collection failed: %v", err)
			} else {
				log.Println("Scheduled data collection completed successfully")
			}
		}
	}()

	log.Printf("Combined mode: API server starting on port %s with daily data collection", config.APIPort)
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
		// Parse query parameters
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")
		marketID := r.URL.Query().Get("market_id")
		exchangeID := r.URL.Query().Get("exchange_id")
		format := r.URL.Query().Get("format")

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
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error," + fmt.Sprintf("Database error: %v", err) + "\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   fmt.Sprintf("Database error: %v", err),
				})
			}
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
			// Filter out entries with empty market_id or exchange_id
			if p.MarketID == "" || p.ExchangeID == "" {
				continue
			}
			prices = append(prices, p)
		}

		if format == "csv" {
			writeCSVResponse(w, prices)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    prices,
				Count:   len(prices),
			})
		}
	}
}

// getLatestPricesHandler returns the latest price for each market/exchange combination
func getLatestPricesHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")
		log.Printf("DEBUG: format parameter = '%s'", format)

		query := `
			SELECT DISTINCT ON (market_id, exchange_id) 
				timestamp, market_id, exchange_id, price 
			FROM price_data 
			ORDER BY market_id, exchange_id, timestamp DESC
		`

		rows, err := db.Query(query)
		if err != nil {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error," + fmt.Sprintf("Database error: %v", err) + "\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   fmt.Sprintf("Database error: %v", err),
				})
			}
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
			// Filter out entries with empty market_id or exchange_id
			if p.MarketID == "" || p.ExchangeID == "" {
				continue
			}
			prices = append(prices, p)
		}

		if format == "csv" {
			log.Printf("DEBUG: Writing CSV response for %d prices", len(prices))
			writeCSVResponse(w, prices)
		} else {
			log.Printf("DEBUG: Writing JSON response for %d prices", len(prices))
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    prices,
				Count:   len(prices),
			})
		}
	}
}

// getPricesByMarketHandler returns prices for a specific market
func getPricesByMarketHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")

		// Extract market ID from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/prices/market/")
		if path == "" {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error,Market ID is required\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   "Market ID is required",
				})
			}
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
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error," + fmt.Sprintf("Database error: %v", err) + "\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   fmt.Sprintf("Database error: %v", err),
				})
			}
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
			// Filter out entries with empty market_id or exchange_id
			if p.MarketID == "" || p.ExchangeID == "" {
				continue
			}
			prices = append(prices, p)
		}

		if format == "csv" {
			writeCSVResponse(w, prices)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    prices,
				Count:   len(prices),
			})
		}
	}
}

// getPricesByExchangeHandler returns prices for a specific exchange
func getPricesByExchangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")

		// Extract exchange ID from URL path
		path := strings.TrimPrefix(r.URL.Path, "/api/prices/exchange/")
		if path == "" {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error,Exchange ID is required\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   "Exchange ID is required",
				})
			}
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
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error," + fmt.Sprintf("Database error: %v", err) + "\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   fmt.Sprintf("Database error: %v", err),
				})
			}
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
			// Filter out entries with empty market_id or exchange_id
			if p.MarketID == "" || p.ExchangeID == "" {
				continue
			}
			prices = append(prices, p)
		}

		if format == "csv" {
			writeCSVResponse(w, prices)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    prices,
				Count:   len(prices),
			})
		}
	}
}

// getPricesByRangeHandler returns prices within a date range
func getPricesByRangeHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		format := r.URL.Query().Get("format")

		startDate := r.URL.Query().Get("start")
		endDate := r.URL.Query().Get("end")
		marketID := r.URL.Query().Get("market_id")
		exchangeID := r.URL.Query().Get("exchange_id")
		limit := r.URL.Query().Get("limit")
		offset := r.URL.Query().Get("offset")

		if startDate == "" || endDate == "" {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error,start and end date parameters are required (format: YYYY-MM-DD)\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   "start and end date parameters are required (format: YYYY-MM-DD)",
				})
			}
			return
		}

		// Parse dates
		start, err := time.Parse("2006-01-02", startDate)
		if err != nil {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error,Invalid start date format. Use YYYY-MM-DD\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   "Invalid start date format. Use YYYY-MM-DD",
				})
			}
			return
		}

		end, err := time.Parse("2006-01-02", endDate)
		if err != nil {
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte("error,Invalid end date format. Use YYYY-MM-DD\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   "Invalid end date format. Use YYYY-MM-DD",
				})
			}
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
			if format == "csv" {
				w.Header().Set("Content-Type", "text/csv")
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("error," + fmt.Sprintf("Database error: %v", err) + "\n"))
			} else {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(APIResponse{
					Success: false,
					Error:   fmt.Sprintf("Database error: %v", err),
				})
			}
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
			// Filter out entries with empty market_id or exchange_id
			if p.MarketID == "" || p.ExchangeID == "" {
				continue
			}
			prices = append(prices, p)
		}

		if format == "csv" {
			writeCSVResponse(w, prices)
		} else {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(APIResponse{
				Success: true,
				Data:    prices,
				Count:   len(prices),
			})
		}
	}
}
