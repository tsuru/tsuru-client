FROM golang:1.10-alpine AS builder

COPY . /go/src/github.com/tsuru/tsuru-client

WORKDIR /go/src/github.com/tsuru/tsuru-client

RUN apk add --update gcc git make musl-dev && \
    make build

FROM alpine:3.8

COPY --from=builder /go/src/github.com/tsuru/tsuru-client/bin/tsuru /bin/tsuru

RUN apk update && \
    apk add --no-cache ca-certificates && \
    rm /var/cache/apk/*

CMD ["tsuru"]
