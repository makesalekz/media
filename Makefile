GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
else
	INTERNAL_PROTO_FILES=$(shell find internal -name *.proto)
	API_PROTO_FILES=$(shell find api -name *.proto)
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
	npm install widdershins -g

.PHONY: run
# run
run:	
	set -a && source .env && set +a && \
	GOFLAGS='-mod=readonly' kratos run

.PHONY: db
# db
db:
	set -a && source .env && set +a && \
	export REGISTRY_IMAGE=busybox && \
	docker compose up -d

.PHONY: start
# start
start:
	set -a && source .env && set +a && \
	export REGISTRY_IMAGE=busybox && \
	docker compose build --ssh default=$$HOME/.ssh/id_rsa dev-service && \
	docker compose --profile=dev up -d dev-service

.PHONY: stop
# stop
stop:
	set -a && source .env && set +a && \
	export REGISTRY_IMAGE=busybox && \
	docker compose --profile=dev down dev-service

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

.PHONY: proto
# proto
proto:
	go mod vendor;
	find vendor/gitlab.calendaria.team -name 'models.proto' -exec sh -c 'f="{}"; d="third_party/api/$$(dirname "$$f" | awk -F/ "{print \$$(NF-1)\"/\"\$$NF}")"; mkdir -p "$$d"; rsync -a "$$f" "$$d"' \;

.PHONY: api
# generate api proto
api:
	go mod vendor;
	protoc --proto_path=. \
		   --proto_path=./third_party \
 	       --go_out=paths=source_relative:. \
 	       --go-http_out=paths=source_relative:. \
 	       --go-grpc_out=paths=source_relative:. \
			--go-errors_out=paths=source_relative:. \
	       --openapi_out=fq_schema_naming=true,default_response=false:. \
	       $(API_PROTO_FILES)

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: generate
# generate
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
