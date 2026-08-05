import { expect, type Locator, type Page } from "@playwright/test"

export const liveAuthStatePath = "test-results/live-auth/candidate.json"

const allowedMutationPaths = new Set([
  "/api/auth/refresh",
  "/api/telemetry",
  "/api/public/telemetry",
])

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

  await page.goto("/login", { waitUntil: "domcontentloaded" })
  try {
    await page.waitForURL((url) => url.origin !== candidateOrigin, { timeout: 30_000 })
  } catch {
    throw new Error("Candidate portal did not redirect to Casdoor")
  }

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
      { timeout: 60_000 },
    )
  } catch {
    throw new Error("Casdoor login did not return to the candidate portal")
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
  const mutationAttempts: string[] = []
  const apiFailures: string[] = []
  const pageErrors: string[] = []
  const consoleErrors: string[] = []
  const requestFailures: string[] = []

  await page.route("**/api/**", async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const method = request.method().toUpperCase()
    const safeMethod = method === "GET" || method === "HEAD" || method === "OPTIONS"

    if (url.origin !== candidateOrigin || safeMethod || allowedMutationPaths.has(url.pathname)) {
      await route.continue()
      return
    }

    mutationAttempts.push(safeRequestLabel(request.url(), method))
    await route.abort("blockedbyclient")
  })

  page.on("response", (response) => {
    const url = new URL(response.url())
    if (url.origin !== candidateOrigin || !url.pathname.startsWith("/api/")) return
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
    if (request.failure()?.errorText === "net::ERR_BLOCKED_BY_CLIENT") return
    requestFailures.push(`${safeRequestLabel(request.url(), request.method())} -> request failed`)
  })

  return {
    assertClean() {
      expect(mutationAttempts, "read-only regression attempted a business mutation").toEqual([])
      expect(apiFailures, "candidate API returned an unexpected error").toEqual([])
      expect(requestFailures, "candidate API request failed").toEqual([])
      expect(pageErrors, "candidate page raised a JavaScript exception").toEqual([])
      expect(consoleErrors, "candidate page wrote errors to the browser console").toEqual([])
    },
  }
}
