import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const candidate = { id: "user-a", candidate_ulid: "candidate-a", name: "Candidate A", email: "a@example.test", role_label: "student", user_group: "candidate", is_member: false, status: "Active", email_verified: true, mfa_enabled: false, last_login_at: "2026-11-23T09:30:00Z", created_at: "2026-01-01T00:00:00Z" }
const staff = { ...candidate, id: "staff", name: "Staff A", candidate_ulid: "staff-candidate-id", user_group: "staff", role_label: "admin", mfa_enabled: true }
const member = { candidate_ulid: "candidate-a", membership_ulid: "plan-a", started_at: "2000-01-01T00:00:00Z", expires_at: "2999-01-01T00:00:00Z" }
function overview(users = [candidate]) {
  return { candidate_total: 2, user_stats: { total: 3, active: 3, admins: 1, members: 1 }, users, user_total: users.length, user_page: 1, user_page_size: 10, stage_buckets: [], stage_buckets_exact: true, today_revenue: [], today_revenue_exact: true, membership_distribution: { candidate_accounts: 2, no_membership_role: 1, roles: [{ role: "role_membership_affiliate", count: 1 }] } }
}
function registration(id: string, status = 1) {
  return { pipeline_ulid: id, pipeline_cc_ulid: "cert-config", status, created_at: "2026-01-01T00:00:00Z" }
}

for (const lang of ["zh", "en"] as const) {
  for (const mobile of [false, true]) {
    test(`separate user groups, actual membership and independent registrations (${lang}, ${mobile ? "mobile" : "desktop"})`, async ({ page }, testInfo) => {
      await seedAuthenticatedAdmin(page, lang)
      await page.setViewportSize({ width: mobile ? 390 : 1440, height: 900 })
      const membershipIds: string[] = []
      const registrationQueries: URLSearchParams[] = []
      const methods: string[] = []
      await installAdminApiMocks(page, ({ method, pathname, url }) => {
        methods.push(method)
        if (pathname === "/api/dashboard/ops") return { data: overview(url.searchParams.get("user_group") === "staff" ? [staff] : [candidate, { ...candidate, id: "user-b", candidate_ulid: "candidate-b", name: "Candidate B", is_member: true }]) }
        if (pathname === "/api/memberships/active") {
          membershipIds.push(url.searchParams.get("candidate_ulid") || "")
          return { data: url.searchParams.get("candidate_ulid") === "candidate-a" ? { membership: member } : {} }
        }
        if (pathname === "/api/memberships/plan-a") return { data: { name: "Affiliate Member" } }
        if (pathname === "/api/memberships/plan-a/translations") return { data: { translations: { "zh-CN": { name: "附属会员" }, "en-US": { name: "Affiliate Member" } } } }
        if (pathname === "/api/prog/pipelines") {
          registrationQueries.push(url.searchParams)
          if (url.searchParams.get("status") === "3") return { data: { pipelines: [registration("completed-registration", 3)], has_more: false } }
          if (url.searchParams.has("cursor")) return { data: { pipelines: [registration("third-registration", 3)], has_more: false } }
          return { data: { pipelines: [registration("first-registration"), registration("second-registration", 2)], has_more: true, next_cursor: "next" } }
        }
        if (pathname === "/api/pipelines/cert-config") return { data: { name: "CFtP Certification" } }
        return undefined
      })
      await page.goto("/dashboard")
      const users = page.locator("section").filter({ has: page.getByRole("heading", { name: lang === "zh" ? "用户管理" : "User Management", exact: true }) })
      const rows = mobile ? users.locator("article") : users.locator("tbody tr")
      await expect(rows).toHaveCount(2)
      await expect(rows.first().getByText(lang === "zh" ? "附属会员" : "Affiliate Member", { exact: true })).toBeVisible()
      await expect(rows.nth(1).getByText(lang === "zh" ? "暂时无法获取" : "Temporarily unavailable", { exact: true })).toBeVisible()
      expect(membershipIds.sort()).toEqual(["candidate-a", "candidate-b"])
      await expect(users.getByRole("button", { name: lang === "zh" ? "全部账号" : "All accounts", exact: true })).toHaveCount(0)
      expect(registrationQueries).toHaveLength(0)
      await rows.first().locator("summary").click()
      await rows.first().getByRole("button", { name: lang === "zh" ? "查看详情" : "View Details", exact: true }).click()
      const dialog = page.getByRole("dialog")
      await expect(dialog.getByText("first-registration", { exact: true })).toBeVisible()
      await expect(dialog.getByText("second-registration", { exact: true })).toBeVisible()
      await expect(dialog.getByText("CFtP Certification", { exact: true })).toHaveCount(2)
      await expect(dialog.getByText(lang === "zh" ? "会员到期日期" : "Membership Expires", { exact: true })).toBeVisible()
      expect(registrationQueries[0]?.get("candidate_ulid")).toBe("candidate-a")
      await dialog.getByRole("button", { name: lang === "zh" ? "加载更多" : "Load More", exact: true }).click()
      await expect(dialog.getByText("third-registration", { exact: true })).toBeVisible()
      expect(registrationQueries[1]?.get("cursor")).toBe("next")
      await dialog.getByRole("combobox").selectOption("3")
      await expect(dialog.getByText("completed-registration", { exact: true })).toBeVisible()
      await expect(dialog.getByText("first-registration", { exact: true })).toHaveCount(0)
      expect(registrationQueries.at(-1)?.has("cursor")).toBe(false)
      await page.screenshot({ path: testInfo.outputPath("candidate-detail.png"), fullPage: true })
      await dialog.getByRole("button", { name: lang === "zh" ? "关闭" : "Close", exact: true }).click()
      await users.getByRole("button", { name: lang === "zh" ? "内部员工" : "Internal Staff", exact: true }).click()
      await expect(rows).toHaveCount(1)
      await expect(rows.first().getByText("Staff A", { exact: true })).toBeVisible()
      await expect(users.getByText(lang === "zh" ? "会员等级" : "Membership Tier", { exact: true })).toHaveCount(0)
      await expect(rows.first().getByText(lang === "zh" ? "已启用" : "Enabled", { exact: true }).last()).toBeVisible()
      await expect(rows.first()).toContainText("2026")
      await rows.first().locator("summary").focus()
      await page.keyboard.press("Enter")
      await expect(rows.first().getByRole("button", { name: lang === "zh" ? "查看详情" : "View Details", exact: true })).toBeVisible()
      await expect(rows.first().getByRole("button", { name: lang === "zh" ? "查看课程报读" : "View Course Enrollments", exact: true })).toHaveCount(0)
      await page.keyboard.press("Escape")
      await expect(rows.first().locator("details")).not.toHaveAttribute("open", "")
      await expect(rows.first().locator("summary")).toBeFocused()
      await page.screenshot({ path: testInfo.outputPath("staff-list.png"), fullPage: true })
      expect(membershipIds).not.toContain("staff-candidate-id")
      expect(methods.every((method) => method === "GET")).toBe(true)
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= window.innerWidth)).toBe(true)
    })
  }
}

test("membership expiry, missing identity and service errors are distinguished", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const ids: string[] = []
  await installAdminApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/dashboard/ops") return { data: overview([
      { ...candidate, candidate_ulid: "expired" },
      { ...candidate, id: "unavailable", candidate_ulid: "unavailable", name: "Unavailable" },
      { ...candidate, id: "unlinked", candidate_ulid: "", name: "Unlinked" },
    ]) }
    if (pathname === "/api/memberships/active") {
      const id = url.searchParams.get("candidate_ulid") || ""
      ids.push(id)
      if (id === "unavailable") return { status: 503 }
      return { data: { membership: { ...member, candidate_ulid: id, expires_at: "2001-01-01T00:00:00Z" } } }
    }
    return undefined
  })
  await page.goto("/dashboard")
  const rows = page.locator("tbody tr")
  await expect(rows.nth(0).getByText("无有效会员", { exact: true })).toBeVisible()
  await expect(rows.nth(1).getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(rows.nth(2).getByText("尚未关联考生身份", { exact: true })).toBeVisible()
  expect(ids.sort()).toEqual(["expired", "unavailable"])
})

test("registration errors retain loaded rows and retry the same cursor", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let reads = 0
  const cursors: string[] = []
  await installAdminApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/dashboard/ops") return { data: overview() }
    if (pathname === "/api/prog/pipelines") {
      reads += 1
      cursors.push(url.searchParams.get("cursor") || "")
      if (reads === 1) return { data: { pipelines: [registration("first")], has_more: true, next_cursor: "next" } }
      if (reads === 2) return { status: 503 }
      return { data: { pipelines: [registration("second")], has_more: false } }
    }
    return undefined
  })
  await page.goto("/dashboard")
  await page.locator("table summary").click()
  await page.locator("table").getByRole("button", { name: "查看详情", exact: true }).click()
  const dialog = page.getByRole("dialog")
  await dialog.getByRole("button", { name: "加载更多", exact: true }).click()
  await expect(dialog.getByRole("alert")).toHaveText("认证报名记录加载失败，请重试")
  await expect(dialog.getByText("first", { exact: true })).toBeVisible()
  await dialog.getByRole("button", { name: "重试", exact: true }).click()
  await expect(dialog.getByText("second", { exact: true })).toBeVisible()
  expect(cursors).toEqual(["", "next", "next"])
})
