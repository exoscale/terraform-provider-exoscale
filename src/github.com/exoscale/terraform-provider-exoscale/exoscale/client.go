package exoscale

import (
	"github.com/exoscale/egoscale"

	"gopkg.in/amz.v2/aws"
	"gopkg.in/amz.v2/s3"
)

const defaultComputeEndpoint = "https://api.exoscale.ch/compute"
const defaultDnsEndpoint = "https://api.exoscale.ch/dns"
const defaultS3Endpoint = "https://sos.exo.io"
const defaultTimeout = 60         // seconds
const defaultDelayBeforeRetry = 5 // seconds

// BaseConfig represents the provider structure
type BaseConfig struct {
	token            string
	secret           string
	timeout          int
	compute_endpoint string
	dns_endpoint     string
	s3_endpoint      string
	async            egoscale.AsyncInfo
}

func getClient(endpoint string, meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	return egoscale.NewClient(endpoint, config.token, config.secret)
}

// GetComputeClient builds a CloudStack client
func GetComputeClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	return getClient(config.compute_endpoint, meta)
}

// GetDnsClient builds a DNS client
func GetDnsClient(meta interface{}) *egoscale.Client {
	config := meta.(BaseConfig)
	return getClient(config.dns_endpoint, meta)
}

// GetS3Client builds a S3 client to (CH-GV1 region)
func GetS3Client(meta interface{}) *s3.S3 {
	config := meta.(BaseConfig)
	var exo1 = aws.Region{
		Name:                 "CH-GV1",
		S3Endpoint:           config.s3_endpoint,
		S3LocationConstraint: false,
	}

	var auth = aws.Auth{
		AccessKey: config.token,
		SecretKey: config.secret,
	}

	return s3.New(auth, exo1)
}
