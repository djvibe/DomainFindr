# Domain Research Memory And Session Artifacts

## Goal

Make domain research work more persistent and reusable between sessions.

## Why

High-value naming sessions generate:

- lessons learned
- strategic conclusions
- saved outputs
- naming shortlists
- brand architecture decisions

Without a durable structure, the same work gets repeated.

## Current V1 Scope

### Tracked memory

- keep durable lessons in tracked docs
- link those docs from `AGENTS.md`
- maintain a concise “what we learned” artifact after major sessions

### Local session outputs

- standardize local report folders under `workspace/results/`
- include:
  - `README.md`
  - `executive-summary.md`
  - `full-report.md`
  - `candidate-domains.md`
  - `availability.md`

### Output conventions

- name folders with topic + date
- keep titles human-readable and recallable
- separate durable lessons from one-off research output

## Current Status

The initial `domainFindr consult` workflow now creates these local session artifacts automatically.

Still open:

- deciding whether to add a tracked session recap artifact in `docs/`
- tightening the brief schema beyond simple Markdown field parsing

## Success Criteria

- future sessions can recover context quickly
- repeated mistakes are reduced
- important research is easy to find without cluttering tracked docs
