# Registrar Verification Follow-Ups

## Goal

Track the next work items after phase 1 registrar-aware verification.

## Status Update

The following slices are now completed:

- explicit registrar `consensus` / `conflict` / `incomplete` labeling
- explicit provider environment labeling in result output
- confidence-aware registrar merge weighting that favors clean production evidence over sandbox, OTE, or partial pricing evidence

See `docs/features/done/RegistrarVerificationConfidenceAndEnvironmentLabels.md` for implementation notes.

## Follow-Up Areas

### 1. Explicit conflict handling

Completed.

### 2. Environment awareness

Completed.

### 3. Registrar retry and timeout hardening

Completed.

Implemented:

- separate verifier-level registrar retry policy
- transient timeout and transient HTTP failure classification for registrar errors
- bounded registrar-only recheck path for transient `registrar_unknown` results

See `docs/features/done/RegistrarRetryAndRecheckHardening.md` for implementation notes.

### 4. Provider ranking / confidence

Partially completed.

Implemented:

- confidence weighting for production vs sandbox / OTE / custom
- confidence weighting for priced vs unpriced available results
- confidence weighting for clean vs partial / errored results

Still open:

- tune weighting against additional real production registrar behavior
- decide whether to expose an explicit numeric confidence score in output

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
