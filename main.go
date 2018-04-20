package main

import (
	"context"
	"fmt"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/ext"

	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

const (
	// Endpoint to send Zipkin spans to.
	zipkinHTTPEndpoint = "http://localhost:9411/api/v1/spans"

	// Debug mode.
	debug = false

	// same span can be set to true for RPC style spans (Zipkin V1) vs Node style (OpenTracing)
	sameSpan = true

	// make Tracer generate 128 bit traceID's for root spans.
	traceID128Bit = true
)

//ci
func main() {
}

type zipkinParams struct {
	serviceHostPort string
	serviceName     string
	serviceEndpoint string
}

func sendToZipkin(zp zipkinParams) error {
	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		return err
	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, debug, zp.serviceHostPort, zp.serviceName)

	// Create our tracer.
	tracer, err := zipkin.NewTracer(
		recorder,
		zipkin.ClientServerSameSpan(sameSpan),
		zipkin.TraceID128Bit(traceID128Bit),
	)
	if err != nil {
		return err
	}

	// Explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)

	// Create Root Span for duration of the interaction with svc1
	span := opentracing.StartSpan("Server")

	// Put root span in context so it will be used in our calls to the client.
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	makeFirstRequest(ctx, zp)
	makeSecondRequest(ctx, zp)

	// Finish our CLI span
	span.Finish()

	// Close collector to ensure spans are sent before exiting.
	collector.Close()
	return nil
}

func makeFirstRequest(ctx context.Context, zp zipkinParams) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s %s", "GET", "first"))
	defer span.Finish()
	req, err := http.NewRequest(http.MethodGet, zp.serviceHostPort+"/first", nil)
	if err != nil {
		return err
	}
	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	span.SetTag(string(ext.SpanKind), ext.SpanKindRPCClient)
	span.SetTag(string(ext.HTTPMethod), req.Method)
	span.SetTag(string(ext.HTTPUrl), req.URL.String())
	span.SetTag(string(ext.PeerService), zp.serviceName)

	if resp != nil {
		span.SetTag(string(ext.HTTPStatusCode), resp.StatusCode)
	}
	return nil
}

func makeSecondRequest(ctx context.Context, zp zipkinParams) error {
	span, ctx := opentracing.StartSpanFromContext(ctx, fmt.Sprintf("%s %s", "GET", "second"))
	defer span.Finish()
	req, err := http.NewRequest(http.MethodGet, zp.serviceHostPort+"/second", nil)
	if err != nil {
		return err
	}
	httpClient := http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	span.SetTag(string(ext.SpanKind), ext.SpanKindRPCClient)
	span.SetTag(string(ext.HTTPMethod), req.Method)
	span.SetTag(string(ext.HTTPUrl), req.URL.String())
	span.SetTag(string(ext.PeerService), zp.serviceName)

	if resp != nil {
		span.SetTag(string(ext.HTTPStatusCode), resp.StatusCode)
	}
	return nil
}
