package exoscale

import "github.com/runseb/egoscale/src/egoscale"

const ComputeEndpoint = "https://api.exoscale.ch/compute"


type Client struct {
	token		string
	secret		string
}

func GetClient(endpoint string, meta interface{}) *egoscale.Client {
	client := meta.(Client)
	return egoscale.NewClient(endpoint, client.token, client.secret)
}
