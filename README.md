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

## Configuration

### Consul

```bash
app/media/AWS_REGION
app/media/AWS_BUCKET
```

### .env

```bash
AWS_ACCESS_KEY_ID={aws-key}
AWS_SECRET_ACCESS_KEY={aws-secret}
```

## Run

### Run locally

```bash
make run
```

### Run in Docker

```bash
make start
```

### Stop docker

```bash
make stop
```
