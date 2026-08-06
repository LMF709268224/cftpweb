# Candidate Portal E2E Regression

This suite runs the candidate portal in Chromium and replaces Casdoor, Stripe,
and backend APIs with deterministic browser-level mocks. It is intended for
fast pull-request regression checks, not for validating third-party uptime.

## Requirements

- Node.js 20 or newer
- Dependencies installed with `npm install`

## Commands

```powershell
npm run test:e2e:install
npm run test:e2e
```

Use `npm run test:e2e:headed` to watch the browser and
`npm run test:e2e:report` to open the latest HTML report.

## Current coverage

- Protected-route login redirect and return-path preservation
- Expired access token refresh
- Expired refresh token session cleanup
- Order status and action consistency
- Hosted payment return synchronization
- Embedded Stripe completion without leaving the orders page
- Complete certification checkout through mocked Stripe and appearance in My Certifications
- Complete course lesson and quiz submission journey
- Empty-data rendering and JavaScript crash detection for the 12 main
  candidate pages

When a candidate UI behavior changes, add a test that fails against the old
behavior. New main pages should also be added to `portal-smoke.spec.ts`.

Real Casdoor, Stripe test-mode, and exam-provider smoke tests belong in a
separate staging suite so this fast regression suite remains repeatable.

The live suite also provides the manually gated `live-journey` project for a
dedicated Stripe Test Mode certification purchase and course-learning account:

```powershell
npm run test:e2e:live:journey
```

It reuses the existing candidate live-test account, selects the first paid,
currently purchasable certification that contains a course, disables default
course exemptions, and continues into the certification created by that
payment. The run creates a paid test order and irreversible candidate progress,
so Admin cleanup and account-state reset are required before rerunning it.

## Live read-only regression

The separate `e2e-live` suite signs in through the real Casdoor test application
and reads the candidate test environment. It does not replace this deterministic
mock suite.

Required environment variables:

- `E2E_CANDIDATE_BASE_URL`
- `E2E_CANDIDATE_USERNAME`
- `E2E_CANDIDATE_PASSWORD`

Run it only against the shared candidate test environment:

```powershell
npm run test:e2e:live
```

The suite blocks unexpected non-read API requests. Only session refresh and
telemetry requests may use non-GET methods. Traces, videos, and automatic
screenshots are disabled so authentication cookies and tokens are not placed in
test artifacts.
