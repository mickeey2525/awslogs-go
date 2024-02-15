package cloudwatch

import (
	"bufio"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"log"
	"os"
	"time"
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

func GetLogEvents(client *cloudwatchlogs.Client, logGroupName, logStreamName string, startTime, endTime time.Time, logChannel chan<- LogEvent) {
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

func WriteLogEvents(logChannel <-chan LogEvent, modeStr string) {
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
			fileName := fmt.Sprintf("%s_%s.log", logEvent.LogStreamName, logEvent.Timestamp.Format("2006-01-02"))
			file, err := os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				log.Fatalf("failed to open file: %v", err)
			}
			writer := bufio.NewWriter(file)
			_, err = writer.WriteString(logEvent.Message + "\n")
			if err != nil {
				log.Fatalf("failed to write to file: %v", err)
			}
			err = writer.Flush()
			if err != nil {
				return
			}
			err = file.Close()
			if err != nil {
				return
			}
		}
	} else {
		// 標準出力に書き出す処理
		for logEvent := range logChannel {
			fmt.Println(logEvent.Message)
		}
	}
}
