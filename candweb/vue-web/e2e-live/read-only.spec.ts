import { expect, test } from "@playwright/test"
import { installReadOnlyGuards, liveEnvironment } from "./support/live"

test.setTimeout(600_000)

const portalPages = [
  { path: "/dashboard", heading: "欢迎来到门户" },
  { path: "/certifications", heading: "商城" },
  { path: "/my-certifications", heading: "我的认证" },
  { path: "/exams", heading: "考试" },
  { path: "/records", heading: "档案" },
  { path: "/resource-packs", heading: "资源包" },
  { path: "/credentials", heading: "资格申请" },
  { path: "/certificates", heading: "证书" },
  { path: "/membership", heading: "会员" },
  { path: "/orders", heading: "订单" },
  { path: "/messages", heading: "消息" },
  { path: "/settings", heading: "设置" },
]

test("real candidate session can read health and current-user APIs", async ({ page }) => {
  const environment = liveEnvironment()
  const healthResponse = await page.context().request.get(new URL("/health", environment.baseURL).toString())
  expect(healthResponse.status()).toBe(200)

  const userResponse = await page.context().request.get(new URL("/api/user/me", environment.baseURL).toString())
  expect(userResponse.status()).toBe(200)

  const payload = await userResponse.json()
  expect(payload.code).toBe(200)
  expect(payload.data).toBeTruthy()
  expect(String(payload.data.name || payload.data.id || "").trim()).not.toBe("")
})

test("candidate main pages can read real test-environment data", async ({ page }) => {
  const guards = await installReadOnlyGuards(page)

  for (const portalPage of portalPages) {
    await test.step(`${portalPage.heading}页面`, async () => {
      guards.reset()
      await page.goto(portalPage.path, { waitUntil: "domcontentloaded" })

      await expect(page).toHaveURL(new RegExp(`${portalPage.path.replaceAll("/", "\\/")}(?:[?#].*)?$`))
      await expect(
        page.getByRole("heading", { name: portalPage.heading, exact: true }).first(),
      ).toBeVisible({ timeout: 45_000 })

      await guards.waitForAPIIdle()
      await guards.assertClean()
    })
  }
})
