import { expect, test } from "@playwright/test"
import {
  candidateUser,
  installCandidateApiMocks,
  seedAuthenticatedCandidate,
} from "./support/candidate"

test("未登录访问受保护页面时保留目标地址并进入登录流程", async ({ page }) => {
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/auth/login-url") {
      return { data: { url: "http://127.0.0.1:4173/e2e-login-target" } }
    }
    return undefined
  })

  await page.goto("/orders", { waitUntil: "domcontentloaded" })

  await expect(page).toHaveURL(/\/e2e-login-target$/)
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem("post_login_redirect"))).toBe("/orders")
})

test("公共商城移动端菜单只保留语言和认证入口", async ({ page }) => {
  await page.setViewportSize({ width: 382, height: 739 })
  await installCandidateApiMocks(page)

  await page.goto("/", { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: "Open menu" }).click()

  await expect(page.getByRole("button", { name: "关于", exact: true })).toHaveCount(0)
  await expect(page.getByRole("button", { name: "中文", exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "English", exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "登录 / 注册", exact: true })).toBeVisible()
})

test("access token 过期时只刷新一次并继续停留在当前页面", async ({ page }) => {
  let userRequests = 0
  let refreshRequests = 0

  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/user/me") {
      userRequests += 1
      if (userRequests === 1) {
        return { status: 401, code: 401, message: "access token expired" }
      }
      return { data: candidateUser }
    }
    if (pathname === "/api/auth/refresh") {
      refreshRequests += 1
      return { data: { refreshed: true } }
    }
    return undefined
  })

  await page.goto("/settings", { waitUntil: "domcontentloaded" })

  await expect(page.getByRole("heading", { name: "设置", exact: true })).toBeVisible()
  await expect.poll(() => userRequests).toBeGreaterThanOrEqual(2)
  await expect.poll(() => refreshRequests).toBe(1)
  await expect.poll(() => page.evaluate(() => localStorage.getItem("is_authenticated"))).toBe("true")
  await expect(page.getByText("登录已过期，请重新登录")).toHaveCount(0)
})

test("refresh token 也失效时清理本地会话并提示重新登录", async ({ page }) => {
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/user/me" || pathname === "/api/auth/refresh") {
      return { status: 401, code: 401, message: "session expired" }
    }
    if (pathname === "/api/auth/login-url") {
      return { data: { url: "http://127.0.0.1:4173/e2e-login-target" } }
    }
    return undefined
  })

  await page.goto("/settings", { waitUntil: "domcontentloaded" })

  await expect(page.getByText("登录已过期，请重新登录")).toBeVisible()
  await expect.poll(() => page.evaluate(() => localStorage.getItem("is_authenticated"))).toBeNull()
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem("post_login_redirect"))).toBe("/settings")
})
