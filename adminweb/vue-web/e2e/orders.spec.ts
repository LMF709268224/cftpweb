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
          business_detail: {
            found: true,
            detail: {
              bundle_order_ulid: "bundle-order-1",
              payment_mode: "FULL_PIPELINE",
              updated_at: "2026-08-11T01:00:00Z",
            },
          },
          price_detail: {
            currency_code: "USD",
            subtotal_minor: 70000,
            discount_total_minor: 7000,
            tax_total_minor: 0,
            total_minor: 63000,
          },
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
  await expect(page.getByText("USD 129.00").first()).toBeVisible()
  expect(requests).toContain("GET /api/mall/orders")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("order detail displays price summary and business metadata without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installOrderReadMocks(page, requests)
  await page.goto("/orders")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Bundle" })
  await expect(detailDialog.getByText("order-1", { exact: true }).first()).toBeVisible()
  const originalPriceRow = detailDialog.getByText("原价", { exact: true }).locator("..")
  const discountRow = detailDialog.getByText("优惠总额", { exact: true }).locator("..")
  const amountPaidRow = detailDialog.getByText("实际支付", { exact: true }).locator("..")
  await expect(originalPriceRow.getByText("USD 700.00", { exact: true })).toBeVisible()
  await expect(discountRow.getByText("-USD 70.00", { exact: true })).toBeVisible()
  await expect(amountPaidRow.getByText("USD 630.00", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("税费", { exact: true })).toHaveCount(0)
  await detailDialog.getByRole("button", { name: /业务详情/ }).click()
  await expect(detailDialog.getByText("bundle-order-1", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("整套认证一次性支付", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/mall/orders/order-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
