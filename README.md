# DomainFindr

DomainFindr is a Go CLI for checking whether domains are registered or available. It supports direct domain arguments for quick checks and batch input from Markdown or CSV files for larger runs.

## Features

- Direct lookup: `domainFindr openai.com`
- Batch lookup from `--input domains.md` or `--input domains.csv`
- Output formats: table, JSON, and CSV
- RDAP-based availability checks
- Worker pool with retries, timeouts, and global request pacing
- Automatic per-run history with saved results and logs

## Install

### Build locally

```bash
go build -o domainFindr ./cmd/domainfindr
./domainFindr openai.com
```

### Install globally

```bash
sudo cp ./domainFindr /usr/local/bin/domainFindr
domainFindr openai.com
```

## Usage

### Single domain

```bash
domainFindr openai.com
```

### Multiple domains

```bash
domainFindr openai.com example.com
```

### Batch from file

```bash
domainFindr --input domains.md
domainFindr --input domains.csv --format json
domainFindr --input domains.csv --format csv --output results.csv
```

### History and logs

By default, each run saves a timestamped history folder under:

```bash
~/.local/state/domainfindr/history/
```

Each run directory contains:

- `results.txt`, `results.json`, or `results.csv`
- `domainfindr.log`

Examples:

```bash
domainFindr openai.com
domainFindr --input ~/Downloads/domains.csv --history-dir ~/domainfindr-history
domainFindr --input ~/Downloads/domains.md --log-file ~/domainfindr.log --no-history
```

### Common flags

- `--input`, `-i`: Markdown or CSV input file
- `--output`, `-o`: write results to a file instead of stdout
- `--format`, `-f`: `table`, `json`, or `csv`
- `--concurrency`, `-c`: number of workers
- `--delay`: minimum delay between outbound lookups
- `--retry`: retry count for transient failures
- `--timeout`: timeout per lookup attempt
- `--verbose`, `-v`: verbose progress logging to stderr
- `--log-file`: write logs to a specific file
- `--history-dir`: override the default history root directory
- `--no-history`: disable automatic history saving

## Project Layout

- `cmd/domainfindr/`: CLI entrypoint
- `internal/app/`: flag parsing and orchestration
- `internal/input/`: Markdown and CSV parsing
- `internal/lookup/`: RDAP client
- `internal/runner/`: concurrency, retries, rate limiting
- `internal/output/`: table, JSON, CSV formatting
- `docs/`: research and planning notes
- `docs/features/planned/`: planned feature or launch docs tracked in Git
- `docs/features/done/`: completed feature or launch docs tracked in Git
- `workspace/`: local-only batch input/output area, ignored by Git

## Suggested Workflow

Use `workspace/` for local batch processing and `docs/features/` for tracked feature planning and completed launch docs.

### Batch searches

Drop incoming Markdown or CSV files into:

```bash
workspace/batches/in/
```

After running them and saving the outputs you want to keep, move the source file into:

```bash
workspace/batches/done/
```

Example:

```bash
domainFindr --input workspace/batches/in/april-batch.csv --format csv
mv workspace/batches/in/april-batch.csv workspace/batches/done/
```

### Feature planning

Keep domains you are still considering in:

```bash
docs/features/planned/
```

Move finalized or completed sets into:

```bash
docs/features/done/
```

## Development

```bash
go test ./...
gofmt -w cmd internal
go build ./cmd/domainfindr
```

If Go cannot write to its default cache in a restricted environment:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache go test ./...
```

## Notes

v1 uses public RDAP lookups only. It does not require API credentials, but live lookups do require working outbound network access.
