import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const membershipSummary = {
  membership_ulid: "membership-1",
  membership_gpath: "/memberships/regression",
  name: "Regression Membership",
  description: "Read-only membership summary",
  duration_in_months: 12,
  tier_level: 2,
  status: "Active",
  version: 4,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installMembershipReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/memberships/configs") {
      return { data: { memberships: [membershipSummary], has_more: false, next_cursor: "", prev_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/memberships/membership-1") {
      return {
        data: {
          ...membershipSummary,
          description: "Read-only membership detail",
          features_json: JSON.stringify(["Priority support", "Course discount"]),
          ideal_for: "Regression users",
          casdoor_role_name: "member-pro",
        },
      }
    }
    return undefined
  })
}

test("membership list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMembershipReadMocks(page, requests)

  await page.goto("/memberships")

  await expect(page.getByText("Regression Membership", { exact: true })).toBeVisible()
  await expect(page.getByText("Read-only membership summary", { exact: true })).toBeVisible()
  await expect(page.getByText("membership-1", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/memberships/configs")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("membership detail reads configuration fields without editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMembershipReadMocks(page, requests)
  await page.goto("/memberships")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  await expect(page.getByRole("heading", { name: "Regression Membership" })).toBeVisible()
  await expect(page.getByText("Read-only membership detail", { exact: true })).toBeVisible()
  await expect(page.getByText("member-pro", { exact: true })).toBeVisible()
  await expect(page.getByText("Regression users", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/memberships/membership-1")
  expect(requests.some((request) => request.includes("/deprecate") || request.startsWith("POST ") || request.startsWith("PUT ") || request.startsWith("PATCH ") || request.startsWith("DELETE "))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
