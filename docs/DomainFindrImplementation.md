# Executive Summary

We want a CLI tool that reads a list of domain names (from a Markdown file or a CSV) and checks each domain’s availability (registered or free). It should output the results (domain name and availability status) as CSV, JSON, or a human-readable table. Key requirements include parsing input files, performing lookups (via WHOIS or DNS), handling concurrency and rate limits, and formatting output. Non-functional needs cover error handling, logging, retries, and cross-platform support.

The core workflow is: read domains, for each domain perform a WHOIS/DNS query (or use an API), then collect and format the results. Because many registries enforce strict rate limits (≈1 lookup per second per IP)【54†L99-L100】, our tool must throttle requests accordingly. We’ll use a worker pool (threads or async tasks) with a delay or token bucket to avoid overwhelming WHOIS servers【54†L99-L100】. For example, the Rust tool “domain-check” processes up to 100 domains in parallel with retries and outputs CSV/JSON【51†L65-L70】 – we’ll adopt a simpler version of that idea. 

We’ll survey existing options to leverage: the standard `whois` CLI (GPLv2)【45†L221-L223】, DNS lookup tools (`dig`/`host`/`nslookup`), plus libraries like Python’s `python-whois` (MIT)【52†L188-L191】, Node’s `whois` (BSD)【36†L44-L47】, and Go’s `likexian/whois` (Apache 2.0)【41†L208-L212】【41†L270-L273】. Each has trade-offs (see table below). For example, system `whois` is ubiquitous on Unix but single-threaded and rate-limited【54†L99-L100】, while the Go `whois` library is fast and concurrent. We’ll likely wrap one of these tools or libraries rather than re-implement WHOIS.

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
  - **Rate-limiting:** Enforce a delay between WHOIS requests (~1/sec per IP)【54†L99-L100】 to comply with registry limits.  
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

To efficiently handle many domains, we’ll launch multiple workers (threads or async tasks). However, many WHOIS servers (especially through resellers like OpenSRS) enforce **1 lookup per second per IP**【54†L99-L100】. To avoid being blocked, we’ll insert delays or use a rate limiter. For example:

- Use a worker pool of size *N*. Each worker fetches a domain and does `whois` or DNS query.  
- After a WHOIS query, a worker waits for `delay` seconds (default 1s) before the next query. This ensures ≲1 query/sec per worker.  
- Alternatively, use a shared token bucket with capacity/rate of 1 token/sec, so only 1 worker can query per second collectively. 

This way, even with concurrency, we don’t exceed ~1 QPS. Tools like the Rust “domain-check” CLI achieve high throughput (e.g. *up to 100 concurrent checks*) by handling retries and outputs as a stream【51†L65-L70】. We’ll aim for a simpler model but with configurable concurrency and delay to adapt. For example, `--concurrency 5 --delay 1` gives 5 lookups per second (with each thread doing 1/s).

We should also catch common rate-limit responses (like “Lookup refused” or WHOIS errors) and back off.

## Error Handling and Retries

- If a domain lookup fails (timeout, connection error, unexpected response), retry up to `--retry` times. Wait a bit between retries.  
- If a domain name is invalid (e.g. bad syntax), skip it and log a warning.  
- For WHOIS: some libraries throw an exception when domain is “NOT FOUND” (meaning available). We’ll interpret that as `available=true`. Any other exception is a lookup error.  
- For DNS: if no A/NS records found, we might guess “likely available” but DNS alone is not reliable. WHOIS is primary.  
- After retries, if still failing, record an “error” in the output or log. Example CSV output columns might be `domain,available,error`; where `error` is blank if lookup succeeded.  

Comprehensive error handling is key. The domain-check example highlights “automatic retry mechanisms, comprehensive error handling”【51†L65-L70】; we’ll do a simpler version.

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

## Tools and Libraries Inventory

Here’s a summary of existing tools and libraries we might leverage:

- **whois (system CLI):** Port-43 WHOIS client (usually GPLv2)【45†L221-L223】. Queries registry WHOIS servers.  
  - *License:* GPLv2+【45†L221-L223】.  
  - *Platforms:* Available on Linux (in `whois` package) and macOS. Windows does not have it by default (could use Cygwin or WinWHOIS).  
  - *Pros:* No extra install on Unix, standard tool.  
  - *Cons:* Single-threaded, no built-in JSON/CSV output, strictly rate-limited (~1 query/sec/IP)【54†L99-L100】, parses text output manually.  

- **dig/host/nslookup:** DNS query tools (part of BIND suite, ISC license).  
  - *License:* ISC/BIND license.  
  - *Platforms:* Cross-platform (Windows has nslookup, Linux/Mac have dig/host).  
  - *Pros:* Fast DNS lookups; no rate-limit; can quickly check if domain has any DNS records.  
  - *Cons:* Only checks DNS, not WHOIS. A domain can be registered with no DNS. So these are an **incomplete proxy** for availability. Not a reliable availability check by itself.  

- **Domainr API (Fastly):** Domain search API (part of Fastly now). Direct registry data.  
  - *License:* Proprietary (API service).  
  - *Platforms:* HTTP REST.  
  - *Pros:* Very accurate (ICANN-accredited); covers all TLDs; returns JSON with availability flags. Fast (hundreds of ms).  
  - *Cons:* Requires an API key (free tier has limits). Usage would require handling API signup and potential rate limits.  

- **Registrar APIs (GoDaddy, Namecheap):** Official domain availability APIs.  
  - *License:* Proprietary.  
  - *Platforms:* HTTP REST.  
  - *Pros:* Official data, reliable.  
  - *Cons:* Need developer account and authentication, usually a paid plan or minimum spend. Not practical for a simple CLI unless key is already available.  

- **python-whois:** Python library (MIT) to query WHOIS and parse the text【52†L188-L191】.  
  - *License:* MIT【52†L188-L191】.  
  - *Platforms:* Cross-platform (runs on any system with Python 3).  
  - *Pros:* Easy to call (`import whois; whois.whois("domain.com")` returns an object). Parses dates, registrar, etc.  
  - *Cons:* Under the hood it still uses WHOIS, so subject to same rate limits. Python can be slower and has to handle edge cases in text parsing.  

- **whois (NodeJS, npm):** Node.js WHOIS client by hjr265 (BSD)【36†L44-L47】. The npm package is named “whois”.  
  - *License:* BSD-2-Clause【36†L44-L47】.  
  - *Platforms:* Cross-platform (requires Node).  
  - *Pros:* Asynchronous, returns raw text or JSON (with parsers).  
  - *Cons:* Last major update was many years ago; may need an updated fork or wrap the CLI via child process.  

- **likexian/whois (Go):** Go module and CLI for WHOIS queries【41†L208-L212】【41†L270-L273】.  
  - *License:* Apache-2.0【41†L270-L273】.  
  - *Platforms:* Cross-platform (Go programs run anywhere).  
  - *Pros:* Fast and efficient, supports IPv4/IPv6/ASN lookups. Can build a static binary. Good for concurrent use.  
  - *Cons:* If using from Go code, requires Go tooling. If used as a CLI, we’d parse its output or use their library directly.  

- **domain-check (Rust CLI):** Open-source CLI (Apache-2.0) that checks domains using RDAP and WHOIS【51†L65-L70】.  
  - *License:* Apache-2.0【51†L65-L70】.  
  - *Platforms:* Cross-platform (Rust builds).  
  - *Features:* Bulk CSV/JSON output, configurable concurrency (up to 100 threads)【51†L65-L70】, retries, caching.  
  - *Notes:* We won’t reuse it directly, but it’s a great example of a mature solution.  

- **whois-wrapper (NodeJS):** A thin npm wrapper that simply calls the system `whois` command.  
  - *License:* MIT (recent).  
  - *Use-case:* If we rely on the installed `whois`, this can simplify Node integration.

Below is a comparison of key tools/libraries:

| Tool/Library                  | Type    | License (See ref.)   | Platforms            | Notes / Pros & Cons                                                                                  |
|-------------------------------|---------|----------------------|----------------------|------------------------------------------------------------------------------------------------------|
| **whois** (system CLI)        | CLI     | GPLv2+【45†L221-L223】   | Unix (Linux/macOS); Windows (needs install) | Standard WHOIS client. Native to many Unix systems. **Pros:** no extra deps. **Cons:** serial, text output only, strict rate-limit (~1/s)【54†L99-L100】.|
| **dig/host/nslookup**         | CLI     | ISC (BIND license)   | Cross (nslookup on Win, dig/host on Unix) | DNS lookups only. **Pros:** fast, no rate-limit. **Cons:** only checks DNS records (not definitive on registration). |
| **python-whois**              | Library | MIT【52†L188-L191】     | Cross-platform       | Python WHOIS parser. **Pros:** Easy to script, returns structured data. **Cons:** Python slowness, still uses WHOIS under the hood (rate limits). |
| **whois (NodeJS)**            | Library | BSD-2-Clause【36†L44-L47】 | Cross-platform       | Node.js WHOIS client. **Pros:** Async interface. **Cons:** Not recently updated, may need maintenance. |
| **likexian/whois (Go)**       | Library | Apache-2.0【41†L270-L273】| Cross-platform (Go)  | Go WHOIS library & CLI. **Pros:** Fast, supports IPv4/6/ASN【41†L208-L212】, easy concurrency. **Cons:** Adds Go dependency. |
| **domain-check (Rust)**       | CLI     | Apache-2.0【51†L65-L70】 | Cross-platform       | Example high-performance CLI. **Pros:** Bulk JSON/CSV, 100-thread concurrency, retries, error handling【51†L65-L70】. Good model. |
| **Domainr API (Fastly)**      | API     | Proprietary          | HTTP REST            | Domain availability API. **Pros:** Accurate (ICANN-accredited) with no false positives【21†L12-L15】. **Cons:** Requires API key, usage limits. |
| **Registrar APIs** (GoDaddy, Namecheap) | API | Proprietary          | HTTP REST            | Official WHOIS/availability APIs. **Pros:** Reliable. **Cons:** Require account/keys, paid plans. |

_(References:_ Licenses and features are documented above【45†L221-L223】【52†L188-L191】【36†L44-L47】【41†L270-L273】【54†L99-L100】【51†L65-L70】 _)._  

## Implementation Options (Languages)

- **Python:** Easy to write and extend. Many libraries (`python-whois`, `csv`, `json`, `argparse`). Concurrency via `threading`, `concurrent.futures`, or `asyncio` (with `aiohttp` if using APIs). Cross-platform out of box. Downside: Python processes are heavier and the GIL limits real parallel threads (though I/O-bound WHOIS calls are mostly waiting). We can also use Python’s subprocess to call system `whois` if needed.
  
- **Go:** Compiled static binary, excellent for CLI tools. Concurrency is natural (goroutines, channels). Good standard library for flags/CSV/JSON. We can import `likexian/whois` to do lookups【41†L208-L212】. The result is a fast, small executable. Downside: longer compile times and more boilerplate code.

- **Node.js:** Good async support for I/O. Modules like `csv-parser`, `commander` for CLI, `whois` for lookups. Cross-platform (requires Node runtime). Concurrency via Promises or `async/await`. Might be a bit heavy on memory, but quick to develop if the team is familiar.

Each language can achieve the task. For example, in Python we might use a ThreadPoolExecutor; in Go, goroutines with a semaphore; in Node, `Promise.all` with a concurrency limiter library (like `p-limit`). The choice depends on team expertise and deployment needs. 

## Architecture & Module Breakdown

A minimal architecture could be:

1. **CLI Module:** Parses arguments, calls the main workflow, handles help messages. (Could be `main.py`, `main.go`, or `index.js` depending on language.)  
2. **Input Parser:** Reads the input file. If Markdown, it removes Markdown syntax and extracts domains (regex or a Markdown library). If CSV, it uses a CSV reader. Returns a list of domains.  
3. **Lookup Engine:** Given a domain, performs the check. This might use:  
   - *WHOIS Library/CLI:* e.g. call `python-whois` or `node-whois`, or spawn `whois domain`.  
   - *DNS Check:* Optionally use `dig` or a DNS library (`dnspython` in Python) to see if any record exists.  
   - *API call:* If an API key is provided (Domainr or registrar API), call that.  
   Decide availability: if WHOIS says “Not found”, mark available; if WHOIS returns data, mark taken.  
4. **Concurrency Controller:** Manages a pool of workers (threads/async tasks) that each take a domain and call the lookup engine. Also enforces rate-limiting (e.g. sleep or semaphore between calls) so we don’t exceed ~1 lookup/sec per IP【54†L99-L100】.  
5. **Output Formatter:** Collects results from all workers, then formats into CSV, JSON, or table. Use standard libraries (`csv.writer`, `json.dump`).  
6. **Logger:** A simple logging setup to print progress or errors.  

Each module should have unit tests. For example, the input parser should be tested with sample Markdown/CSV strings.

## Sample Command-Line Usage and Output

Here are a few examples of how the tool might be used and what it outputs:

- **Check domains from `domains.md`, JSON output:**
  ```
  $ domaincheck -i domains.md -f json -o avail.json
  [{"domain":"example.com","available":false},{"domain":"mynewdomain","available":true}]
  ```
  (Printed is the JSON array.)

- **Check 10 domains from CSV, show table:**
  ```
  $ domaincheck -i list.csv -f table
  DOMAIN              | AVAILABLE
  --------------------|----------
  example.com         | No
  likelyfree-domain.xyz | Yes
  ```
  (Here “No” = taken, “Yes” = available.)

- **Check with retries and verbose logging:**
  ```
  $ domaincheck -i domains.md --concurrency 5 --delay 1 --retry 3 --verbose
  Checking example.com... Registered (found WHOIS data).
  Checking likelyfree.com... Domain not found (available).
  ...
  Done: 50 domains (42 registered, 8 available, 0 errors).
  ```

All output should follow the specified format (CSV, JSON, or aligned text). 

## Flowchart of Workflow (Mermaid)

```mermaid
flowchart TD
    A[Read input file (MD/CSV)] --> B[Extract domain list]
    B --> C{Initialize worker pool}
    C --> D[Worker: get next domain]
    D --> E[Perform WHOIS/DNS query (with rate-limiter)]
    E --> F{Success or error}
    F -- Success --> G[Parse response → available/unavailable]
    F -- Error --> H[Retry or record error]
    G --> I[Collect result (domain,status,error?)]
    H --> I
    I --> D
    C --> J[All domains processed?]
    J -- No --> D
    J -- Yes --> K[Format results (CSV/JSON/Table)]
    K --> L[Write output / print]
```

## Development Timeline & Testing Checklist

- **Week 1:** Finalize requirements and choose tech stack. Set up project repo. Implement input parsing (Markdown & CSV) with tests.  
- **Week 2:** Implement single-threaded domain lookup using a chosen method (e.g. Python’s `python-whois` or Go’s library). Test with a few domains.  
- **Week 3:** Add concurrency (multi-thread/async) and enforce rate limiting (~1s delay). Write tests for concurrency and timing.  
- **Week 4:** Build output formatting (CSV/JSON/table). Refine CLI options and help text. Integration test end-to-end.  
- **Week 5:** Add logging/verbose mode. Polish and document. Final testing on cross-platform.  

**Testing Checklist:**  
- [ ] **Input Parsing:** Covers Markdown bullet lists, tables, CSV (with/without header).  
- [ ] **Domain Lookup:** Validates both registered domains and definitely-free domains. Handles edge cases (new TLDs, punycode).  
- [ ] **Concurrency:** Verifies that multiple lookups run in parallel but respect the delay (check timestamps to ensure ≈1s gap).  
- [ ] **Retries:** Simulate a failing WHOIS call (e.g. by mocking) to ensure it retries the configured number of times.  
- [ ] **Output Formats:** Generated CSV/JSON should parse correctly. Table output should align.  
- [ ] **CLI Behavior:** Flags and help text. Error if input file missing or unreadable.  
- [ ] **Error Handling:** For invalid domains or network errors, ensure the tool logs errors and continues.  
- [ ] **Cross-Platform:** Run on Linux/macOS/Windows (as applicable) to catch any OS-specific issues (e.g. newline handling, availability of `whois` command).

By following this plan—using existing libraries like **python-whois** (MIT)【52†L188-L191】 or **likexian/whois** (Apache)【41†L270-L273】, honoring rate limits【54†L99-L100】, and designing a clean CLI—we’ll build a robust domain-availability checker. The above table summarizes the tools to consider, and the flowchart and requirements guide the implementation.

**Sources:** We referenced standard WHOIS limits【54†L99-L100】, a high-performance example tool【51†L65-L70】, and library metadata for licenses and capabilities【52†L188-L191】【36†L44-L47】【41†L208-L212】. These informed our design and tool choices.