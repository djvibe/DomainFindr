package runner

import (
	"context"
	"errors"
	"io"
	"log"
	"testing"
	"time"

	"github.com/djvibe/domainfindr/internal/model"
)

type fakeChecker struct {
	results []model.Result
	calls   int
}

func (f *fakeChecker) Check(context.Context, string) model.Result {
	result := f.results[f.calls]
	f.calls++
	return result
}

type fakeLimiter struct {
	calls int
}

func (f *fakeLimiter) Wait(context.Context) error {
	f.calls++
	return nil
}

func TestRunnerRetriesLookupErrors(t *testing.T) {
	t.Parallel()

	checker := &fakeChecker{
		results: []model.Result{
			{Domain: "example.com", Status: model.StatusLookupError, Source: model.SourceRDAP, Error: model.StringPtr("temporary")},
			{Domain: "example.com", Status: model.StatusRegistered, Source: model.SourceRDAP, Available: model.BoolPtr(false)},
		},
	}
	limiter := &fakeLimiter{}

	run := New(checker, limiter, Config{
		Retry:       1,
		Concurrency: 1,
		Timeout:     time.Second,
		Logger:      log.New(io.Discard, "", 0),
	})

	results := run.Run(context.Background(), []model.Entry{{Domain: "example.com", Valid: true}})
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != model.StatusRegistered {
		t.Fatalf("expected retry to succeed, got %#v", results[0])
	}
	if limiter.calls != 2 {
		t.Fatalf("expected 2 limiter calls, got %d", limiter.calls)
	}
}

func TestRunnerDoesNotRetryRegistrarUnknown(t *testing.T) {
	t.Parallel()

	checker := &fakeChecker{
		results: []model.Result{
			{Domain: "example.com", Status: model.StatusRegistrarUnknown, Source: model.SourceHTTP, Error: model.StringPtr("timeout")},
		},
	}
	limiter := &fakeLimiter{}

	run := New(checker, limiter, Config{
		Retry:       3,
		Concurrency: 1,
		Timeout:     time.Second,
		Logger:      log.New(io.Discard, "", 0),
	})

	results := run.Run(context.Background(), []model.Entry{{Domain: "example.com", Valid: true}})
	if results[0].Status != model.StatusRegistrarUnknown {
		t.Fatalf("expected registrar_unknown, got %#v", results[0])
	}
	if limiter.calls != 1 {
		t.Fatalf("expected a single limiter call, got %d", limiter.calls)
	}
}

func TestRunnerInvalidInputBypassesLookup(t *testing.T) {
	t.Parallel()

	checker := &fakeChecker{
		results: []model.Result{{Domain: "unused", Status: model.StatusRegistered}},
	}
	run := New(checker, nil, Config{Concurrency: 1, Logger: log.New(io.Discard, "", 0)})

	results := run.Run(context.Background(), []model.Entry{{Domain: "not a domain", Valid: false}})
	if results[0].Status != model.StatusInvalid {
		t.Fatalf("expected invalid status, got %#v", results[0])
	}
	if checker.calls != 0 {
		t.Fatalf("expected no checker calls, got %d", checker.calls)
	}
}

func TestTickerLimiterHonorsContext(t *testing.T) {
	t.Parallel()

	limiter := NewTickerLimiter(100 * time.Millisecond)
	if err := limiter.Wait(context.Background()); err != nil {
		t.Fatalf("unexpected initial wait error: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := limiter.Wait(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
