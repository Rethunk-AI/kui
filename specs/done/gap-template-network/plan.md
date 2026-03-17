# Gap #7 — Template Create: Network from Template

## Overview

When creating a VM from a template (or cloning from a template), the template's domain.xml and meta.yaml reference a network. If that network does not exist on the target host, libvirt fails on `DefineXML` with an opaque error. This plan adds pre-validation: before create/clone, validate the template's network exists on the target host; return 400 with a clear error if not.

**Greenfield only.** No migration paths, no backwards compatibility.

**Reference:** `patchVMConfig` pattern (lines 916–932 in `internal/routes/routes.go`).

---

## Architecture

```
┌─────────────────────┐     POST create-from-template      ┌──────────────────────────────┐
│  CreateVMModal      │ ─────────────────────────────────► │  createVMFromTemplate handler │
│  (template mode)    │     { template_id, host_id, ... }  │  1. Load meta.yaml, domain.xml│
└─────────────────────┘                                    │  2. Resolve network          │
                                                           │  3. ListNetworks + validate  │
                                                           │  4. Create volume, DefineXML │
                                                           └────────────┬────────────────┘
                                                                        │
                        ┌───────────────────────────────────────────────┼───────────────────────────────────┐
                        ▼                                               ▼                                   ▼
               ┌─────────────────┐                          ┌─────────────────┐                  ┌─────────────────┐
               │ template pkg    │                          │ libvirtconn     │                  │ Git (templates/) │
               │ ParseMeta       │                          │ ListNetworks    │                  │ meta.yaml       │
               │ (domainxml)     │                          │ ValidatePool    │                  │ domain.xml      │
               │ extractNetwork  │                          │ CopyVolume      │                  │                 │
               └─────────────────┘                          │ DefineXML       │                  └─────────────────┘
                                                           └─────────────────┘
```

**Flow:**
1. Load template meta.yaml and domain.xml from `<git_base>/templates/<template_id>/`.
2. Resolve network: prefer `meta.Network` if non-empty; else extract first network from domain.xml interfaces.
3. Connect to target host; call `conn.ListNetworks(ctx)`.
4. If resolved network not in list → return 400 with `"network invalid or does not exist on host"`.
5. Proceed with pool validation, volume creation, domain XML build, DefineXML.

---

## Scope

| In scope | Out of scope |
|----------|--------------|
| Create VM from template endpoint with network validation | vmClone (clone from VM) — uses hardcoded "default"; separate flow |
| Network validation before DefineXML (patchVMConfig pattern) | Domain XML edit network validation (Gap #8) |
| Return 400 for invalid network (per gap: "return 400 with clear error") | Bridge-type interfaces |
| Unit tests for create-from-template network validation | Migration, backfill, backwards compatibility |

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

**Note:** This plan uses 400 (Bad Request) for invalid network, per gap requirement ("return 400 with clear error"). patchVMConfig uses 409; for create-from-template, 400 is correct (invalid input).

---

## Implementation Tasks

### Task 1 — Add create-from-template handler and route

**Where:** `internal/routes/routes.go`

**Implementation:**

1. Add route: `router.Post("/api/templates/{template_id}/create", state.createVMFromTemplate())` (after line 271, near other template routes).
2. Add `createVMFromTemplateRequest` struct:
   ```go
   type createVMFromTemplateRequest struct {
       HostID      string `json:"host_id"`
       TargetPool  string `json:"target_pool"`
       DisplayName string `json:"display_name"`
   }
   ```
3. Add `createVMFromTemplate()` handler:
   - Parse template_id from URL.
   - Decode request body.
   - Load meta.yaml and domain.xml from `<git_base>/templates/<template_id>/`.
   - Resolve target host (host_id or config default).
   - Connect to target host via `getConnectorForHost`.
   - **Network validation (Task 2)** — before pool/volume logic.
   - Validate target pool via `conn.ValidatePool`.
   - Create volume from template base image (copy/clone to target pool).
   - Build domain XML from template domain.xml: substitute disk path, set name/uuid, apply meta CPU/RAM/network.
   - `conn.DefineXML`; insert vm_metadata; record audit; return 201.

**Acceptance criteria:**

- [x] POST `/api/templates/{template_id}/create` creates VM on target host
- [x] Template not found → 404
- [x] Invalid request body → 400

**Files:** `internal/routes/routes.go`

---

### Task 2 — Validate template network on target host before create

**Where:** `internal/routes/routes.go` — `createVMFromTemplate()` handler

**Implementation:**

1. After connecting to target host and before pool validation:
   - Resolve network: if `meta.Network != ""` use it; else parse domain.xml into `libvirtxml.Domain`, walk `dom.Devices.Interfaces`, take first `Source.Network.Network` if present; else default to `"default"`.
   - Call `networks, err := conn.ListNetworks(req.Context())`.
   - If error → return 500 with `"failed to list networks"`.
   - Check if resolved network exists in list (match by `n.Name`).
   - If not found → return 400 with `"network invalid or does not exist on host"`.
2. Use same pattern as patchVMConfig (lines 916–932), but return 400 (not 409) per gap requirement.

**Acceptance criteria:**

- [x] Template with network that exists on target host → create proceeds
- [x] Template with network that does not exist on target host → 400, `"network invalid or does not exist on host"`
- [x] ListNetworks error → 500, `"failed to list networks"`
- [x] Network resolved from meta.Network when present; else from domain.xml first interface

**Files:** `internal/routes/routes.go`

---

### Task 3 — Add helper to extract network from template

**Where:** `internal/template/template.go` or `internal/domainxml/validate.go`

**Implementation:**

1. Add function (in `internal/domainxml` if `NetworksFromDomain` exists from Gap #8; else in `internal/template`):
   ```go
   // NetworkFromTemplate returns the network name from template meta and domain XML.
   // Prefers meta.Network if non-empty; else first network from domain interfaces; else "default".
   func NetworkFromTemplate(meta *template.Meta, domainXML string) (string, error)
   ```
2. If `domainxml.NetworksFromDomain` exists, use it to get networks from domainXML; take first. Else: unmarshal into `libvirtxml.Domain`, walk interfaces, extract first `Source.Network.Network`.
3. Return trimmed, non-empty string; default `"default"` if none found.

**Acceptance criteria:**

- [ ] meta.Network = "bridge0" → "bridge0"
- [ ] meta.Network empty, domain has `source network="custom"` → "custom"
- [ ] Both empty / no interfaces → "default"
- [ ] Invalid domain XML → error

**Files:** `internal/template/template.go` (or `internal/domainxml/validate.go` if NetworksFromDomain exists), `internal/template/template_test.go`

---

### Task 4 — Add createVMFromTemplate network validation tests

**Where:** `internal/routes/routes_test.go`

**Implementation:**

1. **TestCreateVMFromTemplate_NetworkNotFound:**
   - Mock connector with `networks: []libvirtconn.NetworkInfo{{Name: "bridge0"}}` (no "default").
   - Template with meta.Network = "default" (or domain.xml with `source network="default"`).
   - Expect 400, body contains `"network invalid or does not exist on host"`.
2. **TestCreateVMFromTemplate_NetworkValid:**
   - Mock connector with `networks: []libvirtconn.NetworkInfo{{Name: "default"}}`.
   - Template with meta.Network = "default".
   - Expect 201 (or appropriate success path).
3. **TestCreateVMFromTemplate_TemplateNotFound:**
   - template_id that does not exist in git.
   - Expect 404.

**Files:** `internal/routes/routes_test.go`

---

## Acceptance Criteria (Overall)

- [ ] Create VM from template validates template's network exists on target host before DefineXML.
- [ ] Invalid or non-existent network returns 400 with `"network invalid or does not exist on host"`.
- [ ] Valid network proceeds to create; VM is defined and metadata inserted.
- [ ] Network resolution: meta.Network preferred; else domain.xml first interface; else "default".
- [ ] Error message is clear and matches patchVMConfig wording.
- [ ] Status code 400 for invalid network (per gap requirement).

---

## Verification Steps

1. `go test ./internal/template/...` — passes (NetworkFromTemplate if in template pkg)
2. `go test ./internal/domainxml/...` — passes (if NetworkFromTemplate in domainxml)
3. `go test ./internal/routes/...` — passes (including createVMFromTemplate network tests)
4. `make all` — passes
5. Manual: Create VM from template with network that does not exist on target host → expect 400 with clear error

---

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| 400 for invalid network | 409 (like patchVMConfig) | Gap explicitly: "return 400 with clear error" |
| New endpoint POST /api/templates/{id}/create | Extend POST /api/vms with template_id | Cleaner separation; createVM is pool+path; template create is distinct flow |
| Resolve network from meta then domain | Domain only | meta.Network is canonical in template spec; domain may have stale value |
| NetworkFromTemplate helper | Inline in handler | Reusable, testable; aligns with domainxml.NetworksFromDomain pattern |

---

## Changelog

- 2026-03-17: Initial plan (Gap #7 from gap-audit)
