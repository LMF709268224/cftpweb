import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const orderSummary = {
  order_ulid: "order-1",
  product_name: "Regression Bundle",
  candidate_ulid: "candidate-1",
  biz_type: "BUNDLE_PURCHASE",
  biz_ref_ulid: "bundle-order-1",
  amount_minor: 12900,
  currency_code: "USD",
  order_status: "PAID",
  payment_status: "PAID",
  created_at: "2026-08-11T00:00:00Z",
}

async function installOrderReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/mall/orders") {
      return {
        data: {
          items: [orderSummary],
          total: 1,
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/mall/orders/order-1") {
      return {
        data: {
          summary: orderSummary,
          items: [{
            item_type: "course",
            item_id: "course-1",
            title: "Certification Course",
            base_price: 15000,
            quantity: 1,
          }],
        },
      }
    }
    return undefined
  })
}

test("order list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installOrderReadMocks(page, requests)

  await page.goto("/orders")

  await expect(page.getByText("Regression Bundle").first()).toBeVisible()
  await expect(page.getByText(/candidate-1/).first()).toBeVisible()
  await expect(page.getByText("USD 129", { exact: true }).first()).toBeVisible()
  expect(requests).toContain("GET /api/mall/orders")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("order detail displays product items without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installOrderReadMocks(page, requests)
  await page.goto("/orders")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Bundle" })
  await expect(detailDialog.getByText("order-1", { exact: true }).first()).toBeVisible()
  await expect(detailDialog.getByText("商品明细", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Certification Course", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("USD 150", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("数量", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("价格", { exact: true })).toHaveCount(0)
  await expect(detailDialog.getByRole("button", { name: /业务详情/ })).toHaveCount(0)

  expect(requests).toContain("GET /api/mall/orders/order-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
