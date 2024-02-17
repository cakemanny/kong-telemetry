package main

import (
	"context"
	"net/http"
	"net/http/httptest"

	"testing"

	"goplugin/test"

	"github.com/Kong/go-pdk"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
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

func setupOTEL(t *testing.T, e sdktrace.SpanExporter) {
	ctx := context.TODO()

	tp := sdktrace.NewTracerProvider(sdktrace.WithSyncer(e))

	otel.SetTextMapPropagator(propagation.TraceContext{})
	otel.SetTracerProvider(tp)
	t.Cleanup(func() { _ = tp.Shutdown(ctx) })
}

func TestPlugin(t *testing.T) {
	chk := assert.New(t)
	env, err := test.New(t, test.Request{
		Method:  "GET",
		Url:     "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{"host": {"localhost"}},
	})
	chk.NoError(err)

	New := mkNew(context.Background())

	env.DoAccess(New())
	chk.Equal(200, env.ClientRes.Status)
	chk.Equal("Go says hello to localhost", env.ClientRes.Headers.Get("x-hello-from-go"))
}

func TestInstrumentation_NoParent(t *testing.T) {
	chk := assert.New(t)

	exporter := NewFakeExporter()
	setupOTEL(t, exporter)

	env, err := test.New(t, test.Request{
		Method:  "GET",
		Url:     "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{"host": {"localhost"}},
	})
	chk.NoError(err)

	New := mkNew(context.Background())

	env.DoAccess(New())

	if chk.Len(*exporter.spans, 4) {
		span0 := (*exporter.spans)[0]
		chk.Equal("Get Host", span0.Name())

		span1 := (*exporter.spans)[1]
		chk.Equal("Set header", span1.Name())

		span2 := (*exporter.spans)[2]
		chk.Equal("Exit 200", span2.Name())

		span3 := (*exporter.spans)[3]
		chk.Equal("GET /plugin", span3.Name())

		chk.False(span3.Parent().IsValid())
	}
}

type accessFunc func(ctx context.Context, kong *pdk.PDK)
type testConfig struct {
	baseContext context.Context
	access      accessFunc
}

func (c *testConfig) Access(kong *pdk.PDK) {
	c.access(c.baseContext, kong)
}
func mkTestNew(ctx context.Context, a accessFunc) func() interface{} {
	New := func() interface{} {
		return &testConfig{
			baseContext: ctx,
			access:      a,
		}
	}
	return New
}

func TestInstrumentation_WithParent(t *testing.T) {

	chk := assert.New(t)

	exporter := NewFakeExporter()
	setupOTEL(t, exporter)

	env, err := test.New(t, test.Request{
		Method: "GET",
		Url:    "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{
			"host":        {"localhost"},
			"traceparent": {"00-f68de45b0b36ac1c97c2a43166c9cb8f-9a94fd01ca53f63d-01"},
		},
	})
	chk.NoError(err)

	access := func(ctx context.Context, kong *pdk.PDK) {
		ctx, span, err := startAccessSpan(ctx, kong)
		if !chk.NoError(err) {
			return
		}
		defer span.End()

		_, childSpan := getTracer(span).Start(ctx, "Get Host")
		_, _ = kong.Request.GetHost()
		childSpan.End()
	}

	New := mkTestNew(context.Background(), access)

	env.DoAccess(New())

	if chk.GreaterOrEqual(len(*exporter.spans), 1, "spans non-empty") {
		finalSpan := (*exporter.spans)[len(*exporter.spans)-1]
		chk.Equal("GET /plugin", finalSpan.Name())

		chk.True(finalSpan.SpanContext().IsValid(), "SpanContext valid")
		chk.False(finalSpan.SpanContext().IsRemote(), "SpanContext remote")
		chk.True(finalSpan.SpanContext().IsSampled(), "SpanContext sampled")
		chk.True(finalSpan.Parent().IsValid(), "Parent valid")
		chk.True(finalSpan.Parent().IsRemote(), "Parent remote")
	}

	if chk.Equal(2, len(*exporter.spans), "has child span") {
		childSpan := (*exporter.spans)[0]
		chk.Equal("Get Host", childSpan.Name())

		chk.True(childSpan.SpanContext().IsValid(), "child valid")
		chk.False(childSpan.SpanContext().IsRemote(), "child remote")
		chk.True(childSpan.SpanContext().IsSampled(), "child sampled")
		chk.True(childSpan.Parent().IsValid(), "child parent valid")
		chk.False(childSpan.Parent().IsRemote(), "child parent remote")
		chk.Equal(trace.SpanKindInternal, childSpan.SpanKind())
	}
}

func TestInstrumentation_WithClientCall(t *testing.T) {
	chk := assert.New(t)

	exporter := NewFakeExporter()
	setupOTEL(t, exporter)

	env, err := test.New(t, test.Request{
		Method: "GET",
		Url:    "http://example.com/plugin?q=search&x=9",
		Headers: map[string][]string{
			"host":        {"localhost"},
			"traceparent": {"00-f68de45b0b36ac1c97c2a43166c9cb8f-9a94fd01ca53f63d-01"},
		},
	})
	chk.NoError(err)

	fakeUpstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tp := r.Header.Get("traceparent")
		chk.Contains(tp, "00-f68de45b0b36ac1c97c2a43166c9cb8f-")
		chk.NotContains(tp, "-9a94fd01ca53f63d-")
		w.Write([]byte("ok"))
	}))
	defer fakeUpstream.Close()

	access := func(ctx context.Context, kong *pdk.PDK) {
		ctx, span, err := startAccessSpan(ctx, kong)
		if !chk.NoError(err) {
			return
		}
		defer span.End()

		// This is almost about checking whether I am able to use
		// opentelemetry correctly than whether the instrumentation works...
		client := http.Client{
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		}

		req, err := http.NewRequestWithContext(ctx, "GET", fakeUpstream.URL+"/", nil)
		if err != nil {
			panic(err)
		}
		resp, err := client.Do(req)
		chk.NoError(err)
		defer resp.Body.Close()
		chk.Equal(200, resp.StatusCode)
	}

	New := mkTestNew(context.Background(), access)

	env.DoAccess(New())

	if chk.GreaterOrEqual(len(*exporter.spans), 1, "spans non-empty") {
		finalSpan := (*exporter.spans)[len(*exporter.spans)-1]
		chk.Equal("GET /plugin", finalSpan.Name())
		chk.True(finalSpan.SpanContext().IsValid(), "valid")
		chk.Equal(trace.SpanKindServer, finalSpan.SpanKind())
	}

	if chk.Equal(2, len(*exporter.spans), "has child span") {
		childSpan := (*exporter.spans)[0]
		chk.Equal("HTTP GET", childSpan.Name())

		chk.True(childSpan.SpanContext().IsValid(), "valid")
		chk.False(childSpan.SpanContext().IsRemote(), "remote")
		chk.True(childSpan.SpanContext().IsSampled(), "sampled")
		chk.True(childSpan.Parent().IsValid(), "parent valid")
		chk.False(childSpan.Parent().IsRemote(), "parent remote")
		chk.Equal(trace.SpanKindClient, childSpan.SpanKind())
	}
}
