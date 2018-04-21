// https://github.com/nwjlyons/nethttpexample/blob/master/middleware/main.go
package main

import (
	"context"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"

	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

type zipkinParams struct {
	serviceHostPort string
	serviceName     string
}

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

func createZipkin(zp zipkinParams) (context.Context, error) {
	// Create our HTTP collector.
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	// Explicitly set our tracer to be the default tracer.
	opentracing.InitGlobalTracer(tracer)

	// Create Root Span for duration of the interaction with svc1
	span := opentracing.StartSpan("Server")
	// Put root span in context so it will be used in our calls to the client.
	ctx := opentracing.ContextWithSpan(context.Background(), span)

	// Finish our CLI span
	span.Finish()

	// Close collector to ensure spans are sent before exiting.
	// collector.Close()
	return ctx, nil
}

type fooHandler struct {
	ctx context.Context
}

func (f fooHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fooSpan, _ := opentracing.StartSpanFromContext(f.ctx, "fooHandler")
	defer fooSpan.Finish()
	fooSpan.SetOperationName("GET fooEndpoint")
	fooSpan.SetTag("tag", "1")
}

type barHandler struct {
	ctx context.Context
}

func (b barHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	barSpan, _ := opentracing.StartSpanFromContext(b.ctx, "barHandler")
	defer barSpan.Finish()
	barSpan.SetOperationName("GET barEndpoint")
	barSpan.SetTag("tag", "2")
}

// string is the URL path and http.Handler is any type that has a ServeHTTP method.
type multiplexer map[string]http.Handler

func (m multiplexer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if handler, ok := m[r.RequestURI]; ok {
		handler.ServeHTTP(w, r)
	} else {
		w.WriteHeader(http.StatusNotFound)
	}
}

func endpoints(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)
	})
}

func main() {
	ctx, err := createZipkin(zipkinParams{
		serviceHostPort: "127.0.0.1",
		serviceName:     "The test service",
	})
	if err != nil {
		panic(err)
	}
	mux := multiplexer{
		"/foo/": fooHandler{
			ctx: ctx,
		},
		"/bar/": barHandler{
			ctx: ctx,
		},
	}

	http.ListenAndServe(":8000", endpoints(mux))
}
