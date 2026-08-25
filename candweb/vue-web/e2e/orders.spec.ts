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
    biz_type: "BUNDLE_PURCHASE",
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
    pricing: {
      available: true,
      source: "GPAY_INVOICE",
      currency_code: "USD",
      billable_subtotal_minor: 15000,
      promotion_discount_minor: 2500,
      tax_minor: 400,
      total_minor: 12900,
      amount_paid_minor: 12900,
      exemption_amount_recorded: false,
      items: [{
        item_type: "course",
        item_ulid: "course-1",
        title: "Certification Course",
        unit_price_minor: 15000,
        quantity: 1,
        subtotal_minor: 15000,
      }],
      coupons: [{ code: "PACKAGE", name: "Package offer" }],
    },
    exemptions: [{
      course_cc_ulid: "course-exempted-1",
      credential_ulid: "credential-1",
    }],
    business_detail: {
      found: true,
      detail: {
        summary: {
          bundle_order_ulid: order.biz_ref_ulid,
          status: "COMPLETED",
        },
        completed_at: "2026-08-05T10:00:00Z",
      },
    },
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

test("订单详情展示价格拆分、免考与业务订单信息", async ({ page }) => {
  const order = orderFixture({
    order_id: "order-priced",
    product_name: "CFtP Priced Bundle",
    order_status: "COMPLETED",
    payment_status: "PAID",
    amount: 129,
  })

  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/orders") return orderListResponse([order])
    if (pathname === `/api/orders/${order.order_id}`) return { data: completedOrderDetail(order) }
    return undefined
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })
  await page.locator(".order-row").filter({ hasText: order.product_name }).click()

  const dialog = page.getByRole("dialog", { name: "订单详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("USD 150.00", { exact: true }).first()).toBeVisible()
  await expect(dialog.getByText("-USD 25.00", { exact: true })).toBeVisible()
  await expect(dialog.getByText("USD 129.00", { exact: true }).last()).toBeVisible()
  await expect(dialog.getByText("Certification Course", { exact: true })).toBeVisible()
  await expect(dialog.getByText("Package offer", { exact: true })).toBeVisible()
  await expect(dialog.getByText("未记录", { exact: true })).toBeVisible()
  await expect(dialog.getByText("course-exempted-1", { exact: true })).toBeVisible()
  await expect(dialog.getByText("认证套餐订单 ID", { exact: true })).toBeVisible()
  await expect(dialog.getByText(order.biz_ref_ulid || "", { exact: true })).toBeVisible()
})

test("订单筛选只应用最后一次请求返回的数据", async ({ page }) => {
  const initialOrder = orderFixture({
    order_id: "order-initial",
    product_name: "INITIAL-ORDER",
  })
  const staleOrder = orderFixture({
    order_id: "order-stale",
    product_name: "STALE-ORDER",
    order_status: "COMPLETED",
    payment_status: "PAID",
  })
  const latestOrder = orderFixture({
    order_id: "order-latest",
    product_name: "LATEST-ORDER",
    order_status: "CANCELLED",
  })
  let completedRequests = 0

  await installCandidateApiMocks(page, async ({ pathname, url }) => {
    if (pathname !== "/api/orders") return undefined

    const status = url.searchParams.get("status")
    if (status === "COMPLETED") {
      completedRequests += 1
      await new Promise((resolve) => setTimeout(resolve, 1_000))
      return orderListResponse([staleOrder])
    }
    if (status === "CANCELLED") return orderListResponse([latestOrder])
    return orderListResponse([initialOrder])
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })
  await expect(page.getByText("INITIAL-ORDER", { exact: true })).toBeVisible()

  const statusFilter = page.locator("#order-status-filter")
  await statusFilter.selectOption("COMPLETED")
  await expect.poll(() => completedRequests).toBe(1)
  await statusFilter.selectOption("CANCELLED")

  await expect(page.getByText("LATEST-ORDER", { exact: true })).toBeVisible()
  await page.waitForTimeout(1_100)
  await expect(page.getByText("LATEST-ORDER", { exact: true })).toBeVisible()
  await expect(page.getByText("STALE-ORDER", { exact: true })).toHaveCount(0)
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
    biz_type: "BUNDLE_PURCHASE",
    biz_ref_ulid: "bundle-order-cancel",
  })
  await expect.poll(() => orderListRequests).toBeGreaterThanOrEqual(2)
  await expect(row).toHaveCount(0)
})
