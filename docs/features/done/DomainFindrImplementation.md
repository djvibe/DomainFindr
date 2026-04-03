# Executive Summary

Historical note: this document describes the original implementation plan before registrar verification and consultant-mode session artifacts were added. The current codebase has moved beyond the purely initial availability-checker scope described below.

We want a CLI tool that reads a list of domain names (from a Markdown file or a CSV) and checks each domain’s availability (registered or free). It should output the results (domain name and availability status) as CSV, JSON, or a human-readable table. Key requirements include parsing input files, performing lookups (via WHOIS or DNS), handling concurrency and rate limits, and formatting output. Non-functional needs cover error handling, logging, retries, and cross-platform support.

The core workflow is: read domains, for each domain perform a WHOIS/DNS query (or use an API), then collect and format the results. Because many registries enforce strict rate limits (≈1 lookup per second per IP), our tool must throttle requests accordingly. We’ll use a worker pool (threads or async tasks) with a delay or token bucket to avoid overwhelming WHOIS servers. For example, the Rust tool “domain-check” processes up to 100 domains in parallel with retries and outputs CSV/JSON – we’ll adopt a simpler version of that idea. 

We’ll survey existing options to leverage: the standard `whois` CLI (GPLv2), DNS lookup tools (`dig`/`host`/`nslookup`), plus libraries like Python’s `python-whois` (MIT), Node’s `whois` (BSD), and Go’s `likexian/whois` (Apache 2.0). Each has trade-offs (see table below). For example, system `whois` is ubiquitous on Unix but single-threaded and rate-limited, while the Go `whois` library is fast and concurrent. We’ll likely wrap one of these tools or libraries rather than re-implement WHOIS.

Finally, we’ll design the CLI interface (flags for input file, output format, concurrency, etc.), outline a development plan, and list tests. The table below compares candidate tools/libraries, and a flowchart illustrates the workflow.

## Technical Specifications

- **Functional Requirements:**  
  - **Input:** Read domains from a Markdown file (e.g. bullet list or table) or a CSV (with or without header).  
  - **Lookup:** For each domain, determine if it’s available (unregistered) or not. This can be done via WHOIS queries or DNS checks.  
  - **Output:** Write results in CSV, JSON, or a formatted text table. (E.g. `domain,available` in CSV or an array of JSON objects.)  
  - **CLI UX:** Provide options like `-i INPUT`, `-o OUTPUT`, `-f FORMAT`, `-c CONCURRENCY`, etc. Include `--help`. For example:  
    ```
    $ domaincheck -i domains.md -o results.csv -f csv -c 5
    domain,available
    example.com,false
    new-domain-test,true
    ```  
    (Here “false” means registered, “true” means available.)
    
- **Non-Functional Requirements:**  
  - **Concurrency:** Support multiple parallel queries to speed up bulk checks.  
  - **Rate-limiting:** Enforce a delay between WHOIS requests (~1/sec per IP) to comply with registry limits.  
  - **Retries:** On transient errors or timeouts, retry a fixed number of times (e.g. 2) before giving up.  
  - **Logging:** Log progress and errors (with a verbose/quiet mode).  
  - **Cross-platform:** Run on Linux, macOS, and ideally Windows (using native tools or fallback libraries).

## Input Formats

- **Markdown (.md):** Could be bullet lists (`- example.com`) or other Markdown content. We’ll strip markdown syntax (e.g. removing `- `, `* `, table characters) to extract plain domain names. A lightweight Markdown parser or regex can do this.  
- **CSV:** Standard CSV (comma or other delimiter) with domain names. We’ll detect a column named “domain” or use the first column by default. Use a robust CSV parser to handle quotes and delimiters.  

We should also handle small variations (tabs, semicolons, extra columns). If the input is not a recognized format, we’ll output an error message.

## Output Formats

- **CSV:** Plain CSV with header, e.g. `domain,available,error`. “available” could be a boolean (true/false) or “Yes/No”.  
- **JSON:** An array of objects, e.g. `[{"domain":"foo.com","available":false}, ...]`.  
- **Table:** ASCII table printed to stdout. For example:

  ```
  DOMAIN             | AVAILABLE
  -------------------|----------
  example.com        | No
  some-freenames.org | Yes
  ```
  
The output format is chosen by a command-line flag (e.g. `--format json`), defaulting to a simple table for console.

## CLI Design

A possible CLI (using `-` or `--` flags):

```
Usage: domaincheck [options] <input-file>
Options:
  -i, --input FILE       Input file (Markdown or CSV) with domain list
  -o, --output FILE      Output file (default: stdout)
  -f, --format [csv,json,table]  Output format
  -c, --concurrency N    Number of parallel workers (default: 1)
  --delay S              Delay between requests in seconds (for rate-limit)
  --retry N              Retries on failure (default: 2)
  -v, --verbose          Verbose logging
  -h, --help             Show help
```

**Example Usage:**

- JSON output, 10 concurrent queries, 0.5s delay:
  ```
  $ domaincheck -i domains.md -o results.json -f json -c 10 --delay 0.5
  [{"domain":"example.com","available":false},{"domain":"mynewdomain","available":true}]
  ```

- CSV output:
  ```
  $ domaincheck -i list.csv -f csv > availability.csv
  domain,available
  example.com,false
  example-test,yes
  ```

If no output file is given, write to standard output. The tool should also support reading from stdin if `-i -` is allowed (optional).

## Concurrency & Rate-Limiting

To efficiently handle many domains, we’ll launch multiple workers (threads or async tasks). However, many WHOIS servers (especially through resellers like OpenSRS) enforce **1 lookup per second per IP**. To avoid being blocked, we’ll insert delays or use a rate limiter. For example:

- Use a worker pool of size *N*. Each worker fetches a domain and does `whois` or DNS query.  
- After a WHOIS query, a worker waits for `delay` seconds (default 1s) before the next query. This ensures ≲1 query/sec per worker.  
- Alternatively, use a shared token bucket with capacity/rate of 1 token/sec, so only 1 worker can query per second collectively. 

This way, even with concurrency, we don’t exceed ~1 QPS. Tools like the Rust “domain-check” CLI achieve high throughput (e.g. *up to 100 concurrent checks*) by handling retries and outputs as a stream. We’ll aim for a simpler model but with configurable concurrency and delay to adapt. For example, `--concurrency 5 --delay 1` gives 5 lookups per second (with each thread doing 1/s).

We should also catch common rate-limit responses (like “Lookup refused” or WHOIS errors) and back off.

## Error Handling and Retries

- If a domain lookup fails (timeout, connection error, unexpected response), retry up to `--retry` times. Wait a bit between retries.  
- If a domain name is invalid (e.g. bad syntax), skip it and log a warning.  
- For WHOIS: some libraries throw an exception when domain is “NOT FOUND” (meaning available). We’ll interpret that as `available=true`. Any other exception is a lookup error.  
- For DNS: if no A/NS records found, we might guess “likely available” but DNS alone is not reliable. WHOIS is primary.  
- After retries, if still failing, record an “error” in the output or log. Example CSV output columns might be `domain,available,error`; where `error` is blank if lookup succeeded.  

Comprehensive error handling is key. The domain-check example highlights “automatic retry mechanisms, comprehensive error handling”; we’ll do a simpler version.

## Logging

Include a verbose mode. By default, print only errors and a final summary (e.g. “Checked 50 domains: 20 available, 30 taken, 0 errors”). With `--verbose`, log each domain check as it happens (including “checking foo.com…”). Use timestamps if helpful. Logging can go to stderr or a log file. 

## Testing Approach

We will test thoroughly:

- **Unit tests:** For parsing Markdown/CSV inputs, to ensure domains are extracted correctly (e.g. with tricky formatting). For output formatters, to ensure CSV/JSON formats are valid.  
- **Lookup tests:** Use a small set of known domains: e.g. “example.com” (registered), “some-unique-test-domain-xyz.com” (almost certainly free). Verify that the tool reports correctly. We can mock WHOIS server responses or use a test WHOIS server for controlled tests.  
- **Concurrency tests:** Check that with concurrency enabled, lookups still obey the delay. (E.g. instrument code to measure time between actual network calls.)  
- **Error tests:** Force a failure (e.g. cut network, use an invalid WHOIS server) and see that retries happen and errors are reported.  
- **CLI integration tests:** Run the compiled tool (or Python script) on sample input files and compare output files to expected results.  
- **Platform tests:** Run on Linux and Windows (for example, on Windows ensure it doesn’t crash if `whois` CLI is missing – maybe fall back to a library).  

A short testing checklist:
- [ ] Input file parsing (MD vs CSV, headers/no-headers).  
- [ ] Successful lookup of known domains.  
- [ ] Correct detection of availability vs taken.  
- [ ] Concurrency does not exceed rate limit (test with timestamps).  
- [ ] Retries work on simulated failures.  
- [ ] Output correctness in all formats.  
- [ ] Command-line flags behave as expected.  
- [ ] Error Handling: For invalid domains or network errors, ensure the tool logs errors and continues.  
- [ ] Cross-Platform: Run on Linux/macOS/Windows (as applicable) to catch any OS-specific issues (e.g. newline handling, availability of `whois` command).

By following this plan—using existing libraries like **python-whois** (MIT) or **likexian/whois** (Apache), honoring rate limits, and designing a clean CLI—we’ll build a robust domain-availability checker. The above table summarizes the tools to consider, and the flowchart and requirements guide the implementation.

**Sources:** We referenced standard WHOIS limits, a high-performance example tool, and library metadata for licenses and capabilities. These informed our design and tool choices.
