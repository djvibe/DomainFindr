# Registrar-Aware Availability Verification

## Status

Completed for the current phase.

This feature now supports registrar-aware verification alongside RDAP, including side-by-side comparison output across multiple providers.

## What Was Built

### Provider abstractions

- added a provider-based verification flow under `internal/lookup/`
- kept RDAP as the registry-level source
- added registrar providers for:
  - GoDaddy OTE
  - Namecheap SBX
- preserved provider-specific results per domain so output can compare sources directly

### Result model expansion

The result model now carries:

- `registry_status`
- `registrar_status`
- `pricing_class`
- `price`
- `currency`
- `registration_period`
- `verification_provider`
- `verifications[]` for per-provider detail

Supported status classes now include:

- `standard_available`
- `premium_available`
- `unavailable`
- `registrar_unknown`

### CLI and environment support

- added `.env` loading from the repo root
- `.env` is git-ignored
- added provider-aware env resolution for:
  - GoDaddy OTE / PRD
  - Namecheap SBX / PRD
- added multi-provider selection through comma-separated `--provider`

Examples:

```bash
domainfindr --provider godaddy,namecheap spabeacon.com allspas.ai
domainfindr --provider namecheap --format json fluxforge984321.com
```

### Output improvements

The CLI now supports side-by-side provider comparison in table output.

Current provider labels:

- `GoDaddy OTE`
- `Namecheap SBX`
- `RDAP`

Table output now shows:

- final merged result
- RDAP result
- merged registrar result
- final selected price
- individual registrar provider columns
- final error column

### Merge behavior

The registrar merge logic was updated to avoid weak or incomplete results overriding stronger clean results.

Current behavior:

- `unavailable` outranks available statuses
- clean `premium_available` outranks `standard_available`
- incomplete `standard_available err` no longer overrides a clean premium result
- registrar disagreements remain visible in provider columns and summary output

### Summary improvements

The CLI summary now reports:

- available / unavailable / registrar unknown totals
- disagreement count
- provider coverage count
- disagreement domain list

## What We Observed In Live Testing

### GoDaddy OTE

- works for live integration testing
- returns availability and pricing data
- produces real premium and standard classifications
- is not reliable enough to treat as final buying truth
- production credentials returned access denied for the domains API

### Namecheap SBX

- works for live sandbox integration testing
- exact availability works
- standard pricing worked for `.com`, `.co`, and `.io` in testing
- `.ai` pricing lookup was incomplete in sandbox
- sandbox availability should not be treated as final buying truth

## Practical Conclusion

This phase is complete as an engineering feature.

The tool can now:

1. screen with RDAP
2. compare registrar-facing verification across multiple providers
3. surface premium vs standard vs unavailable
4. show where providers disagree

What it does not yet provide is production-grade purchase truth across live registrar environments. Sandbox and OTE data remain useful for integration verification, not final buy-now recommendations.

## Verification

Verified with:

```bash
GOCACHE=/tmp/domainfindr-go-build-cache /usr/local/go/bin/go test ./...
```

Live runs were also exercised against:

- GoDaddy OTE
- Namecheap SBX
