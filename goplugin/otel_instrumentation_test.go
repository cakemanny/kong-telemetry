package main_test

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"go.opentelemetry.io/otel/propagation"
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
		t.Log(fmt.Sprintf("spancontext: %q, %q", j, err))
	} else {
		j, err := sc.MarshalJSON()
		t.Log(fmt.Sprintf("invalid span context: %q, %q", j, err))
	}

	if newCtx == ctx {
		t.Fail()
	}
}
