FROM tsuru/alpine-go:latest as builder
COPY . /go/src/github.com/tsuru/tsuru-client
WORKDIR /go/src/github.com/tsuru/tsuru-client
ENV CC=/usr/bin/gcc
ENV GOPATH=/go
RUN go build -i -v --ldflags '-linkmode external -extldflags "-static"' -o ./bin/tsuru ./tsuru

FROM alpine:3.4
RUN apk update && apk add --no-cache ca-certificates
COPY --from=builder /go/src/github.com/tsuru/tsuru-client/bin/tsuru /bin/tsuru
CMD ["tsuru"]
