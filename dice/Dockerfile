# Start by building the application.
FROM golang:1.21 as build

WORKDIR /go/src/app
COPY go.mod go.mod
COPY go.sum go.sum

RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /go/bin/app

# Now copy it into our base image.
FROM gcr.io/distroless/static-debian12
COPY --from=build /go/bin/app /
CMD ["/app"]

ENV OTEL_RESOURCE_ATTRIBUTES=service.name=dice,service.version=0.1.0
