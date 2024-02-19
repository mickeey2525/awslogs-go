package cloudwatch

import (
	"bufio"
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"log"
	"os"
	"sync"
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
			log.Printf("Failed to get log events for stream %s: %v\n", logStreamName, err)
			return
		}

		for _, event := range result.Events {
			eventTime := time.UnixMilli(*event.Timestamp).UTC()
			// Truncate the timestamp to the start of the day (yyyy-mm-dd)
			eventDate := time.Date(eventTime.Year(), eventTime.Month(), eventTime.Day(), 0, 0, 0, 0, eventTime.Location())

			logChannel <- LogEvent{
				LogStreamName: logStreamName,
				Timestamp:     eventDate,
				Message:       *event.Message,
			}
		}

		if lastToken != nil && result.NextBackwardToken != nil && *lastToken == *result.NextBackwardToken {
			break
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

	fileHandles := make(map[string]*os.File)
	var fileMutex sync.Mutex

	if mode == ModeFile {
		defer func() {
			fileMutex.Lock()
			defer fileMutex.Unlock()
			for _, file := range fileHandles {
				err := file.Close()
				if err != nil {
					return
				}
			}
		}()
	}

	for logEvent := range logChannel {
		if mode == ModeFile {
			fileName := fmt.Sprintf("%s_%s.log", logEvent.LogStreamName, logEvent.Timestamp.Format("2006-01-02"))
			fileMutex.Lock()
			file, exists := fileHandles[fileName]
			if !exists {
				var err error
				file, err = os.OpenFile(fileName, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					log.Printf("Failed to open file %s: %v", fileName, err)
					fileMutex.Unlock()
					continue
				}
				fileHandles[fileName] = file
			}
			fileMutex.Unlock()

			writer := bufio.NewWriter(file)
			if _, err := writer.WriteString(logEvent.Message + "\n"); err != nil {
				log.Printf("Failed to write to file %s: %v", fileName, err)
				continue
			}
			err := writer.Flush()
			if err != nil {
				return
			}
		} else {
			fmt.Println(logEvent.Message)
		}
	}
}
