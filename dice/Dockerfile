# Start by building the application.
FROM golang:1.21 as build

RUN go env -w GOMODCACHE=/root/.cache/go-build

WORKDIR /go/src/app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    go mod download && go mod verify

COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build\
    CGO_ENABLED=0 go build -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian12
COPY --from=build --link /go/bin/app /
CMD ["/app"]

ENV OTEL_RESOURCE_ATTRIBUTES=service.name=dice,service.version=0.1.0
