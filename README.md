# awslogs-go

## Overview

This is a simple Go application that demonstrates how to use the AWS SDK for Go to interact with Amazon CloudWatch Logs.

## Running the application

1. Install Go
2. Run the command to install this cli
```bash
go install github.com/mickeey2525/awslogs-go
```

## How to use

```bash
Usage of awslogs-go:
  -e string
        End time (default "2024-02-02")
  -log-group string
        Log group name
  -log-stream string
        Log stream name (comma separated)
  -mode string
        Mode: stdout or file (default "stdout")
  -p string
        aws profile name (default "default")
  -region string
        AWS Region (default "ap-northeast-1")
  -s string
        Start time (default "2024-02-01")
```

## Example

```bash
awslogs-go -log-group /aws/lambda/your-log-group -log-stream your-log-stream -s 2024-02-01 -e 2024-02-02 -mode file
```