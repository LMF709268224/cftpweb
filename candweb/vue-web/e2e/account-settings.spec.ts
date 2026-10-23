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

test("profile saves Casdoor links without changing social login identities", async ({ page }) => {
  const profileRequests: Record<string, unknown>[] = []
  const user = {
    ...candidateUser,
    language: "en",
    gender: "male",
    birthday: "1990-01-01",
    address_text: "1 Main Street",
    postal_code: "123456",
    homepage: "https://old.example.com",
    linkedin: "https://www.linkedin.com/in/old",
    twitter: "https://x.com/old",
  }
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/user/me") return { data: user }
    if (pathname === "/api/user/profile" && method === "PUT") {
      const values = body as Record<string, unknown>
      profileRequests.push(values)
      Object.assign(user, values)
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/settings")
  await expect(page.getByLabel("Website")).toHaveValue("https://old.example.com")
  await expect(page.getByLabel("LinkedIn")).toHaveValue("https://www.linkedin.com/in/old")
  await expect(page.getByLabel("X", { exact: true })).toHaveValue("https://x.com/old")
  await page.getByLabel("Website").fill("https://example.com")
  await page.getByLabel("LinkedIn").fill("https://not-linkedin.example/in/user")
  await page.locator("form.settings-form").first().getByRole("button", { name: "Save Changes" }).click()
  expect(profileRequests).toHaveLength(0)

  await page.getByLabel("LinkedIn").fill("https://www.linkedin.com/in/new")
  await page.getByLabel("X", { exact: true }).fill("https://x.com/new")
  const invalidFields = await page.locator("form.settings-form").first().evaluate((form) =>
    Array.from(form.querySelectorAll<HTMLInputElement | HTMLSelectElement>("input, select"))
      .filter((input) => !input.checkValidity())
      .map((input) => `${input.getAttribute("placeholder") || input.getAttribute("name") || input.tagName}: ${input.validationMessage}`),
  )
  expect(invalidFields).toEqual([])
  await page.locator("form.settings-form").first().getByRole("button", { name: "Save Changes" }).click()
  await expect.poll(() => profileRequests.length).toBe(1)
  expect(profileRequests[0]).toMatchObject({ homepage: "https://example.com", linkedin: "https://www.linkedin.com/in/new", twitter: "https://x.com/new" })

  await page.reload()
  await expect(page.getByLabel("Website")).toHaveValue("https://example.com")
  await expect(page.getByLabel("LinkedIn")).toHaveValue("https://www.linkedin.com/in/new")
  await expect(page.getByLabel("X", { exact: true })).toHaveValue("https://x.com/new")
})

test("profile rejects future and underage birthdates before saving", async ({ page }) => {
  let profileUpdates = 0
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/user/me") return { data: { ...candidateUser, language: "en", province: "Singapore" } }
    if (pathname === "/api/user/profile" && method === "PUT") {
      profileUpdates += 1
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/settings")
  const birthday = page.locator(".localized-date-picker input").first()
  const save = page.locator("form.settings-form").first().getByRole("button", { name: "Save Changes" })
  const today = new Date()
  const dates = [
    { date: new Date(Date.UTC(today.getUTCFullYear() - 17, today.getUTCMonth(), today.getUTCDate())), message: "You must be at least 18 years old" },
    { date: new Date(Date.UTC(today.getUTCFullYear(), today.getUTCMonth(), today.getUTCDate() + 1)), message: "Date of birth cannot be in the future" },
  ]
  for (const { date, message } of dates) {
    const inputDate = `${String(date.getUTCDate()).padStart(2, "0")}/${String(date.getUTCMonth() + 1).padStart(2, "0")}/${date.getUTCFullYear()}`
    await birthday.fill(inputDate)
    await birthday.press("Enter")
    await save.click()
    await expect(page.getByText(message, { exact: true })).toBeVisible()
    expect(profileUpdates).toBe(0)
  }
})

test("email change verifies the current address first and allows retrying a wrong new code", async ({ page }) => {
  const emailRequests: { path: string, body: unknown }[] = []
  let oldAttempts = 0
  let newAttempts = 0
  let loginStarts = 0
  await page.clock.install()
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/auth/login-url") loginStarts += 1
    if (pathname === "/api/user/email/send-code" && method === "POST") return { data: { email: candidateUser.email } }
    if (pathname === "/api/user/email/verify-current" && method === "POST") {
      emailRequests.push({ path: pathname, body })
      oldAttempts += 1
      return oldAttempts === 1
        ? { status: 400, errorCode: "INVALID_REQUEST", message: "current email verification code is incorrect or expired" }
        : { data: { proof: "signed-old-email-proof", expires_in: 600 } }
    }
    if (pathname === "/api/user/email/send-new-code" && method === "POST") {
      emailRequests.push({ path: pathname, body })
      return { data: { email: "new@example.test" } }
    }
    if (pathname === "/api/user/email/confirm" && method === "POST") {
      emailRequests.push({ path: pathname, body })
      newAttempts += 1
      return newAttempts === 1
        ? { status: 400, errorCode: "INVALID_REQUEST", message: "请在 10 分钟内输入正确验证码" }
        : { data: { status: "updated" } }
    }
    return undefined
  })
  await page.goto("/settings?tab=account")
  await expect(page.getByText(candidateUser.email)).toBeVisible()
  await page.getByLabel("新邮箱").fill("new@example.test")
  await expect(page.getByLabel("新邮箱验证码")).toHaveCount(0)
  await page.getByRole("button", { name: "验证当前邮箱" }).click()
  await expect(page.getByText("请先获取并输入当前邮箱收到的 6 位验证码")).toBeVisible()
  await page.getByRole("button", { name: "发送验证码" }).click()
  await page.getByLabel("当前邮箱验证码").fill("111111")
  await page.getByRole("button", { name: "验证当前邮箱" }).click()
  await expect(page.getByText("当前邮箱验证码错误或已过期，请核对后重试")).toBeVisible()
  await page.getByLabel("当前邮箱验证码").fill("123456")
  await page.getByRole("button", { name: "验证当前邮箱" }).click()
  await expect(page.getByText("当前邮箱已验证")).toBeVisible()
  await expect(page.getByLabel("当前邮箱验证码")).toHaveCount(0)
  await expect(page.getByLabel("新邮箱验证码")).toBeVisible()
  await expect(page.getByRole("button", { name: "发送到新邮箱" })).toBeDisabled()
  await page.clock.fastForward(61_000)
  await page.getByRole("button", { name: "发送到新邮箱" }).click()
  await expect.poll(() => emailRequests).toContainEqual({
    path: "/api/user/email/send-new-code", body: { new_email: "new@example.test", proof: "signed-old-email-proof" },
  })
  await page.getByLabel("新邮箱验证码").fill("111111")
  await page.getByRole("button", { name: "确认修改邮箱" }).click()
  await expect(page.getByText("新邮箱验证码错误或已过期，请核对后重试")).toBeVisible()
  await page.getByLabel("新邮箱验证码").fill("654321")
  await page.getByRole("button", { name: "确认修改邮箱" }).click()
  await expect.poll(() => emailRequests.filter((request) => request.path === "/api/user/email/confirm").length).toBe(2)
  await expect(page).toHaveURL(/\/$/)
  await expect.poll(() => page.evaluate(() => localStorage.getItem("is_authenticated"))).toBeNull()
  expect(loginStarts).toBe(0)
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
