# Store Mind Agent Routing

This repository is a monorepo with separate backend and frontend scopes.

## Scope Routing

- Frontend work: follow `frontend/AGENTS.md`
- Backend work: follow backend project docs and conventions under `backend/`

## Notes

- Keep frontend and backend changes isolated by directory.
- Prefer running commands from each subproject root (`frontend/` or `backend/`).

## Design Artifacts Placement

All product design, architecture analysis, and UX prototype artifacts belong under `docs/design/`. This keeps the repository root clean and separates active design deliverables from dated planning documents.

### Directory Structure

```
docs/
├── design/                         # Active design deliverables
│   ├── product.md                  # Project context: users, brand, tone, anti-references
│   ├── design.md                   # Visual design system: colors, typography, components
│   ├── prd/
│   │   └── wang-prd.html           # Product Requirements Document
│   ├── gap-analysis/
│   │   └── gap-analysis.html       # PRD vs current implementation gap analysis
│   └── prototypes/
│       └── v3.html                 # Latest interactive UX prototype
├── plans/                          # Dated design & implementation plans (historical)
│   └── 2026-05-28-customer-qa-agent-design.md ...
├── harness/                        # Harness scripts guide
└── testing/                        # Testing documentation
```

### Placement Rules

| Artifact type | Location | Naming convention |
|---------------|----------|-------------------|
| PRD | `docs/design/prd/` | `{feature}-prd.html` |
| Gap / impact analysis | `docs/design/gap-analysis/` | `{subject}-gap.html` |
| UX prototypes (HTML) | `docs/design/prototypes/` | `v{N}.html` (versioned) or `{feature}-proto.html` |
| Impeccable context (PRODUCT.md) | `docs/design/product.md` | — |
| Visual design system (DESIGN.md) | `docs/design/design.md` | — |
| Implementation plans | `docs/plans/` | `YYYY-MM-DD-{subject}.md` |
| Architecture decision records | `docs/design/` or `docs/plans/` | `YYYY-MM-DD-{decision}.md` |

### Impeccable Context

The `impeccable` skill's `load-context.mjs` searches for `PRODUCT.md` and `DESIGN.md` in the project root first, then `.agents/context/`, then `docs/`. When running `$impeccable` commands, set the context directory override so it loads from the design folder:

```
IMPECCABLE_CONTEXT_DIR=docs/design
```

### Cross-References Between HTML Deliverables

HTML files under `docs/design/` may link to each other. Use relative paths from the file's own location:

- From `docs/design/prd/wang-prd.html` to prototype: `../prototypes/v3.html`
- From `docs/design/gap-analysis/gap-analysis.html` to PRD: `../wang-prd/wang-prd.html`

### CodeGraph Index

CodeGraph is installed and maintains a live index at `.codegraph/codegraph.db`. After any document restructuring, run `codegraph sync` to refresh the index.
