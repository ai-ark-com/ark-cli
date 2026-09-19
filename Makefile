# Developer tasks. Every target has a Docker twin (docker-<target>) for
# machines without a Go toolchain; CI runs the same commands.

GO        ?= go
GOLANGCI  ?= golangci-lint
DOCKER    ?= docker
GO_IMAGE  ?= golang:1.26
LINT_IMAGE?= golangci/golangci-lint:v2.13.2
PWD_HOST  := $(CURDIR)

# Run a command inside the Go image with module and build caches persisted.
define in_docker
$(DOCKER) run --rm -v "$(PWD_HOST):/src" -w /src \
	-v ark-gomod:/go/pkg/mod -v ark-gocache:/root/.cache/go-build \
	-e GOFLAGS=-buildvcs=false $(1) $(2)
endef

.PHONY: build test lint fmt tidy vet check clean \
        docker-build docker-test docker-lint docker-fmt docker-tidy docker-check

build:
	$(GO) build -o ark$(EXE) .

test:
	$(GO) test -race -cover ./...

vet:
	$(GO) vet ./...

lint:
	$(GOLANGCI) run ./...

fmt:
	$(GOLANGCI) fmt ./...

tidy:
	$(GO) mod tidy

check: vet lint test

clean:
	rm -f ark ark.exe coverage.*

docker-build:
	$(call in_docker,$(GO_IMAGE),go build -o ark .)

docker-test:
	$(call in_docker,$(GO_IMAGE),go test -race -cover ./...)

docker-tidy:
	$(call in_docker,$(GO_IMAGE),go mod tidy)

docker-lint:
	$(call in_docker,$(LINT_IMAGE),golangci-lint run ./...)

docker-fmt:
	$(call in_docker,$(LINT_IMAGE),golangci-lint fmt ./...)

docker-check: docker-lint docker-test
