package cloudwatch

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
)

func New(region, profile string) (*cloudwatchlogs.Client, error) {
	ctx := context.TODO()
	cfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(region),
	)
	if profile != "" {
		cfg, err = config.LoadDefaultConfig(ctx,
			config.WithSharedConfigProfile(profile),
			config.WithRegion(region))
	}
	if err != nil {
		return nil, err
	}
	cwLogsClient := cloudwatchlogs.NewFromConfig(cfg)
	return cwLogsClient, nil
}
