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
