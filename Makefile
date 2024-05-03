include .env
export

GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell $(Git_Bash) -c "find internal -name *.proto")
	API_PROTO_FILES=$(shell $(Git_Bash) -c "find api -name *.proto")
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
	REGISTRY_IMAGE=busybox
endif

.PHONY: init
# init env
init:
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
	go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest
	go install github.com/google/wire/cmd/wire@latest
	go install github.com/golang/mock/mockgen@v1.6.0
	npm install widdershins -g

.PHONY: doc
doc:
	go run -mod=mod entgo.io/ent/cmd/ent describe ./ent/schema > ./doc/schema.md
	doc/sed.sh doc/schema.md
	widdershins openapi.yaml -o ./doc/openapi.md --l --code --omitHeader --summary --resolve

.PHONY: run
# run locally
run:	
	GOFLAGS='-mod=readonly' kratos run -w ./configs

.PHONY: db
# run docker db container
db:
	docker compose up -d db

.PHONY: start
# start docker container locally
start:
	docker compose build local-service && \
	docker compose up -d local-service

.PHONY: stop
# stop docker container locally
stop:
	docker compose down local-service

.PHONY: config
# generate internal proto
config:
	protoc --proto_path=./internal \
	       --proto_path=./third_party \
 	       --go_out=paths=source_relative:./internal \
	       $(INTERNAL_PROTO_FILES)

.PHONY: ent
# generate ent
ent:
	go generate ./ent

.PHONY: migrations
# generate migrations
migrations:
	atlas migrate diff init \
		--dir "file://ent/migrate/migrations" \
		--to "ent://ent/schema" \
		--dev-url "docker://postgres/15/test?search_path=public"

.PHONY: hash
# rehash migrations
hash:
	atlas migrate hash --dir "file://ent/migrate/migrations"

.PHONY: proto
# copy proto files from vendor to third_party/api
proto:
	go mod vendor;
	find vendor/gitlab.calendaria.team -name 'models.proto' -exec sh -c 'f="{}"; d="third_party/api/$$(dirname "$$f" | awk -F/ "{print \$$(NF-1)\"/\"\$$NF}")"; mkdir -p "$$d"; rsync -a "$$f" "$$d"' \;

.PHONY: api
# generate api proto files
api:
	protoc --proto_path=. \
		   --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
			--go-errors_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
# build executable file
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate ent & wire
generate:
	go mod tidy
	go get github.com/google/wire/cmd/wire@latest
	GOFLAGS='-mod=readonly' go generate ./...

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

.PHONY: lint
# run linter
lint:
	golangci-lint run -v

.PHONY: test
# run tests
test:
	go test -v -count=1 ./...

.PHONY: race
# run tests with race
race:
	go test -v -race -count=10 ./...

.PHONY: cover
# calculate coverage
cover:
	go test -short -count=1 -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
	rm coverage.out

.PHONY: mock
# generate mock - (example here)
mock:
	mockgen -source internal/data/teams.go -destination internal/data/mock/teams.go -package mock

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
