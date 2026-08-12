import { expect, test } from "@playwright/test"
import { adminPageChecks } from "../e2e/support/admin-pages"
import { installReadOnlyGuards, liveEnvironment } from "./support/live"

test.setTimeout(900_000)

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

for (const adminPage of adminPageChecks) {
  test(`${adminPage.path} reads its primary API without mutations`, async ({ page }) => {
    const guards = await installReadOnlyGuards(page)
    await page.goto(adminPage.path, { waitUntil: "domcontentloaded" })

    await expect(page).toHaveURL(new RegExp(`${adminPage.path.replaceAll("/", "\\/")}(?:[?#].*)?$`))
    await expect(page.locator("h1").first()).toBeVisible({ timeout: 45_000 })
    await guards.waitForAPIIdle()
    await guards.assertRequested(adminPage.endpoint)
    await guards.assertClean()
  })
}
