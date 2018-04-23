package main

import (
	"log"
	"net/http"

	"github.com/davecgh/go-spew/spew"
	uuid "github.com/hashicorp/go-uuid"
)

// middleware provides a convenient mechanism for filtering HTTP requests
// entering the application. It returns a new handler which performs various
// operations and finishes with calling the next HTTP handler.
type middleware func(http.HandlerFunc) http.HandlerFunc

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
func (h handlerOne) logBefore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uuid, err := uuid.GenerateUUID()
		if err != nil {
			spew.Dump(err)
		}
		spew.Dump(string(uuid))
		next.ServeHTTP(w, r)
	}
}

// sendToZipkinAfter fires after the request
func (h handlerOne) logAfter(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		spew.Dump(r.Header)
		next.ServeHTTP(w, r)
		log.Println("After")
	}
}

func (h handlerOne) home(w http.ResponseWriter, r *http.Request) {
	log.Println("Home")
	r.Response = &http.Response{
		StatusCode: 400,
	}
}

func main() {
	handlerOne := handlerOne{
		clientName:  "dummyClient",
		serviceName: "dummyServer",
	}
	mw := chainMiddleware(handlerOne.logAfter, handlerOne.logBefore)
	http.Handle("/home", mw(handlerOne.home))

	if err := http.ListenAndServe(":8080", nil); err != nil {
		panic(err)
	}

}
