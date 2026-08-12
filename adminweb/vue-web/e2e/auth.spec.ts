import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("protected route preserves its destination when redirecting to login", async ({ page }) => {
  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/auth/login-url") return { data: {} }
    return undefined
  })

  await page.goto("/orders?status=FAILED")

  await expect(page).toHaveURL(/\/login\?redirect=/)
  const currentURL = new URL(page.url())
  expect(currentURL.searchParams.get("redirect")).toBe("/orders?status=FAILED")
  await expect(page.locator("h1").first()).toBeVisible()
})

test("expired access session refreshes once and retries the admin request", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let dashboardRequests = 0
  let refreshRequests = 0

  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") {
      dashboardRequests += 1
      if (dashboardRequests === 1) {
        return { status: 401, errorCode: "TOKEN_EXPIRED", message: "expired" }
      }
    }
    if (pathname === "/api/auth/refresh") {
      refreshRequests += 1
      return { data: {} }
    }
    return undefined
  })

  await page.goto("/dashboard")

  await expect.poll(() => dashboardRequests).toBe(2)
  expect(refreshRequests).toBe(1)
  await expect(page).toHaveURL(/\/dashboard$/)
  await expect(page.locator("h1").first()).toBeVisible({ timeout: 20_000 })
})

test("expired refresh session clears local auth and preserves the dashboard destination", async ({ page }) => {
  let dashboardRequests = 0
  let refreshRequests = 0

  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/dashboard/ops") {
      dashboardRequests += 1
      return { status: 401, errorCode: "TOKEN_EXPIRED", message: "expired" }
    }
    if (pathname === "/api/auth/refresh") {
      refreshRequests += 1
      return { status: 401, errorCode: "INVALID_REFRESH_TOKEN", message: "expired refresh" }
    }
    if (pathname === "/api/auth/login-url") return { data: {} }
    return undefined
  })

  await page.goto("/login")
  await page.evaluate(() => {
    localStorage.setItem("is_authenticated", "true")
    localStorage.setItem("user_name", "Regression Admin")
    localStorage.setItem("app_lang", "zh")
  })
  await page.goto("/dashboard")

  await expect(page).toHaveURL(/\/login\?redirect=/)
  const currentURL = new URL(page.url())
  expect(currentURL.searchParams.get("redirect")).toBe("/dashboard")
  expect(dashboardRequests).toBe(1)
  expect(refreshRequests).toBe(1)
  await expect.poll(() => page.evaluate(() => localStorage.getItem("is_authenticated"))).toBeNull()
  await expect(page.locator("h1").first()).toBeVisible()
})

test("expired OAuth state restarts authentication once and then offers a manual retry", async ({ page }) => {
  let loginRequests = 0
  let loginURLRequests = 0

  await installAdminApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/auth/login") {
      loginRequests += 1
      return { status: 401, errorCode: "OAUTH_STATE_INVALID", message: "Unauthorized" }
    }
    if (pathname === "/api/auth/login-url") {
      loginURLRequests += 1
      return { data: {} }
    }
    return undefined
  })

  await page.goto("/callback?code=expired-code&state=expired-state")

  await expect(page).toHaveURL(/\/login\?redirect=/)
  await expect.poll(() => loginURLRequests).toBe(1)
  expect(loginRequests).toBe(1)
  await expect.poll(() => page.evaluate(() => sessionStorage.getItem("admin_oauth_state_recovery"))).toBe("1")

  await page.goto("/callback?code=expired-again&state=expired-again")

  await expect(page.getByText(/This login attempt expired|登录页面停留时间过长/)).toBeVisible()
  await expect(page.getByRole("button", { name: /Sign in again|重新登录/ })).toBeVisible()
  expect(loginRequests).toBe(2)
  expect(loginURLRequests).toBe(1)
})
