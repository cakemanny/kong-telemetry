FROM golang:1.21 as build

WORKDIR /go/src/goplugin
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/goplugin

CMD []
