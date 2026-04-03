# Registrar Verification Follow-Ups

## Goal

Track the next work items after phase 1 registrar-aware verification.

## Follow-Up Areas

### 1. Explicit conflict handling

- add a `conflict` or `consensus` field
- distinguish:
  - aligned registrar confirmation
  - provider disagreement
  - incomplete registrar evidence

### 2. Environment awareness

- mark provider environment explicitly in output
- avoid presenting sandbox or OTE pricing as production-grade purchase truth
- support environment labels in JSON, CSV, and table output

### 3. Registrar retry and timeout hardening

- improve retry behavior for registrar timeouts
- separate RDAP retry policy from registrar retry policy
- consider fallback recheck mode for high-interest domains

### 4. Provider ranking / confidence

- add confidence weighting for:
  - production vs sandbox
  - priced vs unpriced results
  - clean vs partial results
- consider making the merged registrar status confidence-aware instead of rank-only

### 5. Production registrar rollout

- enable GoDaddy production verification when production access is approved
- add Namecheap production testing after production credentials are available
- treat production registrar data as the real shortlist gate

### 6. Additional registrar coverage

- evaluate adding:
  - Porkbun
  - Dynadot
  - NameSilo

### 7. Purchase-prep operations

- GoDaddy agreements
- GoDaddy purchase schema
- purchase validation before any buy flow

## Short-Term Recommendation

Do not spend more time trying to make sandbox or OTE environments behave like final checkout truth.

Use them for:

- integration verification
- output-shape validation
- comparison UX testing

Use production registrar access for:

- shortlist confirmation
- pricing realism
- purchase decisions
