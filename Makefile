VERSION=0.9.6_x-$(shell git rev-parse --short HEAD)
#VERSION=0.9.6_x1

GOOS?=linux
GOARCH?=amd64

PROVIDER=terraform-provider-exoscale
PKG=github.com/exoscale/$(PROVIDER)

SRCS=main.go $(wildcard exoscale/*.go)

DEST=build
OSES=windows darwin linux
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
DEBUG_BIN = $(PROVIDER)_v$(VERSION)


GOPATH := $(CURDIR)/.gopath
export GOPATH
export PATH := $(PATH):$(GOPATH)/bin

.PHONY: default
default: $(DEBUG_BIN)

.PHONY: all
all: deps ($BIN)

.PHONY: build
build: deps $(BIN)

$(DEBUG_BIN): $(SRCS)
	(cd $(GOPATH)/src/$(PKG) && \
		go build \
			-o $@ \
			$<)

$(BIN): $(SRCS)
	(cd $(GOPATH)/src/$(PKG) && \
		CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -ldflags "-s" \
			-o $@ \
			$<)

.PHONY: deps
deps: $(GOPATH)/src/$(PKG)
	(cd $(GOPATH)/src/$(PKG) && dep ensure)

$(GOPATH)/src/$(PKG):
	mkdir -p $(GOPATH)
	go get -u github.com/golang/dep/cmd/dep
	mkdir -p $(shell dirname $(GOPATH)/src/$(PKG))
	ln -sf ../../../.. $(GOPATH)/src/$(PKG)

.PHONY: deps-update
deps-update: deps
	(cd $(GOPATH)/src/$(PKG) && dep ensure -update)

.PHONY: signature
signature: $(BIN).asc

$(BIN).asc: $(BIN)
	rm -f $(BIN).asc
	gpg -a -u ops@exoscale.ch --output $@ --detach-sign $<

.PHONY: release
release: deps
	$(foreach goos, $(OSES), \
		GOOS=$(goos) $(MAKE) signature;)

.PHONY: clean
clean:
	rm -f $(DEBUG_BIN)
	rm -rf $(DEST)
