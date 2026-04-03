# Registrar Retry And Recheck Hardening

## Status

Completed for the current slice.

## What Changed

### Separate registrar retry policy

- added verifier-level registrar policy controls separate from runner-level RDAP retry handling
- added CLI flags for registrar hardening:
  - `--registrar-retry`
  - `--registrar-timeout`
  - `--registrar-recheck`
- runner-level `--retry` and `--timeout` still govern top-level lookup attempts
- registrar providers now use their own retry and timeout budget inside the verifier

### Transient timeout classification

- added internal registrar failure classification for transient and timed-out failures
- context deadline errors and timeout-shaped network failures are now marked retryable
- transient registrar HTTP statuses now carry retryable classification:
  - `408`
  - `429`
  - `502`
  - `503`
  - `504`
- non-transient registrar failures still surface as `registrar_unknown`, but do not trigger extra retry work

### Bounded registrar recheck path

- added a registrar-only second-pass recheck path for `registrar_unknown` results caused by transient failures
- rechecks are bounded by `--registrar-recheck`
- rechecks replace the provider’s prior verification entry rather than duplicating provider columns
- this keeps output stable while still allowing recovery from short-lived registrar failures

## Practical Outcome

The tool is now more resilient when registrar APIs briefly stall or rate-limit.

In particular:

- RDAP retry behavior remains isolated from registrar retry behavior
- registrar timeouts can recover without rerunning the entire domain check stack
- a transient registrar timeout no longer forces a final `registrar_unknown` result if a bounded retry or recheck can recover cleaner evidence

## Verification

Verified with:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache /usr/local/go/bin/go test ./...
```

Live comparison run completed with local credentials:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache /usr/local/go/bin/go run ./cmd/domainfindr --no-history --provider godaddy,namecheap fluxforge984321.com
```

Observed live behavior on April 3, 2026:

- RDAP returned `available`
- GoDaddy OTE returned `standard_available`
- Namecheap sandbox returned `standard_available`
- merged output remained `incomplete` because the registrar evidence was non-production

This run validated that:

- provider wiring still works after registrar retry hardening
- environment labeling remains intact
- provider columns remain stable after verifier-side retry and recheck changes
