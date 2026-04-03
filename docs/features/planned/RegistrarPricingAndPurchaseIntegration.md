# Registrar Pricing And Purchase Integration

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

- add registrar provider abstraction
- support registrar-backed availability checks
- include `price`, `currency`, and `registration_period` in results
- add output columns for pricing in table, CSV, and JSON

### Phase 2: Budget-aware filtering

- add `--budget-max`
- add `--budget-min`
- add `--sort price`
- add `--only-standard-price`
- add summary breakdowns by price tier

### Phase 3: Purchase preparation

- retrieve registrar agreement / consent metadata
- support dry-run purchase payload generation
- validate account configuration before purchase mode

### Phase 4: Purchase execution

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
