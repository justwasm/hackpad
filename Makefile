SHELL := /usr/bin/env bash
GO_VERSION = 1.27
GOROOT =
TAG = go1.27.0-go4js.1
PATH := ${PWD}/cache/go/bin:${PWD}/cache/go/misc/wasm:${PATH}
GOOS = js
GOARCH = wasm
export
LINT_VERSION=1.52.2

.PHONY: serve
serve:
	go run ./server

.PHONY: lint-deps
lint-deps: go
	@if ! which golangci-lint >/dev/null || [[ "$$(golangci-lint version 2>&1)" != *${LINT_VERSION}* ]]; then \
		curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin v${LINT_VERSION}; \
	fi

.PHONY: lint
lint: lint-deps
	golangci-lint run

.PHONY: lint-fix
lint-fix: lint-deps
	golangci-lint run --fix

.PHONY: test-native
test-native:
	GOARCH= GOOS= go test \
		-race \
		-coverprofile=cover.out \
		./...

.PHONY: test-js
test-js: go
	go test \
		-coverprofile=cover_js.out \
		./...

.PHONY: test
test: test-native #test-js  # TODO restore when this is resolved: https://travis-ci.community/t/goos-js-goarch-wasm-go-run-fails-panic-newosproc-not-implemented/1651

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: go-static
go-static: tidy commands

server/public/wasm:
	mkdir -p server/public/wasm

.PHONY: clean
clean:
	rm -rf ./out ./server/public/wasm

cache:
	mkdir -p cache

.PHONY: commands
commands: server/public/wasm/wasm_exec.js server/public/wasm/main.wasm $(patsubst cmd/%,server/public/wasm/%.wasm,$(wildcard cmd/*))

.PHONY: go
go: cache/go${GO_VERSION}

cache/go${GO_VERSION}: cache
	if [[ ! -e cache/go${GO_VERSION} ]]; then \
		set -ex; \
		host=$$(go env GOHOSTOS); \
		arch=$$(go env GOHOSTARCH); \
		TMP=$$(mktemp -d); \
		trap 'rm -rf "$$TMP"' EXIT; \
		curl -sL "https://github.com/justwasm/go/releases/download/${TAG}/${TAG}.$${host}-$${arch}.min.tar.gz" | tar -xzC "$$TMP"; \
		mv "$$TMP/go" cache/go${GO_VERSION}; \
		rm -rf cache/go; \
		ln -sfn go${GO_VERSION} cache/go; \
	fi
	touch cache/go${GO_VERSION}
	touch cache/go.mod  # Makes it so linters will ignore this dir

server/public/wasm/%.wasm: server/public/wasm go
	go build -o $@ ./cmd/$*

server/public/wasm/main.wasm: server/public/wasm go
	go build -o server/public/wasm/main.wasm .

server/public/wasm/wasm_exec.js: go
	mkdir -p server/public/wasm/
	cp cache/go/lib/wasm/wasm_exec.js server/public/wasm/wasm_exec.js

.PHONY: node-static
node-static:
	@echo "No build step needed - frontend is buildless"

.PHONY: watch
watch:
	npx serve server/public -p 3000

.PHONY: watch-go
watch-go:
	nodemon --signal SIGINT -e go -d 2 -x 'make go-static || exit 1' & \
	npx serve server/public -p 3000

.PHONY: build
build: build-docker
	rm -rf ./out
	docker cp $$(docker create --rm hackpad):/usr/share/nginx/html ./out

.PHONY: build-docker
build-docker:
	docker build -t hackpad .

.PHONY: run-docker
run-docker: build-docker
	docker run -it --rm \
		--name hackpad \
		-p 8080:80 \
		hackpad:latest
