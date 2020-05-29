GOPATH?=$(shell pwd)/build
VERSION:=$(shell git describe --tags `git rev-list --tags --max-count=1` | sed 's|^[^0-9]*||')
SWEEP?=us-east-1,us-west-2
TEST?=./...
GOFMT_FILES?=$$(find . -name '*.go' |grep -v vendor)
GOTEST_OPTS=-v -count=1 -failfast -timeout 120m
PKG_NAME=exoscale
WEBSITE_REPO=github.com/hashicorp/terraform-website
GOLANGCI_LINT_VERSION=1.15.0

default: build

build: fmtcheck
	GOPATH=$(GOPATH) GO111MODULE=on go install

install: build
	@mkdir -p $(HOME)/.terraform.d/plugins
	@cp -v $(GOPATH)/bin/terraform-provider-exoscale $(HOME)/.terraform.d/plugins/terraform-provider-exoscale_v$(VERSION)
	@echo "Please run 'terraform init' to finalize provider/plugin installation"

sweep:
	@echo "WARNING: This will destroy infrastructure. Use only in development accounts."
	go test $(TEST) -sweep=$(SWEEP) $(SWEEPARGS)

test: fmtcheck
	go test $(GOTEST_OPTS) $(TESTARGS) $(TEST)

testacc: fmtcheck
	TF_ACC=1 go test $(GOTEST_OPTS) $(TESTARGS) $(TEST)

fmt:
	@echo "==> Fixing source code with gofmt..."
	gofmt -s -w $(GOFMT_FILES)

fmtcheck:
	@sh -c "'$(CURDIR)/scripts/gofmtcheck.sh'"

lint:
	@echo "==> Checking source code against linters..."
	@golangci-lint run ./$(PKG_NAME)

tools:
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v$(GOLANGCI_LINT_VERSION)

errcheck:
	@sh -c "'$(CURDIR)/scripts/errcheck.sh'"

vendor-status:
	@go mod verify

test-compile:
	@if [ "$(TEST)" = "./..." ]; then \
		echo "ERROR: Set TEST to a specific package. For example,"; \
		echo "  make test-compile TEST=./$(PKG_NAME)"; \
		exit 1; \
	fi
	go test $(GOTEST_OPTS) -c $(TESTARGS) $(TEST)

website:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PKG_NAME)

website-test:
ifeq (,$(wildcard $(GOPATH)/src/$(WEBSITE_REPO)))
	echo "$(WEBSITE_REPO) not found in your GOPATH (necessary for layouts and assets), get-ting..."
	git clone https://$(WEBSITE_REPO) $(GOPATH)/src/$(WEBSITE_REPO)
endif
	@$(MAKE) -C $(GOPATH)/src/$(WEBSITE_REPO) website-provider-test PROVIDER_PATH=$(shell pwd) PROVIDER_NAME=$(PKG_NAME)

.PHONY: build install sweep test testacc vet fmt fmtcheck errcheck vendor-status test-compile website website-test

