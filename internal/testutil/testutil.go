// Package testutil provides shared test helpers for HTTP mocking and error simulation.
package testutil

import (
	"io"
	"net/http"
	"strings"
)

// RoundTripFunc is an http.RoundTripper backed by a function.
// Use it to intercept HTTP calls in tests without a real server.
type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// JSONResponse builds a 200 OK response with a JSON body string.
func JSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

// FailingReader is an io.Reader that always returns an error.
type FailingReader struct{}

func (FailingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

// FailingCloser is an io.ReadCloser that always fails on Read.
type FailingCloser struct{}

func (FailingCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (FailingCloser) Close() error             { return nil }
