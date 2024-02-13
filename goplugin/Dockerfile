FROM golang:1.21 as build

WORKDIR /go/src/goplugin
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/goplugin

FROM debian:12-slim
COPY --from=build /go/bin/goplugin /go/bin/goplugin

RUN useradd -u 1000 kong

RUN mkdir -p /mnt/goplugins \
    && /go/bin/goplugin -dump > /mnt/goplugins/dump.json \
    && chown -R kong /mnt/goplugins

USER kong

CMD ["/go/bin/goplugin", "-instrument", "-kong-prefix", "/mnt/goplugins"]
