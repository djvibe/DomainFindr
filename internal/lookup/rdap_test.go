package lookup

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/djvibe/domainfindr/internal/model"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestRDAPCheckerRegistered(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path != "/domain/example.com" {
				t.Fatalf("unexpected path %s", req.URL.Path)
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	checker := NewRDAPChecker(client)
	checker.BaseURL = "https://rdap.test"
	result := checker.Check(context.Background(), "example.com")

	if result.Status != model.StatusRegistered || result.Available == nil || *result.Available {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRDAPCheckerAvailable(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(strings.NewReader(`not found`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	checker := NewRDAPChecker(client)
	checker.BaseURL = "https://rdap.test"
	result := checker.Check(context.Background(), "free-example.com")

	if result.Status != model.StatusAvailable || result.Available == nil || !*result.Available {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestRDAPCheckerServerError(t *testing.T) {
	t.Parallel()

	client := &http.Client{
		Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusBadGateway,
				Body:       io.NopCloser(strings.NewReader(`{"title":"upstream error"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	checker := NewRDAPChecker(client)
	checker.BaseURL = "https://rdap.test"
	result := checker.Check(context.Background(), "broken.example")

	if result.Status != model.StatusLookupError || result.Error == nil {
		t.Fatalf("unexpected result: %#v", result)
	}
}
