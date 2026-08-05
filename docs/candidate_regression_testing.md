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
- the candidate Vue production build
- Playwright regression tests with mocked API responses

These checks must not create orders, schedule exams, submit credential
applications, or modify shared test accounts.

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
- registration of critical candidate routes
- authentication requirements for protected routes
- malformed authentication cookies
- allowed and rejected CORS origins

Handler tests continue to use fake gRPC clients for downstream behavior.

## API Coverage Inventory

The candidate BFF currently registers 98 HTTP method and route combinations.
Coverage is tracked by route group so missing areas remain visible.

| Route group                                     | Routes | Current automated coverage                                    |
| ----------------------------------------------- | -----: | ------------------------------------------------------------- |
| Health, public config, webhook, telemetry, auth |      9 | Partial handler coverage and router smoke coverage            |
| Protected preview endpoints                     |      4 | Authentication boundary coverage                              |
| Public membership and mall catalog              |     11 | Partial pipeline and runtime handler coverage                 |
| User profile                                    |      5 | Authentication and cookie coverage; profile mutations pending |
| Membership                                      |      4 | Pending                                                       |
| Mall purchase and payment                       |      7 | Partial stage, return URL, and payment-state coverage         |
| Pipeline and progress                           |     12 | Partial access, runtime, and progress coverage                |
| Enrollments                                     |      2 | Pending                                                       |
| Resource packs and files                        |      5 | Pending                                                       |
| Quizzes                                         |      6 | Partial downstream-error coverage                             |
| Exams                                           |     10 | Partial history, retake, and callback coverage                |
| Credential applications                         |      9 | Partial candidate-scoped application coverage                 |
| Certificates                                    |      1 | Pending                                                       |
| Orders                                          |      3 | Partial status and cancellation coverage                      |
| Invoices                                        |      2 | URL validation and PDF extraction coverage                    |
| Messages                                        |      5 | Partial pagination response coverage                          |
| Dashboard                                       |      2 | Pending                                                       |

The first expansion priority is authentication, orders, payment, exams, and
credential applications. Statement coverage is a supporting metric, not a
replacement for business-flow assertions.

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
