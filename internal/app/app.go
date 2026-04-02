package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/djvibe/domainfindr/internal/input"
	"github.com/djvibe/domainfindr/internal/lookup"
	"github.com/djvibe/domainfindr/internal/model"
	"github.com/djvibe/domainfindr/internal/output"
	"github.com/djvibe/domainfindr/internal/runner"
)

type Config struct {
	Input       string
	Output      string
	Format      string
	Concurrency int
	Delay       time.Duration
	Retry       int
	Timeout     time.Duration
	Verbose     bool
	Domains     []string
}

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return 1
	}
	if cfg == nil {
		return 0
	}

	logger := log.New(io.Discard, "", 0)
	if cfg.Verbose {
		logger = log.New(stderr, "", log.LstdFlags)
	}

	var entries []model.Entry
	if cfg.Input != "" {
		entries, err = input.ParseFile(cfg.Input)
		if err != nil {
			fmt.Fprintf(stderr, "input error: %v\n", err)
			return 1
		}
	} else {
		entries = input.ParseDomains(cfg.Domains)
	}

	checker := lookup.NewRDAPChecker(&http.Client{})
	run := runner.New(checker, nil, runner.Config{
		Concurrency: cfg.Concurrency,
		Delay:       cfg.Delay,
		Retry:       cfg.Retry,
		Timeout:     cfg.Timeout,
		Logger:      logger,
	})

	results := run.Run(ctx, entries)

	writer := stdout
	if cfg.Output != "" {
		file, err := os.Create(cfg.Output)
		if err != nil {
			fmt.Fprintf(stderr, "output error: %v\n", err)
			return 1
		}
		defer file.Close()
		writer = file
	}

	if err := output.WriteResults(writer, cfg.Format, results); err != nil {
		fmt.Fprintf(stderr, "format error: %v\n", err)
		return 1
	}

	printSummary(stderr, results)
	return 0
}

func parseFlags(args []string, stderr io.Writer) (*Config, error) {
	fs := flag.NewFlagSet("domainfindr", flag.ContinueOnError)
	fs.SetOutput(stderr)

	cfg := &Config{}
	fs.StringVar(&cfg.Input, "input", "", "Input file (Markdown or CSV)")
	fs.StringVar(&cfg.Input, "i", "", "Input file (Markdown or CSV)")
	fs.StringVar(&cfg.Output, "output", "", "Output file (defaults to stdout)")
	fs.StringVar(&cfg.Output, "o", "", "Output file (defaults to stdout)")
	fs.StringVar(&cfg.Format, "format", "table", "Output format: table, csv, or json")
	fs.StringVar(&cfg.Format, "f", "table", "Output format: table, csv, or json")
	fs.IntVar(&cfg.Concurrency, "concurrency", 4, "Number of worker goroutines")
	fs.IntVar(&cfg.Concurrency, "c", 4, "Number of worker goroutines")
	fs.DurationVar(&cfg.Delay, "delay", time.Second, "Minimum delay between outbound lookups")
	fs.IntVar(&cfg.Retry, "retry", 2, "Retries for transient lookup failures")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "Timeout per lookup attempt")
	fs.BoolVar(&cfg.Verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&cfg.Verbose, "v", false, "Enable verbose logging")

	if err := fs.Parse(args); err != nil {
		return nil, err
	}

	cfg.Domains = fs.Args()
	if cfg.Input == "" && len(cfg.Domains) == 0 {
		fs.Usage()
		return nil, fmt.Errorf("provide --input FILE or one or more domains")
	}

	return cfg, nil
}

func printSummary(w io.Writer, results []model.Result) {
	var available, registered, invalid, errors int
	for _, result := range results {
		switch result.Status {
		case model.StatusAvailable:
			available++
		case model.StatusRegistered:
			registered++
		case model.StatusInvalid:
			invalid++
		case model.StatusLookupError:
			errors++
		}
	}

	fmt.Fprintf(
		w,
		"Checked %d domains: %d available, %d registered, %d invalid, %d lookup errors.\n",
		len(results),
		available,
		registered,
		invalid,
		errors,
	)
}
