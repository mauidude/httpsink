package httpsink

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
)

// HTTPSink provides the struct around running the HTTP sync
type HTTPSink struct {
	Addr     string
	Capacity int
	Listener net.Listener

	mux *http.ServeMux

	sync.Mutex
	requests []*http.Request
}

// NewHTTPSink creates a sync running on :0 (random port)
func NewHTTPSink() (*HTTPSink, error) {
	return NewHTTPSinkOnAddr("localhost:0")
}

// NewHTTPSinkOnAddr takes in an adder, such as localhost:0 and
// the returned HTTPSink allows you to run the http server
func NewHTTPSinkOnAddr(addr string) (*HTTPSink, error) {
	s := &HTTPSink{Capacity: 1000, mux: http.NewServeMux()}
	s.mux.HandleFunc("/get", s.getHandler())
	s.mux.HandleFunc("/", s.setHandler())

	var err error
	s.Listener, err = net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	s.Addr = strings.Replace(s.Listener.Addr().String(), "[::]", "localhost", 1)

	return s, nil
}

// StartHTTP is a blocking call that serves the HTTP sync
func (s *HTTPSink) StartHTTP() error {
	return http.Serve(s.Listener, s.mux)
}

// Close closes the listener to free up the port
func (s *HTTPSink) Close() error {
	return s.Listener.Close()
}

func (s *HTTPSink) setHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.Capacity != 0 && len(s.requests) < s.Capacity {
			s.Lock()
			s.requests = append(s.requests, r)
			s.Unlock()

			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(fmt.Sprintf(`{"index":"%d"}`, len(s.requests))))

			return
		}
		w.WriteHeader(http.StatusGone)
		msg := "http sync is at capacity and no longer taking requests"
		json.NewEncoder(w).Encode(syncErr(msg))
	}
}

func (s *HTTPSink) getHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		index, err := strconv.Atoi(r.FormValue("idx"))
		if err != nil || len(s.requests) < index+1 {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(syncErr(fmt.Sprintf("idx value not valid (%s)", r.FormValue("idx"))))
			return
		}

		json.NewEncoder(w).Encode(s.requests[index])
	}
}

// SyncErrorResponse is the standard error response container for errors
// encountered when running HTTPSink
type SyncErrorResponse struct {
	Error string `json:"error"`
}

func syncErr(msg string) SyncErrorResponse {
	return SyncErrorResponse{msg}
}