package main

import (
	"log"
	"time"
)

// Scheduler runs the data collection on a daily schedule
func runScheduler() {
	// Calculate time until next midnight
	now := time.Now()
	midnight := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	timeUntilMidnight := midnight.Sub(now)

	log.Printf("Scheduler will start in %v", timeUntilMidnight)

	// Wait until midnight
	time.Sleep(timeUntilMidnight)

	// Run the scheduler
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	// Run immediately on first start
	runDataCollection()

	// Then run every 24 hours
	for range ticker.C {
		runDataCollection()
	}
}

func runDataCollection() {
	log.Println("Starting daily data collection...")

	// Load configuration from environment variables
	config := Config{
		PrometheusURL: getEnv("PROMETHEUS_URL", "http://54.160.217.166:9090"),
		DBHost:        getEnv("DB_HOST", "localhost"),
		DBPort:        getEnv("DB_PORT", "5432"),
		DBUser:        getEnv("DB_USER", "postgres"),
		DBPassword:    getEnv("DB_PASSWORD", "password"),
		DBName:        getEnv("DB_NAME", "pricefeed"),
	}

	// Initialize database
	db, err := initDB(config)
	if err != nil {
		log.Printf("Failed to initialize database: %v", err)
		return
	}
	defer db.Close()

	// Create table if it doesn't exist
	if err := createTable(db); err != nil {
		log.Printf("Failed to create table: %v", err)
		return
	}

	// Run the data collection
	if err := collectAndStoreData(config, db); err != nil {
		log.Printf("Failed to collect and store data: %v", err)
		return
	}

	log.Println("Daily data collection completed successfully")
}
