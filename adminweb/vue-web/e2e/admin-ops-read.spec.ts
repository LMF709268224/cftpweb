import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

type OpsModuleFixture = {
  label: string
  path: string
  itemKey: string
  idKey: string
  id: string
  detail?: boolean
  requiredFilter?: { label: string; key: string; value: string }
}

const opsModuleFixtures: OpsModuleFixture[] = [
  { label: "Pay Webhooks", path: "/api/pay/webhook-events", itemKey: "events", idKey: "event_id", id: "evt_regression", detail: true },
  { label: "Order Items", path: "/api/pay/order-items", itemKey: "items", idKey: "item_id", id: "item-regression", requiredFilter: { label: "Order ID", key: "order_ulid", value: "order-regression" } },
  { label: "Mall Mail Tasks", path: "/api/mall/mail-tasks", itemKey: "items", idKey: "mail_task_ulid", id: "mail-task-mall-regression", detail: true },
  { label: "Membership Mail Tasks", path: "/api/memberships/mails", itemKey: "mails", idKey: "mail_ulid", id: "mail-membership-regression", detail: true },
  { label: "Mall NATS", path: "/api/mall/nats-messages", itemKey: "items", idKey: "message_ulid", id: "nats-mall-regression", detail: true },
  { label: "Certification Mail Tasks", path: "/api/prog/mail-tasks", itemKey: "tasks", idKey: "mail_task_ulid", id: "mail-task-prog-regression", detail: true, requiredFilter: { label: "Candidate ID", key: "candidate_ulid", value: "candidate-regression" } },
  { label: "Certification Stages", path: "/api/prog/stages", itemKey: "stages", idKey: "stage_ulid", id: "stage-regression", detail: true, requiredFilter: { label: "Certification ID", key: "pipeline_ulid", value: "pipeline-regression" } },
  { label: "Certification Course Units", path: "/api/prog/course-units", itemKey: "course_units", idKey: "course_unit_ulid", id: "course-unit-regression", detail: true, requiredFilter: { label: "Certification ID", key: "pipeline_ulid", value: "pipeline-regression" } },
  { label: "Certification Driver Events", path: "/api/prog/driver-events", itemKey: "items", idKey: "event_ulid", id: "driver-event-regression", detail: true },
  { label: "Certification NATS", path: "/api/prog/nats-messages", itemKey: "items", idKey: "message_ulid", id: "nats-prog-regression", detail: true },
  { label: "Exam Audit", path: "/api/exam-ops/audit-messages", itemKey: "audit_messages", idKey: "message_ulid", id: "01J00000000000000000000000", detail: true },
  { label: "Exam Transitions", path: "/api/exam-ops/status-transitions", itemKey: "transitions", idKey: "msg_fp", id: "transition-regression" },
  { label: "Exam Reminder Mails", path: "/api/exam-ops/reminder-mails", itemKey: "mails", idKey: "mail_ulid", id: "mail-exam-regression", detail: true },
]

const firstSubscription = {
  subscription_ulid: "subscription-page-1",
  customer_ulid: "customer-regression",
  candidate_name: "Regression Candidate",
  stripe_subscription_id: "sub_regression_1",
  status: "ACTIVE",
  amount: 1299,
  currency: "usd",
  created_at: 1786406400,
}

test("admin operations renders subscription data and reads its next cursor page", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname !== "/api/pay/subscriptions") return undefined
    if (url.searchParams.get("cursor") === "next-subscriptions") {
      return {
        data: {
          subscriptions: [{ ...firstSubscription, subscription_ulid: "subscription-page-2", stripe_subscription_id: "sub_regression_2" }],
          total: 1,
          has_more: false,
          next_cursor: "",
        },
      }
    }
    return { data: { subscriptions: [firstSubscription], total: 1, has_more: true, next_cursor: "next-subscriptions" } }
  })

  await page.goto("/admin-ops")
  await expect(page.getByText("Regression Candidate", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("sub_regression_1", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("有效", { exact: true }).first()).toBeVisible()

  await page.getByRole("button", { name: "下一页" }).click()
  await expect(page.getByText("sub_regression_2", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("第 2 页", { exact: true }).first()).toBeVisible()
  expect(requests.some((request) => request.includes("cursor=next-subscriptions"))).toBe(true)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("admin operations forwards read-only subscription filters", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname === "/api/pay/subscriptions") return { data: { subscriptions: [firstSubscription], total: 1, has_more: false, next_cursor: "" } }
    return undefined
  })
  await page.goto("/admin-ops")
  await expect(page.getByText("Regression Candidate", { exact: true }).first()).toBeVisible()

  await page.getByText("客户 ID", { exact: true }).locator("..").getByRole("textbox").fill("customer-regression")
  await page.getByRole("button", { name: "查询" }).click()

  await expect.poll(() => requests.some((request) => request.includes("customer_ulid=customer-regression"))).toBe(true)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

for (const fixture of opsModuleFixtures) {
  test(`admin operations reads ${fixture.label} populated data${fixture.detail ? " and detail" : ""}`, async ({ page }) => {
    await seedAuthenticatedAdmin(page, "en")
    const requests: string[] = []
    await installAdminApiMocks(page, ({ method, pathname, url }) => {
      requests.push(`${method} ${pathname}${url.search}`)
      if (pathname === fixture.path) {
        return {
          data: {
            [fixture.itemKey]: [{
              [fixture.idKey]: fixture.id,
              status: "ACTIVE",
              event_status: "PROCESSED",
              task_status: "SENT",
              created_at: "2026-08-11T00:00:00Z",
            }],
            total: 1,
            has_more: false,
            next_cursor: "",
          },
        }
      }
      if (fixture.detail && pathname === `${fixture.path}/${fixture.id}`) {
        return { data: { [fixture.idKey]: fixture.id, detail_marker: `${fixture.id}-detail` } }
      }
      return undefined
    })

    await page.goto("/admin-ops")
    await page.getByRole("button", { name: fixture.label, exact: true }).click()
    if (fixture.requiredFilter) {
      await page.getByText(fixture.requiredFilter.label, { exact: true }).locator("..").getByRole("textbox").fill(fixture.requiredFilter.value)
      await page.getByRole("button", { name: "Search", exact: true }).click()
    }

    await expect(page.getByText(fixture.id, { exact: true }).first()).toBeVisible()
    await expect.poll(() => requests.some((request) => request.startsWith(`GET ${fixture.path}`))).toBe(true)
    if (fixture.requiredFilter) {
      await expect.poll(() => requests.some((request) => request.includes(`${fixture.requiredFilter?.key}=${fixture.requiredFilter?.value}`))).toBe(true)
    }
    if (fixture.detail) {
      await page.getByRole("button", { name: "View detail", exact: true }).click()
      await expect(page.getByText(`${fixture.id}-detail`, { exact: false })).toBeVisible()
      expect(requests.some((request) => request.startsWith(`GET ${fixture.path}/${fixture.id}`))).toBe(true)
    }
    expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
  })
}

test("admin operations returns to the previous subscription cursor page", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname !== "/api/pay/subscriptions") return undefined
    if (url.searchParams.get("cursor") === "next-subscriptions") {
      return {
        data: {
          subscriptions: [{ ...firstSubscription, subscription_ulid: "subscription-page-2", stripe_subscription_id: "sub_regression_2" }],
          total: 1,
          has_more: false,
          next_cursor: "",
        },
      }
    }
    return { data: { subscriptions: [firstSubscription], total: 1, has_more: true, next_cursor: "next-subscriptions" } }
  })

  await page.goto("/admin-ops")
  await expect(page.getByText("sub_regression_1", { exact: true }).first()).toBeVisible()
  await page.getByRole("button", { name: "Next", exact: true }).click()
  await expect(page.getByText("sub_regression_2", { exact: true }).first()).toBeVisible()
  await page.getByRole("button", { name: "Previous", exact: true }).click()

  await expect(page.getByText("sub_regression_1", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("Page 1", { exact: true }).first()).toBeVisible()
  expect(requests[requests.length - 1]).not.toContain("cursor=")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("admin operations list recovers after a failed read", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const requests: string[] = []
  let subscriptionReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/pay/subscriptions") return undefined
    subscriptionReads += 1
    if (subscriptionReads === 1) return { status: 503, errorCode: "OPS_UNAVAILABLE", message: "Ops list unavailable" }
    return { data: { subscriptions: [firstSubscription], total: 1, has_more: false, next_cursor: "" } }
  })

  await page.goto("/admin-ops")
  await expect(page.getByText(/Failed to load ops list/)).toBeVisible()
  await page.getByRole("button", { name: "Refresh", exact: true }).click()

  await expect(page.getByText("sub_regression_1", { exact: true }).first()).toBeVisible()
  expect(subscriptionReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("admin operations ignores a slow list after switching modules", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname === "/api/pay/subscriptions") {
      return {
        data: { subscriptions: [{ ...firstSubscription, stripe_subscription_id: "sub_stale" }], total: 1, has_more: false, next_cursor: "" },
        delayMs: 600,
      }
    }
    if (pathname === "/api/pay/webhook-events") {
      return { data: { events: [{ event_id: "evt_latest", processed_status: "PROCESSED" }], total: 1, has_more: false, next_cursor: "" } }
    }
    return undefined
  })

  await page.goto("/admin-ops")
  await page.getByRole("button", { name: "Pay Webhooks", exact: true }).click()
  await expect(page.getByText("evt_latest", { exact: true }).first()).toBeVisible()
  await page.waitForTimeout(650)

  await expect(page.getByText("sub_stale", { exact: true })).toHaveCount(0)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("admin operations detail can be reopened after a failed read", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const requests: string[] = []
  let detailReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname === "/api/pay/webhook-events") {
      return { data: { events: [{ event_id: "evt_retry", processed_status: "PROCESSED" }], total: 1, has_more: false, next_cursor: "" } }
    }
    if (pathname === "/api/pay/webhook-events/evt_retry") {
      detailReads += 1
      if (detailReads === 1) return { status: 503, errorCode: "DETAIL_UNAVAILABLE", message: "Detail unavailable" }
      return { data: { event_id: "evt_retry", detail_marker: "detail-recovered" } }
    }
    return undefined
  })

  await page.goto("/admin-ops")
  await page.getByRole("button", { name: "Pay Webhooks", exact: true }).click()
  await expect(page.getByText("evt_retry", { exact: true }).first()).toBeVisible()
  await page.getByRole("button", { name: "View detail", exact: true }).click()
  await expect(page.getByText(/Failed to load detail/)).toBeVisible()
  await page.getByRole("button", { name: "Close", exact: true }).click()
  await page.getByRole("button", { name: "View detail", exact: true }).click()

  await expect(page.getByText("detail-recovered", { exact: false })).toBeVisible()
  expect(detailReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
