PACKAGE := github.com/exoscale/terraform-provider-exoscale
PROJECT_URL = https://$(PACKAGE)
GO_BUILD_EXTRA_ARGS = -v -trimpath
GOLANGCI_LINT_CONFIG = .golangci.yml
EXTRA_ARGS := -parallel=3 -count=1 -failfast

include go.mk/init.mk

GO_LD_FLAGS := "-s -w -X $(PACKAGE)/version.Version=${VERSION} \
							-X $(PACKAGE)/version.Commit=${GIT_REVISION}"
GO_BIN_OUTPUT_NAME = terraform-provider-exoscale_v$(VERSION)

include go.mk/public.mk

.PHONY: test-acc test-verbose test
test: GO_TEST_EXTRA_ARGS=${EXTRA_ARGS}
test-verbose: GO_TEST_EXTRA_ARGS+=$(EXTRA_ARGS)
test-acc: GO_TEST_EXTRA_ARGS=-v $(EXTRA_ARGS)
test-acc: ## Runs acceptance tests (requires valid Exoscale API credentials)
	TF_ACC=1 $(GO) test			\
		-race                   \
		-timeout=90m            \
		-tags=testacc           \
		$(GO_TEST_EXTRA_ARGS)   \
		$(GO_TEST_PKGS)
