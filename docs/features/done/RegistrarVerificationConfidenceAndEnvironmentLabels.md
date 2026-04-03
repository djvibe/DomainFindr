# Registrar Verification Confidence And Environment Labels

## Status

Completed for the current slice.

## What Changed

### Registrar comparison labeling

- added `registrar_consensus` to merged results
- supported values:
  - `consensus`
  - `conflict`
  - `incomplete`
- `incomplete` is used when registrar evidence is present but not strong enough to treat as clean production confirmation

### Environment labeling

- added explicit registrar environment metadata to merged results and per-provider verification entries
- table output now includes an `ENV` column
- provider cells now show environment tags such as:
  - `[ote]`
  - `[sandbox]`
  - `[prod]`
- CSV and JSON output now include environment fields so downstream processing can distinguish production from non-production evidence

### Confidence-aware merge selection

- registrar merge selection is no longer purely status-rank based
- production results now carry more weight than sandbox or OTE results
- available results with clean pricing data carry more weight than available results missing price, currency, or term
- errored or partial registrar results carry less weight than clean registrar results
- provider base URL selection now respects the configured registrar environment so GoDaddy production does not silently fall back to OTE when both base URLs are present

## Practical Outcome

The merged registrar output is now harder to misread as final purchase truth.

In particular:

- sandbox and OTE checks remain visible, but are explicitly labeled as non-production
- registrar provider disagreement is visible as `conflict`
- registrar evidence that is partial, sandbox-only, OTE-only, or otherwise not clean production confirmation is labeled `incomplete`
- clean production registrar evidence can now outrank stronger-looking but lower-confidence sandbox results

## Verification

Verified with:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache /usr/local/go/bin/go test ./...
```

Live comparison run completed with:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache /usr/local/go/bin/go run ./cmd/domainfindr --no-history --provider godaddy,namecheap fluxforge984321.com
```

Observed live behavior:

- GoDaddy OTE returned `standard_available` with pricing
- Namecheap sandbox returned `standard_available` with pricing
- merged output labeled the result `incomplete`
- environment labels clearly showed `ote` and `sandbox`

This is the intended behavior because neither provider was production in that run.

Additional live verification with GoDaddy forced to production:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache GODADDY_API_ENV=prd /usr/local/go/bin/go run ./cmd/domainfindr --no-history --provider godaddy,namecheap fluxforge984321.com
```

Observed behavior:

- GoDaddy was correctly labeled `prod`
- GoDaddy production returned `registrar_unknown`
- Namecheap sandbox returned `standard_available` with pricing
- merged output stayed `incomplete` and selected the sandbox result because the only production evidence was an error, not clean purchase-grade confirmation
