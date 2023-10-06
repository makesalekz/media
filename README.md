# media

## Proto files

### Add a proto template

```bash
kratos proto add api/server/server.proto
```

### Generate the proto code

```bash
kratos proto client api/server/server.proto
```

### Generate the source code of service by proto file

```bash
kratos proto server api/server/server.proto -t internal/service

go generate ./...
```

## Generate other auxiliary files by Makefile

### Download and update dependencies

```bash
make init
```

### Generate API files (include: pb.go, http, grpc, validate, swagger) by proto file

```bash
make api
```

### Generate all files

```bash
make all
```

### Generate migrations

[Install Atlas](https://entgo.io/docs/versioned-migrations#generating-migrations)

```bash
make migrations
```

## Run

Add values to Consul for keys:

1. `app/media/AWS_REGION`
2. `app/media/AWS_BUCKET`

### Run debug

```bash
make run
```

### Build & Run

```bash
export AWS_ACCESS_KEY_ID={aws-key}
export AWS_SECRET_ACCESS_KEY={aws-secret}

go build -o ./bin/ ./...
./bin/media -conf ./configs
```

## Run in Docker

```bash
make start
```

To stop docker:

```bash
make stop
```
