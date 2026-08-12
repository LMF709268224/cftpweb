import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"
import { adminPageChecks } from "./support/admin-pages"

for (const adminPage of adminPageChecks) {
  test(`${adminPage.path} renders its empty state without browser errors`, async ({ page }) => {
    await seedAuthenticatedAdmin(page)
    const pageErrors: string[] = []
    const consoleErrors: string[] = []
    page.on("pageerror", (error) => pageErrors.push(error.message))
    page.on("console", (message) => {
      if (message.type() === "error") consoleErrors.push(message.text())
    })
    const { requestedPaths } = await installAdminApiMocks(page)

    await page.goto(adminPage.path)

    await expect(page).toHaveURL(new RegExp(`${adminPage.path.replaceAll("/", "\\/")}$`))
    await expect.poll(() => requestedPaths.has(adminPage.endpoint)).toBe(true)
    expect(pageErrors).toEqual([])
    expect(consoleErrors).toEqual([])
    await expect(page.locator("h1").first()).toBeVisible({ timeout: 15_000 })
  })
}
