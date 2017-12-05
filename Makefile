# Provide a simple mechanism to ensure a proper build

GO15VENDOREXPERIMENT := 1
export GO15VENDOREXPERIMENT

GOPATH := $(PWD)
export GOPATH

.PHONY: all
all: deps build

.PHONY: build
build:
	go install github.com/exoscale/terraform-provider-exoscale/

.PHONY: deps
deps:
	go get github.com/hashicorp/terraform
	go get github.com/exoscale/egoscale
	go get gopkg.in/amz.v2/s3

.PHONY: devdeps
devdeps: deps
	go get -u github.com/golang/lint/golint

.PHONY: lint
lint:
	bin/golint github.com/exoscale/terraform-provider-exoscale

.PHONY: vet
vet:
	go tool vet src/github.com/exoscale/terraform-provider-exoscale

.PHONY: clean
clean:
	rm -rf bin/
