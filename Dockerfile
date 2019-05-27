FROM golang:1.12-alpine as builder
# MAINTAINER hanchuanchuan <chuanchuanhan@gmail.com>

ENV TZ=Asia/Shanghai
ENV LANG="en_US.UTF-8"

RUN apk add --no-cache \
    wget \
    make \
    git \
    gcc \
    musl-dev

RUN wget -O /usr/local/bin/dumb-init https://github.com/Yelp/dumb-init/releases/download/v1.2.2/dumb-init_1.2.2_amd64 \
 && chmod +x /usr/local/bin/dumb-init

COPY bin/goInception /goInception

# wget percona.com/get/pt-online-schema-change
# COPY bin/pt-online-schema-change /pt-online-schema-change

COPY config/config.toml.default /etc/config.toml


# Executable image
FROM alpine

COPY --from=builder /goInception /goInception
COPY --from=builder /etc/config.toml /etc/config.toml
COPY --from=builder /usr/local/bin/dumb-init /usr/local/bin/dumb-init

# COPY --from=builder /pt-online-schema-change /usr/local/bin/pt-online-schema-change

WORKDIR /

EXPOSE 4000

ENV PERCONA_TOOLKIT_VERSION 3.0.4

RUN set -x \
  && apk add --update perl perl-dbi perl-dbd-mysql perl-io-socket-ssl perl-term-readkey make ca-certificates wget \
  && update-ca-certificates \
  && wget -O /tmp/percona-toolkit.tar.gz https://www.percona.com/downloads/percona-toolkit/${PERCONA_TOOLKIT_VERSION}/source/tarball/percona-toolkit-${PERCONA_TOOLKIT_VERSION}.tar.gz \
  && tar -xzvf /tmp/percona-toolkit.tar.gz -C /tmp \
  && cd /tmp/percona-toolkit-${PERCONA_TOOLKIT_VERSION} \
  && perl Makefile.PL \
  && make \
  && make test \
  && make install \
  && apk del make ca-certificates wget \
  && rm -rf /var/cache/apk/* /tmp/percona-toolkit*

ENTRYPOINT ["/usr/local/bin/dumb-init", "/goInception","--config=/etc/config.toml"]
