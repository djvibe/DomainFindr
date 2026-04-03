# Registrar-Aware Availability Verification

## Goal

Reduce false confidence from RDAP-only availability results by verifying candidate domains against registrar-facing availability where possible.

## Why

Real sessions exposed that:

- RDAP can say “available” while registrars show premium or unavailable
- purchase decisions need more than registry-level RDAP
- registrar truth is what matters at checkout

## Proposed Scope

- support a second verification layer after RDAP
- classify outcomes as:
  - standard_available
  - premium_available
  - unavailable
  - registrar_unknown
- add optional verification mode for shortlisted names

## Suggested Output Fields

- `registry_status`
- `registrar_status`
- `pricing_class`
- `price`
- `currency`
- `verification_provider`

## Example Workflow

1. Run broad RDAP scan across 100-500 names.
2. Shortlist top candidates.
3. Recheck shortlist with registrar-aware verification.
4. Promote only registrar-confirmed options into final recommendations.

## Success Criteria

- fewer misleading “available” recommendations
- better alignment between research output and real checkout reality
