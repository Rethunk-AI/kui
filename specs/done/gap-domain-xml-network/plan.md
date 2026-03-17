# Gap #8 — Domain XML Edit: Network Validation

## Overview

Domain XML edit (`putDomainXML`) accepts arbitrary XML and applies it via `conn.DefineXML`. Unlike `patchVMConfig`, it does not validate that referenced networks exist on the host. A user can change a network interface to a non-existent network; libvirt fails on `DefineXML` with an opaque error. This plan adds pre-apply network validation, matching the `patchVMConfig` pattern (lines 916–932 in `internal/routes/routes.go`).

**Greenfield only.** No migration paths, no backwards compatibility.

---

## Architecture

```
┌─────────────────┐     PUT domain-xml      ┌──────────────────────────┐
│  DomainXMLEditor │ ───────────────────►  │  putDomainXML handler     │
└─────────────────┘                        │  1. ValidateSafe          │
                                            │  2. NetworksFromDomain    │
                                            │  3. ListNetworks + check  │
                                            │  4. DefineXML            │
                                            └────────────┬─────────────┘
                                                         │
                        ┌────────────────────────────────┼────────────────────────────────┐
                        ▼                                ▼                                ▼
               ┌─────────────────┐            ┌─────────────────┐            ┌─────────────────┐
               │ domainxml pkg   │            │ libvirtconn      │            │ audit           │
               │ ValidateSafe   │            │ ListNetworks     │            │ RecordEventWithDiff
               │ NetworksFromDomain (new)     │ DefineXML        │            │                 │
               └─────────────────┘            └─────────────────┘            └─────────────────┘
```

**Flow:**
1. `ValidateSafe` — parse, UUID match, forbidden elements (unchanged).
2. **New:** Extract network names from domain XML via `domainxml.NetworksFromDomain`.
3. **New:** Call `conn.ListNetworks`; for each extracted network, verify it exists on the host.
4. If any network not found → return `409` with `"network invalid or does not exist on host"`.
5. `DefineXML` — apply (unchanged).

---

## Scope

| In scope | Out of scope |
|----------|--------------|
| Validate network-type interfaces (`source network="X"`) in edited domain XML before `DefineXML` | Bridge-type interfaces (`source bridge="Y"`) — host bridges not validated |
| Same error message and status code as `patchVMConfig` (409) | Template network validation (Gap #7) |
| Unit tests for `NetworksFromDomain` | Migration, backfill, backwards compatibility |

---

## Reference: patchVMConfig Pattern

From `internal/routes/routes.go` lines 916–932:

```go
if strings.TrimSpace(payload.Network) != "" {
    networks, err := conn.ListNetworks(req.Context())
    if err != nil {
        writeJSONError(w, http.StatusInternalServerError, "failed to list networks")
        return
    }
    found := false
    for _, n := range networks {
        if n.Name == payload.Network {
            found = true
            break
        }
    }
    if !found {
        writeJSONError(w, http.StatusConflict, "network invalid or does not exist on host")
        return
    }
}
```

---

## Implementation Tasks

### Task 1 — Add `NetworksFromDomain` to domainxml package

**Where:** `internal/domainxml/validate.go` (append), `internal/domainxml/validate_test.go`

**Implementation:**

1. Add function:
   ```go
   // NetworksFromDomain parses domain XML and returns network names from interface
   // source network elements. Returns error on parse failure. Only network-type
   // interfaces are considered; bridge-type are ignored.
   func NetworksFromDomain(xmlStr string) ([]string, error)
   ```
2. Unmarshal into `libvirtxml.Domain`; if error, return `fmt.Errorf("invalid domain XML")`.
3. Walk `dom.Devices.Interfaces`; for each interface with `Source != nil` and `Source.Network != nil`, collect non-empty `Source.Network.Network` into a slice (deduplicate if desired; validation will pass if all exist).
4. Return the slice.

**Acceptance criteria:**

- [x] Valid XML with no network interfaces → `[]string{}`, nil
- [x] Valid XML with one `source network="default"` → `[]string{"default"}`, nil
- [x] Valid XML with multiple network interfaces → returns all network names
- [x] Invalid XML → error
- [x] Bridge-type interfaces ignored (no panic, not in result)

**Files:** `internal/domainxml/validate.go`, `internal/domainxml/validate_test.go`

---

### Task 2 — Validate networks in putDomainXML before DefineXML

**Where:** `internal/routes/routes.go` — `putDomainXML()` (lines 1103–1179)

**Implementation:**

1. After `domainxml.ValidateSafe(afterXML, libvirtUUID)` (line 1146) and before `conn.GetDomainXML` (line 1149):
   - Call `networks, err := domainxml.NetworksFromDomain(afterXML)`
   - If `err != nil` → already handled by ValidateSafe (should not occur; NetworksFromDomain uses same parse). For robustness, return 400 with err.Error().
   - For each `net` in `networks`:
     - Call `conn.ListNetworks(req.Context())`
     - If error → return 500 with `"failed to list networks"`
     - Check if `net` exists in list (match by `n.Name`)
     - If not found → return 409 with `"network invalid or does not exist on host"`
   - Optimization: call `ListNetworks` once before the loop; reuse the result for all networks.
2. Use same status code (409) and message as `patchVMConfig` (line 929).

**Acceptance criteria:**

- [x] Domain XML with valid network (exists on host) → DefineXML proceeds
- [x] Domain XML with non-existent network → 409, `"network invalid or does not exist on host"`
- [x] Domain XML with no network interfaces → no validation, DefineXML proceeds
- [x] ListNetworks error → 500, `"failed to list networks"`

**Files:** `internal/routes/routes.go`

---

### Task 3 — Add putDomainXML network validation tests

**Where:** `internal/routes/routes_test.go`

**Implementation:**

1. **TestPutDomainXML_NetworkNotFound:**
   - Mock connector with `networks: []libvirtconn.NetworkInfo{{Name: "bridge0"}}` (no "default")
   - Domain XML with `<interface type="network"><source network="default"/></interface>`
   - Expect 409, body contains `"network invalid or does not exist on host"`
2. **TestPutDomainXML_NetworkValid:**
   - Mock connector with `networks: []libvirtconn.NetworkInfo{{Name: "default"}}`
   - Domain XML with interface using `source network="default"`
   - Expect 200
3. **TestPutDomainXML_NoNetworkInterfaces:**
   - Domain XML with no `<devices><interface>...</interface></devices>` (or only disk)
   - Expect 200 (no network validation needed)

**Note:** Existing `authHandlerWithClaimedVM` and similar helpers may need to set `mock.networks` for putDomainXML tests. Ensure `TestPutDomainXML_Success` (or equivalent) passes with mock that includes the network used in the domain XML, or uses domain XML without network interfaces.

**Files:** `internal/routes/routes_test.go`

---

## Acceptance Criteria (Overall)

- [x] `NetworksFromDomain` correctly extracts network names from domain XML
- [x] putDomainXML rejects edited XML with non-existent network (409)
- [x] putDomainXML accepts edited XML with valid network (200)
- [x] putDomainXML accepts edited XML with no network interfaces (200)
- [x] Error message matches patchVMConfig: `"network invalid or does not exist on host"`
- [x] Status code 409 for invalid network (matches patchVMConfig)

---

## Verification Steps

1. `go test ./internal/domainxml/...` — passes
2. `go test ./internal/routes/...` — passes (including new putDomainXML network tests)
3. `make all` — passes
4. Manual: Edit domain XML in UI, change network to non-existent name → expect clear error message

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| Validate in putDomainXML | Document libvirt validates | Gap specifies prefer validation; same pattern as patchVMConfig; clearer UX |
| 409 for invalid network | 400 | Matches patchVMConfig (line 929) |
| NetworksFromDomain in domainxml | Inline in routes | Reusable, testable, single responsibility |
| Validate only network-type interfaces | Validate bridge too | Gap is about network; bridge validation is different (host bridge existence) |

---

## Changelog

- 2026-03-17: Initial plan (Gap #8 from gap-audit)
