FROM golang:latest AS builder

COPY . /src
WORKDIR /src
ARG TOKEN

RUN mkdir -p -m 0700 ~/.ssh && \
    ssh-keyscan gitlab.calendaria.team >> ~/.ssh/known_hosts && \
    git config --global url.ssh://git@gitlab.calendaria.team.insteadOf https://gitlab.calendaria.team && \
    touch .env && \
    echo "machine gitlab.calendaria.team login gitlab-ci-token password ${TOKEN}" > ~/.netrc && \
    chmod 600 ~/.netrc && \
    go env -w GO111MODULE='on' GOPRIVATE='gitlab.calendaria.team'

RUN --mount=type=ssh,id=rsa make build

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
