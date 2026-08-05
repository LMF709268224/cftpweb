import { expect, type Locator, type Page, type Request as PlaywrightRequest } from "@playwright/test"

export const liveAuthStatePath = "test-results/live-auth/candidate.json"

const allowedMutationPaths = new Set([
  "/api/auth/refresh",
  "/api/telemetry",
  "/api/public/telemetry",
])

const backgroundAPIPaths = new Set([
  "/api/telemetry",
  "/api/public/telemetry",
  "/api/messages/unread-count",
  "/api/credentials/actionable-count",
])

async function installMutationBlocker(page: Page, candidateOrigin: string) {
  await page.addInitScript(
    ({ origin, allowedPaths }) => {
      const mutationAttempts: string[] = []
      const testWindow = window as typeof window & { __candidateMutationAttempts?: string[] }
      testWindow.__candidateMutationAttempts = mutationAttempts

      const allowed = new Set(allowedPaths)
      const originalFetch = window.fetch

      window.fetch = function candidateReadOnlyFetch(input, init) {
        const request = input instanceof Request ? input : null
        const method = String(init?.method || request?.method || "GET").toUpperCase()
        const rawURL = request?.url || String(input)
        const url = new URL(rawURL, window.location.origin)
        const safeMethod = method === "GET" || method === "HEAD" || method === "OPTIONS"

        if (url.origin === origin && url.pathname.startsWith("/api/") && !safeMethod && !allowed.has(url.pathname)) {
          const label = `${method} ${url.pathname}`
          mutationAttempts.push(label)
          return Promise.reject(new DOMException(`Blocked by candidate read-only regression: ${label}`, "AbortError"))
        }

        return originalFetch.call(window, input, init)
      }
    },
    {
      origin: candidateOrigin,
      allowedPaths: [...allowedMutationPaths],
    },
  )
}

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim()
  if (!value) {
    throw new Error(`${name} is required for the candidate live read-only regression`)
  }
  return value
}

export function liveEnvironment() {
  return {
    baseURL: requiredEnvironment("E2E_CANDIDATE_BASE_URL"),
    username: requiredEnvironment("E2E_CANDIDATE_USERNAME"),
    password: requiredEnvironment("E2E_CANDIDATE_PASSWORD"),
  }
}

async function openCasdoorLogin(page: Page, baseURL: string) {
  const candidateOrigin = new URL(baseURL).origin
  const callbackURL = new URL("/callback", candidateOrigin).toString()
  const loginURL = new URL("/api/auth/login-url", candidateOrigin)
  loginURL.searchParams.set("callback", callbackURL)

  await page.goto(candidateOrigin, { waitUntil: "domcontentloaded" })
  const result = await page.evaluate(async (endpoint) => {
    const response = await fetch(endpoint, {
      credentials: "include",
      headers: { Accept: "application/json" },
    })
    const contentType = response.headers.get("content-type") || "missing"
    const text = await response.text()
    let payload: unknown = null
    try {
      payload = JSON.parse(text)
    } catch {
      // The caller reports a sanitized shape instead of exposing the response body.
    }
    return { status: response.status, contentType, payload }
  }, loginURL.toString())

  if (result.status !== 200) {
    throw new Error(`Candidate login URL request returned HTTP ${result.status}`)
  }

  const payload = result.payload as { code?: unknown, data?: { url?: unknown } } | null
  const rawCasdoorURL = payload?.data?.url
  if (payload?.code !== 200 || typeof rawCasdoorURL !== "string" || !rawCasdoorURL.trim()) {
    const topLevelKeys = payload && typeof payload === "object" ? Object.keys(payload).sort().join(",") : "non-json"
    const dataKeys = payload?.data && typeof payload.data === "object"
      ? Object.keys(payload.data).sort().join(",")
      : "none"
    throw new Error(
      `Candidate login URL response was invalid; content-type=${result.contentType}; `
      + `keys=${topLevelKeys || "none"}; code-type=${typeof payload?.code}; data-keys=${dataKeys || "none"}`,
    )
  }

  let casdoorURL: URL
  try {
    casdoorURL = new URL(rawCasdoorURL)
  } catch {
    throw new Error("Candidate login URL was not an absolute URL")
  }
  if (casdoorURL.origin === candidateOrigin) {
    throw new Error("Candidate login URL did not point to Casdoor")
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

export async function authenticateCandidate(page: Page) {
  const environment = liveEnvironment()
  const candidateOrigin = new URL(environment.baseURL).origin

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
      (url) => url.origin === candidateOrigin && url.pathname !== "/callback" && url.pathname !== "/login",
      { timeout: 90_000 },
    )
  } catch {
    const currentPath = new URL(page.url()).pathname
    throw new Error(`Casdoor login did not return to the candidate portal; stopped at ${currentPath}`)
  }

  await expect.poll(
    () => page.evaluate(() => localStorage.getItem("is_authenticated")),
    { timeout: 20_000 },
  ).toBe("true")

  await page.evaluate(() => localStorage.setItem("app_lang", "zh"))

  const currentUser = await page.context().request.get(new URL("/api/user/me", environment.baseURL).toString())
  if (currentUser.status() !== 200) {
    throw new Error(`Authenticated candidate check returned HTTP ${currentUser.status()}`)
  }
  const payload = await currentUser.json().catch(() => null)
  if (!payload || payload.code !== 200 || !payload.data) {
    throw new Error("Authenticated candidate check returned an invalid response")
  }
}

function safeRequestLabel(rawURL: string, method?: string) {
  const url = new URL(rawURL)
  return `${method ? `${method} ` : ""}${url.pathname}`
}

function isIgnoredOptionalResponse(pathname: string, status: number) {
  if (allowedMutationPaths.has(pathname)) return true
  return status === 404 && pathname.includes("/thumbnail-url")
}

export async function installReadOnlyGuards(page: Page) {
  const environment = liveEnvironment()
  const candidateOrigin = new URL(environment.baseURL).origin
  const apiFailures: string[] = []
  const pageErrors: string[] = []
  const consoleErrors: string[] = []
  const requestFailures: string[] = []
  const requestedAPIPaths = new Map<string, string>()
  const completedAPIPaths = new Set<string>()
  let lastAPIActivityAt = Date.now()

  function trackedAPIPath(request: PlaywrightRequest) {
    const url = new URL(request.url())
    const tracked = url.origin === candidateOrigin
      && url.pathname.startsWith("/api/")
      && !backgroundAPIPaths.has(url.pathname)
    return tracked ? url.pathname : ""
  }

  await installMutationBlocker(page, candidateOrigin)

  page.on("request", (request) => {
    const pathname = trackedAPIPath(request)
    if (!pathname || requestedAPIPaths.has(pathname)) return
    requestedAPIPaths.set(pathname, safeRequestLabel(request.url(), request.method()))
    lastAPIActivityAt = Date.now()
  })
  page.on("response", (response) => {
    const url = new URL(response.url())
    if (url.origin !== candidateOrigin || !url.pathname.startsWith("/api/")) return
    if (!backgroundAPIPaths.has(url.pathname) && !completedAPIPaths.has(url.pathname)) {
      completedAPIPaths.add(url.pathname)
      lastAPIActivityAt = Date.now()
    }
    if (response.status() < 400 || isIgnoredOptionalResponse(url.pathname, response.status())) return
    apiFailures.push(`${safeRequestLabel(response.url())} -> HTTP ${response.status()}`)
  })
  page.on("pageerror", (error) => pageErrors.push(error.message))
  page.on("console", (message) => {
    if (message.type() !== "error") return
    if (message.text().startsWith("Failed to load resource:")) return
    consoleErrors.push(message.text())
  })
  page.on("requestfailed", (request) => {
    const url = new URL(request.url())
    if (url.origin !== candidateOrigin || !url.pathname.startsWith("/api/")) return
    if (!backgroundAPIPaths.has(url.pathname) && !completedAPIPaths.has(url.pathname)) {
      completedAPIPaths.add(url.pathname)
      lastAPIActivityAt = Date.now()
    }
    if (request.failure()?.errorText === "net::ERR_BLOCKED_BY_CLIENT") return
    requestFailures.push(`${safeRequestLabel(request.url(), request.method())} -> request failed`)
  })

  return {
    reset() {
      apiFailures.length = 0
      pageErrors.length = 0
      consoleErrors.length = 0
      requestFailures.length = 0
      requestedAPIPaths.clear()
      completedAPIPaths.clear()
      lastAPIActivityAt = Date.now()
    },
    async waitForAPIIdle(timeout = 45_000) {
      const deadline = Date.now() + timeout
      let idleSince = Date.now()

      while (Date.now() < deadline) {
        const outstanding = [...requestedAPIPaths.keys()].filter((pathname) => !completedAPIPaths.has(pathname))
        if (outstanding.length === 0) {
          if (Date.now() - Math.max(idleSince, lastAPIActivityAt) >= 1_000) return
        } else {
          idleSince = Date.now()
        }
        await page.waitForTimeout(250)
      }

      const pending = [...requestedAPIPaths]
        .filter(([pathname]) => !completedAPIPaths.has(pathname))
        .map(([, label]) => label)
      throw new Error(`Candidate API requests did not settle within ${timeout}ms: ${pending.join(", ")}`)
    },
    async assertClean() {
      const mutationAttempts = await page.evaluate(() => {
        const testWindow = window as typeof window & { __candidateMutationAttempts?: string[] }
        return testWindow.__candidateMutationAttempts || []
      })
      expect(mutationAttempts, "read-only regression attempted a business mutation").toEqual([])
      expect(apiFailures, "candidate API returned an unexpected error").toEqual([])
      expect(requestFailures, "candidate API request failed").toEqual([])
      expect(pageErrors, "candidate page raised a JavaScript exception").toEqual([])
      expect(consoleErrors, "candidate page wrote errors to the browser console").toEqual([])
    },
  }
}
