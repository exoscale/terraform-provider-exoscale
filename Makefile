# Provide a simple mechanism to ensure a proper build

PROVIDER=terraform-provider-exoscale
DEST=bin
BIN= $(DEST)/$(PROVIDER)
BINS=\
		$(BIN)        \
		$(BIN)-static

GOPATH := $(PWD)
export GOPATH

.PHONY: all
all: deps build

.PHONY: build
build: $(BIN)

.PHONY: $(BIN)
$(BIN):
	go install github.com/exoscale/terraform-provider-exoscale/

.PHONY: $(BIN)-static
$(BIN)-static:
	env CGO_ENABLED=0 GOOS=linux go build -ldflags "-s" \
		-o $@ \
		github.com/exoscale/terraform-provider-exoscale

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

.PHONY: release
release: $(BINS)
	$(foreach bin,$^,\
		rm -f $(bin).asc;\
		gpg -a -u ops@exoscale.ch --output $(bin).asc --detach-sign $(bin);)

.PHONY: clean
clean:
	rm -rf $(DEST)
