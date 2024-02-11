// Originally derived from https://github.com/Kong/go-plugins/blob/master/go-hello.go
/*
A "hello world" plugin in Go,
which reads a request header and sets a response header.
*/
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/Kong/go-pdk"
	"github.com/Kong/go-pdk/server"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
)

const (
	pluginName    = "goplugin"
	pluginVersion = "0.1.0"
)

type Config struct {
	Message string `json:"message"`
	Timeout int    `json:"timeout_ms"`

	// Hide context in the config
	baseContext context.Context
}

func mkNew(ctx context.Context) func() interface{} {
	New := func() interface{} {
		return &Config{
			baseContext: ctx,
		}
	}
	return New
}

func (conf Config) Access(kong *pdk.PDK) {

	ctx, span, err := startAccessSpan(conf.baseContext, kong)
	if err != nil {
		kong.Log.Err(err.Error())
		kong.Response.ExitStatus(500)
		return
	}
	defer span.End()

	_, childSpan := tracer.Start(ctx, "Get Host")
	host, err := kong.Request.GetHeader("Host")
	childSpan.End()
	childSpan = nil
	if err != nil {
		kong.Log.Err(err.Error())
	}
	message := conf.Message
	if message == "" {
		message = "hello"
	}
	_, childSpan = tracer.Start(ctx, "Set header")
	kong.Response.SetHeader("x-hello-from-go", fmt.Sprintf("Go says %s to %s", message, host))
	childSpan.End()
	childSpan = nil

	_, childSpan = tracer.Start(ctx, "Exit 200")
	kong.Response.ExitStatus(200)
	childSpan.End()
	childSpan = nil
	// todo: think of a nice way to do this always
	span.SetAttributes(semconv.HTTPResponseStatusCode(200))
}

func main() {
	if isDumpOrHelp() {
		server.StartServer(mkNew(context.Background()), pluginVersion, 0)
		return
	}

	if err := run(); err != nil {
		log.Fatalln(err)
	}
}

// We have to avoid OTEL setup .... this is is an awful hack
func isDumpOrHelp() bool {
	flagSet := flag.NewFlagSet(os.Args[0], flag.ContinueOnError)
	var (
		// Must be kept in sync it
		// https://github.com/Kong/go-pdk/blob/master/server/os.go#L14-L16
		_    = flagSet.String("kong-prefix", "/usr/local/kong", "")
		dump = flagSet.Bool("dump", false, "")
		help = flagSet.Bool("help", false, "")
	)
	err := flagSet.Parse(os.Args[1:])
	if err != nil || *dump || *help {
		return true
	}
	return false
}

func run() (err error) {
	// Handle SIGINT (CTRL+C) gracefully.
	ctx, stop := signal.NotifyContext(
		context.Background(), os.Interrupt, syscall.SIGTERM,
	)
	defer stop()

	// Set up OpenTelemetry.
	otelShutdown, err := setupOTelSDK(ctx)
	if err != nil {
		return
	}
	// Handle shutdown properly so nothing leaks.
	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Start Plugin server.
	srvErr := make(chan error, 1)
	go func() {
		srvErr <- server.StartServer(mkNew(ctx), pluginVersion, 0)
	}()

	// Wait for interruption.
	select {
	case err = <-srvErr:
		// Error when starting Plugin server.
		return
	case <-ctx.Done():
		// Wait for first CTRL+C.
		// Stop receiving signal notifications as soon as possible.
		stop()
	}
	return
}
