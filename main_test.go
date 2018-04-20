package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func testClient() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/first":
			w.WriteHeader(http.StatusOK)
		case "/second":
			w.WriteHeader(http.StatusUnauthorized)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func TestMain(t *testing.T) {
	ts := testClient()
	defer ts.Close()
	err := sendToZipkin(zipkinParams{
		serviceHostPort: ts.URL,
		serviceName:     "Test Service",
		serviceEndpoint: ts.URL,
	})
	if err != nil {
		t.Fatal(err)
	}
}
