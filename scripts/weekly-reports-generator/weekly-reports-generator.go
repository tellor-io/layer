package main

import (
	"bufio"
	"context"
	"encoding/csv"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/tellor-io/layer/utils"
)

const (
	// File names
	VoteExtParticipationFile = "vote_extension_participation.csv"
	BridgeValidatorFile      = "bridge_validator_timestamps.csv"

	// Analysis thresholds
	LowParticipationThreshold = 80.0

	// Schedule
	ReportDay  = time.Friday
	ReportHour = 9 // 9 AM
)

type VoteExtData struct {
	Height            uint64
	Timestamp         uint64
	ParticipationRate float64
}

type BridgeValidatorData struct {
	Timestamp time.Time
}

type WeeklyReport struct {
	// Vote extension analysis
	AvgParticipationRate    float64
	LowParticipationCount   int
	LowestParticipationRate float64
	TotalBlocks             int

	// Bridge validator analysis
	AvgUpdateFrequency   time.Duration
	ShortestUpdatePeriod time.Duration
	LongestUpdatePeriod  time.Duration
	TotalUpdates         int

	// Time period
	StartDate time.Time
	EndDate   time.Time
}

func main() {
	var (
		dataFolder                  string
		valsetUpdateChannel         string
		voteExtParticipationChannel string
		runOnce                     bool
	)

	flag.StringVar(&dataFolder, "data-folder", "", "Path to folder containing CSV files")
	flag.StringVar(&valsetUpdateChannel, "valset-update-channel", "", "Discord webhook URL for validator set update reports")
	flag.StringVar(&voteExtParticipationChannel, "vote-ext-participation-channel", "", "Discord webhook URL for vote extension participation reports")
	flag.BoolVar(&runOnce, "run-once", false, "Run the report once immediately instead of waiting for Friday")
	flag.Parse()

	if dataFolder == "" {
		log.Fatal("Usage: go run ./scripts/weekly-reports-generator/weekly-reports-generator.go -data-folder=<path> -valset-update-channel=<webhook_url> -vote-ext-participation-channel=<webhook_url>")
	}

	if valsetUpdateChannel == "" || voteExtParticipationChannel == "" {
		log.Fatal("Both -valset-update-channel and -vote-ext-participation-channel are required")
	}

	// Validate that the data folder exists
	if _, err := os.Stat(dataFolder); os.IsNotExist(err) {
		log.Fatalf("Data folder does not exist: %s", dataFolder)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if runOnce {
		log.Println("Running report once immediately...")
		if err := generateAndSendReports(ctx, dataFolder, valsetUpdateChannel, voteExtParticipationChannel); err != nil {
			log.Printf("Error generating reports: %v", err)
		}
		return
	}

	// Schedule reports for Fridays at 9 AM
	log.Println("Starting weekly reports generator...")
	log.Printf("Reports will be generated every %s at %d:00", ReportDay, ReportHour)

	for {
		nextRun := getNextFriday()
		log.Printf("Next report scheduled for: %s", nextRun.Format("2006-01-02 15:04:05"))

		// Wait until next Friday
		time.Sleep(time.Until(nextRun))

		log.Println("Generating weekly reports...")
		if err := generateAndSendReports(ctx, dataFolder, valsetUpdateChannel, voteExtParticipationChannel); err != nil {
			log.Printf("Error generating reports: %v", err)
		}
	}
}

func getNextFriday() time.Time {
	now := time.Now()

	// Calculate days until next Friday
	daysUntilFriday := int(ReportDay - now.Weekday())
	if daysUntilFriday <= 0 {
		daysUntilFriday += 7
	}

	// Set to next Friday at 9 AM
	nextFriday := now.AddDate(0, 0, daysUntilFriday)
	return time.Date(nextFriday.Year(), nextFriday.Month(), nextFriday.Day(), ReportHour, 0, 0, 0, nextFriday.Location())
}

func generateAndSendReports(ctx context.Context, dataFolder, valsetUpdateChannel, voteExtParticipationChannel string) error {
	// Calculate the time period for the last week
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	log.Printf("Analyzing data from %s to %s", startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	// Generate the report
	report, err := generateWeeklyReport(dataFolder, startDate, endDate)
	if err != nil {
		return fmt.Errorf("failed to generate weekly report: %w", err)
	}

	// Send reports to Discord channels
	var wg sync.WaitGroup
	wg.Add(2)

	// Send vote extension participation report
	go func() {
		defer wg.Done()
		if err := sendVoteExtParticipationReport(report, voteExtParticipationChannel); err != nil {
			log.Printf("Error sending vote extension participation report: %v", err)
		}
	}()

	// Send bridge validator set update report
	go func() {
		defer wg.Done()
		if err := sendBridgeValidatorReport(report, valsetUpdateChannel); err != nil {
			log.Printf("Error sending bridge validator report: %v", err)
		}
	}()

	wg.Wait()
	log.Println("Weekly reports completed successfully")
	return nil
}

func generateWeeklyReport(dataFolder string, startDate, endDate time.Time) (*WeeklyReport, error) {
	report := &WeeklyReport{
		StartDate: startDate,
		EndDate:   endDate,
	}

	// Analyze vote extension participation data
	voteExtData, err := loadVoteExtParticipationData(filepath.Join(dataFolder, VoteExtParticipationFile), startDate)
	if err != nil {
		return nil, fmt.Errorf("failed to load vote extension data: %w", err)
	}

	// Filter data for the last week
	var weeklyVoteExtData []VoteExtData
	for _, data := range voteExtData {
		blockTime := time.Unix(int64(data.Timestamp), 0)
		if blockTime.After(startDate) {
			weeklyVoteExtData = append(weeklyVoteExtData, data)
		}
	}

	if len(weeklyVoteExtData) > 0 {
		report.AvgParticipationRate, report.LowParticipationCount, report.LowestParticipationRate = analyzeVoteExtParticipation(weeklyVoteExtData)
		report.TotalBlocks = len(weeklyVoteExtData)
	}

	// Analyze bridge validator set update data
	bridgeData, err := loadBridgeValidatorData(filepath.Join(dataFolder, BridgeValidatorFile))
	if err != nil {
		return nil, fmt.Errorf("failed to load bridge validator data: %w", err)
	}

	// Filter data for the last week
	var weeklyBridgeData []BridgeValidatorData
	for _, data := range bridgeData {
		if data.Timestamp.After(startDate) {
			weeklyBridgeData = append(weeklyBridgeData, data)
		}
	}

	if len(weeklyBridgeData) > 1 {
		report.AvgUpdateFrequency, report.ShortestUpdatePeriod, report.LongestUpdatePeriod = analyzeBridgeValidatorUpdates(weeklyBridgeData)
		report.TotalUpdates = len(weeklyBridgeData)
	}

	return report, nil
}

func loadVoteExtParticipationData(filePath string, startDate time.Time) ([]VoteExtData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open vote extension file: %w", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 {
		return nil, fmt.Errorf("insufficient data in vote extension file")
	}

	var data []VoteExtData
	var validRecords [][]string
	validRecords = append(validRecords, records[0]) // Keep header

	for i, record := range records {
		if i == 0 {
			// Skip header processing
			continue
		}

		if len(record) < 3 {
			log.Printf("Warning: skipping malformed record %d (expected 3 columns, got %d)", i, len(record))
			continue
		}

		height, err := strconv.ParseUint(record[0], 10, 64)
		if err != nil {
			log.Printf("Warning: failed to parse height %s: %v", record[0], err)
			continue
		}

		timestamp, err := strconv.ParseUint(record[1], 10, 64)
		if err != nil {
			log.Printf("Warning: failed to parse timestamp %s: %v", record[1], err)
			continue
		}

		participationRate, err := strconv.ParseFloat(record[2], 64)
		if err != nil {
			log.Printf("Warning: failed to parse participation rate %s: %v", record[2], err)
			continue
		}

		blockTime := time.Unix(int64(timestamp), 0)

		// Only keep data that's not older than startDate
		if blockTime.After(startDate) || blockTime.Equal(startDate) {
			data = append(data, VoteExtData{
				Height:            height,
				Timestamp:         timestamp,
				ParticipationRate: participationRate,
			})
			validRecords = append(validRecords, record)
		}
	}

	// Clean up the CSV file by removing old data
	if err := cleanupVoteExtCSV(filePath, validRecords); err != nil {
		log.Printf("Warning: failed to cleanup CSV file: %v", err)
	}

	return data, nil
}

func cleanupVoteExtCSV(filePath string, validRecords [][]string) error {
	// Create a temporary file
	tempFile, err := os.CreateTemp(filepath.Dir(filePath), "vote_ext_cleanup_*.csv")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up temp file if we fail

	// Write valid records to temp file
	writer := csv.NewWriter(tempFile)
	if err := writer.WriteAll(validRecords); err != nil {
		tempFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	writer.Flush()
	tempFile.Close()

	// Replace original file with cleaned up version
	if err := os.Rename(tempFile.Name(), filePath); err != nil {
		return fmt.Errorf("failed to replace original file: %w", err)
	}

	log.Printf("Cleaned up vote extension CSV file, removed %d old records", len(validRecords)-1) // -1 for header
	return nil
}

func loadBridgeValidatorData(filePath string) ([]BridgeValidatorData, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open bridge validator file: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var data []BridgeValidatorData

	// Skip header
	if scanner.Scan() {
		header := scanner.Text()
		if header != "validator_set_update_timestamps" {
			log.Printf("Warning: unexpected CSV header: %s", header)
		}
	}

	for scanner.Scan() {
		timestampStr := strings.TrimSpace(scanner.Text())
		if timestampStr == "" {
			continue
		}

		// Parse Unix timestamp
		unixTime, err := strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			log.Printf("Warning: failed to parse timestamp %s: %v", timestampStr, err)
			continue
		}

		timestamp := time.Unix(unixTime, 0)
		data = append(data, BridgeValidatorData{
			Timestamp: timestamp,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return data, nil
}

func analyzeVoteExtParticipation(data []VoteExtData) (avgRate float64, lowCount int, lowestRate float64) {
	if len(data) == 0 {
		return 0, 0, 0
	}

	var totalRate float64
	lowestRate = 100.0 // Start with 100% as the lowest

	for _, d := range data {
		totalRate += d.ParticipationRate

		if d.ParticipationRate < LowParticipationThreshold {
			lowCount++
		}

		if d.ParticipationRate < lowestRate {
			lowestRate = d.ParticipationRate
		}
	}

	avgRate = totalRate / float64(len(data))
	return avgRate, lowCount, lowestRate
}

func analyzeBridgeValidatorUpdates(data []BridgeValidatorData) (avgFrequency, shortestPeriod, longestPeriod time.Duration) {
	if len(data) < 2 {
		return 0, 0, 0
	}

	// Sort by timestamp
	sort.Slice(data, func(i, j int) bool {
		return data[i].Timestamp.Before(data[j].Timestamp)
	})

	var timeDiffs []time.Duration
	for i := 1; i < len(data); i++ {
		diff := data[i].Timestamp.Sub(data[i-1].Timestamp)
		timeDiffs = append(timeDiffs, diff)
	}

	// Calculate average
	var totalDuration time.Duration
	for _, diff := range timeDiffs {
		totalDuration += diff
	}
	avgFrequency = totalDuration / time.Duration(len(timeDiffs))

	// Find shortest and longest periods
	shortestPeriod = timeDiffs[0]
	longestPeriod = timeDiffs[0]
	for _, diff := range timeDiffs {
		if diff < shortestPeriod {
			shortestPeriod = diff
		}
		if diff > longestPeriod {
			longestPeriod = diff
		}
	}

	return avgFrequency, shortestPeriod, longestPeriod
}

func sendVoteExtParticipationReport(report *WeeklyReport, webhookURL string) error {
	message := fmt.Sprintf("**Weekly Vote Extension Participation Report**\n\n"+
		"**Period:** %s to %s\n\n"+
		"**Summary:**\n"+
		"• Total blocks analyzed: %d\n"+
		"• Average participation rate: %.2f%%\n"+
		"• Blocks below %d%%: %d\n"+
		"• Lowest participation rate: %.2f%%\n\n"+
		"**Purpose:**\n"+
		"This report covers the vote extension participation rates for the past week. ",
		report.StartDate.Format("2006-01-02"),
		report.EndDate.Format("2006-01-02"),
		report.TotalBlocks,
		report.AvgParticipationRate,
		int(LowParticipationThreshold),
		report.LowParticipationCount,
		report.LowestParticipationRate)

	discordNotifier := utils.NewDiscordNotifier(webhookURL)
	return discordNotifier.SendAlert(message)
}

func sendBridgeValidatorReport(report *WeeklyReport, webhookURL string) error {
	message := fmt.Sprintf("**Weekly Bridge Validator Set Update Report**\n\n"+
		"**Period:** %s to %s\n\n"+
		"**Summary:**\n"+
		"• Total updates: %d\n"+
		"• Average frequency: %s\n"+
		"• Shortest period: %s\n"+
		"• Longest period: %s\n\n"+
		"**Purpose:**\n"+
		"This report analyzes the frequency of bridge validator set updates for the past week using the csv file created by the monitor on validator set update events.",
		report.StartDate.Format("2006-01-02"),
		report.EndDate.Format("2006-01-02"),
		report.TotalUpdates,
		formatDuration(report.AvgUpdateFrequency),
		formatDuration(report.ShortestUpdatePeriod),
		formatDuration(report.LongestUpdatePeriod))

	discordNotifier := utils.NewDiscordNotifier(webhookURL)
	return discordNotifier.SendAlert(message)
}

func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else {
		days := int(d.Hours()) / 24
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	}
}
