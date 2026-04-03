# Registrar Pricing And Purchase Integration

## Status

Partially completed.

Phases 1 and 2 are effectively done in the current codebase:

- registrar provider abstraction exists
- registrar-backed availability checks exist
- results already include `price`, `currency`, and `registration_period`
- table, CSV, and JSON outputs already surface registrar pricing fields
- budget filtering exists via `--budget-min` and `--budget-max`
- price sorting exists via `--sort price`
- standard-price filtering exists via `--only-standard-price`
- summary output now includes price-tier breakdowns

Related completed work:

- `docs/features/done/RegistrarAwareAvailabilityVerification.md`
- `docs/features/done/RegistrarVerificationConfidenceAndEnvironmentLabels.md`
- `docs/features/done/RegistrarRetryAndRecheckHardening.md`

Remaining work is concentrated in:

- phase 3 purchase-prep metadata and dry-run payload generation
- phase 4 explicit purchase execution

## Goal

Extend `DomainFindr` beyond RDAP-only availability checks so it can surface registrar pricing and eventually support purchase workflows.

## Why

Availability alone is not enough for real buying decisions.

Key gaps exposed in domain strategy sessions:

- a domain may appear available in RDAP but be premium or brokered at registrars
- pricing materially changes recommendations
- users need a way to compare “available at hand-reg fee” vs “premium acquisition”

## Proposed Scope

### Phase 1: Read-only registrar pricing

Completed.

Implemented via registrar-aware verification, merged registrar result selection, pricing fields in the result model, and provider-aware output formatting.

### Phase 2: Budget-aware filtering

Completed.

Implemented via:

- `--budget-max`
- `--budget-min`
- `--sort price`
- `--only-standard-price`
- summary breakdowns for priced available results

- add `--budget-max`
- add `--budget-min`
- add `--sort price`
- add `--only-standard-price`
- add summary breakdowns by price tier

### Phase 3: Purchase preparation

- not yet implemented
- retrieve registrar agreement / consent metadata
- support dry-run purchase payload generation
- validate account configuration before purchase mode

### Phase 4: Purchase execution

- not yet implemented
- explicit purchase command
- confirmation step before order placement
- purchase logs and order receipt capture

## Suggested CLI Shape

```bash
domainFindr --input domains.md --provider godaddy --include-pricing
domainFindr --input domains.md --provider godaddy --budget-max 100
domainFindr buy --provider godaddy --domain spaatlas.ai
```

## Risks

- registrar APIs may have account-level restrictions
- premium domain handling may differ from standard registrations
- pricing may vary between registry, registrar, and marketplace surfaces

## Success Criteria

- user can distinguish standard-priced registrations from premium domains
- output makes buying decisions easier, not more confusing
- purchase support is gated behind explicit confirmation
