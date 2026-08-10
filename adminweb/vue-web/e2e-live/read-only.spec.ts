import { expect, test } from "@playwright/test"
import { installReadOnlyGuards, liveEnvironment } from "./support/live"

test.setTimeout(900_000)

const adminPages = [
  "/dashboard",
  "/applications",
  "/exams",
  "/prog",
  "/lms",
  "/pipelines",
  "/bundles",
  "/memberships",
  "/resource-packs",
  "/resource-pack-files",
  "/credentials",
  "/pdf-templates",
  "/orders",
  "/invoices",
  "/messages",
  "/mails",
  "/permissions",
  "/admin-ops",
  "/audit/logs",
  "/audit/webhooks",
  "/pdf-requests",
  "/settings",
]

test("real admin session can read health and current-user APIs", async ({ page }) => {
  const environment = liveEnvironment()
  const healthResponse = await page.context().request.get(new URL("/health", environment.baseURL).toString())
  expect(healthResponse.status()).toBe(200)

  const userResponse = await page.context().request.get(new URL("/api/user/me", environment.baseURL).toString())
  expect(userResponse.status()).toBe(200)
  const payload = await userResponse.json()
  expect(payload.code).toBe(200)
  expect(payload.data).toBeTruthy()
  expect(String(payload.data.name || payload.data.id || "").trim()).not.toBe("")
})

test("admin main pages can read real test-environment data without mutations", async ({ page }) => {
  const guards = await installReadOnlyGuards(page)

  for (const path of adminPages) {
    await test.step(`${path} page`, async () => {
      guards.reset()
      await page.goto(path, { waitUntil: "domcontentloaded" })

      await expect(page).toHaveURL(new RegExp(`${path.replaceAll("/", "\\/")}(?:[?#].*)?$`))
      await expect(page.locator("h1").first()).toBeVisible({ timeout: 45_000 })
      await guards.waitForAPIIdle()
      await guards.assertClean()
    })
  }
})
