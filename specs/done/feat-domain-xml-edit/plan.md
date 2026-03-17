# feat-domain-xml-edit Plan

## Overview

Add domain XML editing (libvirt domain define) for claimed VMs: API for fetch/update, validation (XML schema, libvirt, safety checks), and UI editor with save flow and error handling. Greenfield; no migration; no stubs. Aligns with existing `patchVMConfig` patterns (auth, audit, VM-stopped precondition).

## Tech Stack

- **Backend**: Go, chi router, libvirtconn, libvirtxml
- **Frontend**: Vanilla TS + Vite, modal pattern (CreateVMModal/CloneVMModal)
- **DB**: SQLite (audit_events), Git (audit diffs)

## Architecture

```
┌─────────────────┐     GET/PUT domain-xml      ┌──────────────────┐
│  DomainXMLEditor │ ◄────────────────────────► │  routes handler   │
│  (modal)         │                            │  getDomainXML     │
└─────────────────┘                            │  putDomainXML     │
        │                                       └─────────┬────────┘
        │                                                 │
        │                                                 ▼
        │                                       ┌──────────────────┐
        │                                       │ domainxml pkg    │
        │                                       │ - ValidateSafe   │
        │                                       │ - RejectForbidden│
        │                                       └─────────┬────────┘
        │                                                 │
        │                                                 ▼
        │                                       ┌──────────────────┐
        │                                       │ libvirtconn      │
        │                                       │ GetDomainXML     │
        │                                       │ DefineXML        │
        │                                       └──────────────────┘
```

## Components

| Component | Description | Tech |
|-----------|-------------|------|
| **getDomainXML** | HTTP handler: fetch domain XML for a VM | routes, libvirtconn.GetDomainXML |
| **putDomainXML** | HTTP handler: validate, define domain XML | routes, domainxml.ValidateSafe, libvirtconn.DefineXML |
| **domainxml** | Validation package: safety checks, forbidden elements | internal/domainxml |
| **DomainXMLEditor** | Modal: textarea editor, Save/Cancel, error display | web/src/components/DomainXMLEditor.ts |
| **VMList** | Add "Edit XML" action to VM row | web/src/components/VMList.ts |

## APIs

### GET /api/hosts/{host_id}/vms/{libvirt_uuid}/domain-xml

**Auth**: JWT required (same as patchVMConfig).

**Response**: `200 OK` with `Content-Type: application/xml`; body = raw domain XML string.

**Errors**:
- `401` unauthorized
- `404` VM not found (not claimed or host missing)
- `500` GetDomainXML failed

### PUT /api/hosts/{host_id}/vms/{libvirt_uuid}/domain-xml

**Auth**: JWT required.

**Request**: `Content-Type: application/xml`; body = raw domain XML string.

**Preconditions**:
- VM must be stopped (DomainStateShutoff). Return `400` with message "VM must be stopped to edit domain XML" otherwise.
- UUID in submitted XML must match `libvirt_uuid` path param. Reject `400` if mismatch.

**Validation order**:
1. **Parse**: Unmarshal into `libvirtxml.Domain`. Fail `400` with "invalid domain XML" on parse error.
2. **Safety**: Reject forbidden elements (see below). Fail `400` with "domain XML contains forbidden elements: <list>".
3. **Libvirt**: Call `conn.DefineXML`. Fail `400` or `409` with libvirt error message on define failure.

**Response**: `200 OK` with `Content-Type: application/json`; body = `vmDetailResponse` (same as patchVMConfig).

**Audit**: `vm_config_change` with Git diff; path `audit/vm/<host_id>/<libvirt_uuid>/<timestamp>.diff`. Diff format: domain XML before/after (unified diff).

**Errors**:
- `400` VM running, UUID mismatch, parse error, safety rejection, libvirt define error
- `401` unauthorized
- `404` VM not found
- `409` libvirt conflict (e.g., duplicate name)

## Data Models

No schema changes. `vm_metadata` unchanged. Domain XML is transient (libvirt); audit stores diffs in Git.

## Validation

### Domain XML Safety Checks

Reject XML containing any of these elements (namespace-aware):

| Element | Namespace | Reason |
|---------|-----------|--------|
| `qemu:commandline` | `http://libvirt.org/schemas/domain/qemu/1.0` | Arbitrary QEMU args; exec bypass |
| `qemu:arg` | same | Part of qemu:commandline |
| `qemu:env` | same | Arbitrary env vars to QEMU |
| `init` (domain init) | `http://libvirt.org/schemas/domain/qemu/1.0` | Arbitrary init script path |

**Implementation**: Parse XML with `encoding/xml` or `libvirtxml.Domain`; walk DOM for forbidden elements. Return explicit list of forbidden elements found.

**UUID check**: After unmarshaling, compare `domain.UUID` with path param. If different, reject.

### Libvirt Validation

`conn.DefineXML` performs libvirt validation. Invalid XML (schema, semantic errors) returns libvirt error. Surface error message to client (sanitized if needed).

## Security

| Area | Approach |
|------|----------|
| Authn | JWT via `mw.UserFromContext`; same as patchVMConfig |
| Authz | VM must be claimed; host must exist; connector from config |
| Input | Reject forbidden elements before DefineXML; validate UUID |
| Audit | Full before/after diff in Git; vm_config_change audit_events row |
| Output | No sensitive data in domain XML response (operator-managed) |

## Performance

- No size limit on domain XML in MVP; typical domain XML is &lt;10KB. If abuse observed, add configurable max size (e.g., 256KB).
- Single GET/PUT per edit; no polling.

## Risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| User edits break VM (e.g., wrong disk path) | VM unusable | Libvirt DefineXML validates; operator must fix via libvirt. Document in UI. |
| Forbidden elements list incomplete | Potential exec bypass | Start with qemu:commandline/env; add others as needed; security-auditor review |
| Large XML DoS | Memory/CPU | Consider max body size in chi middleware (e.g., 256KB) |

## Testing

- **Unit**: `internal/domainxml/validate_test.go` — forbidden elements, UUID check, parse errors
- **Integration**: `internal/routes/routes_test.go` — getDomainXML, putDomainXML (success, VM running, forbidden XML, UUID mismatch, parse error, 404)
- **E2E**: Manual; no automated E2E in scope
- **Verification**: `go test ./...`, `make all`

## Rollout

- **Config**: None. Uses existing hosts, JWT, auth.
- **Deploy**: Standard deploy; no migration.

## Decision Log

| Decision | Alternatives | Rationale |
|----------|--------------|-----------|
| Separate GET/PUT endpoints for domain XML | Extend patchVMConfig with `domain_xml` field | Clear separation; domain XML is large; different content-type (XML vs JSON) |
| Reject qemu:commandline, qemu:env | Allow with sanitization | Sanitization is fragile; reject is safer (libvirt secure usage) |
| VM must be stopped | Allow live edit | Libvirt DefineXML on running domain requires specific flags; stopped is simpler and matches patchVMConfig |
| UUID must match | Allow UUID change | Changing UUID would create new domain; reject to avoid accidental undefine+redefine |
| ValidateSafe in new package | Inline in routes | Reusable; testable; single responsibility |

## Ownership

**In-scope**:
- `internal/routes/routes.go` — getDomainXML, putDomainXML handlers
- `internal/domainxml/` — new package (ValidateSafe, forbidden elements)
- `web/src/components/DomainXMLEditor.ts` — new component
- `web/src/components/VMList.ts` — add Edit XML action
- `web/src/lib/api.ts` — fetchDomainXML, putDomainXML
- `web/src/main.ts` — wire openDomainXMLEditor
- `internal/routes/routes_test.go` — handler tests

**Out-of-scope**:
- `internal/libvirtconn` — no changes
- `internal/audit` — no changes (uses existing RecordEventWithDiff)
- `internal/db` — no changes

## Assumptions

- Operators editing domain XML have libvirt knowledge; no guided wizard for XML.
- Forbidden elements list is minimal; expand if security audit finds more risks.
- No syntax highlighting in MVP; plain textarea for domain XML.

## Open Questions

- None; scope is complete for MVP.

---

## Approval Checklist

- [ ] Scope matches intent (no extra features)
- [ ] File paths and ownership are clear
- [ ] Data model is correct and safe (greenfield: no migration/backfill)
- [ ] Authn/authz + context scoping are correct
- [ ] API contracts are specified (requests/responses/errors)
- [ ] Test plan + verification steps are included
- [ ] Rollout/ops notes are sufficient (if needed)

---

## Changelog

- 2026-03-16: Initial plan
