# Candidate Regression Testing

## Purpose

The candidate system uses layered regression tests. No single test layer is
expected to validate the browser, BFF, downstream services, Casdoor, and Stripe
at the same time.

## Automatic Push Checks

Every push to `v2` or `main`, and every pull request targeting those branches,
runs:

- `go vet ./...` in `candbff`
- `go test ./...` in `candbff`
- a 20% minimum statement-coverage floor for `candbff/handler` and
  `candbff/server`
- the candidate Vue production build
- Playwright regression tests with mocked API responses

These checks must not create orders, schedule exams, submit credential
applications, or modify shared test accounts.

## Test Locations

Regression tests are grouped by layer instead of being placed throughout the
application:

- `candbff/handler/*_test.go` contains focused Handler business tests. Each
  business module uses its corresponding test file, such as
  `payment_test.go`, `invoice_test.go`, `membership_test.go`, and
  `message_test.go`.
- `candbff/server/router_test.go` contains the complete HTTP route contract,
  authentication-boundary, CORS, health, and not-found tests.
- `candweb/vue-web/e2e/*.spec.ts` contains deterministic browser regression
  tests. Shared browser API mocks remain under `candweb/vue-web/e2e/support`.
- `.github/workflows/candidate-e2e.yml` is the single GitHub Actions entry
  point that runs the backend and frontend layers.

Tests must not be added to unrelated production directories. A new business
module should receive a focused `*_test.go` or `*.spec.ts` file in the
appropriate location above rather than extending one global test file.

## Private Go Module Access

`candbff` depends on the private repository
`github.com/afnandelfin620-star/cftptest`. GitHub Actions cannot download this
module with the workflow repository's built-in token.

Create the repository secret `CFTPTEST_READ_TOKEN` with a fine-grained token
that can read only the `cftptest` repository. The token needs repository
`Contents: Read-only` permission. Do not grant write, administration, Actions,
or organization permissions.

The workflow limits the authenticated Git URL rewrite to the single
`cftptest` repository. The token must not be stored in source files, local test
data, workflow YAML, or test artifacts.

## HTTP Regression Scope

The `candbff/server` tests cover the HTTP boundary without connecting to real
microservices:

- health response
- JSON 404 response
- an exact contract for all 98 HTTP method and route combinations
- authentication requirements for all 78 protected routes
- malformed authentication cookies
- allowed and rejected CORS origins

Handler tests continue to use fake gRPC clients for downstream behavior.
The coverage floor prevents large regressions, but it does not replace route,
authorization, state-transition, or ownership assertions.

## API Coverage Inventory

The candidate BFF currently registers 98 HTTP method and route combinations.
Coverage is tracked by route group so missing areas remain visible.

| Route group                                     | Routes | Current automated coverage                                          |
| ----------------------------------------------- | -----: | ------------------------------------------------------------------- |
| Health, public config, webhook, telemetry, auth |      9 | Full route contract; public config, callback, cookie, refresh, and logout coverage |
| Protected preview endpoints                     |      4 | Full route and authentication boundary coverage                     |
| Public membership and mall catalog              |     11 | Full route contract; partial pipeline and runtime coverage          |
| User profile                                    |      5 | Input validation, normalization, authentication, and cookie coverage |
| Membership                                      |      4 | Plan filtering, candidate scope, pagination, and cancellation coverage |
| Mall purchase and payment                       |      7 | Partial stage, return URL, and payment-state coverage               |
| Pipeline and progress                           |     12 | Partial access, runtime, and progress coverage                      |
| Enrollments                                     |      2 | Candidate scope, filters, pagination, detail, and validation coverage |
| Resource packs and files                        |      5 | Candidate scope, ownership, pagination, preview, and view coverage  |
| Quizzes                                         |      6 | Candidate forwarding, answer preservation, and validation coverage |
| Exams                                           |     10 | History, retake, callback, and request-validation coverage          |
| Credential applications                         |      9 | Candidate scope and request-validation coverage                     |
| Certificates                                    |      1 | Candidate scope, credential enrichment, and file mapping coverage   |
| Orders                                          |      3 | Filters, totals, detail ownership, status, and cancellation coverage |
| Invoices                                        |      2 | Ownership, completed-state, query, trusted URL, and PDF coverage    |
| Messages                                        |      5 | Candidate scope, rendering, pagination, unread, read, and delete coverage |
| Dashboard                                       |      2 | Candidate-scoped aggregate response coverage                        |

The remaining isolated-test priorities are deeper successful flows for mall
purchase, pipeline access, exams, and credential applications. Statement
coverage is a supporting metric, not a replacement for business-flow
assertions.

## Candidate UI Regression Scope

The deterministic Playwright suite covers:

- protected-route login redirect and post-login return-path preservation
- expired access-token refresh and expired refresh-token cleanup
- order status and action consistency
- hosted and embedded payment completion returning to the orders page
- empty-data rendering for the 12 main candidate pages: dashboard,
  marketplace, my certifications, exams, records, resource packs, credential
  applications, certificates, membership, orders, messages, and settings
- JavaScript page-crash detection for those main candidate pages

These tests use browser-level API and Stripe mocks. They validate candidate UI
behavior and request handling without requiring shared test accounts or
creating real business records.

## Change Requirements

Every candidate-system change must keep the regression suite current:

- A new, removed, renamed, or method-changed BFF route must update
  `candidateRouteContract`. The exact route-contract test rejects untracked
  route changes.
- New Handler behavior must add focused tests for successful forwarding,
  invalid input, candidate ownership or authorization, downstream errors, and
  state transitions where those cases apply.
- New candidate UI behavior must add or update a Playwright test when the
  behavior can be reproduced with deterministic mocks.
- Every bug fix must include a regression test that would fail without the
  fix.
- Tests that mutate real data belong in the live-environment suite and must
  define cleanup behavior before they are automated.

The pull request template repeats these requirements so reviewers can see
which layers were updated.

## Live Environment Scope

Live tests target the candidate test environment and use credentials supplied
through environment variables or GitHub Actions secrets:

- `E2E_CANDIDATE_BASE_URL`
- `E2E_CANDIDATE_USERNAME`
- `E2E_CANDIDATE_PASSWORD`

Credentials must never be stored in tracked files, workflow YAML, test output,
screenshots, traces, or browser videos.

Live read-only smoke tests may run on a schedule. Tests that create or mutate
business data must be manually triggered until automatic cleanup is available.

## Test Account Policy

Use separate candidate accounts for different purposes:

- smoke account: login and read-only page checks
- payment account: order creation, cancellation, and Stripe test payments
- business account: exam registration and credential application flows

Tests must not change account passwords, account email addresses, or production
data.

## Order Cleanup Policy

Unpaid test orders should be cancelled by the candidate test after assertions
complete.

Paid orders cannot be cancelled by the candidate. A live payment test must
record the order ID and completion time in its test result. Paid-order cleanup
remains a manual Admin operation until the Admin cancellation contract and
refund behavior are explicitly verified.

Do not automate Admin login with a broad-privilege account. If automatic cleanup
is required later, use a dedicated least-privilege test account or a reviewed
test-only cleanup API.
