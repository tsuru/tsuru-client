# Copyright © 2023 tsuru-client authors
# Use of this source code is governed by a BSD-style
# license that can be found in the LICENSE file.

FROM golang:1.20-alpine AS builder

RUN apk add --update --no-cache \
        gcc \
        git \
        make \
        musl-dev \
    && :

WORKDIR /go/src/github.com/tsuru/tsuru-client
COPY go.mod go.sum ./
RUN go mod download

COPY . /go/src/github.com/tsuru/tsuru-client

ARG DOCKER_BUILD_TSURU_VERSION
RUN make build && echo 1


FROM alpine:3.18

RUN apk update && \
    apk add --no-cache ca-certificates && \
    rm /var/cache/apk/*

COPY --from=builder /go/src/github.com/tsuru/tsuru-client/build/tsuru /bin/tsuru

ENTRYPOINT ["tsuru"]
