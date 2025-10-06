package main

import (
	"log"
	"time"
)

// Scheduler runs the data collection on a daily schedule
func runSchedulerWithDelay() {
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
