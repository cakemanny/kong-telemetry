# syntax = docker/dockerfile:1.4
FROM golang:1.21 as build

# This is the default, but let's pin it
RUN go env -w GOMODCACHE=/root/.cache/go-build

WORKDIR /go/src/goplugin

COPY --link go.mod go.sum ./

RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download -x && go mod verify

COPY --link . .

RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -o /go/bin/goplugin

CMD []
