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

## Initial coverage

- Protected-route login redirect and return-path preservation
- Expired access token refresh
- Expired refresh token session cleanup
- Order status and action consistency
- Hosted payment return synchronization
- Embedded Stripe completion without leaving the orders page

Real Casdoor, Stripe test-mode, and exam-provider smoke tests belong in a
separate staging suite so this fast regression suite remains repeatable.
