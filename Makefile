BINARY=prommerge
IMAGE=prommerge
VERSION := $(shell git describe --tags --always --dirty)
GIT_COMMIT := $(shell git rev-list -1 HEAD)

.DEFAULT_GOAL := build
.PHONY: help lint-prepare lint unittest clean install build run stop

version: ## Show version
	@echo $(VERSION) \(git commit: $(GIT_COMMIT)\)

build:
	go build -o server cmd/server/*.go

build-exporter:
	go build -o exporter cmd/exporter-server/*.go

run: build
	./server

run-exporter: build-exporter
	./exporter

lint: # Run linters
	golangci-lint run --color always ./...

test: ## Run only quick tests
	go test  ./...

bench:
	go test -bench=. -benchmem

