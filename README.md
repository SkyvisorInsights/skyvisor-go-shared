# Skyvisor Go Shared

Shared Go module for Skyvisor product types and the typed HTTP client against **skyvisor-api**.

## Packages

| Package | Purpose |
|---|---|
| `domain` | JSON contract types (Flight, Trip, Watch, Entitlements, …) |
| `apiclient` | HTTP client for `/v1/*` and `/public/shares/*` |

**Do not** put handlers, Templ, DB drivers, or Aviationstack secrets here.

## Local workspace

Until this is its own remote repo, dependents use:

```go
replace github.com/FACorreiaa/skyvisor-go-shared => ../skyvisor-go-shared
```

## Entitlements

`domain.FreeWatchLimit` is the Free-plan concurrent watch cap (aligned with `PRICING.md`).
