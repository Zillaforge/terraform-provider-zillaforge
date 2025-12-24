-include .env
export

PARALLEL ?= 2

default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint cache clean
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

# Set PARALLEL to control maximum parallel tests (default: 2)
# Example: make testacc PARALLEL=4
# Use TESTARGS to pass additional `go test` flags, for example:
#   make testacc TESTARGS='-run=TestAccImagesDataSource' PARALLEL=1
TESTARGS ?=

testacc:
	@echo "Running acceptance tests with -parallel=$(PARALLEL)"; \
	TF_ACC=1 go test -coverprofile=coverage.out --count 1 -v -cover -timeout 120m -failfast \
	-parallel $(PARALLEL) \
	./... $(TESTARGS)

coverage: testacc
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

verification_apply:
	terraform -chdir=./examples/verification plan
	terraform -chdir=./examples/verification apply

verification_destroy:
	terraform -chdir=./examples/verification destroy

.PHONY: fmt lint test testacc build install generate verification_apply verification_destroy
