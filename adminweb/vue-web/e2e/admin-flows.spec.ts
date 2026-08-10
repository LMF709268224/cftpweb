import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("message and mail navigation badges use independent counts", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/system/reddots") {
      return {
        data: {
          applications: 0,
          exams: 0,
          prog: 0,
          orders: 0,
          invoices: 0,
          messages: 2,
          mails: 5,
        },
      }
    }
    return undefined
  })

  await page.goto("/dashboard")

  const messageBadge = page.locator('a[href="/messages"] span').last()
  const mailBadge = page.locator('a[href="/mails"] span').last()
  await expect(messageBadge).toHaveText("2")
  await expect(mailBadge).toHaveText("5")
})
