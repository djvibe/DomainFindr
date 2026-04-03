# DomainFindr

DomainFindr is a Go CLI for domain availability checks and domain-strategy workflow support. It supports direct domain arguments for quick checks, batch input from Markdown or CSV files for larger runs, registrar-backed verification, and a `consult` workflow that produces reusable strategy session artifacts.

## Features

- Direct lookup: `domainFindr openai.com`
- Batch lookup from `--input domains.md` or `--input domains.csv`
- Consultant session workflow: `domainFindr consult --brief brief.md --input candidates.md`
- Output formats: table, JSON, and CSV
- RDAP-based screening with optional registrar verification
- Worker pool with retries, timeouts, and global request pacing
- Automatic per-run history with saved results and logs
- Consultant session artifacts under `workspace/results/`

## Install

### Build locally

```bash
go build -o domainFindr ./cmd/domainfindr
./domainFindr openai.com
./domainFindr consult --brief brief.md
```

### Install globally

```bash
sudo cp ./domainFindr /usr/local/bin/domainFindr
domainFindr openai.com
```

### Consultant session

```bash
domainFindr consult --brief brief.md
domainFindr consult --brief brief.md --style authority,startup --input workspace/batches/in/candidates.md
domainFindr consult --brief brief.md --style authority,startup --provider godaddy spaatlas.ai sparank.ai
```

The `consult` workflow:

- parses a structured brief from Markdown
- creates a session folder under `workspace/results/`
- writes `README.md`, `executive-summary.md`, `full-report.md`, `candidate-domains.md`, and `availability.md`
- optionally runs the existing availability pipeline when an input file or positional domains are supplied

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
domainFindr --input docs/example-domains.md
domainFindr --input workspace/batches/in/april-batch.csv --format json
domainFindr --input workspace/batches/in/april-batch.csv --format csv --output results.csv
```

Supported batch input patterns:

- Markdown bullet lists such as `- example.com`
- Markdown tables with a `domain` column
- Plain domain lines inside fenced code blocks
- CSV files with a `domain` header or domains in the first column

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

### Consultant flags

- `consult`: enter consultant workflow mode
- `--brief`: Markdown brief file for the session
- `--style`: comma-separated naming lanes such as `authority,startup,directory`
- `--results-dir`: override the default consultant artifact root (`workspace/results`)
- `--provider`: optional registrar provider list for verification during consultant sweeps

## Project Layout

- `cmd/domainfindr/`: CLI entrypoint
- `internal/app/`: flag parsing and orchestration
- `internal/consult/`: brief parsing and consultant artifact generation
- `internal/input/`: Markdown and CSV parsing
- `internal/lookup/`: RDAP client
- `internal/runner/`: concurrency, retries, rate limiting
- `internal/output/`: table, JSON, CSV formatting
- `docs/features/`: tracked feature planning and completed launch docs
- `docs/features/planned/`: planned feature or launch docs tracked in Git
- `docs/features/done/`: completed feature or launch docs tracked in Git
- `workspace/`: local-only batch input/output area, ignored by Git

## Suggested Workflow

Use `workspace/` for local batch processing and consultant session outputs, and `docs/features/` for tracked feature planning and completed launch docs.

### Batch searches

Create the local batch folders if they do not exist yet:

```bash
mkdir -p workspace/batches/in workspace/batches/done
```

Drop incoming Markdown or CSV files into:

```bash
workspace/batches/in/
```

Recommended import formats:

~~~md
# Batch Name

```text
example.com
anotherexample.com
```
~~~

or:

```csv
domain
example.com
anotherexample.com
```

Avoid prose mixed directly with raw domain lines outside bullets, tables, or fenced code blocks.

After running them and saving the outputs you want to keep, move the source file into:

```bash
workspace/batches/done/
```

Example:

```bash
domainFindr --input workspace/batches/in/april-batch.csv --format csv
mv workspace/batches/in/april-batch.csv workspace/batches/done/
```

### Consultant sessions

Consultant runs write local-only strategy artifacts under:

```bash
workspace/results/
```

Recommended brief format:

~~~md
# Spa Atlas

- Business model: directory for premium spas
- Launch wedge: British Columbia spa discovery
- Long-term ambition: wellness platform
- Preferred naming styles: authority, startup, directory
- Geography: Canada
- Budget: under $2,000
- Registrar preference: GoDaddy
~~~

Example:

```bash
domainFindr consult --brief workspace/briefs/spa-atlas.md --style authority,startup --input workspace/batches/in/spa-atlas-candidates.md
```

Each session folder is named with a slug plus date and contains:

- `README.md`
- `executive-summary.md`
- `full-report.md`
- `candidate-domains.md`
- `availability.md`

### Feature planning

Keep domains you are still considering in:

```bash
docs/features/planned/
```

Move finalized or completed sets into:

```bash
docs/features/done/
```

Consultant workflow references now live in:

```bash
docs/features/done/
```

## Development

```bash
go test ./...
gofmt -w cmd internal
go build -o domainFindr ./cmd/domainfindr
```

If Go cannot write to its default cache in a restricted environment:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache go test ./...
```

## Notes

RDAP screening works without provider credentials. Registrar-backed verification requires the relevant environment variables when you use providers such as GoDaddy or Namecheap. Live lookups still require working outbound network access.
