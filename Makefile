# Every compiler and package manager runs in a Linux container.
NODE ?= node
CONTAINER := $(NODE) scripts/container.mjs
all: build

doctor:
	bash scripts/doctor.sh

build:
	$(CONTAINER) build

release:
	$(CONTAINER) release

build-api: build

run:
	pwsh -NoProfile -File scripts/start.ps1

dev: run

stop:
	pwsh -NoProfile -File scripts/start.ps1 -Stop

test:
	$(CONTAINER) test

lint:
	$(CONTAINER) lint

cross:
	$(CONTAINER) vet

ui:
	$(CONTAINER) ui

ui-check:
	$(CONTAINER) ui-check

verify:
	$(CONTAINER) verify

schema:
	$(CONTAINER) schema

schema-write:
	$(CONTAINER) schema-write

bench:
	$(CONTAINER) bench

ci: verify lint test cross ui-check schema

help:
	@echo "build/release: Linux application image; run/dev/stop: persistent Windows launcher"
	@echo "test/lint/cross/ui/ui-check/verify/schema/bench/ci: Linux container checks"

.PHONY: all doctor build release build-api run dev stop test lint cross ui ui-check verify schema schema-write bench ci help
