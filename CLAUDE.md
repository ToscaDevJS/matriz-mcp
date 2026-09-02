# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Repository state

Greenfield. There is **no Go code yet** — the only tracked file is
`matriz-mcp-handoff.md` (Spanish, 1194 lines), which is the **source of truth**
for this project: exact contracts, work plan, tests and hard rules.

Read the handoff before writing code. Its section numbers (§5.1, §7.4, …) are
the canonical references, and this file only points at them.

If a task in the handoff contradicts a hard rule in its §7, **the rule wins and
the document is wrong**: stop, record it in `PENDING.md`, and report to the user.

## Commands

The module does not exist yet. PR-0 creates it with `go mod init github.com/<user>/matriz` (Go 1.25).

```bash
go build ./...                                  # must be clean on every PR
go vet ./...                                    # must be clean on every PR
go test ./...                                   # baseline: zero failures

go test ./internal/core -run TestResize         # single test
go test ./internal/core -run TestResize -update # regenerate golden fixtures, then eyeball them
```

Manual acceptance (§6, A-01): run the MCP server against the inspector and check
the thumbnail renders as an **image**, not raw base64.

```bash
go build -o bin/matriz-mcp ./cmd/matriz-mcp
npx @modelcontextprotocol/inspector ./bin/matriz-mcp
```

## Architecture

Local MCP server (stdio) that gives an LLM agent the ability to generate, edit
and export the images of a client website, plus a Go TUI over the same engine.
No HTTP, no auth, no multi-tenancy.

```
cmd/matriz-mcp   ─┐
                  ├─→ internal/core ──→ internal/providers (Provider interface)
cmd/matriz-tui   ─┘                └──→ internal/budget
```

- **One core, two frontends.** All logic lives in `internal/core`. `cmd/` holds
  no business logic. `internal/core` never imports `cmd/` or `internal/mcpserver`
  — the dependency runs one way only.
- **Deterministic vs generative is the central split.** Crop / resize / adjust /
  encode are local, free and reproducible (`img_transform`). Generation,
  inpaint, outpaint, background removal cost money and take seconds
  (`img_generate_drafts`, `img_refine`).
  The split is visible in tool names and descriptions: generative descriptions
  start with `COSTS MONEY`, deterministic ones with `FREE` (test T-15 greps for it).
- **Providers are swappable.** The core only knows the `Provider` interface
  (§5.1). Adding a provider must not touch the core or the tools. v0.1.0 ships
  Google Gemini (`google.golang.org/genai`) plus a deterministic `fakeProvider`
  that every test uses. fal.ai is PR-6 — **deferred**, with explicit triggers.
- **Budget guard before any paid call** (`internal/budget`, §5.6).
  `EstimateCostUSD` must never hit the network, so the guard can refuse a
  request before it spends.
- **Reproducibility by sidecar.** Every produced file writes
  `<name>.<ext>.meta.json` next to it (§5.4) with provider, model, prompt, seed
  and params. Sidecars are **not** gitignored.
- **Manifest as MCP resource.** `matriz.json` (§5.14) exposed at
  `matriz://project/manifest`: slots, palette and asset inventory. This is what
  stops the model generating a square hero for a 21:9 slot.

Delivery order: PR-0 scaffold → PR-1 deterministic pipeline → PR-2 providers +
budget guard → PR-3 MCP server → PR-4 manifest → PR-5 TUI. Each unit ends
compiling, green, and useful on its own. Full file matrix in §8.

## Non-negotiable rules (§7 — do not "improve" these)

1. A deterministic operation never calls a generative provider. Never merge
   `img_transform` and `img_refine` into one tool.
2. Full-resolution bytes never travel through the protocol. Tools return a
   thumbnail (`maxEdge` 512 px, §5.12) plus structured output; the big file
   stays on disk behind an `AssetRef`.
3. API keys live only in environment variables and memory. Never on disk, in a
   log, in an error message, or in the repo.
4. Every `ref` crossing the MCP boundary goes through `ResolveRef` (§5.8) —
   including refs a previous tool returned. LLM-supplied paths are untrusted.
5. Client input files are never modified in place or deleted. Results are always
   new files. There is no delete tool in v0.1.0; do not add one.
6. `origin` (`client` | `generated` | `derived`) is mandatory on every asset, and
   a `generated` image never fills a slot that presents it as a real client or
   real work. Ambiguous slot ⇒ ask, don't assume.
7. `stdout` belongs to the stdio protocol. No `fmt.Print*` outside stderr in
   `cmd/matriz-mcp`.
8. Do not build on MCP roots, sampling or logging — deprecated since spec
   `2026-07-28` (SEP-2577).
9. Tool failures return `CallToolResult{IsError: true}` with actionable text
   (§5.7 template), never a Go error propagated to the protocol — the LLM would
   not see it and could not correct itself.
10. Model IDs are configuration, never Go constants (`gemini-3-pro-image-preview`
    is a preview ID and will be deprecated).
11. An unknown cost is never estimated as zero — a model missing from the pricing
    table estimates at the known worst case.
12. Seeds are never invented. No seed from the provider ⇒ sidecar says
    `"seed": 0` with `"seeded": false`.

## Configuration (environment only — no config file in v0.1.0)

`MATRIZ_PROVIDER`, `GOOGLE_API_KEY` (this exact name — the Google SDK reads it),
`MATRIZ_PROJECT_ROOT`, `MATRIZ_MODEL_DRAFT`, `MATRIZ_MODEL_FINAL`,
`MATRIZ_BUDGET_USD` (default 2.00), `MATRIZ_MAX_GENERATIVE_CALLS` (default 20),
`MATRIZ_DRAFT_MAX_EDGE` (default 768). Details in §5.9.

## Conventions

| Aspect | Convention |
|---|---|
| Code, comments, tool names | English |
| Documentation (`docs/`, `README.md`) | Spanish |
| Commits | English, imperative, `feat:` / `fix:` / `test:` / `docs:` |
| Branches | `pr-0-scaffold`, `pr-1-deterministic`, … |
| Discipline | RED → implement → GREEN; the test comes before the implementation |
| Tests | Table-driven; image transforms use golden files compared byte for byte |

Contracts in §5 are copied **as-is**. Substitute the `<...>` placeholders; do
not "improve" the rest — every line has a decision behind it.

## Unverified assumptions

The handoff was written against a repository that did not exist, and **nothing
in it was compiled**. Treat these as open:

- No dependency in §5.2 was built. PR-0's first task is a spike that compiles a
  `main` importing all of them. If `gen2brain/avif` fails, the evaluated
  fallback is dropping AVIF from v0.1.0 — do not invent a third option.
- `mcp.ImageContent.Data` base64 handling is assumed, not observed. Confirm with
  A-01 before relying on it.
- Prices in §2.2 come from Feb–Aug 2026 sources. Re-check the official Gemini
  pricing page before filling `pricing.go`, and record the consultation date in
  `docs/proveedores.md`.

When reality diverges from the document, **do not adapt silently**: capture the
literal evidence (real output, exact error, exact version), record it in
`PENDING.md` with its condition, and decide it with the user. Reality beats the
document.
