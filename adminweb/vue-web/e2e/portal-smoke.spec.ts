import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const adminPages = [
  { path: "/dashboard", endpoint: "/api/dashboard/ops" },
  { path: "/resource-packs", endpoint: "/api/lms/resource-packs" },
  { path: "/resource-pack-files", endpoint: "/api/lms/resource-pack-files" },
  { path: "/lms", endpoint: "/api/lms/courses" },
  { path: "/pipelines", endpoint: "/api/pipelines" },
  { path: "/bundles", endpoint: "/api/mall/bundles" },
  { path: "/memberships", endpoint: "/api/memberships/configs" },
  { path: "/prog", endpoint: "/api/prog/pipelines" },
  { path: "/exams", endpoint: "/api/exams" },
  { path: "/messages", endpoint: "/api/messages/templates" },
  { path: "/mails", endpoint: "/api/mails/templates" },
  { path: "/orders", endpoint: "/api/mall/orders" },
  { path: "/invoices", endpoint: "/api/mall/invoices" },
  { path: "/credentials", endpoint: "/api/credentials/definitions" },
  { path: "/applications", endpoint: "/api/applications" },
  { path: "/pdf-templates", endpoint: "/api/pdf-templates" },
  { path: "/pdf-requests", endpoint: "/api/pdf-requests" },
  { path: "/admin-ops", endpoint: "/api/pay/subscriptions" },
  { path: "/audit/logs", endpoint: "/api/audit/logs" },
  { path: "/audit/webhooks", endpoint: "/api/audit/webhooks" },
  { path: "/permissions", endpoint: "/api/credentials/definitions" },
  { path: "/settings", endpoint: "/api/user/me" },
]

for (const adminPage of adminPages) {
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
