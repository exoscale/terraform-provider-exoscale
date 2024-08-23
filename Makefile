GO_MK_REF := v2.0.3

# make go.mk a dependency for all targets
.EXTRA_PREREQS = go.mk

ifndef MAKE_RESTARTS
# This section will be processed the first time that make reads this file.

# This causes make to re-read the Makefile and all included
# makefiles after go.mk has been cloned.
Makefile:
	@touch Makefile
endif

.PHONY: go.mk
.ONESHELL:
go.mk:
	@if [ ! -d "go.mk" ]; then
		git clone https://github.com/exoscale/go.mk.git
	fi
	@cd go.mk
	@if ! git show-ref --quiet --verify "refs/heads/${GO_MK_REF}"; then
		git fetch
	fi
	@if ! git show-ref --quiet --verify "refs/tags/${GO_MK_REF}"; then
		git fetch --tags
	fi
	git checkout --quiet ${GO_MK_REF}

PACKAGE := github.com/exoscale/terraform-provider-exoscale
PROJECT_URL = https://$(PACKAGE)
GO_BUILD_EXTRA_ARGS = -v -trimpath
GOLANGCI_LINT_CONFIG = .golangci.yml
EXTRA_ARGS := -parallel=3 -count=1 -failfast

go.mk/init.mk:
include go.mk/init.mk

GO_LD_FLAGS := "-s -w -X $(PACKAGE)/version.Version=${VERSION} \
							-X $(PACKAGE)/version.Commit=${GIT_REVISION}"
GO_BIN_OUTPUT_NAME = terraform-provider-exoscale_v$(VERSION)

go.mk/public.mk:
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
