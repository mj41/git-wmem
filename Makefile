# Git-Wmem Tools Makefile

.PHONY: build clean test test-verbose test-unit test-init test-commit test-log test-workflow test-validations test-data test-advanced help

# Build all tools
build: bin/git-wmem bin/git-wmem-init bin/git-wmem-commit bin/git-wmem-log

bin/git-wmem: cmd/git-wmem/main.go internal/*.go cmd/git-wmem/git-wmem.md cmd/git-wmem/help.txt
	go build -ldflags="-X main.GitSHA=$(shell git rev-parse --short HEAD)" -o bin/git-wmem ./cmd/git-wmem

bin/git-wmem-init: cmd/git-wmem-init/main.go internal/*.go
	go build -o bin/git-wmem-init ./cmd/git-wmem-init

bin/git-wmem-commit: cmd/git-wmem-commit/main.go internal/*.go
	go build -o bin/git-wmem-commit ./cmd/git-wmem-commit

bin/git-wmem-log: cmd/git-wmem-log/main.go internal/*.go
	go build -o bin/git-wmem-log ./cmd/git-wmem-log

# Clean build artifacts
clean:
	rm -rf bin/

# Run tests with built tools in PATH
test: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -timeout 10m

test-verbose: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -timeout 10m

# Run unit tests
test-unit: build
	export PATH=$(PWD)/bin:$$PATH && cd tests && go test -v -timeout 5m

# Test individual components
test-init: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestGitWmemInit -timeout 5m

test-commit: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestGitWmemCommit -timeout 5m

test-log: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestGitWmemLog -timeout 5m

test-workflow: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestBasicDevelopmentWorkflow -timeout 5m

test-validations: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestValidations -timeout 5m

test-data: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestDataStructures -timeout 5m

test-advanced: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestWmemMerge -timeout 5m
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -v -run TestCommitWorkdir -timeout 5m

# Test with coverage
test-coverage: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -cover -timeout 10m

test-coverage-html: build
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go test -coverprofile=coverage.out -timeout 10m
	export PATH=$(PWD)/bin:$$PATH && cd tests_e2e && go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: tests_e2e/coverage.html"

# Development helpers
deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

# Help target
help:
	@echo "Git-Wmem Tools - Development Commands"
	@echo ""
	@echo "Available targets:"
	@echo "  build          Build all git-wmem tools"
	@echo "  clean          Remove built binaries"
	@echo "  test           Run all e2e tests"
	@echo "  test-verbose   Run all e2e tests with verbose output"
	@echo "  test-unit      Run unit tests"
	@echo "  test-init      Run git-wmem-init tests"
	@echo "  test-commit    Run git-wmem-commit tests"
	@echo "  test-log       Run git-wmem-log tests"
	@echo "  test-workflow  Run complete workflow tests"
	@echo "  test-validations Run validation tests"
	@echo "  test-data      Run data structure tests"
	@echo "  test-advanced  Run advanced scenario tests"
	@echo "  test-coverage  Run tests with coverage report"
	@echo "  test-coverage-html Generate HTML coverage report"
	@echo "  deps           Download dependencies"
	@echo "  fmt            Format Go code"
	@echo "  vet            Run go vet"
	@echo "  help           Show this help message"
