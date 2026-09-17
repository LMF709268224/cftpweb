# Admin audit event contract

Admin BFF publishes one audit event after every authenticated request that can
change business state. Read-only requests, previews, rendering, existence
checks, and temporary upload/view URL requests are deliberately excluded.

The authoritative route taxonomy is `adminbff/server/audit.go`. A route test
fails when a new protected `POST`, `PUT`, `PATCH`, or `DELETE` endpoint is not
explicitly classified as audited or excluded.

## Search fields

These fields are the stable search contract. Do not put environment names,
display labels, IDs, or request-specific values into `action` or
`resource_type`.

| Field | Admin BFF rule | Example |
| --- | --- | --- |
| `source_service` | Always the emitting service name | `adminbff` |
| `action` | Lowercase `verb_object` name; route semantics, not HTTP method | `publish_course` |
| `resource_type` | Lowercase singular domain noun | `course` |
| `resource_id` | Canonical business ID/ULID; empty only when the API exposes no ID | `01J...` |
| `operator_id` | Internal administrator ULID resolved during authentication | `01J...` |
| `status` | `SUCCESS` for HTTP `< 400`, otherwise `FAILED` | `SUCCESS` |
| `created_at` | Request start time in UTC RFC3339Nano | `2026-09-17T08:00:00Z` |

Use `source_service + action` to find an operation family, and
`resource_type + resource_id` to reconstruct one resource's history. Use
`operator_id` for an administrator's activity and `status` for failed attempts.
An application rejection that completes normally therefore uses action
`reject_application` with status `SUCCESS`; `status` describes execution of the
admin command, not the resulting business state.

## Field semantics

- `summary_text` is human-readable English derived from the canonical action.
  It is useful for the keyword box, but it is not a category key.
- `operator` includes the authenticated administrator ID, name, email, and the
  fixed role `admin`. `gaudit` therefore does not need to guess the actor.
- `context` contains client IP, User-Agent, trace/request ID, request path, and
  optional client geo header.
- `details` uses schema `admin.audit.v1` and contains only transport metadata:
  HTTP method, route pattern, status, request ID, duration, safe error code, and
  route parameters.
- Raw request bodies and response bodies are never persisted. This prevents
  passwords, tokens, mail contents, and other secrets from leaking into audit
  storage. Bodies are inspected in memory only to locate allowlisted resource
  ID fields when a create endpoint returns its new ID.

## Naming rules

Actions and resource types must match `^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`.
Actions use a specific business verb such as `create`, `update`, `delete`,
`publish`, `deprecate`, `grant`, `revoke`, `retry`, `ignore`, `sync`, `import`,
or `reorder`. Do not use generic values such as `POST`, `UPDATE`, `operate`, or
localized display text.

Resource types remain stable across actions. For example, create, update,
publish, and delete operations for an LMS course all use `course`. Similar
names from different domains are disambiguated where necessary, such as
`certification_pipeline` and `candidate_pipeline`.

Changing an existing action or resource type breaks historical filters and
must be treated as an interface migration. Add a new action only when the
business meaning is genuinely different.

## BFF and domain-service events

An `adminbff` event records the administrator command and its HTTP outcome.
A domain microservice may additionally publish a richer event describing the
committed state transition or old/new field diff. Those are not duplicates:
their different `source_service` values distinguish command audit from domain
audit. Domain services should reuse the same action and resource vocabulary
where the meanings match.
