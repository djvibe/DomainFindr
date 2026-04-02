package app

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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
	LogFile     string
	HistoryDir  string
	NoHistory   bool
}

func Run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	cfg, err := parseFlags(args, stderr)
	if err != nil {
		return 1
	}
	if cfg == nil {
		return 0
	}

	history, err := prepareHistory(cfg)
	if err != nil {
		fmt.Fprintf(stderr, "history error: %v\n", err)
		return 1
	}
	defer closeQuietly(history.logFile)
	defer closeQuietly(history.outputFile)

	logWriter := io.Writer(io.Discard)
	if cfg.Verbose {
		logWriter = stderr
	}
	if history.logFile != nil {
		if cfg.Verbose {
			logWriter = io.MultiWriter(stderr, history.logFile)
		} else {
			logWriter = history.logFile
		}
	}
	logger := log.New(logWriter, "", log.LstdFlags)

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
	} else if history.outputFile != nil {
		writer = io.MultiWriter(stdout, history.outputFile)
	}

	if err := output.WriteResults(writer, cfg.Format, results); err != nil {
		fmt.Fprintf(stderr, "format error: %v\n", err)
		return 1
	}

	printSummary(stderr, results)
	if history.dir != "" {
		fmt.Fprintf(stderr, "History saved in %s\n", history.dir)
	}
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
	fs.StringVar(&cfg.LogFile, "log-file", "", "Write logs to a file")
	fs.StringVar(&cfg.HistoryDir, "history-dir", "", "Directory for timestamped result and log history")
	fs.BoolVar(&cfg.NoHistory, "no-history", false, "Disable automatic history saving")

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

type historyFiles struct {
	dir        string
	logFile    *os.File
	outputFile *os.File
}

func prepareHistory(cfg *Config) (historyFiles, error) {
	if cfg.NoHistory {
		if cfg.LogFile == "" {
			return historyFiles{}, nil
		}
		logFile, err := os.Create(cfg.LogFile)
		if err != nil {
			return historyFiles{}, err
		}
		return historyFiles{logFile: logFile}, nil
	}

	dir := cfg.HistoryDir
	if dir == "" {
		dir = defaultHistoryRoot()
	}
	runDir := filepath.Join(dir, time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return historyFiles{}, err
	}

	var logPath string
	if cfg.LogFile != "" {
		logPath = cfg.LogFile
	} else {
		logPath = filepath.Join(runDir, "domainfindr.log")
	}
	logFile, err := os.Create(logPath)
	if err != nil {
		return historyFiles{}, err
	}

	var outputFile *os.File
	if cfg.Output == "" {
		outputPath := filepath.Join(runDir, "results."+outputExtension(cfg.Format))
		outputFile, err = os.Create(outputPath)
		if err != nil {
			logFile.Close()
			return historyFiles{}, err
		}
	}

	return historyFiles{
		dir:        runDir,
		logFile:    logFile,
		outputFile: outputFile,
	}, nil
}

func defaultHistoryRoot() string {
	if stateHome := os.Getenv("XDG_STATE_HOME"); stateHome != "" {
		return filepath.Join(stateHome, "domainfindr", "history")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".", ".domainfindr-history")
	}
	return filepath.Join(home, ".local", "state", "domainfindr", "history")
}

func outputExtension(format string) string {
	switch strings.ToLower(format) {
	case "json":
		return "json"
	case "csv":
		return "csv"
	default:
		return "txt"
	}
}

func closeQuietly(file *os.File) {
	if file != nil {
		_ = file.Close()
	}
}
