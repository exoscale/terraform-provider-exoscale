@RTK.md

# CLAUDE.md — terraform-provider-exoscale

This file provides scaffolding context and conventions for AI-assisted development in this repository.

---

## Project Overview

Vendored upstream Go Terraform/OpenTofu provider for Exoscale (`github.com/exoscale/terraform-provider-exoscale`). This copy is maintained in the monorepo for reference and to support local builds and patches. The canonical upstream is at `github.com/exoscale/terraform-provider-exoscale`.

---

## Repository Structure

```text
terraform-provider-exoscale/
├── exoscale/         # Provider resource and data source implementations (Go)
├── pkg/              # Shared internal packages
├── docs/             # Provider resource documentation (Markdown)
├── examples/         # Terraform usage examples per resource
├── templates/        # Docs generation templates
├── vendor/           # Vendored Go dependencies
├── scripts/          # Build/release scripts
├── bin/              # Compiled binaries (git-ignored)
├── .github/          # CI workflow definitions
├── go.mod / go.sum   # Go module definition
├── Makefile          # Build targets
└── .goreleaser.yml   # Release configuration
```

---

## Toolchain

- **Go** 1.16+ — provider implementation language
- **Make** — `make build` produces `bin/terraform-provider-exoscale_vdev`
- **goreleaser** — release automation
- **OpenTofu / Terraform** 0.15+ — consumer of the provider

---

## Things to Watch Out For

- **This is a vendored upstream copy** — changes here may be overwritten on the next vendor sync. Prefer contributing patches upstream; only apply local patches when upstream is too slow or unavailable
- **Provider registry reference** — the kubiqo-platform catalogs reference `registry.opentofu.org/hashicorp/exoscale 0.66.0`; check this version matches the code in this repo before making catalog-impacting changes
- **Go module path** — the module is `github.com/exoscale/terraform-provider-exoscale`; do not rename or move packages without updating all import paths
- **`docs/` is generated** — resource documentation is generated from templates; edit `templates/` not `docs/` directly
- **Acceptance tests require live Exoscale credentials** — do not run acceptance tests in CI without a valid `EXOSCALE_API_KEY` / `EXOSCALE_API_SECRET`

---

## How to Help Effectively

- When investigating a provider resource bug, start in `exoscale/` — each resource is typically one file named after the resource type
- Cross-reference the `docs/` directory to understand the expected user-facing behaviour before proposing code changes
- Do not upgrade the Go version or add new dependencies without checking compatibility with the vendored dependency set
- On a branch: write to `CHANGELOG-feat-<branch-purpose>.md`, `TROUBLESHOOTING-feat-<branch-purpose>.md`, and `README-feat-<branch-purpose>.md` — not the main files
- Do not auto-commit — always wait for explicit user instruction
