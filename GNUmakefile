-include .env
export

default: fmt lint install generate

build:
	go build -v ./...

install: build
	go install -v ./...

lint:
	golangci-lint run

generate:
	cd tools; go generate ./...

fmt:
	gofmt -s -w -e .

test:
	go test -v -cover -timeout=120s -parallel=10 ./...

testacc:
	TF_ACC=1 go test -v -cover -timeout 120m ./...

verification_apply:
	terraform -chdir=./examples/verification plan
	terraform -chdir=./examples/verification apply

verification_destroy:
	terraform -chdir=./examples/verification destroy

.PHONY: fmt lint test testacc build install generate verification_apply verification_destroy
