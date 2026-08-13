import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const bundleSummary = {
  bundle_ulid: "bundle-1",
  bundle_gpath: "/bundles/regression",
  name: "Regression Bundle",
  description: "Read-only bundle summary",
  items_json: "[]",
  pricing_json: "{}",
  display_amount_min: 12900,
  display_amount_max: 12900,
  display_currency: "USD",
  status: "Active",
  version: 2,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installBundleReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/mall/bundles") {
      return { data: { total: 1, exact: true, bundles: [bundleSummary], has_more: false, next_cursor: "", prev_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/mall/bundles/bundle-1") {
      return {
        data: {
          bundle: {
            ...bundleSummary,
            description: "Read-only bundle detail",
            items_json: JSON.stringify([{ item_type: "pipeline", ref_ulid: "pipeline-1" }]),
          },
        },
      }
    }
    return undefined
  })
}

test("bundle list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installBundleReadMocks(page, requests)

  await page.goto("/bundles")

  await expect(page.getByText("Regression Bundle", { exact: true })).toBeVisible()
  await expect(page.getByText("Read-only bundle summary", { exact: true })).toBeVisible()
  await expect(page.getByText("bundle-1", { exact: false })).toBeVisible()
  expect(requests).toContain("GET /api/mall/bundles")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("bundle detail reads summary and linked items without editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installBundleReadMocks(page, requests)
  await page.goto("/bundles")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  await expect(page.getByRole("heading", { name: "Regression Bundle" })).toBeVisible()
  await expect(page.getByText("Read-only bundle detail", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("pipeline-1", { exact: false })).toBeVisible()

  expect(requests).toContain("GET /api/mall/bundles/bundle-1")
  expect(requests.some((request) => request.includes("/publish") || request.includes("/deprecate") || request.includes("/sync-display-pricing") || request.startsWith("DELETE "))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("membership-only bundle replaces its stale membership reference", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const oldMembershipID = "01JKM1AFF1ATE0000000000001"
  const newMembershipID = "01KZD3FYABXSWJ31RDG1ATQCR6"
  const membershipBundle = {
    ...bundleSummary,
    status: "Draft",
    items_json: JSON.stringify([{ item_type: "membership", ref_ulid: oldMembershipID }]),
    pricing_json: JSON.stringify({
      memberships: [{
        membership_id: oldMembershipID,
        duration_months: 0,
        stripe_product_id: "prod_membership",
        stripe_price_id: "price_membership",
      }],
    }),
  }
  const captured = { updateBody: null as Record<string, unknown> | null }

  page.on("request", (request) => {
    if (request.method() === "PUT" && new URL(request.url()).pathname === "/api/mall/bundles/pricing") {
      captured.updateBody = request.postDataJSON() as Record<string, unknown>
    }
  })

  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method === "GET" && pathname === "/api/mall/bundles") {
      return { data: { total: 1, bundles: [membershipBundle], has_more: false, next_cursor: "", prev_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/mall/bundles/bundle-1") {
      return { data: { bundle: membershipBundle } }
    }
    if (method === "GET" && pathname === "/api/pipelines") {
      return { data: { pipelines: [], next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/memberships") {
      return {
        data: {
          memberships: [{
            membership_ulid: newMembershipID,
            membership_gpath: "/membership/affiliate",
            name: "Affiliate",
            duration_in_months: 12,
            status: "Active",
            version: 1,
          }],
          next_cursor: "",
        },
      }
    }
    if (method === "PUT" && pathname === "/api/mall/bundles/pricing") {
      return { data: membershipBundle }
    }
    if (method === "POST" && pathname === "/api/mall/bundles/sync-display-pricing") {
      return { data: {} }
    }
    return undefined
  })

  await page.goto("/bundles")
  await page.getByRole("button", { name: "查看详情" }).first().click()
  await page.getByRole("button", { name: /结构与价格/ }).click()

  await expect(page.getByRole("heading", { name: "替换绑定会员" })).toBeVisible()
  await expect(page.getByRole("heading", { name: "替换绑定认证" })).toHaveCount(0)
  await page.getByLabel("新绑定会员").selectOption(newMembershipID)
  await page.getByRole("button", { name: "替换会员并保存" }).click()

  await expect.poll(() => captured.updateBody).not.toBeNull()
  const items = JSON.parse(String(captured.updateBody?.items_json))
  const pricing = JSON.parse(String(captured.updateBody?.pricing_json))
  expect(items[0].ref_ulid).toBe(newMembershipID)
  expect(pricing.memberships[0]).toMatchObject({
    membership_id: newMembershipID,
    duration_months: 12,
    discount_coupon: "",
    stripe_product_id: "prod_membership",
    stripe_price_id: "price_membership",
  })
})
