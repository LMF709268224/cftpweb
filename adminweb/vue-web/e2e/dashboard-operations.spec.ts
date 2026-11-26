import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const dashboard = {
  candidate_total: 3,
  user_stats: { total: 8, active: 6, inactive: 2, admins: 2, members: 3, email_verified: 7 },
  user_role_stats: [],
  users: [],
  user_total: 0,
  user_page: 1,
  user_page_size: 10,
  stage_buckets: [],
  stage_buckets_exact: true,
  today_revenue: [{ currency: "USD", amount_minor: 140000, order_count: 2 }],
  today_revenue_exact: true,
  revenue_48h: [{ currency: "USD", previous_amount_minor: 100000, current_amount_minor: 140000 }],
  membership_distribution: { candidate_accounts: 3, no_membership_role: 1, roles: [
    { role: "role_membership_affiliate", count: 1 },
    { role: "role_membership_associate", count: 1 },
    { role: "role_membership_charterholder", count: 0 },
  ] },
  new_candidates_48h: 2,
  unverified_candidates: 1,
  aggregation_sample_limit: 500,
  generated_at: "2026-11-24T00:00:00Z",
}

for (const lang of ["zh", "en"] as const) {
  for (const mobile of [false, true]) {
    test(`dashboard operational metrics and mail links (${lang}, ${mobile ? "mobile" : "desktop"})`, async ({ page }, testInfo) => {
      await seedAuthenticatedAdmin(page, lang)
      await page.setViewportSize({ width: mobile ? 390 : 1440, height: 900 })
      const requests: string[] = []
      const reviewQueries: URLSearchParams[] = []
      const mailQueries: URLSearchParams[] = []
      await installAdminApiMocks(page, ({ method, pathname, url }) => {
        requests.push(`${method} ${pathname}`)
        if (pathname === "/api/dashboard/ops") return { data: dashboard }
        if (pathname === "/api/dashboard/pending-age") return { data: { created_older_48h: 3, exact: true, aggregation_sample_limit: 500 } }
        if (pathname === "/api/applications") {
          reviewQueries.push(url.searchParams)
          return { data: { applications: [], total: 5, total_label: "5", total_exact: true, has_more: false } }
        }
        if (pathname === "/api/mails/stats") return { data: { status_counts: { SENT: 142, FAILED: 2, SCHEDULING: 3, CANCELLED: 1 } } }
        if (pathname === "/api/mails/sent") {
          mailQueries.push(url.searchParams)
          return { data: { mails: [], total: 0, has_more: false } }
        }
        return undefined
      })
      await page.goto("/dashboard")
      const metrics = page.getByRole("region", { name: lang === "zh" ? "关键指标" : "Key Metrics", exact: true })
      const reviews = metrics.getByTestId("dashboard-reviews")
      const business = page.getByTestId("dashboard-business")
      await expect(reviews.getByText("5", { exact: true })).toBeVisible()
      await expect(reviews).toHaveAttribute("href", "/applications")
      await expect(metrics.getByTestId("dashboard-revenue")).toHaveAttribute("href", "/orders")
      await expect(metrics.getByText("USD 1400", { exact: true })).toBeVisible()
      await expect(metrics.getByText("+40%", { exact: true })).toBeVisible()
      const membershipRoles = page.getByTestId("dashboard-memberships")
      await expect(membershipRoles.getByText(lang === "zh" ? "会员角色分布" : "Membership Role Distribution", { exact: true })).toBeVisible()
      await expect(membershipRoles.getByText(lang === "zh" ? "无会员角色" : "No membership role", { exact: true })).toBeVisible()
      await expect(membershipRoles.getByText(lang === "zh" ? "附属会员" : "Affiliate", { exact: true })).toBeVisible()
      await expect(membershipRoles.getByText(lang === "zh" ? "3 个考生账号" : "3 candidate accounts", { exact: true })).toBeVisible()
      await expect(reviews.getByTestId("dashboard-pending-age").getByText(lang === "zh" ? /创建超过 48 小时仍待审: 3/ : /Created over 48h ago, still pending: 3/)).toBeVisible()
      await expect(business.getByText(lang === "zh" ? "邮箱未验证的考生" : "Candidate emails unverified")).toBeVisible()
      expect(reviewQueries[0]?.get("status")).toBe("Pending")
      expect(reviewQueries[0]?.get("include_status_subtotals")).toBe("false")
      await expect(business.getByText(lang === "zh" ? "待审核申请" : "Pending Applications", { exact: true })).toHaveCount(0)
      await expect(business.getByText(lang === "zh" ? "按当前发送状态统计全部历史邮件记录。" : "All-time records by current sending status.", { exact: true })).toBeVisible()
      const failedLink = business.locator('a[href="/mails?tab=sent&status=FAILED"]')
      await expect(failedLink.getByText("2", { exact: true })).toBeVisible()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
      await page.screenshot({ path: testInfo.outputPath("dashboard.png"), fullPage: true })
      await failedLink.click()
      await expect(page).toHaveURL(/\/mails\?tab=sent&status=FAILED$/)
      await expect.poll(() => mailQueries.at(-1)?.get("status")).toBe("FAILED")
      expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
      expect(requests).not.toContain("GET /api/dashboard/funnel")
    })
  }
}

test("dashboard failures stay unavailable and refresh independent metrics", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let reviewReads = 0
  let mailReads = 0
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: dashboard }
    if (pathname === "/api/applications") {
      reviewReads += 1
      if (reviewReads === 1) return { status: 503, message: "Review service unavailable" }
      return { data: { applications: [], total: 500, total_label: "500+", total_exact: false } }
    }
    if (pathname === "/api/mails/stats") {
      mailReads += 1
      if (mailReads === 2) return { status: 503, message: "Mail service unavailable" }
      return { data: { status_counts: { SENT: 10 } } }
    }
    return undefined
  })
  await page.goto("/dashboard")
  const reviews = page.getByTestId("dashboard-reviews")
  const business = page.getByTestId("dashboard-business")
  await expect(reviews.getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(reviews.getByText("暂无待审核申请。", { exact: true })).toHaveCount(0)
  await expect(business.getByText("10", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "刷新", exact: true }).click()
  await expect(reviews.getByText("500+", { exact: true })).toBeVisible()
  await expect(reviews.getByText("此处为已统计数量，可能仍有更多待审核申请。", { exact: true })).toBeVisible()
  await expect(business.getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(business.getByText("10", { exact: true })).toHaveCount(0)
  expect(reviewReads).toBe(2)
  expect(mailReads).toBe(2)
})

test("dashboard distinguishes a confirmed zero from malformed counts", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let reviewReads = 0
  let mailReads = 0
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: dashboard }
    if (pathname === "/api/applications") {
      reviewReads += 1
      return { data: reviewReads === 1 ? { total: 0, total_label: "0", total_exact: true } : { applications: [] } }
    }
    if (pathname === "/api/mails/stats") {
      mailReads += 1
      return { data: mailReads === 1 ? { status_counts: { SENT: 0, FAILED: 0, SCHEDULING: 0, CANCELLED: 0 } } : { status_counts: { FAILED: "unknown" } } }
    }
    return undefined
  })
  await page.goto("/dashboard")
  const reviews = page.getByTestId("dashboard-reviews")
  const business = page.getByTestId("dashboard-business")
  await expect(reviews.getByText("0", { exact: true })).toBeVisible()
  await expect(reviews.getByText("暂无待审核申请。", { exact: true })).toBeVisible()
  await expect(business.getByText("0", { exact: true })).toHaveCount(4)
  await page.getByRole("button", { name: "刷新", exact: true }).click()
  await expect(reviews.getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(business.getByText("暂时无法获取", { exact: true })).toBeVisible()
})

test("dashboard marks incomplete revenue and pending counts as lower bounds", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: { ...dashboard, today_revenue_exact: false } }
    if (pathname === "/api/dashboard/pending-age") return { data: { created_older_48h: 4, exact: false, aggregation_sample_limit: 500 } }
    if (pathname === "/api/applications") return { data: { applications: [], total: 5, total_label: "5", total_exact: true } }
    if (pathname === "/api/mails/stats") return { data: { status_counts: { SENT: 1 } } }
    return undefined
  })
  await page.goto("/dashboard")
  await expect(page.getByTestId("dashboard-revenue").getByText("订单扫描未覆盖全部记录，暂不展示 48 小时对比。")).toBeVisible()
  await expect(page.getByTestId("dashboard-pending-age").getByText(/≥4/)).toBeVisible()
  await expect(page.getByTestId("dashboard-memberships").getByText("3 个考生账号", { exact: true })).toBeVisible()
})
