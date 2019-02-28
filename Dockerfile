# Builder image
FROM golang:1.10.1-alpine as builder

RUN apk add --no-cache \
    make \
    git

COPY . /go/src/github.com/hanchuanchuan/goInception

WORKDIR /go/src/github.com/hanchuanchuan/goInception/

RUN make

# Executable image
FROM scratch

COPY --from=builder /go/src/github.com/hanchuanchuan/goInception/bin/goInception /goInception

WORKDIR /

EXPOSE 4000

ENTRYPOINT ["/goInception"]

