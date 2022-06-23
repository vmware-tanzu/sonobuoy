.PHONY: build test update_golden help
.DEFAULT_GOAL := help

build: ## Builds the Sonobuoy binary for your local OS
	# This is for local use; when building for other os/arch, use functions in ./scripts/build_funcs.sh directly.
	@echo "Building the sonobuoy binary..."
	go build
	@echo "Complete. Use sonobuoy in the current working directory (e.g. ./sonobuoy)"

test: ## Runs go tests
	@echo "Running unit tests..."
	source ./scripts/build_funcs.sh; unit_local

golden: ## Runs go generate to produce version info
	source ./scripts/build_funcs.sh; update_local
	@echo "Run 'git status' and 'git diff' to see updated golden files."

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'