# Change these variables as necessary.
main_package_path = .
binary_name = infragraph

# Version injection — mirrors GoReleaser ldflags.
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0-dev")
GIT_COMMIT ?= $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
GIT_TREE_STATE ?= $(shell test -z "$(shell git status --porcelain 2>/dev/null)" && echo "clean" || echo "dirty")
BUILD_DATE ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS = -s -w \
	-X github.com/timkrebs/infragraph/version.Version=$(VERSION) \
	-X github.com/timkrebs/infragraph/version.GitCommit=$(GIT_COMMIT) \
	-X github.com/timkrebs/infragraph/version.GitTreeState=$(GIT_TREE_STATE) \
	-X github.com/timkrebs/infragraph/version.BuildDate=$(BUILD_DATE)

# ==================================================================================== #
# HELPERS
# ==================================================================================== #

## help: print this help message
.PHONY: help
help:
	@echo 'Usage:'
	@sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' |  sed -e 's/^/ /'

.PHONY: confirm
confirm:
	@echo -n 'Are you sure? [y/N] ' && read ans && [ $${ans:-N} = y ]

.PHONY: no-dirty
no-dirty:
	@test -z "$(shell git status --porcelain)"


# ==================================================================================== #
# QUALITY CONTROL
# ==================================================================================== #

## audit: run quality control checks
.PHONY: audit
audit: test
	go mod tidy -diff
	go mod verify
	test -z "$(shell gofmt -l .)"
	go vet ./...
	go run honnef.co/go/tools/cmd/staticcheck@latest -checks=all,-ST1000,-U1000 ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

## test: run all tests
.PHONY: test
test:
	go test -v -race -buildvcs ./...

## test/cover: run all tests and display coverage
.PHONY: test/cover
test/cover:
	go test -v -race -buildvcs -coverprofile=/tmp/coverage.out ./...
	go tool cover -html=/tmp/coverage.out

## upgradeable: list direct dependencies that have upgrades available
.PHONY: upgradeable
upgradeable:
	@go run github.com/oligot/go-mod-upgrade@latest


# ==================================================================================== #
# DEVELOPMENT
# ==================================================================================== #

## tidy: tidy modfiles and modernize and format .go files
.PHONY: tidy
tidy:
	go mod tidy -v
	go fix ./...
	go fmt ./...

## build: build the application
.PHONY: build
build:
	go build -ldflags="$(LDFLAGS)" -o=bin/${binary_name} ${main_package_path}

## run: run the application
.PHONY: run
run: build
	./bin/${binary_name}

## clean: remove build artifacts
.PHONY: clean
clean:
	rm -rf bin/ dist/

## version: print the version that will be baked into the binary
.PHONY: version
version:
	@echo "$(VERSION) ($(GIT_COMMIT), $(GIT_TREE_STATE), $(BUILD_DATE))"


# ==================================================================================== #
# RELEASE
# ==================================================================================== #

## release/dry-run: run GoReleaser in snapshot mode (no publish)
.PHONY: release/dry-run
release/dry-run:
	goreleaser release --snapshot --clean

## release/local: build release artifacts locally without publishing
.PHONY: release/local
release/local:
	goreleaser build --snapshot --clean

## release/check: validate .goreleaser.yml
.PHONY: release/check
release/check:
	goreleaser check

# ==================================================================================== #
# UI
# ==================================================================================== #

## ui/install: install UI dependencies
.PHONY: ui/install
ui/install:
	cd ui && npm install

## ui/dev: start UI dev server with hot-reload
.PHONY: ui/dev
ui/dev:
	cd ui && npm run dev

## ui/build: build the production UI bundle
.PHONY: ui/build
ui/build:
	cd ui && npm run build

## build/full: build UI then Go binary (single binary with embedded UI)
.PHONY: build/full
build/full: ui/build build

## local-dev: build everything, run tests, and start the server with dev config
.PHONY: local-dev
local-dev: ui/build build test
	@echo "==> Starting InfraGraph server (dev config)..."
	./bin/${binary_name} server start --config example/dev.hcl
