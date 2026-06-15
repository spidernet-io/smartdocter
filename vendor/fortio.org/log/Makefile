
GO_BIN?=GOTOOLCHAIN=local go

all: info test lint size-check coverage example

info:
	@echo "### Go (using GO_BIN=\"$(GO_BIN)\") version:"
	$(GO_BIN) version

test:
	$(GO_BIN) test -race ./...
	$(GO_BIN) test -tags no_json ./...
	$(GO_BIN) test -tags no_http ./...

local-coverage: coverage
	$(GO_BIN) test -coverprofile=coverage.out ./...
	$(GO_BIN) tool cover -html=coverage.out

coverage:
	$(GO_BIN) test -coverprofile=coverage1.out ./...
	$(GO_BIN) test -tags no_net -coverprofile=coverage2.out ./...
	$(GO_BIN) test -tags no_json -coverprofile=coverage3.out ./...
	$(GO_BIN) test -tags no_http,no_json -coverprofile=coverage4.out ./...
	# cat coverage*.out > coverage.out
	$(GO_BIN) install github.com/wadey/gocovmerge@b5bfa59ec0adc420475f97f89b58045c721d761c
	gocovmerge coverage?.out > coverage.out

example:
	@echo "### Colorized (default) ###"
	$(GO_BIN) run ./levelsDemo
	@echo "### JSON: (redirected stderr) ###"
	$(GO_BIN) run ./levelsDemo 3>&1 1>&2 2>&3 | jq -c

line:
	@echo

# Suitable to make a screenshot with a bit of spaces around for updating color.png
screenshot: line example
	@echo

size-check:
	@echo "### Size of the binary:"
	CGO_ENABLED=0 $(GO_BIN) build -ldflags="-w -s" -trimpath -o ./fullsize ./levelsDemo
	ls -lh ./fullsize
	CGO_ENABLED=0 $(GO_BIN) build -tags no_net -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	CGO_ENABLED=0 $(GO_BIN) build -tags no_http,no_json -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	gsa ./smallsize # go install github.com/Zxilly/go-size-analyzer/cmd/gsa@master


lint: .golangci.yml
	golangci-lint run
	golangci-lint run --build-tags no_json

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml


.PHONY: all info test lint size-check local-coverage example screenshot line coverage
