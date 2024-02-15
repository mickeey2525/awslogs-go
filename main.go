package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

var (
	region    = flag.String("region", "ap-northeast-1", "AWS Region")
	startTime = flag.String("s", "2024-02-01", "Start time")
	endTime   = flag.String("e", "2024-02-02", "End time")
	mode      = flag.String("mode", "stdout", "Mode: stdout or file")
	logGroup  = flag.String("log-group", "", "Log group name")
	logStream = flag.String("log-stream", "", "Log stream name (comma separated)")
)

type Mode int

const (
	ModeStdout Mode = iota
	ModeFile
)

type LogEvent struct {
	LogStreamName string
	Timestamp     time.Time
	Message       string
}

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
	startTime, err := time.Parse(shortForm, *startTime)
	if err != nil {
		log.Fatalf("Failed to parse start time: %v", err)
	}
	endTime, err := time.Parse(shortForm, *endTime)
	if err != nil {
		log.Fatalf("Failed to parse end time: %v", err)
	}
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(*region),
	)
	if err != nil {
		log.Fatalf("configuration error, " + err.Error())
	}

	cwLogsClient := cloudwatchlogs.NewFromConfig(cfg)
	var wg sync.WaitGroup

	logChannel := make(chan LogEvent, 10)
	go writeLogEvents(logChannel, *mode)

	for _, logStreamName := range logStreamNames {
		wg.Add(1)
		go func(logStreamName string) {
			defer wg.Done()
			getLogEvents(cwLogsClient, *logGroup, logStreamName, startTime, endTime, logChannel)
		}(logStreamName)
	}

	wg.Wait()
	close(logChannel)
}

func getLogEvents(client *cloudwatchlogs.Client, logGroupName, logStreamName string, startTime, endTime time.Time, logChannel chan<- LogEvent) {
	var lastToken *string

	input := &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
		StartTime:     aws.Int64(startTime.UnixMilli()),
		EndTime:       aws.Int64(endTime.UnixMilli()),
	}

	for {
		result, err := client.GetLogEvents(context.Background(), input)
		if err != nil {
			fmt.Printf("failed to get log events for stream %s, %v\n", logStreamName, err)
			return
		}

		for _, event := range result.Events {
			logChannel <- LogEvent{
				LogStreamName: logStreamName,
				Timestamp:     startTime,
				Message:       *event.Message,
			}
		}

		if lastToken != nil && result.NextBackwardToken != nil && *lastToken == *result.NextBackwardToken {
			return
		}

		lastToken = result.NextBackwardToken
		input.NextToken = result.NextBackwardToken
	}
}

func writeLogEvents(logChannel <-chan LogEvent, modeStr string) {
	var mode Mode
	switch modeStr {
	case "stdout":
		mode = ModeStdout
	case "file":
		mode = ModeFile
	default:
		log.Fatalf("Invalid mode: %s", modeStr)
	}

	if mode == ModeFile {
		for logEvent := range logChannel {
			fileName := fmt.Sprintf("%s_%s.log", logEvent.LogStreamName, logEvent.Timestamp.Format(shortForm))
			file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open file: %v", err)
			}
			writer := bufio.NewWriter(file)
			_, err = writer.WriteString(logEvent.Message + "\n")
			if err != nil {
				log.Fatalf("failed to write to file: %v", err)
			}
			writer.Flush()
			file.Close()
		}
	} else {
		// 標準出力に書き出す処理
		for logEvent := range logChannel {
			fmt.Println(logEvent.Message)
		}
	}
}
