# Provide a simple mechanism to ensure a proper build

GO15VENDOREXPERIMENT := 1
export GO15VENDOREXPERIMENT

GOPATH := $(PWD)
export GOPATH

all: deps build

build:
	go install github.com/exoscale/terraform-provider-exoscale/

deps:
	go get github.com/hashicorp/terraform
	go get github.com/pyr/egoscale/src/egoscale
	go get gopkg.in/amz.v2/s3

clean:
	rm -rf bin/
