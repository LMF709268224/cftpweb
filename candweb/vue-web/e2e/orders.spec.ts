import { expect, test } from "@playwright/test"
import {
  installCandidateApiMocks,
  installStripeCheckoutMock,
  seedAuthenticatedCandidate,
  type ApiMockResponse,
} from "./support/candidate"

type OrderFixture = {
  order_id: string
  product_name: string
  order_status: string
  payment_status: string
  amount?: number
  currency?: string
  biz_type?: string
  biz_ref_ulid?: string
}

function orderFixture(overrides: Partial<OrderFixture> = {}): OrderFixture {
  return {
    order_id: "order-pending",
    product_name: "CFtP Pending Bundle",
    order_status: "WAIT_PAYMENT",
    payment_status: "UNPAID",
    amount: 63000,
    currency: "USD",
    biz_type: "PIPELINE_BUNDLE",
    biz_ref_ulid: "bundle-cftp",
    ...overrides,
  }
}

function orderListResponse(orders: OrderFixture[]): ApiMockResponse {
  return {
    data: {
      orders,
      total_orders: orders.length,
      total_label: String(orders.length),
      total_pages: 1,
      has_more: false,
      next_cursor: "",
      prev_cursor: "",
    },
  }
}

function completedOrderDetail(order: OrderFixture) {
  return {
    found: true,
    summary: {
      order_id: order.order_id,
      biz_type: order.biz_type,
      biz_ref_ulid: order.biz_ref_ulid,
      product_name: order.product_name,
      order_status: "COMPLETED",
      payment_status: "PAID",
      amount: order.amount,
      currency: order.currency,
      meta: {
        product_name: order.product_name,
      },
    },
    paid_at: "2026-08-05T10:00:00Z",
  }
}

test.beforeEach(async ({ page }) => {
  await seedAuthenticatedCandidate(page)
})

test("订单状态与可执行按钮保持一致", async ({ page }) => {
  const pendingOrder = orderFixture()
  const completedOrder = orderFixture({
    order_id: "order-completed",
    product_name: "CFtP Completed Bundle",
    order_status: "COMPLETED",
    payment_status: "PAID",
  })
  const paidPendingOrder = orderFixture({
    order_id: "order-paid-pending",
    product_name: "CFtP Paid Pending Bundle",
    order_status: "WAIT_PAYMENT",
    payment_status: "PAID",
  })

  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/orders") {
      return orderListResponse([pendingOrder, completedOrder, paidPendingOrder])
    }
    return undefined
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })

  const pendingRow = page.locator(".order-row").filter({ hasText: pendingOrder.product_name })
  await expect(pendingRow.getByText("待支付", { exact: true })).toBeVisible()
  await expect(pendingRow.getByRole("button", { name: "继续支付" })).toBeVisible()
  await expect(pendingRow.getByRole("button", { name: "取消支付" })).toBeVisible()

  const completedRow = page.locator(".order-row").filter({ hasText: completedOrder.product_name })
  await expect(completedRow.getByText("已完成", { exact: true })).toBeVisible()
  await expect(completedRow.getByRole("button", { name: "继续支付" })).toHaveCount(0)

  const paidPendingRow = page.locator(".order-row").filter({ hasText: paidPendingOrder.product_name })
  await expect(paidPendingRow.getByText("已支付，处理中", { exact: true })).toBeVisible()
  await expect(paidPendingRow.getByRole("button", { name: "继续支付" })).toHaveCount(0)
  await expect(paidPendingRow.getByRole("button", { name: "取消支付" })).toHaveCount(0)
})

test("从托管支付返回后清理查询参数并同步为已完成", async ({ page }) => {
  const order = orderFixture({
    order_id: "order-hosted-return",
    product_name: "CFtP Hosted Return Bundle",
  })

  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/orders") return orderListResponse([order])
    if (pathname === `/api/orders/${order.order_id}`) {
      return { data: completedOrderDetail(order) }
    }
    return undefined
  })

  await page.goto(`/orders?payment_status=success&payment_action=orders&order_id=${order.order_id}`, {
    waitUntil: "domcontentloaded",
  })

  await expect(page).toHaveURL("http://127.0.0.1:4173/orders")
  const row = page.locator(".order-row").filter({ hasText: order.product_name })
  await expect(row.getByText("已完成", { exact: true })).toBeVisible()
  await expect(row.getByRole("button", { name: "继续支付" })).toHaveCount(0)
})

test("订单页嵌入式支付成功后留在订单页并刷新订单状态", async ({ page }) => {
  const order = orderFixture({
    order_id: "order-embedded-payment",
    product_name: "CFtP Embedded Payment Bundle",
  })
  let paymentCompleted = false
  let orderListRequests = 0
  let successURL = ""

  await installStripeCheckoutMock(page)
  await installCandidateApiMocks(page, ({ pathname, body }) => {
    if (pathname === "/api/orders") {
      orderListRequests += 1
      return orderListResponse([
        paymentCompleted
          ? { ...order, order_status: "COMPLETED", payment_status: "PAID" }
          : order,
      ])
    }
    if (pathname === "/api/mall/payments/preview") {
      return {
        data: {
          subtotal: 63000,
          total: 63000,
          currency: "USD",
          invalid: [],
          discounts: [],
        },
      }
    }
    if (pathname === "/api/mall/payments/initiate") {
      successURL = String((body as { success_url?: string } | null)?.success_url || "")
      return { data: { payment_key: "cs_test_playwright_secret" } }
    }
    if (pathname === `/api/orders/${order.order_id}`) {
      return { data: completedOrderDetail(order) }
    }
    return undefined
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })

  const initialRow = page.locator(".order-row").filter({ hasText: order.product_name })
  await initialRow.getByRole("button", { name: "继续支付" }).click()
  await expect(page.getByRole("heading", { name: "继续支付", exact: true })).toBeVisible()

  const completePaymentButton = page.getByTestId("fake-stripe-complete")
  await expect(completePaymentButton).toBeVisible()
  await expect.poll(() => successURL).toContain("/orders?payment_status=success")

  paymentCompleted = true
  await completePaymentButton.click()

  await expect(page).toHaveURL("http://127.0.0.1:4173/orders")
  await expect(page.getByRole("heading", { name: "继续支付", exact: true })).toHaveCount(0)
  await expect.poll(() => orderListRequests).toBeGreaterThanOrEqual(2)

  const completedRow = page.locator(".order-row").filter({ hasText: order.product_name })
  await expect(completedRow.getByText("已完成", { exact: true })).toBeVisible()
  await expect(completedRow.getByRole("button", { name: "继续支付" })).toHaveCount(0)
})

test("取消未支付订单后刷新列表并隐藏该订单", async ({ page }) => {
  const order = orderFixture({
    order_id: "order-cancel",
    product_name: "CFtP Cancel Bundle",
    biz_ref_ulid: "bundle-order-cancel",
  })
  let cancelled = false
  let cancelBody: unknown
  let orderListRequests = 0

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/orders") {
      orderListRequests += 1
      return orderListResponse(cancelled ? [] : [order])
    }
    if (pathname === "/api/orders/cancel" && method === "POST") {
      cancelBody = body
      cancelled = true
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })
  const row = page.locator(".order-row").filter({ hasText: order.product_name })
  await row.getByRole("button", { name: "取消支付" }).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog).toBeVisible()
  await dialog.getByRole("button", { name: "取消订单" }).click()

  await expect.poll(() => cancelBody).toEqual({
    biz_type: "PIPELINE_BUNDLE",
    biz_ref_ulid: "bundle-order-cancel",
  })
  await expect.poll(() => orderListRequests).toBeGreaterThanOrEqual(2)
  await expect(row).toHaveCount(0)
})
