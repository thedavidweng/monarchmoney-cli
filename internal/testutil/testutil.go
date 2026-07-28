package testutil

import (
	"io"
	"net/http"
	"strings"
)

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func JSONResponse(body string) *http.Response {
	return &http.Response{
		StatusCode: http.StatusOK,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

type FailingReader struct{}

func (FailingReader) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }

type FailingCloser struct{}

func (FailingCloser) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (FailingCloser) Close() error             { return nil }
