# Builder image
FROM golang:1.12-alpine as builder

RUN apk add --no-cache \
    wget \
    make \
    git \
    gcc \
    musl-dev

RUN wget -O /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 \
 && chmod +x /usr/local/bin/dumb-init

COPY bin/goInception /goInception
COPY config/config.toml.example /etc/config.toml


WORKDIR /

# Executable image
FROM alpine

COPY --from=builder /goInception /goInception
COPY --from=builder /etc/config.toml /etc/config.toml
COPY --from=builder /usr/local/bin/dumb-init /usr/local/bin/dumb-init

WORKDIR /

EXPOSE 4000

ENTRYPOINT ["/usr/local/bin/dumb-init", "/goInception","--config=/etc/config.toml"]
