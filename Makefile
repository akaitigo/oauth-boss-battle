.PHONY: build test lint format check quality clean tidy

# Go build settings (backend)
GOFLAGS ?= -trimpath
LDFLAGS ?= -s -w

build:
	cd backend && go build $(GOFLAGS) -ldflags "$(LDFLAGS)" ./...
	cd frontend && npm run build

test:
	cd backend && go test -v -race -count=1 -coverprofile=coverage.out ./...
	cd frontend && npm test -- --passWithNoTests

lint:
	cd backend && golangci-lint run ./...
	cd frontend && npx oxlint .

format:
	cd backend && gofumpt -w . && goimports -w .
	cd frontend && npx biome format --write .

tidy:
	cd backend && go mod tidy

check: format tidy lint test build
	@echo "All checks passed."

# quality は自動化可能な品質チェック。check とは別に実行する。
quality:
	@echo "=== Quality Gate ==="
	@test -f LICENSE || { echo "ERROR: LICENSE missing. Fix: add MIT LICENSE file"; exit 1; }
	@! grep -rn "TODO\|FIXME\|HACK\|console\.log\|println\|print(" backend/cmd/ backend/internal/ frontend/src/ 2>/dev/null | grep -v "node_modules" || { echo "ERROR: debug output or TODO found. Fix: remove before ship"; exit 1; }
	@! grep -rn "password=\|secret=\|api_key=\|sk-\|ghp_" backend/cmd/ backend/internal/ frontend/src/ 2>/dev/null | grep -v '\$${' | grep -v "node_modules" || { echo "ERROR: hardcoded secrets. Fix: use env vars with no default"; exit 1; }
	@test ! -f PRD.md || ! grep -q "\[ \]" PRD.md || { echo "ERROR: unchecked acceptance criteria in PRD.md"; exit 1; }
	@test ! -f CLAUDE.md || [ $$(wc -l < CLAUDE.md) -le 50 ] || { echo "ERROR: CLAUDE.md is $$(wc -l < CLAUDE.md) lines (max 50). Fix: remove build details, use pointers only"; exit 1; }
	@echo "OK: automated quality checks passed"
	@echo "Manual checks required: README quickstart, demo GIF, input validation, ADR >=1"

clean:
	cd backend && go clean -cache -testcache && rm -f coverage.out
	cd frontend && rm -rf node_modules/.cache dist/
