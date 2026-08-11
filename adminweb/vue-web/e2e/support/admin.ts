import type { Page } from "@playwright/test"

export type ApiMockContext = {
  method: string
  pathname: string
  url: URL
}

export type ApiMockResult = {
  status?: number
  data?: unknown
  errorCode?: string
  message?: string
  delayMs?: number
}

export type ApiMockResolver = (context: ApiMockContext) => ApiMockResult | undefined

function emptyAdminData(pathname: string) {
  if (pathname === "/api/system/reddots") {
    return {
      applications: 0,
      exams: 0,
      prog: 0,
      orders: 0,
      invoices: 0,
      messages: 0,
      mails: 0,
    }
  }

  return {
    items: [],
    list: [],
    courses: [],
    packs: [],
    files: [],
    pipelines: [],
    bundles: [],
    memberships: [],
    exams: [],
    messages: [],
    mails: [],
    orders: [],
    invoices: [],
    definitions: [],
    credentials: [],
    applications: [],
    templates: [],
    requests: [],
    webhooks: [],
    events: [],
    users: [],
    logs: [],
    tasks: [],
    stages: [],
    units: [],
    subscriptions: [],
    total: 0,
    total_count: 0,
    count: 0,
    has_more: false,
    next_cursor: "",
    prev_cursor: "",
    next_page_token: "",
    candidate_total: 0,
    user_stats: {
      total: 0,
      active: 0,
      inactive: 0,
      admins: 0,
      members: 0,
      email_verified: 0,
    },
    user_role_stats: [],
    profile_completion_percent: 0,
    user_total: 0,
    user_page: 1,
    user_page_size: 10,
    stage_buckets: [],
    today_revenue: [],
    generated_at: "2026-08-10T00:00:00Z",
    id: "admin-1",
    name: "Regression Admin",
    email: "admin@example.test",
  }
}

export async function installAdminApiMocks(page: Page, resolver?: ApiMockResolver) {
  const requestedPaths = new Set<string>()

  await page.route("**/api/**", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    requestedPaths.add(url.pathname)

    const result = resolver?.({
      method: request.method(),
      pathname: url.pathname,
      url,
    })
    if (result?.delayMs) await new Promise((resolve) => setTimeout(resolve, result.delayMs))
    const status = result?.status ?? 200
    const payload = status >= 400
      ? {
          code: status,
          error_code: result?.errorCode || "REQUEST_FAILED",
          message: result?.message || "Request failed",
        }
      : {
          code: status,
          error_code: "OK",
          message: "OK",
          data: result?.data ?? emptyAdminData(url.pathname),
        }

    await route.fulfill({ status, contentType: "application/json", json: payload })
  })

  return { requestedPaths }
}

export async function seedAuthenticatedAdmin(page: Page, lang: "zh" | "en" = "zh") {
  await page.addInitScript((selectedLang) => {
    localStorage.setItem("is_authenticated", "true")
    localStorage.setItem("user_name", "Regression Admin")
    localStorage.setItem("app_lang", selectedLang)
  }, lang)
}
