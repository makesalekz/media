FROM golang:latest AS builder

COPY . /src
WORKDIR /src
ARG TOKEN

RUN git config --global url.https://gitlab-ci-token:${TOKEN}@gitlab.calendaria.team.insteadOf https://gitlab.calendaria.team && \
    export GOPRIVATE=gitlab.calendaria.team && \
    touch .env

RUN make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

ARG ENV
COPY --from=builder /src/bin /app
COPY --from=builder /src/configs/config.${ENV}.yaml /app/config.yaml

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
