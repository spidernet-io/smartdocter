
all: test example

test:
	go test . -race ./...

example:
	@echo "### Colorized (default) ###"
	go run ./levelsDemo
	@echo "### JSON: (redirected stderr) ###"
	go run ./levelsDemo 3>&1 1>&2 2>&3 | jq -c

line:
	@echo

# Suitable to make a screenshot with a bit of spaces around for updating color.png
screenshot: line example
	@echo


lint: .golangci.yml
	golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml


.PHONY: all test example screenshot line lint
