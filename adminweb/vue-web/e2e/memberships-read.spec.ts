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

const deprecatedMembershipSummary = {
  ...membershipSummary,
  membership_ulid: "membership-deprecated",
  membership_gpath: "/memberships/deprecated",
  name: "Deprecated Membership",
  description: "Deprecated membership summary",
  status: "Deprecated",
}

async function installMembershipReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/memberships/configs") {
      return { data: { memberships: [membershipSummary, deprecatedMembershipSummary], has_more: false, next_cursor: "", prev_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/memberships/membership-1") {
      return {
        data: {
          ...membershipSummary,
          description: "Read-only membership detail",
          features_json: JSON.stringify(["Priority support", "Course discount"]),
          ideal_for: "Regression users",
          casdoor_role_name: "member-pro",
          course_discount_coupon: "member-course-discount",
          required_cred_respaths: ["/credentials/member-pro"],
        },
      }
    }
    if (method === "POST" && pathname === "/api/memberships") {
      return { status: 201, data: { success: true } }
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
  await expect(page.getByText("已启用", { exact: true })).toBeVisible()
  await expect(page.getByText("已下架", { exact: true })).toBeVisible()
  await expect(page.getByText("Active", { exact: true })).toHaveCount(0)
  await expect(page.getByText("Deprecated", { exact: true })).toHaveCount(0)
  expect(requests).toContain("GET /api/memberships/configs")
  expect(requests).not.toContain("GET /api/mall/bundles")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("membership can be copied into a prefilled next-tier create form", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMembershipReadMocks(page, requests)
  await page.goto("/memberships")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  await page.getByRole("button", { name: "复制创建" }).click()

  const editor = page.getByRole("dialog", { name: "复制会员方案" })
  await expect(editor).toBeVisible()
  await expect(editor.getByLabel(/会员方案路径/)).toHaveValue("/memberships/regression")
  await expect(editor.getByLabel(/会员等级/)).toHaveValue("3")
  await expect(editor.getByLabel(/^\*?\s*名称$/)).toHaveValue("Regression Membership")
  await expect(editor.getByLabel(/Casdoor 角色名称/)).toHaveValue("member-pro")
  await expect(editor.getByText("价格不会复制", { exact: false })).toBeVisible()

  await editor.getByLabel(/^\*?\s*名称$/).fill("Regression Membership Plus")
  await editor.getByLabel(/Casdoor 角色名称/).fill("member-pro-plus")
  const createRequest = page.waitForRequest((request) => {
    const url = new URL(request.url())
    return request.method() === "POST" && url.pathname === "/api/memberships"
  })
  await editor.getByRole("button", { name: "创建副本" }).click()

  const request = await createRequest
  const body = request.postDataJSON() as Record<string, unknown>
  expect(body).toMatchObject({
    membership_gpath: "/memberships/regression",
    name: "Regression Membership Plus",
    description: "Read-only membership detail",
    features_json: JSON.stringify(["Priority support", "Course discount"], null, 2),
    ideal_for: "Regression users",
    duration_in_months: 12,
    casdoor_role_name: "member-pro-plus",
    tier_level: 3,
    course_discount_coupon: "member-course-discount",
    required_cred_respaths: ["/credentials/member-pro"],
  })
  expect(String(body.membership_ulid)).toMatch(/^[0-7][0-9A-HJKMNP-TV-Z]{25}$/)
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
  await expect(page.getByText("价格", { exact: true })).toHaveCount(0)

  expect(requests).toContain("GET /api/memberships/membership-1")
  expect(requests.some((request) => request.includes("/deprecate") || request.startsWith("POST ") || request.startsWith("PUT ") || request.startsWith("PATCH ") || request.startsWith("DELETE "))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("membership list ignores a stale response after creation refreshes it", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method === "POST" && pathname === "/api/memberships") {
      return { data: { membership_ulid: "01KZZKAW2FZDZD9S68NFAYPAN5" } }
    }
    return undefined
  })

  const staleMembership = {
    ...membershipSummary,
    membership_ulid: "membership-stale",
    membership_gpath: "/memberships/stale",
    name: "Stale Membership",
  }
  const latestMembership = {
    ...membershipSummary,
    membership_ulid: "membership-latest",
    membership_gpath: "/memberships/latest",
    name: "Latest Membership",
  }
  let listReads = 0
  let releaseStaleResponse: () => void = () => {}
  const staleResponseGate = new Promise<void>((resolve) => {
    releaseStaleResponse = () => resolve()
  })

  await page.route("**/api/memberships/configs**", async (route) => {
    listReads += 1
    const isStaleResponse = listReads === 1
    if (isStaleResponse) await staleResponseGate
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      headers: { "x-membership-response": isStaleResponse ? "stale" : "latest" },
      json: {
        code: 200,
        error_code: "OK",
        message: "OK",
        data: {
          memberships: [isStaleResponse ? staleMembership : latestMembership],
          has_more: false,
          next_cursor: "",
          prev_cursor: "",
        },
      },
    })
  })

  const initialRequest = page.waitForRequest((request) => new URL(request.url()).pathname === "/api/memberships/configs")
  await page.goto("/memberships")
  await initialRequest

  await page.getByRole("button", { name: "新增会员方案" }).click()
  const editor = page.getByRole("dialog", { name: "新增会员方案" })
  await editor.getByLabel(/会员方案路径/).fill("/memberships/latest")
  await editor.getByLabel(/^\*?\s*名称$/).fill("Latest Membership")
  await editor.getByLabel(/Casdoor 角色名称/).fill("member-latest")
  await editor.getByRole("button", { name: "创建会员方案" }).click()

  await expect(page.getByText("Latest Membership", { exact: true })).toBeVisible()
  expect(listReads).toBe(2)

  const staleResponse = page.waitForResponse((response) => response.headers()["x-membership-response"] === "stale")
  releaseStaleResponse()
  await (await staleResponse).finished()
  await page.waitForTimeout(50)

  await expect(page.getByText("Latest Membership", { exact: true })).toBeVisible()
  await expect(page.getByText("Stale Membership", { exact: true })).toHaveCount(0)
})
