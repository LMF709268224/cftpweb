import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

function dashboardData(name: string, overrides: Record<string, unknown> = {}) {
  return {
    candidate_total: 3,
    user_stats: { total: 8, active: 6, inactive: 2, admins: 2, members: 3, email_verified: 7 },
    user_role_stats: [{ role: "admin", count: 2 }],
    profile_completion_percent: 75,
    users: [{
      id: `user-${name}`,
      candidate_ulid: `candidate-${name}`,
      name,
      email: `${name.toLowerCase().replaceAll(" ", ".")}@example.test`,
      location: "Shanghai",
      role_label: "admin",
      status: "active",
      email_verified: true,
      created_at: "2026-08-11T00:00:00Z",
    }],
    user_total: 12,
    user_page: 1,
    user_page_size: 10,
    stage_buckets: [{ stage_id: "stage-regression", status: "ACTIVE", count: 2 }],
		stage_buckets_exact: true,
    today_revenue: [{ currency: "USD", amount_minor: 12900, order_count: 1 }],
		today_revenue_exact: true,
		aggregation_sample_limit: 500,
    generated_at: "2026-08-11T00:00:00Z",
    ...overrides,
  }
}

test("dashboard renders populated user and summary data with read-only requests", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname === "/api/dashboard/ops") return { data: dashboardData("Regression Admin") }
    return undefined
  })

  await page.goto("/dashboard")

  await expect(page.locator("table").getByText("Regression Admin", { exact: true })).toBeVisible()
  await expect(page.locator("table").getByText("candidate-Regression Admin", { exact: true })).toBeVisible()
  await expect(page.getByText("75%", { exact: true })).toBeVisible()
	await expect(page.getByText("USD 129", { exact: true })).toBeVisible()
  await expect(page.getByText("第 1 页 / 共 12 人", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/dashboard/ops")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("dashboard labels truncated aggregates instead of presenting them as exact", async ({ page }) => {
	await seedAuthenticatedAdmin(page)
	await installAdminApiMocks(page, ({ pathname }) => {
		if (pathname === "/api/dashboard/ops") {
			return {
				data: dashboardData("Regression Admin", {
					stage_buckets_exact: false,
					today_revenue_exact: false,
				}),
			}
		}
		return undefined
	})

	await page.goto("/dashboard")
	await expect(page.getByText("数据超过单次扫描上限，当前结果基于最近 500 条记录。")).toHaveCount(2)
})

test("dashboard keeps the latest filter result when an older request finishes late", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/dashboard/ops") return undefined
    const keyword = url.searchParams.get("user_keyword") || ""
    if (keyword === "Slow") return { data: dashboardData("Stale User"), delayMs: 700 }
    if (keyword === "Latest") return { data: dashboardData("Latest User", { user_total: 1 }) }
    return { data: dashboardData("Initial User") }
  })
  await page.goto("/dashboard")
  await expect(page.locator("table").getByText("Initial User", { exact: true })).toBeVisible()

  const search = page.getByPlaceholder("搜索用户姓名或邮箱...")
  const slowRequest = page.waitForRequest((request) => new URL(request.url()).searchParams.get("user_keyword") === "Slow")
  const slowResponse = page.waitForResponse((response) => new URL(response.url()).searchParams.get("user_keyword") === "Slow")
  await search.fill("Slow")
  await slowRequest
  await search.fill("Latest")

  await expect(page.locator("table").getByText("Latest User", { exact: true })).toBeVisible()
  await slowResponse
  await expect(page.getByText("Stale User", { exact: true })).toHaveCount(0)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("dashboard recovers after its initial read fails", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let dashboardReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/dashboard/ops") return undefined
    dashboardReads += 1
    if (dashboardReads === 1) return { status: 503, errorCode: "DASHBOARD_UNAVAILABLE", message: "Dashboard unavailable" }
    return { data: dashboardData("Recovered Administrator", { user_total: 1 }) }
  })

  await page.goto("/dashboard")
  await expect(page.getByText("工作台加载失败", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "刷新", exact: true }).click()

  await expect(page.locator("table").getByText("Recovered Administrator", { exact: true })).toBeVisible()
  expect(dashboardReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
