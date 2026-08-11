import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const adminProfile = {
  name: "regression-admin",
  email: "regression-admin@example.test",
  display_name: "Regression Administrator",
  real_name: "Admin Reader",
  affiliation: "CFTP",
  title: "Operations",
  bio: "Read-only regression profile",
  gender: "female",
  birthday: "1990-08-11T00:00:00Z",
  education: "Regression University",
}

test("settings displays the complete administrator profile using only GET", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname === "/api/user/me") return { data: adminProfile }
    return undefined
  })

  await page.goto("/settings")

  await expect(page.getByLabel("登录 ID")).toHaveValue("regression-admin")
  await expect(page.getByLabel("邮箱")).toHaveValue("regression-admin@example.test")
  await expect(page.getByLabel("显示名称")).toHaveValue("Regression Administrator")
  await expect(page.getByLabel("真实姓名")).toHaveValue("Admin Reader")
  await expect(page.getByLabel("性别")).toHaveValue("女")
  await expect(page.getByLabel("生日")).toHaveValue("1990-08-11")
  await expect(page.getByLabel("机构")).toHaveValue("CFTP")
  await expect(page.getByLabel("职位")).toHaveValue("Operations")
  await expect(page.getByLabel("教育背景")).toHaveValue("Regression University")
  await expect(page.getByLabel("简介")).toHaveValue("Read-only regression profile")
  expect(requests).toContain("GET /api/user/me")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("settings recovers from a failed profile read when reloaded", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let profileReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/user/me") return undefined
    profileReads += 1
    if (profileReads === 1) return { status: 503, errorCode: "PROFILE_UNAVAILABLE", message: "Profile unavailable" }
    return { data: adminProfile }
  })
  await page.goto("/settings")
  await expect(page.getByText("个人资料加载失败", { exact: true })).toBeVisible()

  await page.getByRole("button", { name: "重新加载" }).click()
  await expect(page.getByLabel("显示名称")).toHaveValue("Regression Administrator")
  expect(profileReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
