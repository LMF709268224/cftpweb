import { expect, test } from "@playwright/test"
import { installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

const baseMessage = {
  message_id: "message-feedback",
  msg_type: 1,
  status: 0,
  title: "反馈测试消息",
  content: "反馈测试内容",
  created_at: "2026-08-11T08:00:00Z",
}

test.beforeEach(async ({ page }) => {
  await seedAuthenticatedCandidate(page)
})

test("单条消息操作失败后可以在当前行重试", async ({ page }) => {
  let readAttempts = 0

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/messages" && method === "GET") {
      return { data: { messages: [baseMessage], has_more: false } }
    }
    if (pathname === "/api/messages/read" && method === "PUT") {
      readAttempts += 1
      if (readAttempts === 1) return { status: 503, message: "temporarily unavailable" }
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  const messageRow = page.locator(".message-row").filter({ hasText: baseMessage.title })
  await messageRow.getByRole("button", { name: "更多操作" }).click()
  await messageRow.getByRole("button", { name: "标记为已读" }).click()

  await expect(messageRow.getByRole("alert")).toContainText("标记已读失败")
  await messageRow.getByRole("button", { name: "重试" }).click()
  await expect(messageRow.getByText("未读", { exact: true })).toHaveCount(0)
  await expect(messageRow.getByRole("alert")).toHaveCount(0)
  expect(readAttempts).toBe(2)
})

test("消息详情加载失败后可以在弹层内重试", async ({ page }) => {
  let detailAttempts = 0
  const readMessage = { ...baseMessage, status: 1 }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/messages" && method === "GET") {
      return { data: { messages: [readMessage], has_more: false } }
    }
    if (pathname === `/api/messages/${baseMessage.message_id}`) {
      detailAttempts += 1
      if (detailAttempts === 1) return { status: 503, message: "temporarily unavailable" }
      return { data: { ...readMessage, content: "重试后加载的消息详情" } }
    }
    return undefined
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  await page.getByText(baseMessage.title).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog.getByRole("heading", { name: "消息详情加载失败" })).toBeVisible()
  await dialog.getByRole("button", { name: "重新加载" }).click()
  await expect(dialog.getByText("重试后加载的消息详情")).toBeVisible()
  expect(detailAttempts).toBe(2)
})

test("较早的筛选请求不会覆盖最后一次筛选结果", async ({ page }) => {
  let releaseUnreadRequest!: () => void
  let unreadRequestStarted = false
  let unreadRequestReturned = false
  const unreadRequestGate = new Promise<void>((resolve) => {
    releaseUnreadRequest = resolve
  })

  await installCandidateApiMocks(page, async ({ pathname, method, url }) => {
    if (pathname !== "/api/messages" || method !== "GET") return undefined

    const status = url.searchParams.get("status")
    if (status === "unread") {
      unreadRequestStarted = true
      await unreadRequestGate
      unreadRequestReturned = true
      return { data: { messages: [{ ...baseMessage, title: "较早的未读结果" }], has_more: false } }
    }
    if (status === "read") {
      return { data: { messages: [{ ...baseMessage, status: 1, title: "最后的已读结果" }], has_more: false } }
    }
    return { data: { messages: [baseMessage], has_more: false } }
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  const statusTabs = page.locator(".messages-tabs")
  await statusTabs.getByRole("button", { name: /^未读/ }).click()
  await expect.poll(() => unreadRequestStarted).toBe(true)
  await statusTabs.getByRole("button", { name: "已读", exact: true }).click()
  await expect(page.getByText("最后的已读结果")).toBeVisible()

  const staleResponse = page.waitForResponse((response) => {
    const responseUrl = new URL(response.url())
    return responseUrl.pathname === "/api/messages" && responseUrl.searchParams.get("status") === "unread"
  })
  releaseUnreadRequest()
  await staleResponse
  expect(unreadRequestReturned).toBe(true)
  await expect(page.getByText("最后的已读结果")).toBeVisible()
  await expect(page.getByText("较早的未读结果")).toHaveCount(0)
})

test("消息正文支持语言分段和安全站内链接", async ({ page }) => {
  const localizedMessage = {
    ...baseMessage,
    status: 1,
    title: "<!--lang:en-US-->Localized message<!--lang:zh-CN-->本地化消息",
    content: [
      "<!--lang:en-US-->",
      "### Dear Candidate",
      "English message body.",
      "[View My Certifications]({{CandidatePortalBaseURL}}/my-certifications)",
      "[Direct internal link](/my-certifications)",
      "[External help](https://example.com/help)",
      "[Unsafe link](javascript:alert(1))",
      "",
      "<!--lang:zh-CN-->",
      "### 尊敬的考生",
      "中文消息正文。",
      "[查看我的专业认证]({{CandidatePortalBaseURL}}/my-certifications)",
    ].join("\n"),
  }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/messages" && method === "GET") {
      return { data: { messages: [localizedMessage], has_more: false } }
    }
    if (pathname === `/api/messages/${baseMessage.message_id}` && method === "GET") {
      return { data: localizedMessage }
    }
    return undefined
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  await page.getByText("本地化消息", { exact: true }).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog.getByRole("heading", { name: "尊敬的考生" })).toBeVisible()
  await expect(dialog.getByText("中文消息正文。")).toBeVisible()
  await expect(dialog.getByText("English message body.")).toHaveCount(0)
  await expect(dialog.getByRole("link", { name: "查看我的专业认证" })).toHaveAttribute("href", "/my-certifications")

  await page.evaluate(async () => {
    const modulePath = "/src/lib/language.ts"
    const { changeLanguage } = await import(modulePath)
    changeLanguage("en")
  })

  await expect(dialog.getByRole("heading", { name: "Dear Candidate" })).toBeVisible()
  await expect(dialog.getByText("English message body.")).toBeVisible()
  await expect(dialog.getByText("中文消息正文。")).toHaveCount(0)

  const markedInternalLink = dialog.getByRole("link", { name: "View My Certifications" })
  await expect(markedInternalLink).toHaveAttribute("href", "/my-certifications")
  await expect(dialog.getByRole("link", { name: "Direct internal link" })).toHaveAttribute("href", "/my-certifications")
  await expect(dialog.getByRole("link", { name: "External help" })).toHaveAttribute("target", "_blank")
  await expect(dialog.getByRole("link", { name: "Unsafe link" })).toHaveCount(0)

  await markedInternalLink.click()
  await expect(page).toHaveURL(/\/my-certifications$/)
})

test("没有语言标记的旧消息正文保持完整显示", async ({ page }) => {
  const legacyMessage = {
    ...baseMessage,
    status: 1,
    content: "Dear Candidate / 尊敬的考生\n\nLegacy body / 旧消息正文",
  }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/messages" && method === "GET") {
      return { data: { messages: [legacyMessage], has_more: false } }
    }
    if (pathname === `/api/messages/${baseMessage.message_id}` && method === "GET") {
      return { data: legacyMessage }
    }
    return undefined
  })

  await page.goto("/messages", { waitUntil: "domcontentloaded" })
  await page.getByText(baseMessage.title).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog.getByText("Dear Candidate / 尊敬的考生")).toBeVisible()
  await expect(dialog.getByText("Legacy body / 旧消息正文")).toBeVisible()
})
