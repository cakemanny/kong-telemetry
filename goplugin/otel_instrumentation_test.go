package main

import (
	"context"
	"net/http"

	"testing"

	"goplugin/test"

	pkgassert "github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestExtraction(t *testing.T) {

	headers := map[string][]string{
		"traceparent": {"00-f68de45b0b36ac1c97c2a43166c9cb8f-9a94fd01ca53f63d-01"},
	}
	carrier := http.Header{}
	for k, vs := range headers {
		for _, v := range vs {
			carrier.Add(k, v)
		}
	}

	propagator := propagation.TraceContext{}
	ctx := context.TODO()

	newCtx := propagator.Extract(ctx, propagation.HeaderCarrier(carrier))

	sc := trace.SpanContextFromContext(newCtx)
	if sc.IsValid() {
		j, err := sc.MarshalJSON()
		t.Logf("spancontext: %q, %q", j, err)
	} else {
		j, err := sc.MarshalJSON()
		t.Logf("invalid span context: %q, %q", j, err)
	}

	if newCtx == ctx {
		t.Fail()
	}
}

type fakeExporter struct {
	spans *[]sdktrace.ReadOnlySpan
}

func NewFakeExporter() *fakeExporter {
	spans := []sdktrace.ReadOnlySpan{}
	return &fakeExporter{&spans}
}

func (fe fakeExporter) ExportSpans(ctx context.Context, spans []sdktrace.ReadOnlySpan) error {
	*fe.spans = append(*fe.spans, spans...)
	return nil
}
func (fe fakeExporter) Shutdown(ctx context.Context) error {
	return nil
}

var _ sdktrace.SpanExporter = fakeExporter{}

func TestPlugin(t *testing.T) {
	assert := pkgassert.New(t)
	env, err := test.New(t, test.Request{
		Method:  "GET",
		Url:     "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{"host": {"localhost"}},
	})
	assert.NoError(err)

	New := mkNew(context.Background())

	env.DoAccess(New())
	assert.Equal(200, env.ClientRes.Status)
	assert.Equal("Go says hello to localhost", env.ClientRes.Headers.Get("x-hello-from-go"))
}

func TestInstrumentation_NoParent(t *testing.T) {
	assert := pkgassert.New(t)
	ctx := context.TODO()

	exporter := NewFakeExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(ctx) })

	env, err := test.New(t, test.Request{
		Method:  "GET",
		Url:     "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{"host": {"localhost"}},
	})
	assert.NoError(err)

	New := mkNew(context.Background())

	env.DoAccess(New())

	if assert.Len(*exporter.spans, 4) {
		span0 := (*exporter.spans)[0]
		assert.Equal("Get Host", span0.Name())

		span1 := (*exporter.spans)[1]
		assert.Equal("Set header", span1.Name())

		span2 := (*exporter.spans)[2]
		assert.Equal("Exit 200", span2.Name())

		span3 := (*exporter.spans)[3]
		assert.Equal("GET /plugin", span3.Name())

		assert.False(span3.Parent().IsValid())
	}
}

func TestInstrumentation_WithParent(t *testing.T) {
	assert := pkgassert.New(t)
	ctx := context.TODO()

	exporter := NewFakeExporter()
	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(exporter))

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(ctx) })

	env, err := test.New(t, test.Request{
		Method: "GET",
		Url:    "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{
			"host":        {"localhost"},
			"traceparent": {"00-f68de45b0b36ac1c97c2a43166c9cb8f-9a94fd01ca53f63d-01"},
		},
	})
	assert.NoError(err)

	New := mkNew(context.Background())

	env.DoAccess(New())

	if assert.Len(*exporter.spans, 4) {
		span3 := (*exporter.spans)[3]
		assert.Equal("GET /plugin", span3.Name())

		assert.True(span3.SpanContext().IsValid(), "SpanContext valid")
		assert.False(span3.SpanContext().IsRemote(), "SpanContext remote")
		assert.True(span3.SpanContext().IsSampled(), "SpanContext sampled")
		assert.True(span3.Parent().IsValid(), "Parent valid")
		assert.True(span3.Parent().IsRemote(), "SpanContext remote")
	}

}
