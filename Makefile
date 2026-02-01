all: build test

# Build the application
build:
	@echo -- Building
	@go build -o refract ./cmd

# Run the application
run:
	@echo -- Running
	@go run ./cmd $(filter-out run,$(MAKECMDGOALS))

# Test the application
test:
	@echo -- Testing
	@go test ./... -v

# Clean the binary
clean:
	@echo -- Cleaning
	@rm -f refract

.PHONY: all build run test clean

%:
	@: