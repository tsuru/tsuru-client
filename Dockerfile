FROM golang:1.26.11-alpine AS builder

RUN apk add --update --no-cache \
        gcc \
        git \
        make \
        musl-dev \
    && :

WORKDIR /go/src/github.com/tsuru/tsuru-client
COPY . /go/src/github.com/tsuru/tsuru-client

ARG TSURU_BUILD_VERSION
RUN make build && echo 1

FROM alpine:3.24.3

RUN apk add --no-cache ca-certificates

COPY --from=builder /go/src/github.com/tsuru/tsuru-client/bin/tsuru /bin/tsuru

CMD ["tsuru"]
