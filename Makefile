BINARY_NAME=leafwiki
CMD_DIR=./cmd/leafwiki
VERSION ?= $(shell ./scripts/resolve-version.sh)
RELEASE_DIR := releases
DOCKER_BUILDER := Dockerfile.builder
UI_DIR := ui
HTTP_DIST := internal/http/dist
# The lint target reads its tool version from one shared file.
GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version)
EMBED_LDFLAGS := -X github.com/perber/wiki/internal/http.EmbedFrontend=true -X github.com/perber/wiki/internal/http.Environment=production
LDFLAGS := -X main.Version=$(VERSION) $(EMBED_LDFLAGS)

PLATFORMS := \
  linux/amd64 \
  linux/386 \
  linux/arm64 \
  windows/amd64 \
  darwin/amd64 \
  darwin/arm64

# Release targets other than linux, checked by `make cross` (and in CI) so a
# platform-specific compile error doesn't first show up at the release tag.
CROSS_PLATFORMS ?= windows/amd64 darwin/arm64

all: build

# Build the Vite SPA into internal/http/dist for go:embed.
ui:
	@echo "Building frontend..."
	cd $(UI_DIR) && npm ci --ignore-scripts && VITE_API_URL=/ npm run build
	rm -rf $(HTTP_DIST)
	mkdir -p $(HTTP_DIST)
	cp -R $(UI_DIR)/dist/. $(HTTP_DIST)/
	touch $(HTTP_DIST)/.gitkeep
	@test -f $(HTTP_DIST)/index.html || (echo "Frontend build missing $(HTTP_DIST)/index.html" && exit 1)

# Self-contained binary with embedded UI (matches release/Docker builds).
build: ui
	go build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)

# API-only binary for local Vite-proxied development (no embedded SPA).
build-api:
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY_NAME) $(CMD_DIR)

# npm dependencies for the frontend (ui-deps) and the tests/e2e tests. Install
# scripts stay off, like in CI and `make ui`.
setup: ui-deps
	cd tests/e2e && npm ci --ignore-scripts

# Only for running the E2E suites locally (make run-e2e-local); Chromium is
# the browser the suites use. On Linux, `npx playwright install-deps chromium`
# in tests/e2e/ adds missing system libraries (needs sudo).
setup-e2e: setup
	cd tests/e2e && npx playwright install chromium

# Toolchain versions against .nvmrc and go.mod; exits 1 on a missing tool or
# the wrong Node major.
doctor:
	@./scripts/doctor.sh

# Backend and Vite dev server together; Ctrl-C stops both. ARGS go to the
# backend.
dev:
	@./scripts/dev.sh $(ARGS)

run:
	go run -ldflags "-X main.Version=$(VERSION)" $(CMD_DIR)

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(RELEASE_DIR)

test:
	go test ./...

# Built with the toolchain from go.mod, so the result matches CI.
lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION) run

# Frontend checks run separately or together through ui-check.
ui-deps:
	cd $(UI_DIR) && npm ci --ignore-scripts

ui-lint: ui-deps
	cd $(UI_DIR) && npm run lint -- --max-warnings=0 && npm run format:check && npm run check:architecture

# `npm run build` is the type check (tsc -b).
ui-test: ui-deps
	cd $(UI_DIR) && npm test && npm run build

ui-check: ui-lint ui-test

# Run repository validation, Go checks, cross compilation and frontend checks.
ci: verify lint test cross ui-check

bench:
	go test -bench=. -benchmem -benchtime=3s ./internal/core/links/... ./internal/core/revision/...

# Build and vet (which also type-checks _test.go files) for each of
# CROSS_PLATFORMS, without cgo, like the release builds.
cross:
	@for platform in $(CROSS_PLATFORMS); do \
	  goos=$${platform%/*}; goarch=$${platform#*/}; \
	  echo "Building and vetting for $$goos/$$goarch..."; \
	  CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go build ./... || exit 1; \
	  CGO_ENABLED=0 GOOS=$$goos GOARCH=$$goarch go vet ./... || exit 1; \
	done

# Validate repository references and toolchain configuration.
verify:
	./scripts/check-private-references.sh
	./scripts/check-toolchain-versions.sh
	./scripts/check-toolchain-versions.test.sh

# Validate Conventional Commit subjects for BASE..HEAD.
BASE ?= origin/main
verify-commits:
	./scripts/check-commit-messages.sh "$(BASE)..HEAD"

# Build all platform targets
release: $(PLATFORMS)
	@echo "✅ All builds complete."

# Build for each platform using Docker
$(PLATFORMS):
	@mkdir -p $(RELEASE_DIR)
	@GOOS=$(word 1,$(subst /, ,$@)) ; \
	 GOARCH=$(word 2,$(subst /, ,$@)) ; \
	 EXT=$$( [ "$$GOOS" = "windows" ] && echo ".exe" || echo "" ) ; \
	 OUTPUT=$(BINARY_NAME)-$(VERSION)-$$GOOS-$$GOARCH$$EXT ; \
	 echo "📦 Building $$OUTPUT..." ; \
	 docker build -f $(DOCKER_BUILDER) \
		--build-arg GOOS=$$GOOS \
		--build-arg GOARCH=$$GOARCH \
		--build-arg APP_VERSION=$(VERSION) \
		--build-arg OUTPUT=$(BINARY_NAME) \
		-t leafwiki-builder-$$GOOS-$$GOARCH . ; \
	 ID=$$(docker create leafwiki-builder-$$GOOS-$$GOARCH) ; \
	 docker cp $$ID:/out/$(BINARY_NAME) $(RELEASE_DIR)/$$OUTPUT ; \
	 docker rm $$ID ; \
	 echo "✅ Binary done: $(RELEASE_DIR)/$$OUTPUT" ; \
	 (cd $(RELEASE_DIR) && sha256sum $$OUTPUT > $$OUTPUT.sha256) ; \
	 zip -j $(RELEASE_DIR)/$$OUTPUT.zip $(RELEASE_DIR)/$$OUTPUT ; \
	 tar -czf $(RELEASE_DIR)/$$OUTPUT.tar.gz -C $(RELEASE_DIR) $$OUTPUT ; \
	 echo "📦 Compressed: zip and tar.gz"

run-e2e:
	@echo "🚀 Starting end-to-end tests..."
	@./tests/e2e/run.sh

run-proxy-e2e:
	@echo "🔐 Starting proxy auth E2E tests..."
	@./tests/proxy/run.sh

run-e2e-local:
	@echo "⚡ Starting end-to-end tests (local fast path)..."
	@E2E_RUN_MODE=local ./tests/e2e/run.sh

# Skip the Vite build when dist/ is already up to date; useful when iterating
# on tests without changing the frontend. Pass GREP=<pattern> to filter tests,
# e.g.: make run-e2e-local-fast GREP="shoutout"
run-e2e-local-fast:
	@echo "⚡ Starting end-to-end tests (local, skip UI build)..."
	@E2E_RUN_MODE=local E2E_SKIP_UI_BUILD=1 ./tests/e2e/run.sh $(if $(GREP),--grep "$(GREP)",)

help:
	@echo "Available commands:"
	@echo "  make setup                – Install npm dependencies (frontend and tests/e2e)"
	@echo "  make setup-e2e            – make setup plus the Playwright browser for local E2E runs"
	@echo "  make doctor               – Show Go, Node, npm and mise versions; check them against go.mod and .nvmrc"
	@echo "  make dev                  – Run backend and Vite dev server together (http://localhost:5173)"
	@echo "                              Optional: ARGS=\"<backend flags>\""
	@echo "  make build                – Build self-contained binary with embedded UI (needs Node.js)"
	@echo "  make build-api            – Build API-only binary (use with Vite in Dev Setup)"
	@echo "  make ui                   – Build frontend into internal/http/dist for embedding"
	@echo "  make release              – Cross-compile binaries for all platforms (via Docker)"
	@echo "  make clean                – Clean all generated files"
	@echo "  make test                 – Run all Go tests"
	@echo "  make lint                 – Run golangci-lint (version from .golangci-lint-version)"
	@echo "  make ui-check             – Frontend install, lint, format check, tests and build"
	@echo "                              (ui-lint and ui-test run the two halves)"
	@echo "  make ci                   – verify, lint, test, cross and ui-check, like CI"
	@echo "  make verify               – Run small repository checks (private references, toolchain versions)"
	@echo "  make verify-commits       – Check commit subjects since BASE (default origin/main)"
	@echo "  make bench                – Run Go benchmarks for links and revision"
	@echo "  make cross                – Build and vet for windows/amd64 and darwin/arm64"
	@echo "                              Optional: CROSS_PLATFORMS=\"<goos>/<goarch> ...\""
	@echo "  make run-e2e              – Run end-to-end tests (using Docker)"
	@echo "  make run-proxy-e2e        – Run reverse-proxy auth E2E tests (Docker + nginx)"
	@echo "  make run-e2e-local        – Run end-to-end tests via local fast path"
	@echo "  make run-e2e-local-fast   – Run E2E tests locally, skip UI build (use when dist/ is current)"
	@echo "                              Optional: GREP=<pattern> to filter tests"
	@echo "  make run                  – Run development server (API only; use Vite for UI)"

.PHONY: all build build-api ui setup setup-e2e doctor dev run clean test lint ui-deps ui-lint ui-test ui-check ci bench cross verify verify-commits help run-e2e run-e2e-local run-e2e-local-fast run-proxy-e2e
