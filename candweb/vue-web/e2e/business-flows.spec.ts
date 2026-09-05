import { expect, test } from "@playwright/test"
import {
  candidateUser,
  installCandidateApiMocks,
  seedAuthenticatedCandidate,
  type ApiMockContext,
} from "./support/candidate"

test.beforeEach(async ({ page }) => {
  await seedAuthenticatedCandidate(page)
})

test("Casdoor 所有地区配置展开为完整电话区号列表", async ({ page }) => {
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/user/me") {
      return {
        data: {
          ...candidateUser,
          phone_country_code: "All",
        },
      }
    }
    if (pathname === "/api/public/config/organization") {
      return { data: { country_codes: ["All"] } }
    }
    return undefined
  })

  await page.goto("/settings", { waitUntil: "domcontentloaded" })

  const phoneCountryCode = page.getByTestId("settings-phone-country-code")
  await expect(phoneCountryCode).toHaveValue("SG")
  await expect(phoneCountryCode.locator('option[value="All"]')).toHaveCount(0)
  await expect(phoneCountryCode.locator('option[value="SG"]')).toHaveText("+65 · Singapore")
  await expect(phoneCountryCode.locator('option[value="DM"]')).toHaveText("+1 · Dominica")
  await expect(phoneCountryCode.locator('option[value="DO"]')).toHaveText(
    "+1 · Dominican Republic",
  )
  expect(await phoneCountryCode.locator("option").count()).toBeGreaterThan(200)
})

test("消息可以批量标记已读并删除", async ({ page }) => {
  const messages = [
    {
      message_id: "message-unread",
      msg_type: 1,
      status: 0,
      title: "回归测试未读消息",
      content: "这是一条未读消息",
      created_at: "2026-08-06T08:00:00Z",
    },
    {
      message_id: "message-read",
      msg_type: 2,
      status: 1,
      title: "回归测试已读消息",
      content: "这是一条已读消息",
      created_at: "2026-08-05T08:00:00Z",
    },
  ]
  const readBodies: unknown[] = []
  const deleteBodies: unknown[] = []

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/messages" && method === "GET") {
      return { data: { messages, has_more: false, total: messages.length } }
    }
    if (pathname === "/api/messages/unread-count") {
      return { data: { unread_count: messages.filter((message) => message.status === 0).length } }
    }
    if (pathname === "/api/messages/read" && method === "PUT") {
      readBodies.push(body)
      const ids = new Set((body as { message_ids?: string[] })?.message_ids || [])
      for (const message of messages) {
        if (ids.has(message.message_id)) message.status = 1
      }
      return { data: { success: true } }
    }
    if (pathname === "/api/messages/delete" && method === "POST") {
      deleteBodies.push(body)
      const ids = new Set((body as { message_ids?: string[] })?.message_ids || [])
      for (let index = messages.length - 1; index >= 0; index -= 1) {
        if (ids.has(messages[index].message_id)) messages.splice(index, 1)
      }
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  await expect(page.getByText("回归测试未读消息")).toBeVisible()

  await page.getByRole("button", { name: "全部标为已读" }).click()
  await expect.poll(() => readBodies).toEqual([{ message_ids: ["message-unread"] }])
  await expect(page.getByRole("button", { name: "全部标为已读" })).toHaveCount(0)

  const readMessageRow = page.locator(".group").filter({ hasText: "回归测试已读消息" })
  await readMessageRow.getByRole("button", { name: "更多操作" }).click()
  await readMessageRow.getByRole("button", { name: "删除" }).click()

  await expect.poll(() => deleteBodies).toEqual([{ message_ids: ["message-read"] }])
  await expect(page.getByText("回归测试已读消息")).toHaveCount(0)
})

test("会员取消续费后刷新为已取消状态", async ({ page }) => {
  let cancelled = false
  let cancelBody: unknown

  const activeRecord = () => ({
    membership_record_ulid: "membership-record-1",
    membership_ulid: "membership-plan-1",
    membership_gpath: "affiliate",
    membership_name: "Affiliate Membership",
    status: "ACTIVE",
    auto_renew: true,
    started_at: "2026-01-01T00:00:00Z",
    expires_at: "2027-01-01T00:00:00Z",
    cancelled_at: cancelled ? "2026-08-06T09:00:00Z" : "",
  })

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/membership/plans") {
      return {
        data: {
          memberships: [{
            membership_ulid: "membership-plan-1",
            membership_gpath: "affiliate",
            name: "Affiliate Membership",
            tier_level: 1,
          }],
        },
      }
    }
    if (pathname === "/api/membership/history") {
      return { data: { user_memberships: [activeRecord()], total: 1 } }
    }
    if (pathname === "/api/membership/billings") {
      return { data: { billings: [], total: 0 } }
    }
    if (pathname === "/api/membership/active") {
      return { data: { membership: activeRecord() } }
    }
    if (pathname === "/api/membership/cancel" && method === "POST") {
      cancelBody = body
      cancelled = true
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/membership", { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: "取消续费" }).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog).toBeVisible()
  await dialog.getByRole("button", { name: "取消续费" }).click()

  await expect.poll(() => cancelBody).toEqual({
    membership_record_ulid: "membership-record-1",
    reason: "user_requested",
  })
  await expect(page.getByRole("button", { name: "已取消续费" })).toBeDisabled()
})

test("会员升级先预览分摊费用并以幂等请求确认", async ({ page }) => {
  let upgraded = false
  let upgradeOrderPolls = 0
  let previewBody: unknown
  let upgradeBody: unknown

  const plans = [
    {
      membership_ulid: "membership-plan-affiliate",
      membership_gpath: "/memberships/gfi",
      name: "GFI Affiliate Member",
      tier_level: 1,
      duration_in_months: 12,
    },
    {
      membership_ulid: "membership-plan-charterholder",
      membership_gpath: "/memberships/gfi",
      name: "GFI Charterholder Member",
      tier_level: 2,
      duration_in_months: 12,
    },
  ]
  const activeRecord = () => ({
    membership_record_ulid: upgraded ? "membership-record-2" : "membership-record-1",
    membership_ulid: upgraded ? "membership-plan-charterholder" : "membership-plan-affiliate",
    membership_gpath: "/memberships/gfi",
    membership_name: upgraded ? "GFI Charterholder Member" : "GFI Affiliate Member",
    tier_level: upgraded ? 2 : 1,
    status: "ACTIVE",
    started_at: "2026-08-28T00:00:00Z",
    expires_at: "2027-08-28T00:00:00Z",
  })

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/membership/plans") return { data: { memberships: plans } }
    if (pathname === "/api/membership/history") {
      return { data: { user_memberships: [activeRecord()], total: 1 } }
    }
    if (pathname === "/api/membership/billings") return { data: { billings: [], total: 0 } }
    if (pathname === "/api/membership/active") return { data: { membership: activeRecord() } }
    if (pathname === "/api/membership/upgrade/preview" && method === "POST") {
      previewBody = body
      return {
        data: {
          eligible: true,
          immediate_charge_amount_minor: 1560,
          currency: "usd",
          proration_date: 1780000000,
          current_period_ends_at: "2027-08-28T00:00:00Z",
          next_cycle_renewal_amount_minor: 30000,
          target_membership_name: "GFI Charterholder Member",
          current_membership_name: "GFI Affiliate Member",
        },
      }
    }
    if (pathname === "/api/membership/upgrade" && method === "POST") {
      upgradeBody = body
      return {
        data: {
          success: true,
          status: "PENDING_PAYMENT",
          order_ulid: "membership-upgrade-order-1",
          paid_amount_minor: 1560,
          currency: "usd",
        },
      }
    }
    if (pathname === "/api/orders/membership-upgrade-order-1" && method === "GET") {
      upgradeOrderPolls += 1
      upgraded = upgradeOrderPolls >= 2
      return {
        data: {
          summary: {
            order_status: upgraded ? "COMPLETED" : "WAIT_PAYMENT",
            payment_status: upgraded ? "PAID" : "WAIT_PAY",
          },
        },
      }
    }
    return undefined
  })

  await page.goto("/membership?tab=levels&upgrade=membership-plan-charterholder", { waitUntil: "domcontentloaded" })
  await expect(page.getByRole("button", { name: "当前方案" })).toBeDisabled()

  const dialog = page.getByRole("dialog", { name: "确认会员升级" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("USD 15.6", { exact: true })).toBeVisible()
  await expect.poll(() => previewBody).toEqual({
    target_membership_ulid: "membership-plan-charterholder",
  })

  await dialog.getByRole("button", { name: "确认并支付" }).click()
  await expect.poll(() => upgradeBody).toMatchObject({
    target_membership_ulid: "membership-plan-charterholder",
    proration_date: 1780000000,
  })
  expect((upgradeBody as { currency?: unknown }).currency).toBeUndefined()
  expect((upgradeBody as { idempotency_key?: string }).idempotency_key).toMatch(/^[0-9a-f-]{36}$/)
  await expect.poll(() => upgradeOrderPolls).toBeGreaterThan(1)
  await expect(page.locator(".membership-current-name")).toContainText("GFI Charterholder Member")
})

test("会员升级等待支付或3DS验证时不会提前提示成功", async ({ page }) => {
  let upgradeStatus = "PENDING_PAYMENT"
  let upgradeOrderPolls = 0

  await page.addInitScript(() => {
    ;(window as any).Stripe = () => ({
      confirmCardPayment: async (clientSecret: string) => {
        ;(window as any).__membershipUpgradeClientSecret = clientSecret
        return { paymentIntent: { status: "succeeded" } }
      },
    })
  })

  const plans = [
    {
      membership_ulid: "membership-plan-affiliate",
      membership_gpath: "/memberships/gfi",
      name: "GFI Affiliate Member",
      tier_level: 1,
      duration_in_months: 12,
    },
    {
      membership_ulid: "membership-plan-charterholder",
      membership_gpath: "/memberships/gfi",
      name: "GFI Charterholder Member",
      tier_level: 2,
      duration_in_months: 12,
    },
  ]
  const activeRecord = {
    membership_record_ulid: "membership-record-1",
    membership_ulid: "membership-plan-affiliate",
    membership_gpath: "/memberships/gfi",
    membership_name: "GFI Affiliate Member",
    tier_level: 1,
    status: "ACTIVE",
    started_at: "2026-08-28T00:00:00Z",
    expires_at: "2027-08-28T00:00:00Z",
  }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/membership/plans") return { data: { memberships: plans } }
    if (pathname === "/api/membership/history") return { data: { user_memberships: [activeRecord], total: 1 } }
    if (pathname === "/api/membership/billings") return { data: { billings: [], total: 0 } }
    if (pathname === "/api/membership/active") return { data: { membership: activeRecord } }
    if (pathname === "/api/membership/upgrade/preview" && method === "POST") {
      return {
        data: {
          eligible: true,
          immediate_charge_amount_minor: 1560,
          currency: "usd",
          proration_date: 1780000000,
          current_period_ends_at: "2027-08-28T00:00:00Z",
          next_cycle_renewal_amount_minor: 30000,
          target_membership_name: "GFI Charterholder Member",
          current_membership_name: "GFI Affiliate Member",
        },
      }
    }
    if (pathname === "/api/membership/upgrade" && method === "POST") {
      return {
        data: {
          success: true,
          status: upgradeStatus,
          order_ulid: `membership-upgrade-${upgradeStatus.toLowerCase()}`,
          client_secret: upgradeStatus === "REQUIRES_ACTION" ? "pi_membership_3ds_secret" : "",
        },
      }
    }
    if (pathname.startsWith("/api/orders/membership-upgrade-") && method === "GET") {
      upgradeOrderPolls += 1
      return {
        data: {
          summary: {
            order_status: "WAIT_PAYMENT",
            payment_status: "WAIT_PAY",
          },
        },
      }
    }
    return undefined
  })

  await page.goto("/membership", { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: "会员等级", exact: true }).click()

  await page.getByRole("button", { name: "升级会员" }).click()
  let dialog = page.getByRole("dialog", { name: "确认会员升级" })
  await dialog.getByRole("button", { name: "确认并支付" }).click()
  await expect(page.getByText("升级订单处理中，请稍候。", { exact: true })).toBeVisible()
  await expect(dialog).toBeHidden()
  await expect(page.getByText("会员升级成功", { exact: true })).toHaveCount(0)
  await expect.poll(() => upgradeOrderPolls).toBeGreaterThan(0)

  upgradeStatus = "REQUIRES_ACTION"
  await page.getByRole("button", { name: "会员等级", exact: true }).click()
  await page.getByRole("button", { name: "升级会员" }).click()
  dialog = page.getByRole("dialog", { name: "确认会员升级" })
  await dialog.getByRole("button", { name: "确认并支付" }).click()

  await expect.poll(() => page.evaluate(() => (window as any).__membershipUpgradeClientSecret)).toBe("pi_membership_3ds_secret")
  await expect(page.getByText("支付成功，会员权益正在生效。", { exact: true })).toBeVisible()
  await expect(dialog).toBeHidden()
  await expect(page.getByText("会员升级成功", { exact: true })).toHaveCount(0)
})

test("会员刷新只应用最后一次请求返回的记录", async ({ page }) => {
  const membershipRecord = (id: string, name: string) => ({
    membership_record_ulid: id,
    membership_ulid: "membership-plan-1",
    membership_gpath: "affiliate",
    membership_name: name,
    status: "ACTIVE",
    started_at: "2026-01-01T00:00:00Z",
    expires_at: "2027-01-01T00:00:00Z",
  })
  const billingRecord = (id: string, type: string) => ({
    billing_record_ulid: id,
    billing_type: type,
    status: "PAID",
    amount_minor: 1000,
    currency: "USD",
    period_start: "2026-01-01T00:00:00Z",
    period_end: "2026-02-01T00:00:00Z",
  })
  let historyRequests = 0
  let billingRequests = 0

  await installCandidateApiMocks(page, async ({ pathname }: ApiMockContext) => {
    if (pathname === "/api/membership/plans") {
      return {
        data: {
          memberships: [{
            membership_ulid: "membership-plan-1",
            membership_gpath: "affiliate",
            name: "Affiliate Membership",
          }],
        },
      }
    }
    if (pathname === "/api/membership/history") {
      const requestNumber = ++historyRequests
      if (requestNumber === 2) await new Promise((resolve) => setTimeout(resolve, 1_000))
      const record = requestNumber === 1
        ? membershipRecord("membership-initial", "INITIAL-MEMBERSHIP")
        : requestNumber === 2
          ? membershipRecord("membership-stale", "STALE-MEMBERSHIP")
          : membershipRecord("membership-latest", "LATEST-MEMBERSHIP")
      return { data: { user_memberships: [record], total: 1 } }
    }
    if (pathname === "/api/membership/billings") {
      const requestNumber = ++billingRequests
      if (requestNumber === 2) await new Promise((resolve) => setTimeout(resolve, 1_000))
      const record = requestNumber === 1
        ? billingRecord("billing-initial", "INITIAL-BILLING")
        : requestNumber === 2
          ? billingRecord("billing-stale", "STALE-BILLING")
          : billingRecord("billing-latest", "LATEST-BILLING")
      return { data: { billings: [record], total: 1 } }
    }
    if (pathname === "/api/membership/active") {
      const name = historyRequests >= 3 ? "LATEST-MEMBERSHIP" : "INITIAL-MEMBERSHIP"
      return { data: { membership: membershipRecord("membership-active", name) } }
    }
    return undefined
  })

  await page.goto("/membership", { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: "会员历史", exact: true }).click()
  await expect(page.getByText("INITIAL-MEMBERSHIP", { exact: true })).toBeVisible()

  const refreshButton = page.locator(".membership-refresh-btn")
  await refreshButton.dispatchEvent("click")
  await expect.poll(() => historyRequests).toBe(2)
  await expect.poll(() => billingRequests).toBe(2)
  await refreshButton.dispatchEvent("click")

  await expect.poll(() => historyRequests).toBe(3)
  await expect.poll(() => billingRequests).toBe(3)
  await expect(page.getByText("LATEST-MEMBERSHIP", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "账单记录", exact: true }).click()
  await expect(page.getByText("LATEST-BILLING", { exact: true })).toBeVisible()

  await page.waitForTimeout(1_100)
  await expect(page.getByText("STALE-MEMBERSHIP", { exact: true })).toHaveCount(0)
  await expect(page.getByText("STALE-BILLING", { exact: true })).toHaveCount(0)
  await expect(page.getByText("LATEST-BILLING", { exact: true })).toBeVisible()
})

test("测验选择答案后同步草稿并提交结果", async ({ page }) => {
  const requests: Array<{ pathname: string; body: unknown }> = []
  let releaseDraftSync: (() => void) | undefined
  const draftSyncGate = new Promise<void>((resolve) => {
    releaseDraftSync = resolve
  })

  await installCandidateApiMocks(page, async ({ pathname, method, body }) => {
    if (pathname === "/api/quizzes/attempts/attempt-1/paper") {
      return {
        data: {
          attempt_id: "attempt-1",
          title: "回归测试测验",
          remaining_seconds: 600,
          questions: [{
            question_id: "question-1",
            question_text: "Playwright 的主要用途是什么？",
            question_type: 1,
            points: 10,
            options: [
              { option_id: "option-a", option_text: "浏览器自动化测试" },
              { option_id: "option-b", option_text: "数据库备份" },
            ],
          }],
        },
      }
    }
    if (pathname === "/api/quizzes/attempts/attempt-1/draft" && method === "POST") {
      requests.push({ pathname, body })
      await draftSyncGate
      return { data: { success: true } }
    }
    if (pathname === "/api/quizzes/attempts/attempt-1/submit" && method === "POST") {
      requests.push({ pathname, body })
      return { data: { score: 10, max_score: 10, pass_status: 1 } }
    }
    return undefined
  })

  await page.goto("/quizzes?attemptId=attempt-1", { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: /浏览器自动化测试/ }).click()

  const expectedBody = {
    submissions: [{ question_id: "question-1", selected_option_ids: ["option-a"] }],
  }
  await expect.poll(() => requests.find((request) => request.pathname.endsWith("/draft"))?.body).toEqual(expectedBody)

  await page.getByRole("button", { name: "提交答卷" }).click()
  await page.getByRole("dialog").getByRole("button", { name: "提交答卷" }).click()

  await page.waitForTimeout(100)
  expect(requests.some((request) => request.pathname.endsWith("/submit"))).toBe(false)
  releaseDraftSync?.()

  await expect.poll(() => requests.find((request) => request.pathname.endsWith("/submit"))?.body).toEqual(expectedBody)
  await expect(page.getByRole("heading", { name: "测验已完成" })).toBeVisible()
  await expect(page.getByText("10", { exact: true }).first()).toBeVisible()
})

test("待报名考试从列表进入对应报名页面", async ({ page }) => {
  await installCandidateApiMocks(page, ({ pathname }: ApiMockContext) => {
    if (pathname === "/api/exams") {
      return {
        data: {
          exams: [{
            exam_id: "exam-1",
            exam_name: "CFtP Regression Exam",
            course_unit_ulid: "course-unit-1",
            pipeline_ulid: "pipeline-1",
            course_unit_status: "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
            exam_status: "EXAM_STATUS_UNSPECIFIED",
          }],
          total: 1,
        },
      }
    }
    return undefined
  })

  await page.goto("/exams", { waitUntil: "domcontentloaded" })
  await page.getByRole("link", { name: "去报名考试" }).click()

  await expect(page).toHaveURL(
    "http://127.0.0.1:4173/exams/signup?unitId=course-unit-1&pipelineId=pipeline-1&returnTo=%2Fexams",
  )
})

test("等待教授批卷的考试不显示为不合格或开放成绩入口", async ({ page }) => {
  await installCandidateApiMocks(page, ({ pathname }: ApiMockContext) => {
    if (pathname === "/api/exams") {
      return {
        data: {
          exams: [{
            exam_id: "exam-pending-grading",
            exam_code: "CFTA-E6",
            exam_status: "DONE",
            result_status: "PENDING_GRADING",
            total_score: 52,
            is_passed: false,
            confirmation_number: "CFTP-PENDING-386",
            appointment_start_time: "2026-08-28T08:18:18Z",
            appointment_end_time: "2026-08-28T10:18:18Z",
          }],
          total: 1,
        },
      }
    }
    return undefined
  })

  await page.goto("/exams", { waitUntil: "domcontentloaded" })

  await expect(page.getByText("等待教授批卷", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("主观题正在等待教授批改", { exact: false })).toBeVisible()
  await expect(page.getByText("不合格", { exact: true })).toHaveCount(0)
  await expect(page.getByRole("link", { name: "查看结果" })).toHaveCount(0)
  await expect(page.getByText("52.00", { exact: true })).toHaveCount(0)
})

test("考试列表只应用最后一次刷新返回的数据", async ({ page }) => {
  let examRequests = 0
  await installCandidateApiMocks(page, async ({ pathname }: ApiMockContext) => {
    if (pathname !== "/api/exams") return undefined

    examRequests += 1
    if (examRequests === 2) {
      await new Promise((resolve) => setTimeout(resolve, 1_000))
      return { data: { exams: [{ exam_id: "stale-exam", exam_code: "STALE-EXAM" }], total: 1 } }
    }

    const examCode = examRequests === 1 ? "INITIAL-EXAM" : "LATEST-EXAM"
    return { data: { exams: [{ exam_id: `exam-${examRequests}`, exam_code: examCode }], total: 1 } }
  })

  await page.goto("/exams", { waitUntil: "domcontentloaded" })
  await expect(page.getByText("INITIAL-EXAM", { exact: true })).toBeVisible()

  const refreshButton = page.locator(".exam-refresh-btn")
  await refreshButton.dispatchEvent("click")
  await expect.poll(() => examRequests).toBe(2)
  await refreshButton.dispatchEvent("click")

  await expect(page.getByText("LATEST-EXAM", { exact: true })).toBeVisible()
  await page.waitForTimeout(1_100)
  await expect(page.getByText("LATEST-EXAM", { exact: true })).toBeVisible()
  await expect(page.getByText("STALE-EXAM", { exact: true })).toHaveCount(0)
})
