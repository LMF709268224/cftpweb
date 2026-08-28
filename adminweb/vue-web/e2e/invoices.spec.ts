import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const invoiceSummary = {
  id: "invoice-1",
  order_id: "order-1",
  email: "candidate-1",
  amount: 129,
  currency: "USD",
  status: "COMPLETED",
  created_at: "2026-08-11T00:00:00Z",
  paid_at: "2026-08-11T01:00:00Z",
}

async function installInvoiceReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/mall/invoices") {
      return {
        data: {
          invoices: [invoiceSummary],
          total: 1,
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/mall/invoices/order-1/pdf") {
      return {
        data: {
          pdf_url: "https://pay.stripe.com/invoice/acct_test/invoice_test/pdf?s=preview",
        },
      }
    }
    return undefined
  })
}

test("invoice list renders the returned read-only payment summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installInvoiceReadMocks(page, requests)

  await page.goto("/invoices")

  await expect(page.getByText("invoice-1", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("订单：order-1", { exact: true })).toBeVisible()
  await expect(page.getByText("129 USD", { exact: true }).first()).toBeVisible()
  expect(requests).toContain("GET /api/mall/invoices")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("invoice detail reuses read-only list data without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installInvoiceReadMocks(page, requests)
  await page.goto("/invoices")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByText("发票详情", { exact: true }).locator("..", { hasText: "invoice-1" })
  await expect(page.getByText("candidate-1", { exact: true })).toBeVisible()
  await expect(page.getByText("2026", { exact: false }).first()).toBeVisible()

  expect(await detailDialog.count()).toBeGreaterThan(0)
  expect(requests.filter((request) => request === "GET /api/mall/invoices")).toHaveLength(1)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("invoice detail loads the Stripe PDF preview on demand", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await page.route("https://pay.stripe.com/**", async (route) => {
    await route.fulfill({ status: 200, contentType: "application/pdf", body: "%PDF-1.4\n%%EOF" })
  })
  await installInvoiceReadMocks(page, requests)
  await page.goto("/invoices")

  await page.getByRole("button", { name: "查看详情" }).click()
  await page.getByRole("button", { name: "预览 PDF" }).click()

  const pdfURL = "https://pay.stripe.com/invoice/acct_test/invoice_test/pdf?s=preview"
  await expect(page.getByTitle("发票 PDF 预览")).toHaveAttribute("src", pdfURL)
  await expect(page.getByRole("link", { name: "新窗口打开" })).toHaveAttribute("href", pdfURL)
  expect(requests).toContain("GET /api/mall/invoices/order-1/pdf")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
