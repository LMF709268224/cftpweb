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

for (const lang of ["zh", "en"] as const) {
  for (const mobile of [false, true]) {
    test(`candidate enrollments are scoped, paginated and read-only (${lang}, ${mobile ? "mobile" : "desktop"})`, async ({ page }) => {
      await page.setViewportSize({ width: mobile ? 390 : 1440, height: 900 })
      await seedAuthenticatedAdmin(page, lang)
      const requests: string[] = []
      const enrollmentQueries: URLSearchParams[] = []
      await installAdminApiMocks(page, ({ method, pathname, url }) => {
        requests.push(`${method} ${pathname}`)
        if (pathname === "/api/dashboard/ops") {
          return { data: dashboardData("Candidate A", {
            users: [
              { ...dashboardData("Candidate A").users[0], id: "casdoor-user-a", candidate_ulid: "candidate-a", role_label: "member" },
              { ...dashboardData("Candidate B").users[0], id: "casdoor-user-b", candidate_ulid: "candidate-b", role_label: "student" },
              { ...dashboardData("Staff").users[0], candidate_ulid: "" },
            ],
          }) }
        }
        if (pathname !== "/api/lms/enrollments") return undefined
        enrollmentQueries.push(url.searchParams)
        if (url.searchParams.get("candidate_id") === "candidate-b") return { data: { enrollments: [], has_more: false } }
        const secondPage = url.searchParams.get("cursor") === "page-two"
        return { data: {
          enrollments: [{ enrollment_id: secondPage ? "enrollment-2" : "enrollment-1", course_ulid: secondPage ? "course-2" : "course-1", status: secondPage ? "Completed" : "Learning", progress_percentage: secondPage ? 100 : 0, joined_at: "2026-08-11T00:00:00Z" }],
          has_more: !secondPage,
          next_cursor: secondPage ? "" : "page-two",
        } }
      })
      await page.goto("/dashboard")
      const rows = mobile ? page.locator("section article").filter({ has: page.getByRole("button", { name: lang === "zh" ? "查看详情" : "View Details", exact: true }) }) : page.locator("tbody tr")
      const entryLabel = lang === "zh" ? "查看课程报读" : "View Course Enrollments"
      await expect(rows.nth(2)).toBeVisible()
      await expect(rows.nth(2).getByRole("button", { name: entryLabel, exact: true })).toHaveCount(0)
      expect(enrollmentQueries).toHaveLength(0)
      const trigger = rows.first().getByRole("button", { name: entryLabel, exact: true })
      await trigger.click()
      const dialog = page.getByRole("dialog", { name: lang === "zh" ? "课程报读" : "Course Enrollments", exact: true })
      await expect(dialog.getByText("candidate-a", { exact: true })).toBeVisible()
      await expect(dialog.getByText("enrollment-1", { exact: true })).toBeVisible()
      await expect(dialog.getByText("course-1", { exact: true })).toBeVisible()
      await expect(dialog.getByText("0%", { exact: true })).toBeVisible()
      expect(enrollmentQueries[0]?.get("candidate_id")).toBe("candidate-a")
      expect(enrollmentQueries[0]?.get("page_size")).toBe("20")
      await dialog.getByRole("button", { name: lang === "zh" ? "加载更多" : "Load More", exact: true }).click()
      await expect(dialog.getByText("enrollment-2", { exact: true })).toBeVisible()
      await expect(dialog.getByText("enrollment-1", { exact: true })).toBeVisible()
      await expect(dialog.getByText("100%", { exact: true })).toBeVisible()
      expect(enrollmentQueries[1]?.get("candidate_id")).toBe("candidate-a")
      expect(enrollmentQueries[1]?.get("cursor")).toBe("page-two")
      await expect(dialog.getByRole("button", { name: lang === "zh" ? "加载更多" : "Load More", exact: true })).toHaveCount(0)
      await page.keyboard.press("Escape")
      await expect(dialog).toHaveCount(0)
      await expect(trigger).toBeFocused()
      await rows.nth(1).getByRole("button", { name: entryLabel, exact: true }).click()
      await expect(dialog.getByText(lang === "zh" ? "暂无课程报读记录" : "No course enrollments", { exact: true })).toBeVisible()
      await expect(dialog.getByText("course-1", { exact: true })).toHaveCount(0)
      expect(enrollmentQueries[2]?.get("candidate_id")).toBe("candidate-b")
      expect(enrollmentQueries[2]?.has("cursor")).toBe(false)
      await dialog.getByRole("button", { name: lang === "zh" ? "关闭" : "Close", exact: true }).click()
      expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
    })
  }
}

test("candidate enrollments show errors, retry the same page and ignore closed requests", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let reads = 0
  const cursors: string[] = []
  await installAdminApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/dashboard/ops") return { data: dashboardData("Candidate A") }
    if (pathname !== "/api/lms/enrollments") return undefined
    reads += 1
    cursors.push(url.searchParams.get("cursor") || "")
    if (reads === 1 || reads === 3) return { status: 503, message: "GLMS unavailable" }
    if (reads === 2) return { data: { enrollments: [{ enrollment_id: "first-page", course_ulid: "course-a" }], has_more: true, next_cursor: "next" } }
    if (reads === 4) return { data: { enrollments: [{ enrollment_id: "second-page", course_ulid: "course-b" }], has_more: false } }
    return { data: { enrollments: [{ enrollment_id: reads === 5 ? "stale-enrollment" : "fresh-enrollment" }], has_more: false }, delayMs: reads === 5 ? 700 : 0 }
  })
  await page.goto("/dashboard")
  const trigger = page.locator("table").getByRole("button", { name: "查看课程报读", exact: true })
  const dialog = page.getByRole("dialog", { name: "课程报读", exact: true })
  await trigger.click()
  await expect(dialog.getByRole("alert")).toHaveText("课程报读加载失败，请重试")
  await expect(dialog.getByText("暂无课程报读记录", { exact: true })).toHaveCount(0)
  await dialog.getByRole("button", { name: "重试", exact: true }).click()
  await expect(dialog.getByText("first-page", { exact: true })).toBeVisible()
  await dialog.getByRole("button", { name: "加载更多", exact: true }).click()
  await expect(dialog.getByRole("alert")).toBeVisible()
  await expect(dialog.getByText("first-page", { exact: true })).toBeVisible()
  await dialog.getByRole("button", { name: "重试", exact: true }).click()
  await expect(dialog.getByText("second-page", { exact: true })).toBeVisible()
  expect(cursors).toEqual(["", "", "next", "next"])
  await dialog.getByRole("button", { name: "关闭", exact: true }).click()
  const slowResponse = page.waitForResponse((response) => new URL(response.url()).pathname === "/api/lms/enrollments")
  await trigger.click()
  await expect.poll(() => reads).toBe(5)
  await page.keyboard.press("Escape")
  await trigger.click()
  await expect(dialog.getByText("fresh-enrollment", { exact: true })).toBeVisible()
  await slowResponse
  await expect(dialog.getByText("stale-enrollment", { exact: true })).toHaveCount(0)
})

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
  await expect(page.getByText("资料完整度", { exact: true })).toHaveCount(0)
  await expect(page.getByText("75%", { exact: true })).toHaveCount(0)
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

for (const lang of ["zh", "en"] as const) {
  for (const mobile of [false, true]) {
    test(`dashboard user columns and verification marks match feedback (${lang}, ${mobile ? "mobile" : "desktop"})`, async ({ page }) => {
      await page.setViewportSize({ width: mobile ? 390 : 1440, height: 900 })
      await seedAuthenticatedAdmin(page, lang)
      const queries: URLSearchParams[] = []
      const requests: string[] = []
      await installAdminApiMocks(page, ({ method, pathname, url }) => {
        requests.push(`${method} ${pathname}`)
        if (pathname !== "/api/dashboard/ops") return undefined
        queries.push(url.searchParams)
        const users = ["admin", "student", "member"].map((role, index) => ({
          ...dashboardData(`User ${index}`).users[0],
          role_label: role,
          phone: index === 1 ? "" : `+861380000000${index}`,
          status: index === 1 ? "inactive" : "active",
          email_verified: index !== 1,
        }))
        return { data: dashboardData("Admin", { users, user_page: Number(url.searchParams.get("user_page") || 1) }) }
      })
      await page.goto("/dashboard")
      const section = page.locator("section").filter({ has: page.getByRole("heading", { name: lang === "zh" ? "用户管理" : "User Management", exact: true }) })
      const roles = lang === "zh" ? ["管理员", "学生", "会员"] : ["admin", "student", "member"]
      const verified = lang === "zh" ? "已验证" : "Verified"
      const unverified = lang === "zh" ? "未验证" : "Unverified"
      const columns = lang === "zh" ? ["用户", "邮箱", "角色", "状态", "邮箱验证", "创建时间"] : ["User", "Email", "Role", "Status", "Email Verified", "Created"]
      const rows = mobile ? section.locator("article") : section.locator("tbody tr")
      await expect(rows.first()).toBeVisible()
      await expect(section.getByText("Shanghai", { exact: true })).toHaveCount(0)
      await expect(section.getByText(lang === "zh" ? "地区" : "Location", { exact: true })).toHaveCount(0)
      if (!mobile) await expect(section.getByRole("columnheader")).toHaveText([...columns, lang === "zh" ? "操作" : "Actions"])
      for (const [index, role] of roles.entries()) {
        await expect(rows.nth(index).getByText(role, { exact: true })).toBeVisible()
        await expect(section.getByRole("option", { name: role, exact: true })).toHaveCount(1)
        const mark = rows.nth(index).locator(`[title="${index === 1 ? unverified : verified}"]`)
        await expect(mark).toBeVisible()
        await expect(mark.locator(index === 1 ? "svg.lucide-x" : "svg.lucide-check")).toBeVisible()
        await expect(mark.locator(".sr-only")).toHaveText(index === 1 ? unverified : verified)
      }
      if (mobile) {
        await expect(rows.first().locator(".mt-3 > div > span:first-child")).toHaveText(columns.slice(1))
      }
      const initialRequests = [...requests]
      for (const [index, role] of roles.entries()) {
        const trigger = rows.nth(index).getByRole("button", { name: lang === "zh" ? "查看详情" : "View Details", exact: true })
        await trigger.click()
        const dialog = page.getByRole("dialog", { name: lang === "zh" ? "用户详情" : "User Details", exact: true })
        await expect(dialog).toBeVisible()
        await expect(dialog.locator("dt")).toHaveText([columns[0], columns[1], lang === "zh" ? "电话" : "Phone", ...columns.slice(2)])
        const values = dialog.locator("dd")
        await expect(values.nth(0)).toHaveText(`User ${index}`)
        await expect(values.nth(1)).toHaveText(`user.${index}@example.test`)
        await expect(values.nth(2)).toHaveText(index === 1 ? "-" : `+861380000000${index}`)
        await expect(values.nth(3)).toHaveText(role)
        await expect(values.nth(4)).toHaveText(lang === "zh" ? (index === 1 ? "停用" : "活跃") : (index === 1 ? "Inactive" : "Active"))
        await expect(values.nth(5)).toHaveText(index === 1 ? unverified : verified)
        await expect(values.nth(6)).not.toHaveText("-")
        await expect(dialog.locator("input, select, textarea")).toHaveCount(0)
        const close = dialog.getByRole("button", { name: lang === "zh" ? "关闭" : "Close", exact: true })
        await expect(close).toBeFocused()
        await page.keyboard.press("Tab")
        await expect(close).toBeFocused()
        if (index === 0) await page.keyboard.press("Escape")
        else await close.click()
        await expect(dialog).toHaveCount(0)
        await expect(trigger).toBeFocused()
      }
      expect(requests).toEqual(initialRequests)
      await section.getByRole("combobox").first().selectOption("student")
      await expect.poll(() => queries.at(-1)?.get("user_role")).toBe("student")
      await expect(rows.first()).toBeVisible()
      await section.getByRole("button", { name: lang === "zh" ? "下一页" : "Next", exact: true }).click()
      await expect.poll(() => queries.at(-1)?.get("user_page")).toBe("2")
      expect(queries.at(-1)?.get("user_role")).toBe("student")
      expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
    })
  }
}
