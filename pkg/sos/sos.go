package sos

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/s3"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	awscredentials "github.com/aws/aws-sdk-go-v2/credentials"
)

func NewSOSClient(ctx context.Context, zone, sosEndpoint, exoAPIKey, exoAPISecret string) (*s3.Client, error) {
	if sosEndpoint == "" {
		sosEndpoint = "https://sos-" + zone + ".exo.io"
	}
	cfg, err := awsconfig.LoadDefaultConfig(
		ctx,
		awsconfig.WithRegion(zone),
		awsconfig.WithCredentialsProvider(
			awscredentials.NewStaticCredentialsProvider(
				exoAPIKey, exoAPISecret, "")),

		// To get detailed logging for debugging, uncomment this:
		// awsconfig.WithClientLogMode(aws.LogRequest|aws.LogResponse),
	)
	if err != nil {
		return nil, err
	}

	sosClient := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &sosEndpoint
		o.UsePathStyle = true
	})

	return sosClient, nil
}
