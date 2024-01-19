FROM golang:latest AS builder

COPY . /src
WORKDIR /src

RUN mkdir -p -m 0700 ~/.ssh && \
    ssh-keyscan gitlab.calendaria.team >> ~/.ssh/known_hosts && \
    git config --global url."ssh://git@gitlab.calendaria.team/".insteadOf https://gitlab.calendaria.team/

RUN --mount=type=ssh GOPRIVATE=gitlab.calendaria.team make build

FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
		ca-certificates  \
        netbase \
        ffmpeg \
        && rm -rf /var/lib/apt/lists/ \
        && apt-get autoremove -y && apt-get autoclean -y

COPY --from=builder /src/bin /app

WORKDIR /app

EXPOSE 8000
EXPOSE 9000
VOLUME /data/conf

CMD ["./media", "-conf", "/data/conf/config.yaml"]
