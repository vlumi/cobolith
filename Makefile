.DEFAULT_GOAL := help

MARKDOWNLINT := npx --yes markdownlint-cli2@0.23.3

.PHONY: help
help:  ## List the available commands
	@awk 'BEGIN {FS = ":.*## "} /^[a-zA-Z0-9_-]+:.*## / {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: lint-md
lint-md:  ## Lint the Markdown files
	@$(MARKDOWNLINT) "**/*.md" "!CLAUDE.md"

.PHONY: lint-go
lint-go:  ## Lint the Go code and check its formatting
	golangci-lint run ./...

.PHONY: test
test:  ## Run the tests
	go test ./...

.PHONY: conformance
conformance:  ## Read NIST's COBOL-85 test suite, fetching it first, and rewrite its report
	@scripts/fetch-ccvs85.sh
	go run ./conformance -suite .cache/ccvs85/newcob.val -out conformance/README.md

.PHONY: ci
ci: lint-md lint-go test  ## Full local CI
