package runner

import (
	"context"
	"io"
	"log"
	"sync"
	"time"

	"github.com/djvibe/domainfindr/internal/lookup"
	"github.com/djvibe/domainfindr/internal/model"
)

type Limiter interface {
	Wait(context.Context) error
}

type Config struct {
	Concurrency int
	Delay       time.Duration
	Retry       int
	Timeout     time.Duration
	Logger      *log.Logger
}

type job struct {
	index int
	entry model.Entry
}

type indexedResult struct {
	index  int
	result model.Result
}

type Runner struct {
	checker lookup.Checker
	limiter Limiter
	config  Config
}

func New(checker lookup.Checker, limiter Limiter, cfg Config) *Runner {
	if cfg.Concurrency <= 0 {
		cfg.Concurrency = 4
	}
	if cfg.Delay <= 0 {
		cfg.Delay = time.Second
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.Logger == nil {
		cfg.Logger = log.New(io.Discard, "", 0)
	}
	if limiter == nil {
		limiter = NewTickerLimiter(cfg.Delay)
	}

	return &Runner{
		checker: checker,
		limiter: limiter,
		config:  cfg,
	}
}

func (r *Runner) Run(ctx context.Context, entries []model.Entry) []model.Result {
	results := make([]model.Result, len(entries))
	jobs := make(chan job)
	out := make(chan indexedResult)

	var wg sync.WaitGroup
	for i := 0; i < r.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for item := range jobs {
				out <- indexedResult{
					index:  item.index,
					result: r.processEntry(ctx, item.entry),
				}
			}
		}()
	}

	go func() {
		for idx, entry := range entries {
			jobs <- job{index: idx, entry: entry}
		}
		close(jobs)
		wg.Wait()
		close(out)
	}()

	for item := range out {
		results[item.index] = item.result
	}

	return results
}

func (r *Runner) processEntry(ctx context.Context, entry model.Entry) model.Result {
	if !entry.Valid {
		return model.Result{
			Domain: entry.Domain,
			Status: model.StatusInvalid,
			Source: model.SourceInput,
			Error:  model.StringPtr("invalid domain"),
		}
	}

	attempts := r.config.Retry + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := r.limiter.Wait(ctx); err != nil {
			return model.Result{
				Domain: entry.Domain,
				Status: model.StatusLookupError,
				Source: model.SourceRDAP,
				Error:  model.StringPtr(err.Error()),
			}
		}

		if attempt > 1 {
			r.config.Logger.Printf("retrying %s (%d/%d)", entry.Domain, attempt, attempts)
		} else {
			r.config.Logger.Printf("checking %s", entry.Domain)
		}

		attemptCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
		result := r.checker.Check(attemptCtx, entry.Domain)
		cancel()

		if result.Status != model.StatusLookupError || attempt == attempts {
			return result
		}
	}

	return model.Result{
		Domain: entry.Domain,
		Status: model.StatusLookupError,
		Source: model.SourceRDAP,
		Error:  model.StringPtr("lookup failed"),
	}
}

type tickerLimiter struct {
	mu   sync.Mutex
	next time.Time
	step time.Duration
}

func NewTickerLimiter(delay time.Duration) Limiter {
	if delay <= 0 {
		return noopLimiter{}
	}

	return &tickerLimiter{step: delay}
}

func (l *tickerLimiter) Wait(ctx context.Context) error {
	l.mu.Lock()
	target := l.next
	now := time.Now()
	if target.IsZero() || now.After(target) {
		target = now
	}
	l.next = target.Add(l.step)
	l.mu.Unlock()

	waitFor := time.Until(target)
	if waitFor <= 0 {
		return nil
	}

	timer := time.NewTimer(waitFor)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type noopLimiter struct{}

func (noopLimiter) Wait(context.Context) error {
	return nil
}
