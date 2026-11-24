import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const user = { id: "user-a", candidate_ulid: "candidate-a", name: "Candidate A", email: "a@example.test", role_label: "student", user_group: "candidate", status: "Active", email_verified: true, created_at: "2026-01-01T00:00:00Z" }
const overview = { users: [user], user_total: 1, user_page: 1, user_page_size: 10 }

test("missing dashboard counts and partial mail statistics never become zero", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let reads = 0
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: overview }
    if (pathname === "/api/mails/stats") return { data: ++reads === 1 ? {} : { status_counts: { FAILED: 0 } } }
    return undefined
  })
  await page.goto("/dashboard")
  const business = page.getByTestId("dashboard-business")
  await expect(business.getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(page.getByTestId("dashboard-revenue").getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(page.getByText("暂无收款", { exact: true })).toHaveCount(0)
  await page.getByRole("button", { name: "刷新", exact: true }).click()
  await expect(business.locator('a[href$="status=FAILED"]').getByText("0", { exact: true })).toBeVisible()
  await expect(business.getByText("暂时无法获取", { exact: true })).toHaveCount(3)
  await expect(business.getByText("0", { exact: true })).toHaveCount(1)
})

test("missing user pagination is an error, not an empty or inferred page", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname }) => pathname === "/api/dashboard/ops" ? { data: { users: [user] } } : undefined)
  await page.goto("/dashboard")
  await expect(page.locator("table").getByRole("alert")).toHaveText("工作台加载失败")
  await expect(page.locator("table").getByText("Candidate A")).toHaveCount(0)
  await expect(page.getByText("暂无用户", { exact: true })).toHaveCount(0)
})

test("membership translation cannot be replaced by the original plan name", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const paths: string[] = []
  await installAdminApiMocks(page, ({ pathname }) => {
    paths.push(pathname)
    if (pathname === "/api/dashboard/ops") return { data: overview }
    if (pathname === "/api/memberships/active") return { data: { membership: { candidate_ulid: "candidate-a", membership_ulid: "plan-a", started_at: "2000-01-01T00:00:00Z", expires_at: "2999-01-01T00:00:00Z" } } }
    if (pathname === "/api/memberships/plan-a/translations") return { data: { translations: { "en-US": { name: "English only" } } } }
    if (pathname === "/api/memberships/plan-a") return { data: { name: "Original name" } }
    return undefined
  })
  await page.goto("/dashboard")
  await expect(page.locator("tbody tr td").nth(3)).toHaveText("暂时无法获取")
  expect(paths).not.toContain("/api/memberships/plan-a")
  await expect(page.getByText(/English only|Original name/)).toHaveCount(0)
})

test("missing membership and account fields do not imply negative statuses", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/dashboard/ops") return { data: { ...overview, users: [{ ...user, user_group: url.searchParams.get("user_group"), email_verified: undefined }] } }
    if (pathname === "/api/memberships/active") return { data: {} }
    return undefined
  })
  await page.goto("/dashboard")
  const row = page.locator("tbody tr")
  await expect(row.getByText("暂时无法获取", { exact: true })).toHaveCount(3)
  await expect(row.getByText("无有效会员", { exact: true })).toHaveCount(0)
  await expect(row.getByText("未验证", { exact: true })).toHaveCount(0)
  await page.getByRole("button", { name: "内部员工", exact: true }).click()
  await expect(row.getByText("暂时无法获取", { exact: true })).toHaveCount(2)
  await expect(row.getByText("未启用", { exact: true })).toHaveCount(0)
})

test("missing registration pagination is reported without claiming an empty list", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: overview }
    if (pathname === "/api/prog/pipelines") return { data: {} }
    return undefined
  })
  await page.goto("/dashboard")
  await page.locator("table summary").click()
  await page.locator("table").getByRole("button", { name: "查看详情", exact: true }).click()
  await expect(page.getByRole("dialog").getByRole("alert")).toHaveText("认证报名记录加载失败，请重试")
})

test("absent course progress stays unavailable and zero remains zero", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") return { data: overview }
    if (pathname === "/api/lms/enrollments") return { data: { enrollments: [{ enrollment_id: "missing" }, { enrollment_id: "zero", progress_percentage: 0 }], has_more: false } }
    return undefined
  })
  await page.goto("/dashboard")
  await page.locator("table summary").click()
  await page.locator("table").getByRole("button", { name: "查看课程报读", exact: true }).click()
  const rows = page.getByRole("dialog").locator("article")
  await expect(rows.first().getByText("暂时无法获取", { exact: true })).toBeVisible()
  await expect(rows.first().getByText("0%", { exact: true })).toHaveCount(0)
  await expect(rows.nth(1).getByText("0%", { exact: true })).toBeVisible()
})
