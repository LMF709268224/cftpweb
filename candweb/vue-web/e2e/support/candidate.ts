import type { Page, Route } from "@playwright/test"

export type ApiMockResponse = {
  status?: number
  code?: number
  message?: string
  data?: unknown
}

export type ApiMockContext = {
  pathname: string
  url: URL
  method: string
  body: unknown
}

export type ApiMockHandler = (
  context: ApiMockContext,
) => ApiMockResponse | undefined | Promise<ApiMockResponse | undefined>

export const candidateUser = {
  id: "candidate-e2e",
  name: "Regression Candidate",
  display_name: "Regression Candidate",
  email: "candidate-e2e@example.test",
  first_name: "Regression",
  last_name: "Candidate",
  gender: "male",
  birthday: "1990-01-01",
  phone: "+65 8000 0000",
  phone_country_code: "SG",
  country: "SG",
  region: "SG",
  city: "Singapore",
  address_text: "1 Regression Road",
  postal_code: "018956",
}

function responseCode(status: number, code?: number) {
  if (typeof code === "number") return code
  return status >= 200 && status < 300 ? 200 : status
}

async function fulfillApi(route: Route, response: ApiMockResponse) {
  const status = response.status ?? 200
  await route.fulfill({
    status,
    contentType: "application/json; charset=utf-8",
    body: JSON.stringify({
      code: responseCode(status, response.code),
      message: response.message ?? (status >= 200 && status < 300 ? "ok" : "request failed"),
      data: response.data ?? null,
    }),
  })
}

function requestBody(route: Route) {
  try {
    return route.request().postDataJSON()
  } catch {
    return route.request().postData()
  }
}

function defaultApiResponse(pathname: string): ApiMockResponse {
  switch (pathname) {
    case "/api/user/me":
      return { data: candidateUser }
    case "/api/messages/unread-count":
      return { data: { unread_count: 0 } }
    case "/api/credentials/actionable-count":
      return { data: { actionable_count: 0 } }
    case "/api/public/config":
      return { data: { stripe_publishable_key: "pk_test_playwright" } }
    case "/api/public/config/organization":
      return { data: { country_codes: ["SG", "CN", "US"] } }
    case "/api/auth/logout":
    case "/api/telemetry":
    case "/api/public/telemetry":
      return { data: { success: true } }
    default:
      return { data: {} }
  }
}

export async function seedAuthenticatedCandidate(page: Page) {
  await page.addInitScript(() => {
    localStorage.setItem("is_authenticated", "true")
    localStorage.setItem("user_name", "Regression Candidate")
    localStorage.setItem("app_lang", "zh")

    const expiresAt = Date.now() + 60_000
    localStorage.setItem("candidate_unread_count_cache", JSON.stringify({ value: 0, expiresAt }))
    localStorage.setItem("candidate_actionable_credential_count_cache", JSON.stringify({ value: 0, expiresAt }))
  })
}

export async function installCandidateApiMocks(page: Page, handler?: ApiMockHandler) {
  await page.route("**/api/**", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const context: ApiMockContext = {
      pathname: url.pathname,
      url,
      method: request.method(),
      body: requestBody(route),
    }
    const customResponse = await handler?.(context)
    await fulfillApi(route, customResponse ?? defaultApiResponse(url.pathname))
  })
}

export async function installStripeCheckoutMock(page: Page) {
  await page.route("https://js.stripe.com/v3/**", async (route) => {
    await route.fulfill({
      status: 200,
      contentType: "application/javascript; charset=utf-8",
      body: `
        window.Stripe = function () {
          return {
            initEmbeddedCheckout: async function (options) {
              return {
                mount: function (selector) {
                  var container = document.querySelector(selector);
                  if (!container) throw new Error("Stripe mount container not found");
                  var button = document.createElement("button");
                  button.type = "button";
                  button.dataset.testid = "fake-stripe-complete";
                  button.textContent = "完成测试支付";
                  button.style.margin = "24px";
                  button.style.padding = "12px 20px";
                  button.addEventListener("click", function () {
                    options.onComplete();
                  });
                  container.appendChild(button);
                },
                destroy: function () {},
                unmount: function () {}
              };
            }
          };
        };
      `,
    })
  })
}
