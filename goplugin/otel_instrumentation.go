package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/Kong/go-pdk"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const ScopeName = "goplugin"

var (
	tracer = otel.Tracer(ScopeName)
)

func newTracer(tp trace.TracerProvider) trace.Tracer {
	return tp.Tracer(ScopeName)
}

// Inspiration ...
// "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

func startAccessSpan(octx context.Context, kong *pdk.PDK) (context.Context, trace.Span, error) {
	var ctx context.Context = octx
	// Maybe we could fire these all off as separate goroutines?
	opts := []trace.SpanStartOption{}

	var tracer trace.Tracer
	// Certain actions here seem to break the trace.
	// But when it doesn' break, it doesn't seem to do much at all.. :(
	pick := rand.Int()%2 == 0
	pick2 := rand.Int()%2 == 0
	if pick {
		headers, err := kong.Request.GetHeaders(-1)
		if err != nil {
			return octx, nil, err
		}
		ctx := otel.GetTextMapPropagator().Extract(
			octx, propagation.HeaderCarrier(normalizeHeaders(headers)))

		if pick2 {
			// We start a new root for our plugin
			opts = append(opts, trace.WithNewRoot())
			if s := trace.SpanContextFromContext(ctx); s.IsValid() && s.IsRemote() {
				opts = append(opts, trace.WithLinks(trace.Link{SpanContext: s}))
			}
		}

		if span := trace.SpanFromContext(octx); span.SpanContext().IsValid() {
			tracer = newTracer(span.TracerProvider())
		} else {
			tracer = newTracer(otel.GetTracerProvider())
		}
	}

	// Idea: span each kong access?
	method, err := kong.Request.GetMethod()
	if err != nil {
		return ctx, nil, err
	}

	path, err := kong.Request.GetPath()
	if err != nil {
		return ctx, nil, err
	}
	opts = append(opts, trace.WithAttributes(
		semconv.HTTPRequestMethodKey.String(method),
		semconv.URLPath(path),
	))

	ctx, span := tracer.Start(ctx, fmt.Sprintf("%s %s", method, path))

	span.SetAttributes(attribute.Bool("context.propagated", pick))
	span.SetAttributes(attribute.Bool("root.new", pick2))

	return ctx, span, nil
}

// If the headers aren't in normal form, they're not found
func normalizeHeaders(headers map[string][]string) http.Header {
	result := http.Header{}
	for k, vs := range headers {
		for _, v := range vs {
			result.Add(k, v)
		}
	}
	return result
}
