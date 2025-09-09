package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/tellor-io/layer/utils"
)

type DailyReport struct {
	Date                    string      `json:"date"`
	TotalBlocks             int         `json:"total_blocks"`
	AverageParticipation    float64     `json:"average_participation"`
	MinParticipation        float64     `json:"min_participation"`
	MaxParticipation        float64     `json:"max_participation"`
	LowParticipationBlocks  int         `json:"low_participation_blocks"`
	HighParticipationBlocks int         `json:"high_participation_blocks"`
	AlertsSent              int         `json:"alerts_sent"`
	DataPoints              []DataPoint `json:"data_points"`
}

type DataPoint struct {
	Height            uint64  `json:"height"`
	Timestamp         uint64  `json:"timestamp"`
	ParticipationRate float64 `json:"participation_rate"`
}

func analyzeDailyData(fileName, date string) (*DailyReport, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", fileName, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("failed to read CSV: %w", err)
	}

	if len(records) < 2 { // Header + at least one data row
		return nil, fmt.Errorf("insufficient data in file %s", fileName)
	}

	// Skip header row
	dataRows := records[1:]
	report := &DailyReport{
		Date:       date,
		DataPoints: make([]DataPoint, 0, len(dataRows)),
	}

	var participationRates []float64
	var totalParticipation float64

	for _, row := range dataRows {
		if len(row) < 3 {
			continue // Skip malformed rows
		}

		height, err := strconv.ParseUint(row[0], 10, 64)
		if err != nil {
			continue
		}

		timestamp, err := strconv.ParseUint(row[1], 10, 64)
		if err != nil {
			continue
		}

		participationRate, err := strconv.ParseFloat(row[2], 64)
		if err != nil {
			continue
		}

		report.DataPoints = append(report.DataPoints, DataPoint{
			Height:            height,
			Timestamp:         timestamp,
			ParticipationRate: participationRate,
		})

		participationRates = append(participationRates, participationRate)
		totalParticipation += participationRate
		report.TotalBlocks++
	}

	if len(participationRates) == 0 {
		return nil, fmt.Errorf("no valid data points found in file %s", fileName)
	}

	// Calculate statistics
	sort.Float64s(participationRates)
	report.AverageParticipation = totalParticipation / float64(len(participationRates))
	report.MinParticipation = participationRates[0]
	report.MaxParticipation = participationRates[len(participationRates)-1]

	// Count low/high participation blocks
	for _, rate := range participationRates {
		if rate < 80.0 {
			report.LowParticipationBlocks++
		}
		if rate > 95.0 {
			report.HighParticipationBlocks++
		}
	}

	return report, nil
}

func sendDailyReport(report *DailyReport) error {
	// Format the report message
	message := formatDailyReport(report)

	// Send via Discord
	eventType, ok := eventTypeMap["daily-report"]
	if !ok {
		// Fall back to vote-ext-part-rate if daily-report not configured
		eventType, ok = eventTypeMap["vote-ext-part-rate"]
		if !ok {
			return fmt.Errorf("no suitable event type found for daily report")
		}
	}

	discordNotifier := utils.NewDiscordNotifier(eventType.WebhookURL)
	if err := discordNotifier.SendAlert(message); err != nil {
		return fmt.Errorf("failed to send daily report: %w", err)
	}

	log.Printf("Daily report sent for %s", report.Date)
	return nil
}

func formatDailyReport(report *DailyReport) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("**📊 Daily Vote Extension Report - %s**\n\n", report.Date))

	sb.WriteString("**�� Summary Statistics:**\n")
	sb.WriteString(fmt.Sprintf("• Total Blocks Processed: %d\n", report.TotalBlocks))
	sb.WriteString(fmt.Sprintf("• Average Participation Rate: %.2f%%\n", report.AverageParticipation))
	sb.WriteString(fmt.Sprintf("• Min Participation Rate: %.2f%%\n", report.MinParticipation))
	sb.WriteString(fmt.Sprintf("• Max Participation Rate: %.2f%%\n", report.MaxParticipation))

	sb.WriteString("\n**⚠️ Alert Statistics:**\n")
	sb.WriteString(fmt.Sprintf("• Low Participation Blocks (<80%%): %d\n", report.LowParticipationBlocks))
	sb.WriteString(fmt.Sprintf("• High Participation Blocks (>95%%): %d\n", report.HighParticipationBlocks))

	if report.LowParticipationBlocks > 0 {
		sb.WriteString(fmt.Sprintf("\n**🚨 Warning:** %d blocks had low participation rates\n", report.LowParticipationBlocks))
	}

	if report.AverageParticipation < 85.0 {
		sb.WriteString("\n**🔴 Critical:** Daily average participation rate is below 85%%\n")
	} else if report.AverageParticipation < 90.0 {
		sb.WriteString("\n**🟡 Warning:** Daily average participation rate is below 90%%\n")
	} else {
		sb.WriteString("\n**🟢 Good:** Daily average participation rate is healthy\n")
	}

	return sb.String()
}
