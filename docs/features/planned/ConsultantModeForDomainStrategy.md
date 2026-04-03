# Consultant Mode For Domain Strategy

## Goal

Evolve `DomainFindr` from a bulk availability checker into a domain consultant workflow that does more of the strategic heavy lifting.

## Why

The highest-value user outcome is not a raw available/taken list. It is a decision-ready recommendation that includes:

- strategic fit
- brand role
- fallback quality
- pricing reality
- purchase guidance

## Proposed Capabilities

### Idea generation

- create candidate names from user strategy prompts
- support multiple naming styles:
  - authority
  - discovery
  - SaaS
  - marketplace
  - startup coined

### Batch creation

- auto-generate clean batch files in local workspace
- group names into:
  - exact matches
  - premium alt TLDs
  - clean `.com` options
  - coined brandables
  - fallback constructions

### Recommendation scoring

- score names by:
  - clarity
  - memorability
  - premium feel
  - breadth
  - startup fit
  - SEO/category clarity

### Final report generation

- produce executive summary
- produce full report
- include buy-now recommendation
- include skip list and rationale

## Suggested CLI / Workflow Direction

This may start as a doc-driven workflow before becoming a full command.

Possible future command:

```bash
domainFindr consult --brief brief.md --style authority,startup --provider godaddy
```

## Success Criteria

- users can go from business idea to buy-now shortlist in a single workflow
- reports are easier to act on than raw availability output
- results reflect both naming quality and acquisition reality
