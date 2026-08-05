## Change summary

Describe the user-visible behavior and the affected candidate or Admin flows.

## Regression checklist

- [ ] New or changed BFF routes are reflected in `candidateRouteContract`.
- [ ] New or changed Handler logic has success, validation, authorization, and downstream-error tests where applicable.
- [ ] New or changed candidate UI behavior has a Playwright regression test where applicable.
- [ ] Every bug fix includes a test that fails without the fix.
- [ ] New business mutations document test-data creation and cleanup requirements.
- [ ] Live-environment testing remains separate from deterministic push checks.
- [ ] `go vet ./...`, `go test ./...`, the candidate build, and Playwright pass locally or in GitHub Actions.

If a checkbox does not apply, explain why in the change summary.
