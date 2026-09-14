# Dubbo Call Map Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a fast, dedicated Dubbo call-map screen backed by server-side filtering, grouping, and pagination.

**Architecture:** Extend persisted link decoding with contract metadata, expose a bounded v2 Dubbo report endpoint, and render service-centric list and focused call-map views in React. Existing generic graph and links surfaces remain unchanged.

**Tech Stack:** Go 1.26, net/http ServeMux, React 18, TypeScript, TanStack Query, existing Grafel UI primitives, Vitest.

**Spec:** `docs/superpowers/specs/2026-09-11-dubbo-call-map.md`

## Global Constraints

- Do not download the complete cross-repo link set on the Dubbo screen.
- Default page size is 25 and maximum page size is 100.
- Only exact fully-qualified Dubbo contracts appear in the dedicated screen.
- Existing Graph, Links, Topology, and MCP behavior must remain compatible.
- Preserve deterministic ordering for stable tests and UI rendering.

---

### Task 1: Preserve Dubbo Link Metadata

**Files:**
- Modify: `internal/dashboard/graphstate.go`
- Test: `internal/dashboard/graphstate_test.go`

**Interfaces:**
- Produces: `CrossRepoLink.Identifier string` and `CrossRepoLink.Properties map[string]string`.

- [ ] Add a failing JSON decoding test for `identifier` and `properties`.
- [ ] Run the focused dashboard test and confirm the fields are absent.
- [ ] Extend `CrossRepoLink` and its custom unmarshaler.
- [ ] Re-run the focused test and confirm it passes.

### Task 2: Add Dubbo Report API

**Files:**
- Create: `internal/dashboard/handlers_dubbo.go`
- Create: `internal/dashboard/handlers_dubbo_test.go`
- Modify: `internal/dashboard/server.go`

**Interfaces:**
- Consumes: enriched `CrossRepoLink` metadata.
- Produces: `GET /api/v2/groups/{group}/dubbo` returning summary, facets, paginated services, and endpoint details.

- [ ] Add failing parser tests for interface, group, version, protocol, method, and arity.
- [ ] Add failing handler tests for exact selection, filtering, deterministic grouping, facets, and pagination.
- [ ] Implement contract parsing and report data types.
- [ ] Implement bounded filtering before endpoint enrichment.
- [ ] Register the v2 route and make the handler tests pass.

### Task 3: Add Frontend Data Contract

**Files:**
- Modify: `webui-v2/src/data/types.ts`
- Modify: `webui-v2/src/lib/api.ts`
- Create: `webui-v2/src/hooks/use-dubbo.ts`

**Interfaces:**
- Produces: `DubboReport`, `DubboService`, `DubboEndpoint`, `DubboFilters`, and `useDubboReport`.

- [ ] Define TypeScript response and filter types matching the Go response.
- [ ] Add a query-string API client that omits empty filters.
- [ ] Add a TanStack Query hook keyed by group and all filter values.
- [ ] Run TypeScript checking to catch wire-shape drift.

### Task 4: Build Focused Dubbo UI

**Files:**
- Create: `webui-v2/src/routes/dubbo.tsx`
- Create: `webui-v2/src/lib/dubbo-call-map.ts`
- Create: `webui-v2/src/lib/dubbo-call-map.test.ts`
- Modify: `webui-v2/src/routes/router.tsx`
- Modify: `webui-v2/src/components/chrome/nav-rail.tsx`

**Interfaces:**
- Consumes: `useDubboReport` and `DubboService`.
- Produces: `/g/:groupId/dubbo` with list and focused call-map modes.

- [ ] Add failing unit tests for endpoint deduplication and bounded call-map columns.
- [ ] Implement the pure call-map view-model builder.
- [ ] Build summary cards, filters, pagination, service cards, and focused map.
- [ ] Add source details and graph deep-links for consumer/provider declarations.
- [ ] Register the route and navigation item.
- [ ] Run frontend tests and type checking.

### Task 5: Build, Install, and Verify

**Files:**
- Generated: `webui-v2/dist/**`
- Generated: `internal/dashboard/dist/**`
- Output: `build/grafel-full.exe`

**Interfaces:**
- Produces: locally installed Grafel binary with the Dubbo screen.

- [ ] Run focused Go and frontend tests.
- [ ] Run full frontend lint, tests, and Vite build.
- [ ] Copy the dashboard bundle and run `cmd/verify-dashboard`.
- [ ] Build the CGO-enabled Windows binary with embedded UI.
- [ ] Run Go vet, coverage gate, quality ratchet, and diff checks.
- [ ] Back up and replace the installed Grafel binary.
- [ ] Restart one daemon with one supervised engine.
- [ ] Verify the real `ossac-platform` Dubbo page and network payload in Chrome.
