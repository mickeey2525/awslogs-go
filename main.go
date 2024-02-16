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
	startTime = flag.String("s", "2024-02-01", "Start time")
	endTime   = flag.String("e", "2024-02-02", "End time")
	mode      = flag.String("mode", "stdout", "Mode: stdout or file")
	logGroup  = flag.String("log-group", "", "Log group name")
	logStream = flag.String("log-stream", "", "Log stream name (comma separated)")
	profile   = flag.String("p", "", "aws profile name")
)

func splitLogStreamNames(logStream string) []string {
	return strings.Split(logStream, ",")
}

const shortForm = "2006-01-02"

func init() {
	flag.Parse()
	startTime, err := time.Parse(shortForm, *startTime)
	if err != nil {
		log.Fatalf("Failed to parse start time: %v", err)
	}
	endTime, err := time.Parse(shortForm, *endTime)
	if err != nil {
		log.Fatalf("Failed to parse end time: %v", err)
	}

	if startTime.After(endTime) {
		log.Fatalf("Start time is after end time")
	}
	if *logGroup == "" {
		log.Fatalf("Log group name is required")
	}
	if *logStream == "" {
		log.Fatalf("Log stream name is required")
	}
}

func main() {
	logStreamNames := splitLogStreamNames(*logStream)
	startTime, _ := time.Parse(shortForm, *startTime)
	endTime, _ := time.Parse(shortForm, *endTime)

	cwLogsClient, err := cloudwatch.New(*region, *profile)
	if err != nil {
		log.Fatalf("failed %s", err)
	}
	var wg sync.WaitGroup

	logChannel := make(chan cloudwatch.LogEvent, 10)
	go cloudwatch.WriteLogEvents(logChannel, *mode)

	for _, logStreamName := range logStreamNames {
		wg.Add(1)
		go func(logStreamName string) {
			defer wg.Done()
			cloudwatch.GetLogEvents(cwLogsClient, *logGroup, logStreamName, startTime, endTime, logChannel)
		}(logStreamName)
	}

	wg.Wait()
	close(logChannel)
}
