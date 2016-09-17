package exoscale

import (
    "github.com/pyr/egoscale/src/egoscale"

    "gopkg.in/amz.v2/aws"
    "gopkg.in/amz.v2/s3"
)

const ComputeEndpoint = "https://api.exoscale.ch/compute"
const DNSEndpoint = "https://api.exoscale.ch/dns"
const S3Endpoint = "https://sos.exo.io"

type Client struct {
	token  string
	secret string
}

func GetClient(endpoint string, meta interface{}) *egoscale.Client {
	client := meta.(Client)
	return egoscale.NewClient(endpoint, client.token, client.secret)
}

func GetS3Client(meta interface{}) *s3.S3 {
    client := meta.(Client)
    var exo1 = aws.Region{
        Name: "CH-GV1",
        S3Endpoint: S3Endpoint,
        S3LocationConstraint: false,
    }

    var auth = aws.Auth{
        AccessKey: client.token,
        SecretKey: client.secret,
    }

    return s3.New(auth, exo1)
}
