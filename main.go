package main

import (
	"flag"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/mickeey2525/awslogs-go/cloudwatch"
)

var (
	region    = flag.String("region", "ap-northeast-1", "AWS Region")
	startDate = flag.String("s", "2024-02-01", "Start date (inclusive) in YYYY-MM-DD format")
	endDate   = flag.String("e", "2024-02-02", "End date (exclusive) in YYYY-MM-DD format")
	mode      = flag.String("mode", "stdout", "Output mode: stdout or file")
	logGroup  = flag.String("log-group", "", "Log group name")
	logStream = flag.String("log-stream", "", "Log stream name (comma separated)")
	profile   = flag.String("profile", "default", "AWS profile name")
)

func splitLogStreamNames(logStream string) []string {
	return strings.Split(logStream, ",")
}

const shortForm = "2006-01-02"

func main() {
	flag.Parse()

	// Parse start and end dates
	startTime, err := time.Parse(shortForm, *startDate)
	if err != nil {
		log.Fatalf("Failed to parse start date: %v", err)
	}
	endTime, err := time.Parse(shortForm, *endDate)
	if err != nil {
		log.Fatalf("Failed to parse end date: %v", err)
	}

	// Validate input
	if startTime.After(endTime) {
		log.Fatalf("Start date must be before end date")
	}
	if *logGroup == "" {
		log.Fatalf("Log group name is required")
	}
	if *logStream == "" {
		log.Fatalf("Log stream name is required")
	}

	// Initialize AWS CloudWatch Logs client
	cwLogsClient, err := cloudwatch.New(*region, *profile)
	if err != nil {
		log.Fatalf("Failed to create AWS CloudWatch Logs client: %v", err)
	}

	// Process log streams
	logStreamNames := splitLogStreamNames(*logStream)
	logChannel := make(chan cloudwatch.LogEvent, 10)
	go cloudwatch.WriteLogEvents(logChannel, *mode)

	var wg sync.WaitGroup
	for _, logStreamName := range logStreamNames {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			cloudwatch.GetLogEvents(cwLogsClient, *logGroup, name, startTime, endTime, logChannel)
		}(logStreamName)
	}

	wg.Wait()
	close(logChannel)
}
