import { expect, type Locator, type Page, type Request as PlaywrightRequest, type Response } from "@playwright/test"

export const liveAuthStatePath = "test-results/live-auth/admin.json"

const allowedMutationPaths = new Set([
  "/api/auth/refresh",
])

const backgroundAPIPaths = new Set([
  "/api/system/reddots",
])

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim()
  if (!value) throw new Error(`${name} is required for the admin live regression`)
  return value
}

export function liveEnvironment() {
  const baseURL = requiredEnvironment("E2E_ADMIN_BASE_URL")
  const expectedOrigin = requiredEnvironment("E2E_ADMIN_EXPECTED_ORIGIN")
  let baseOrigin: string
  let configuredOrigin: string
  try {
    baseOrigin = new URL(baseURL).origin
    configuredOrigin = new URL(expectedOrigin).origin
  } catch {
    throw new Error("E2E_ADMIN_BASE_URL and E2E_ADMIN_EXPECTED_ORIGIN must be absolute URLs")
  }
  if (baseOrigin !== configuredOrigin) {
    throw new Error(`Admin live regression target ${baseOrigin} does not match E2E_ADMIN_EXPECTED_ORIGIN`)
  }
  if (new URL(baseURL).protocol !== "https:") {
    throw new Error("Admin live regression target must use HTTPS")
  }

  return {
    baseURL,
    expectedOrigin: configuredOrigin,
    username: requiredEnvironment("E2E_ADMIN_USERNAME"),
    password: requiredEnvironment("E2E_ADMIN_PASSWORD"),
  }
}

async function openCasdoorLogin(page: Page, baseURL: string) {
  const adminOrigin = new URL(baseURL).origin
  const loginURL = new URL("/api/auth/login-url", adminOrigin)
  loginURL.searchParams.set("callback", new URL("/callback", adminOrigin).toString())

  const response = await page.context().request.get(loginURL.toString(), {
    headers: { Accept: "application/json" },
  })
  const payload = await response.json().catch(() => null) as { code?: unknown, data?: { url?: unknown } } | null
  if (response.status() !== 200 || payload?.code !== 200) {
    throw new Error(`Admin login URL request returned HTTP ${response.status()}`)
  }

  const rawCasdoorURL = payload.data?.url
  if (typeof rawCasdoorURL !== "string" || !rawCasdoorURL.trim()) {
    throw new Error("Admin login URL response did not include a URL")
  }

  let casdoorURL: URL
  try {
    casdoorURL = new URL(rawCasdoorURL)
  } catch {
    throw new Error("Admin login URL was not an absolute URL")
  }
  if (casdoorURL.origin === adminOrigin) {
    throw new Error("Admin login URL did not point to Casdoor")
  }

  await page.goto(casdoorURL.toString(), { waitUntil: "domcontentloaded" })
}

async function visibleLocator(page: Page, selectors: string, description: string) {
  const locator = page.locator(selectors).first()
  try {
    await locator.waitFor({ state: "visible", timeout: 20_000 })
  } catch {
    throw new Error(`Casdoor ${description} field was not found`)
  }
  return locator
}

async function submitLocator(page: Page): Promise<Locator> {
  const submit = page.locator('button[type="submit"], input[type="submit"]').first()
  if (await submit.isVisible().catch(() => false)) return submit

  const namedButton = page.getByRole("button", { name: /sign in|log in|login|登录/i }).first()
  try {
    await namedButton.waitFor({ state: "visible", timeout: 10_000 })
  } catch {
    throw new Error("Casdoor login button was not found")
  }
  return namedButton
}

export async function authenticateAdmin(page: Page) {
  const environment = liveEnvironment()
  const adminOrigin = new URL(environment.baseURL).origin

  await openCasdoorLogin(page, environment.baseURL)
  const username = await visibleLocator(
    page,
    [
      'input[name="username"]',
      'input[autocomplete="username"]',
      'input[id*="username" i]',
      'input[placeholder*="username" i]',
      'input[placeholder*="账号"]',
    ].join(","),
    "username",
  )
  const password = await visibleLocator(
    page,
    [
      'input[name="password"]',
      'input[autocomplete="current-password"]',
      'input[type="password"]',
    ].join(","),
    "password",
  )

  await username.fill(environment.username)
  await password.fill(environment.password)
  await (await submitLocator(page)).click()

  try {
    await page.waitForURL(
      (url) => url.origin === adminOrigin && url.pathname !== "/callback" && url.pathname !== "/login",
      { timeout: 90_000 },
    )
  } catch {
    const currentPath = new URL(page.url()).pathname
    throw new Error(`Casdoor login did not return to the admin portal; stopped at ${currentPath}`)
  }

  await expect.poll(
    () => page.evaluate(() => localStorage.getItem("is_authenticated")),
    { timeout: 20_000 },
  ).toBe("true")
  await page.evaluate(() => localStorage.setItem("app_lang", "zh"))

  const currentUser = await page.context().request.get(new URL("/api/user/me", environment.baseURL).toString())
  if (currentUser.status() !== 200) {
    throw new Error(`Authenticated admin check returned HTTP ${currentUser.status()}`)
  }
  const payload = await currentUser.json().catch(() => null)
  if (!payload || payload.code !== 200 || !payload.data) {
    throw new Error("Authenticated admin check returned an invalid response")
  }
}

function safeRequestLabel(rawURL: string, method?: string) {
  const url = new URL(rawURL)
  return `${method ? `${method} ` : ""}${url.pathname}`
}

export async function installReadOnlyGuards(page: Page) {
  const environment = liveEnvironment()
  const adminOrigin = new URL(environment.baseURL).origin
  const apiFailures: string[] = []
  const pageErrors: string[] = []
  const consoleErrors: string[] = []
  const requestFailures: string[] = []
  const resourceFailures: string[] = []
  const mutationAttempts: string[] = []
  const successfulAPIPaths = new Set<string>()
  const pendingRequests = new Set<PlaywrightRequest>()
  const responseInspections = new Set<Promise<void>>()
  let lastAPIActivityAt = Date.now()

  function isTracked(request: PlaywrightRequest) {
    const url = new URL(request.url())
    return url.origin === adminOrigin
      && url.pathname.startsWith("/api/")
      && !backgroundAPIPaths.has(url.pathname)
  }

  async function inspectAPIResponse(response: Response) {
    const url = new URL(response.url())
    if (url.origin !== adminOrigin || !url.pathname.startsWith("/api/")) return
    if (response.status() >= 400) {
      apiFailures.push(`${safeRequestLabel(response.url())} -> HTTP ${response.status()}`)
      return
    }

    const contentType = response.headers()["content-type"]?.toLowerCase() || ""
    if (!contentType.includes("json")) return
    const payload = await response.json().catch(() => null) as { code?: unknown } | null
    if (!payload || payload.code !== 200) {
      apiFailures.push(`${safeRequestLabel(response.url())} -> invalid success envelope`)
      return
    }
    successfulAPIPaths.add(url.pathname)
  }

  await page.route("**/*", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method().toUpperCase()
    const safeMethod = method === "GET" || method === "HEAD" || method === "OPTIONS"
    if (url.origin === adminOrigin && !safeMethod && !allowedMutationPaths.has(url.pathname)) {
      mutationAttempts.push(`${method} ${url.pathname}`)
      await route.abort("blockedbyclient")
      return
    }
    await route.continue()
  })
  page.on("request", (request) => {
    if (!isTracked(request)) return
    pendingRequests.add(request)
    lastAPIActivityAt = Date.now()
  })
  page.on("response", (response) => {
    const request = response.request()
    if (isTracked(request)) {
      pendingRequests.delete(request)
      lastAPIActivityAt = Date.now()
    }
    const url = new URL(response.url())
    if (url.origin === adminOrigin && !url.pathname.startsWith("/api/") && response.status() >= 400) {
      resourceFailures.push(`${safeRequestLabel(response.url(), request.method())} -> HTTP ${response.status()}`)
    }
    const inspection = inspectAPIResponse(response)
    responseInspections.add(inspection)
    void inspection.finally(() => responseInspections.delete(inspection))
  })
  page.on("requestfailed", (request) => {
    const url = new URL(request.url())
    if (url.origin !== adminOrigin) return
    pendingRequests.delete(request)
    lastAPIActivityAt = Date.now()
    if (request.failure()?.errorText === "net::ERR_BLOCKED_BY_CLIENT") return
    requestFailures.push(`${safeRequestLabel(request.url(), request.method())} -> request failed`)
  })
  page.on("pageerror", (error) => pageErrors.push(error.message))
  page.on("console", (message) => {
    if (message.type() !== "error") return
    consoleErrors.push(message.text())
  })

  return {
    async waitForAPIIdle(timeout = 45_000) {
      const deadline = Date.now() + timeout
      while (Date.now() < deadline) {
        if (pendingRequests.size === 0 && responseInspections.size === 0 && Date.now() - lastAPIActivityAt >= 1_000) return
        await page.waitForTimeout(250)
      }
      const pending = [...pendingRequests].map((request) => safeRequestLabel(request.url(), request.method()))
      throw new Error(`Admin API requests did not settle within ${timeout}ms: ${pending.join(", ")}`)
    },
    async assertRequested(pathname: string) {
      await expect.poll(
        () => successfulAPIPaths.has(pathname),
        { message: `expected a successful ${pathname} response`, timeout: 10_000 },
      ).toBe(true)
    },
    async assertClean() {
      expect(mutationAttempts, "read-only regression attempted a business mutation").toEqual([])
      expect(apiFailures, "admin API returned an unexpected error").toEqual([])
      expect(requestFailures, "admin API request failed").toEqual([])
      expect(resourceFailures, "admin page resource returned an unexpected error").toEqual([])
      expect(pageErrors, "admin page raised a JavaScript exception").toEqual([])
      expect(consoleErrors, "admin page wrote errors to the browser console").toEqual([])
    },
  }
}
