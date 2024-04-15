BINARY=prommerge
IMAGE=prommerge
VERSION := $(shell git describe --tags --always --dirty)
GIT_COMMIT := $(shell git rev-list -1 HEAD)

.DEFAULT_GOAL := build
.PHONY: help lint-prepare lint unittest clean install build run stop

version: ## Show version
	@echo $(VERSION) \(git commit: $(GIT_COMMIT)\)

lint: # Run linters
	golangci-lint run --color always ./...

test: ## Run only quick tests
	go test  ./...

bench:
	go test -bench=. -benchmem

