VERSION=0.9.3_x-$(shell git rev-parse --short HEAD)

GOOS?=linux
GOARCH?=amd64

PROVIDER=terraform-provider-exoscale
DEST=bin
S=_v$(VERSION)
ifeq ($(GOOS),windows)
	SUFFIX=$(S).exe
endif
ifeq ($(GOOS),darwin)
	SUFFIX=$(S)_$(GOOS)-$(GOARCH)
endif
ifeq ($(GOOS),linux)
	SUFFIX=$(S)
endif

BIN= $(DEST)/$(PROVIDER)$(SUFFIX)

GOPATH := $(PWD)
export GOPATH

.PHONY: all
all: deps build

.PHONY: build
build: $(BIN)

.PHONY: $(BIN)
$(BIN):
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s" \
		-o $@ \
		github.com/exoscale/terraform-provider-exoscale

.PHONY: deps
deps:
	go get github.com/hashicorp/terraform
	go get github.com/exoscale/egoscale

.PHONY: devdeps
devdeps: deps
	go get -u github.com/golang/lint/golint

.PHONY: lint
lint:
	bin/golint github.com/exoscale/terraform-provider-exoscale

.PHONY: vet
vet:
	go tool vet src/github.com/exoscale/terraform-provider-exoscale

.PHONY: release
release: $(BIN)
	$(foreach bin,$^,\
		rm -f $(bin).asc;\
		gpg -a -u ops@exoscale.ch --output $(bin).asc --detach-sign $(bin);)

.PHONY: clean
clean:
	rm -rf $(DEST)
