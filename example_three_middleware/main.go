package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	opentracing "github.com/opentracing/opentracing-go"

	zipkin "github.com/openzipkin/zipkin-go-opentracing"
)

// middleware provides a convenient mechanism for filtering HTTP requests
// entering the application. It returns a new handler which performs various
// operations and finishes with calling the next HTTP handler.
type middleware func(http.HandlerFunc) http.HandlerFunc

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

func (h handlerOne) createZipkin(r *http.Request) (context.Context, error) {
	collector, err := zipkin.NewHTTPCollector(zipkinHTTPEndpoint)
	if err != nil {
		return nil, err
	}

	// Create our recorder.
	recorder := zipkin.NewRecorder(collector, debug, "127.0.0.1", h.serviceName)

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
	ctx := opentracing.ContextWithSpan(r.Context(), span)

	// Finish our CLI span
	span.Finish()

	// Close collector to ensure spans are sent before exiting.
	// collector.Close()
	return ctx, nil
}

// chainMiddleware provides syntactic sugar to create a new middleware
// which will be the result of chaining the ones received as parameters.
func chainMiddleware(mw ...middleware) middleware {
	return func(final http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			last := final
			for i := len(mw) - 1; i >= 0; i-- {
				last = mw[i](last)
			}
			last(w, r)
		}
	}
}

type handlerOne struct {
	clientName  string
	serviceName string
}

// sendToZipkin fires after the request
func (h handlerOne) sendToZipkinBefore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		span := opentracing.SpanFromContext(r.Context())
		defer span.Finish()

		span.SetOperationName(fmt.Sprintf("before %s %s", r.Method, r.URL.Path))
		span.SetTag("url", r.URL.String())
		next.ServeHTTP(w, r)
	}
}

// sendToZipkinAfter fires after the request
func (h handlerOne) sendToZipkinAfter(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, err := h.createZipkin(r)
		if err != nil {
			panic(err)
		}
		r = r.WithContext(ctx)

		next.ServeHTTP(w, r)
		span, _ := opentracing.StartSpanFromContext(ctx, h.clientName)
		defer span.Finish()

		span.SetOperationName(fmt.Sprintf("after %s %s", r.Method, r.URL.Path))
		span.SetTag("statusCode", r.Response.StatusCode)
		span.SetTag("url", r.URL.String())
	}
}

func (h handlerOne) home(w http.ResponseWriter, r *http.Request) {
	r.Response = &http.Response{
		StatusCode: 400,
	}
}

func main() {
	handlerOne := handlerOne{
		clientName:  "dummyClient",
		serviceName: "dummyServer",
	}
	mw := chainMiddleware(handlerOne.sendToZipkinAfter, handlerOne.sendToZipkinBefore)
	http.Handle("/home", mw(handlerOne.home))

	log.Fatal(http.ListenAndServe(":8080", nil))
}
