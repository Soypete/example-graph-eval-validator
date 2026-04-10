ONTOLOGY_PARSER := $(HOME)/code/misc/ontology-parser
TTL_FILE := genre.ttl
EVAL_DIR := src/evals
ENDPOINT := http://pedrogpt:8080

.PHONY: validate eval build clean

## validate: Run the SKOS validator against genre.ttl using ontology-go
validate:
	@echo "==> Validating $(TTL_FILE)..."
	cd $(ONTOLOGY_PARSER) && go run ./cmd/validate/ $(CURDIR)/$(TTL_FILE)

## validate-errors: Show only errors (skip info/warnings)
validate-errors:
	@echo "==> Validating $(TTL_FILE) (errors only)..."
	cd $(ONTOLOGY_PARSER) && go run ./cmd/validate/ --errors-only $(CURDIR)/$(TTL_FILE)

## validate-json: Output validation results as JSON
validate-json:
	cd $(ONTOLOGY_PARSER) && go run ./cmd/validate/ --format json $(CURDIR)/$(TTL_FILE)

## build: Build the eval binary
build:
	cd $(EVAL_DIR) && go build -o evals .

## eval: Run the full eval suite (9 questions x 5 runs) against the LLM endpoint
eval: build
	@echo "==> Running eval against $(ENDPOINT)..."
	cd $(EVAL_DIR) && ./evals -ttl ../../$(TTL_FILE) -endpoint $(ENDPOINT)

## eval-quick: Run eval with a single question for smoke testing
eval-quick: build
	@echo "==> Quick smoke test..."
	cd $(EVAL_DIR) && ./evals -ttl ../../$(TTL_FILE) -endpoint $(ENDPOINT)

## clean: Remove build artifacts and log files
clean:
	rm -f $(EVAL_DIR)/evals
	rm -f $(EVAL_DIR)/eval_*.log

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //' | column -t -s ':'
