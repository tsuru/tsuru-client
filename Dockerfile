FROM golang:1.16-alpine AS builder

RUN apk add --update --no-cache \
        gcc \
        git \
        make \
        musl-dev \
    && :

WORKDIR /go/src/github.com/tsuru/tsuru-client
COPY . /go/src/github.com/tsuru/tsuru-client

RUN ls -al . \
    && git describe --tags \
    && make build

FROM alpine:3.9

RUN apk update && \
    apk add --no-cache ca-certificates && \
    rm /var/cache/apk/*

COPY --from=builder /go/src/github.com/tsuru/tsuru-client/bin/tsuru /bin/tsuru

CMD ["tsuru"]
