import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("external courseware detail displays configuration and token inventory", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
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
          base_url: "https://partner.example/learn/{token}",
          token_param_name: "auth_token",
          updated_at: "2026-08-26T00:00:00Z",
        },
      }
    }
    if (method === "GET" && pathname === "/api/lms/external-coursewares/courseware-1/token-stats") {
      return { data: { total_count: 10, available_count: 7, allocated_count: 3 } }
    }
    if (method === "GET" && pathname === "/api/lms/external-coursewares/courseware-1/tokens") {
      return { data: { items: [{ token_id: "token-1", token: "abcd1234secret", candidate_ulid: "candidate-1" }] } }
    }
    return undefined
  })

  await page.goto("/external-coursewares")
  await expect(page.getByRole("heading", { name: "Partner Academy" })).toBeVisible()
  await expect(page.getByText("https://partner.example/learn/{token}", { exact: true })).toBeVisible()
  await expect(page.getByText("auth_token", { exact: true })).toBeVisible()
  await expect(page.getByText("abcd...cret", { exact: true })).toBeVisible()
  await expect(page.getByText("candidate-1", { exact: true })).toBeVisible()
})
