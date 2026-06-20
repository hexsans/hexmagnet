GO ?= go
GIT_TAG ?= $(shell git describe --tags --always --dirty)
LDFLAGS ?= -X github.com/hexsans/hexmagnet/internal/version.GitTag=$(GIT_TAG)

.PHONY: gen gen-go gen-schema gen-sqlc gen-gql-enums gen-gql gen-mockery gen-webui-graphql

gen: gen-go gen-schema gen-sqlc gen-gql-enums gen-gql gen-mockery gen-webui-graphql

gen-go:
	$(GO) generate ./...

gen-schema:
	cat database/migrations/*.up.sql > database/schema/schema.sql

gen-sqlc: gen-schema
	$(GO) run github.com/sqlc-dev/sqlc/cmd/sqlc generate

gen-gql-enums:
	$(GO) run ./internal/gql/enums/gen/genenums.go

gen-gql:
	$(GO) run github.com/99designs/gqlgen generate --config ./internal/gql/gqlgen.yml

gen-mockery:
	$(GO) run github.com/vektra/mockery/v2

gen-webui-graphql:
	cd webui && npm run codegen

.PHONY: lint lint-webui lint-golangci

lint: lint-webui

lint-webui:
	cd webui && npm run lint

lint-golangci:
	golangci-lint run --timeout=10m

.PHONY: test-go

test-go:
	$(GO) test -count=1 -timeout 120s -v ./internal/...

.PHONY: build build-go build-webui

build-webui:
	cd webui && npm run build

build-go:
	$(GO) build -ldflags "$(LDFLAGS)"

build-go-debug:
	$(GO) build -tags debug -ldflags "$(LDFLAGS)"

build: build-webui build-go
