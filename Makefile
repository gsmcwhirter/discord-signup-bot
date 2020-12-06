BUILD_DATE := `date -u +%Y%m%d`
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo v0.0.1)
GIT_SHA := $(shell git rev-parse HEAD)

APP_NAME := trials-bot
DUMP_NAME := trials-dump
CLEANUP_NAME := trials-cleanup
PROJECT := github.com/gsmcwhirter/discord-signup-bot

SERVER := discordbot@evogames.org:~/eso-discord/
CONF_FILE := ./trials-bot-config.toml
SERVICE_FILE := ./eso-trials-bot.service
START_SCRIPT := ./start-bot.sh
INSTALLER := ./trials-bot-install.sh

GOPROXY ?= https://proxy.golang.org

# can specify V=1 on the line with `make` to get verbose output
V ?= 0
Q = $(if $(filter 1,$V),,@)

.DEFAULT_GOAL := help

build-debug: version generate
	$Q GOPROXY=$(GOPROXY) go build -v -ldflags "-X main.AppName=$(APP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(APP_NAME) -race $(PROJECT)/cmd/$(APP_NAME)
	$Q GOPROXY=$(GOPROXY) go build -v -ldflags "-X main.AppName=$(DUMP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(DUMP_NAME) -race $(PROJECT)/cmd/$(DUMP_NAME)
	$Q GOPROXY=$(GOPROXY) go build -v -ldflags "-X main.AppName=$(DUMP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(CLEANUP_NAME) -race $(PROJECT)/cmd/$(CLEANUP_NAME)

build-release: version generate
	$Q GOPROXY=$(GOPROXY) GOOS=linux go build -v -ldflags "-s -w -X main.AppName=$(APP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(APP_NAME) $(PROJECT)/cmd/$(APP_NAME)
	$Q GOPROXY=$(GOPROXY) GOOS=linux go build -v -ldflags "-s -w -X main.AppName=$(DUMP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(DUMP_NAME) $(PROJECT)/cmd/$(DUMP_NAME)
	$Q GOPROXY=$(GOPROXY) GOOS=linux go build -v -ldflags "-s -w -X main.AppName=$(DUMP_NAME) -X main.BuildVersion=$(VERSION) -X main.BuildSHA=$(GIT_SHA) -X main.BuildDate=$(BUILD_DATE)" -o bin/$(CLEANUP_NAME) $(PROJECT)/cmd/$(CLEANUP_NAME)

generate:  ## do a go generate
	$Q GOPROXY=$(GOPROXY) go generate ./...

build-release-bundles: build-release
	$Q gzip -k -f bin/$(APP_NAME)
	$Q cp bin/$(APP_NAME).gz bin/$(APP_NAME)-$(VERSION).gz
	$Q gzip -k -f bin/$(DUMP_NAME)
	$Q cp bin/$(DUMP_NAME).gz bin/$(DUMP_NAME)-$(VERSION).gz
	$Q gzip -k -f bin/$(CLEANUP_NAME)
	$Q cp bin/$(CLEANUP_NAME).gz bin/$(CLEANUP_NAME)-$(VERSION).gz

clean:  ## Remove compiled artifacts
	$Q rm bin/*

debug: generate test build-debug  ## Debug build: create a dev build (enable race detection, don't strip symbols)

release: generate test build-release-bundles  ## Release build: create a release build (disable race detection, strip symbols)

deps:  ## download dependencies
	$Q GOPROXY=$(GOPROXY) go mod download
	$Q GOPROXY=$(GOPROXY) go get github.com/golangci/golangci-lint/cmd/golangci-lint@v1.31.0
	$Q GOPROXY=$(GOPROXY) go get github.com/mailru/easyjson/easyjson
	$Q GOPROXY=$(GOPROXY) go get github.com/valyala/quicktemplate/qtc
	$Q GOPROXY=$(GOPROXY) go get golang.org/x/tools/cmd/stringer
	$Q GOPROXY=$(GOPROXY) go get golang.org/x/tools/cmd/goimports

test:  ## Run the tests
	$Q GOPROXY=$(GOPROXY) go test -cover ./...

version:  ## Print the version string and git sha that would be recorded if a release was built now
	$Q echo $(VERSION) $(GIT_SHA)

vet: deps generate ## run various linters and vetters
	$Q bash -c 'for d in $$(go list -f {{.Dir}} ./...); do gofmt -s -w $$d/*.go; done'
	$Q bash -c 'for d in $$(go list -f {{.Dir}} ./...); do goimports -w -local $(PROJECT) $$d/*.go; done'
	$Q golangci-lint run -E golint,gosimple,staticcheck ./...
	$Q golangci-lint run -E deadcode,depguard,errcheck,gocritic,gofmt,goimports,gosec,govet,ineffassign,nakedret,prealloc,structcheck,typecheck,unconvert,varcheck ./...

release-upload: release upload

setup: deps generate  ## attempt to get everything set up to do a build (deps and generate)

upload:
	$Q scp $(CONF_FILE) $(SERVICE_FILE) $(START_SCRIPT) $(INSTALLER) $(SERVER)
	$Q scp  ./bin/$(APP_NAME).gz ./bin/$(APP_NAME)-$(VERSION).gz $(SERVER)
	$Q scp ./bin/$(DUMP_NAME).gz ./bin/$(DUMP_NAME)-$(VERSION).gz $(SERVER)
	$Q scp ./bin/$(CLEANUP_NAME).gz ./bin/$(CLEANUP_NAME)-$(VERSION).gz $(SERVER)

help:  ## Show the help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' ./Makefile
