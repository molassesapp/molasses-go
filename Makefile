CUR_DIR = $(CURDIR)


all: check-style test build

## Build application
.PHONY: Build
build:
	go build molasses.go

## Test application
.PHONY: Test
test:
	go test -v .

## Runs golangci-lint with docker
.PHONY: check-style
check-style: golangci-lint
	@echo Checking for style guide compliance

golangci-lint:
	docker run --rm -v $(CUR_DIR):/app -w /app golangci/golangci-lint:v1.31.0 golangci-lint run ./...

# Help documentation à la https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
help:
	@cat Makefile | grep -v '\.PHONY' |  grep -v '\help:' | grep -B1 -E '^[a-zA-Z0-9_.-]+:.*' | sed -e "s/:.*//" | sed -e "s/^## //" |  grep -v '\-\-' | sed '1!G;h;$$!d' | awk 'NR%2{printf "\033[36m%-30s\033[0m",$$0;next;}1' | sort