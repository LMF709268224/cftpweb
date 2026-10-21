import { expect, test } from "@playwright/test"
import { candidateUser, installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

test("profile keeps identity fields read-only and saves the preferred language", async ({ page }) => {
  const languageRequests: unknown[] = []
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/user/me") return { data: { ...candidateUser, language: "en" } }
    if (pathname === "/api/user/language" && method === "PUT") {
      languageRequests.push(body)
      return { data: { success: true } }
    }
    return undefined
  })
  await page.goto("/settings")
  await expect(page.getByRole("heading", { name: "Profile", exact: true })).toBeVisible()
  await expect(page.getByText("Login ID")).toHaveCount(0)
  await expect(page.getByText("Display Name")).toHaveCount(0)
  await expect(page.getByText("Real Name")).toHaveCount(0)
  await page.getByLabel("Language preference").selectOption("zh")
  await expect.poll(() => languageRequests).toContainEqual({ language: "zh" })
  await expect(page.getByRole("heading", { name: "设置", exact: true }).first()).toBeVisible()
})

test("security settings starts email verification and confirmation clears the session", async ({ page }) => {
  const emailRequests: unknown[] = []
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/user/email/send-code" && method === "POST") return { data: { email: candidateUser.email } }
    if (pathname === "/api/user/email/request" && method === "POST") {
      emailRequests.push(body)
      return { data: { status: "sent" } }
    }
    if (pathname === "/api/auth/email/confirm" && method === "POST") return { data: { status: "updated" } }
    return undefined
  })
  await page.goto("/settings?tab=account")
  await expect(page.getByText(candidateUser.email)).toBeVisible()
  await page.getByRole("button", { name: "发送验证码" }).click()
  await page.getByLabel("新邮箱").fill("new@example.test")
  await page.getByLabel("当前邮箱验证码").fill("123456")
  await page.getByRole("button", { name: "发送确认链接" }).click()
  await expect.poll(() => emailRequests).toContainEqual({ new_email: "new@example.test", code: "123456" })

  await page.goto("/email-confirm?id=test-request&side=new&token=test-token")
  await expect(page.getByText("邮箱修改完成，请使用新邮箱重新登录。")).toBeVisible()
  await expect(page).toHaveURL(/\/email-confirm$/)
  await expect.poll(() => page.evaluate(() => localStorage.getItem("is_authenticated"))).toBeNull()
})

test("profile photo crop previews pixels and uploads the cropped PNG", async ({ page }) => {
  let uploaded = false
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/user/avatar" && method === "POST") {
      uploaded = true
      return { data: { avatar: "https://example.test/avatar.png" } }
    }
    return undefined
  })
  await page.goto("/settings")
  const imageDataURL = await page.evaluate(() => {
    const canvas = document.createElement("canvas")
    canvas.width = 2
    canvas.height = 2
    const ctx = canvas.getContext("2d")!
    ctx.fillStyle = "#e62626"
    ctx.fillRect(0, 0, 2, 2)
    return canvas.toDataURL("image/png")
  })
  await page.getByLabel("个人头像", { exact: true }).setInputFiles({
    name: "sample.png", mimeType: "image/png", buffer: Buffer.from(imageDataURL.split(",")[1], "base64"),
  })
  await expect(page.getByRole("button", { name: "保存照片" })).toBeEnabled()
  const pixel = await page.getByLabel("头像裁剪预览").evaluate((canvas: HTMLCanvasElement) =>
    Array.from(canvas.getContext("2d")!.getImageData(128, 128, 1, 1).data),
  )
  expect(pixel[0]).toBeGreaterThan(200)
  expect(pixel[3]).toBe(255)
  await page.getByRole("button", { name: "保存照片" }).click()
  await expect.poll(() => uploaded).toBe(true)
})
