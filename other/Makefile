# 构建、测试和依赖安装由共享 Node 入口交给 Linux 容器，宿主负责调用工具。
# 根目录按此 Makefile 的实际位置计算，不依赖调用命令时所在目录。
NODE ?= node
ROOT := $(abspath $(dir $(lastword $(MAKEFILE_LIST)))/..)
CONTAINER := $(NODE) $(ROOT)/other/scripts/container.mjs
all: build

doctor:
	bash $(ROOT)/other/scripts/doctor.sh

build:
	$(CONTAINER) build

release:
	$(CONTAINER) release

# build-api 复用完整应用构建，后端二进制仍包含同次构建得到的前端页面。
build-api: build

run:
	pwsh -NoProfile -File $(ROOT)/other/scripts/start.ps1

dev: run

# 停止与启动使用同一 PowerShell 入口，由它校验保存的容器和进程归属。
stop:
	pwsh -NoProfile -File $(ROOT)/other/scripts/start.ps1 -Stop

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

# ci 汇集已有检查目标；make 是否并行执行由调用参数控制，本文件不强制并行。
ci: verify lint test cross ui-check schema

help:
	@echo "build/release: Linux application image; run/dev/stop: persistent Windows launcher"
	@echo "test/lint/cross/ui/ui-check/verify/schema/bench/ci: Linux container checks"

.PHONY: all doctor build release build-api run dev stop test lint cross ui ui-check verify schema schema-write bench ci help
