import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("external courseware detail displays and imports dedicated access URLs", async ({ page }) => {
  let importBody: unknown
  await seedAuthenticatedAdmin(page)
  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/lms/external-coursewares/courseware-1/tokens/import") {
      importBody = request.postDataJSON()
    }
  })
  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method === "GET" && pathname === "/api/lms/external-coursewares") {
      return { data: { items: [{ courseware_ulid: "courseware-1", name: "Partner Academy" }] } }
    }
    if (method === "GET" && pathname === "/api/lms/external-coursewares/courseware-1") {
      return {
        data: {
          courseware_ulid: "courseware-1",
          name: "Partner Academy",
          description: "Token protected learning content",
          base_url: "https://partner.example/learn",
          updated_at: "2026-08-26T00:00:00Z",
        },
      }
    }
    if (method === "GET" && pathname === "/api/lms/external-coursewares/courseware-1/token-stats") {
      return { data: { total_count: 10, available_count: 7, allocated_count: 3 } }
    }
    if (method === "GET" && pathname === "/api/lms/external-coursewares/courseware-1/tokens") {
      return { data: { items: [{ token_id: "token-1", token_url: "https://partner.example/token/abcd1234secret", candidate_ulid: "candidate-1" }] } }
    }
    if (method === "POST" && pathname === "/api/lms/external-coursewares/courseware-1/tokens/import") {
      return { data: { imported_count: 1, duplicate_count: 0 } }
    }
    return undefined
  })

  await page.goto("/external-coursewares")
  await expect(page.getByRole("heading", { name: "Partner Academy" })).toBeVisible()
  await expect(page.getByText("https://partner.example/learn", { exact: true })).toBeVisible()
  await expect(page.getByText("http...cret", { exact: true })).toBeVisible()
  await expect(page.getByText("candidate-1", { exact: true })).toBeVisible()

  await page.getByRole("button", { name: "导入专属 URL" }).click()
  await page.locator("textarea[placeholder^='https://partner.example/token/user-001']").fill("https://partner.example/token/new-user")
  await page.getByRole("button", { name: "导入", exact: true }).click()
  await expect.poll(() => importBody).toEqual({ token_urls: ["https://partner.example/token/new-user"] })
})
