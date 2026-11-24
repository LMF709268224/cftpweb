import { expect, test, type Page } from "@playwright/test"
import { candidateUser, installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

const thumbnail = "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+jC1sAAAAASUVORK5CYII="
const pack = { pack_id: "catalog", title: "Insights", can_access: false }
const member = { ...candidateUser, account_status: { membership: { available: true, is_member: true } } }
declare global {
  interface Window {
    openedResources: string[]
  }
}

async function recordOpenedResources(page: Page) {
  await page.addInitScript(() => {
    window.openedResources = []
    window.open = (url) => {
      window.openedResources.push(String(url))
      return null
    }
  })
}

async function switchToEnglish(page: Page) {
  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en")
    window.dispatchEvent(new Event("lang_change"))
  })
}

test("无权限合集仍可浏览缩略图，支持分页、中英文和移动端", async ({ page }) => {
  const cursors: string[] = []
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/resource-packs") {
      cursors.push(url.searchParams.get("page_token") || "")
      return { data: url.searchParams.has("page_token")
        ? { packs: [{ pack_id: "allowed", title: "Other resources", can_access: true }], next_cursor: "" }
        : { packs: [pack], next_cursor: "catalog-page-2" } }
    }
    if (pathname === "/api/resource-packs/catalog/files") return { data: { files: [
      { file_id: "report", title: "Locked report", file_type: 2, can_access: false, thumbnail_url: thumbnail },
    ] } }
  })
  await page.goto("/resource-packs")
  const catalog = page.getByRole("link").filter({ hasText: "Insights" })
  await expect(catalog).toContainText("内容已锁定")
  await expect(catalog).toContainText("浏览资料")
  await page.getByRole("button", { name: "加载更多" }).click()
  await expect(page.getByRole("link").filter({ hasText: "Other resources" })).not.toContainText("内容已锁定")
  expect(cursors).toEqual(["", "catalog-page-2"])
  await catalog.click()
  await expect(page).toHaveURL(/\/resource-packs\/catalog$/)
  const card = page.getByTestId("resource-file-card")
  await expect(card.getByRole("button")).toBeDisabled()
  await expect(card.getByRole("img")).toBeVisible()
  await expect.poll(() => card.getByRole("img").evaluate((img: HTMLImageElement) => img.naturalWidth)).toBeGreaterThan(0)
  const notice = page.getByTestId("resource-access-notice")
  await expect(notice).toContainText("完整内容需要访问权限")
  await expect(notice.getByRole("link")).toHaveAttribute("href", "/membership")
  await switchToEnglish(page)
  await expect(card).toContainText("Content locked")
  await expect(notice).toContainText("Access required for full content")
  await expect(notice.getByRole("link")).toHaveText("View Membership Plans")
  await page.setViewportSize({ width: 390, height: 844 })
  await expect(notice).toBeVisible()
  await expect.poll(() => page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth)).toBe(true)
})

for (const fileType of [1, 2, 3]) {
  test(`资源类型 ${fileType} 仅明确授权才可打开，会员标签不会覆盖文件权限`, async ({ page }) => {
    const contentRequests: string[] = []
    const thumbnails: string[] = []
    const files = [false, undefined, "true", true].map((access, index) => ({
      file_id: `file-${index}`, title: `Resource ${index}`, file_type: fileType, can_access: access,
      ...(index === 0 ? { thumbnail_object_key: "locked-cover.png" } : { thumbnail_url: thumbnail }),
    }))
    await seedAuthenticatedCandidate(page)
    await recordOpenedResources(page)
    await installCandidateApiMocks(page, ({ pathname }) => {
      if (pathname === "/api/user/me") return { data: member }
      if (pathname === "/api/resource-packs/catalog/files") return { data: { files } }
      if (pathname.endsWith("/thumbnail-url")) {
        thumbnails.push(pathname)
        return { data: { url: thumbnail } }
      }
      if (pathname.endsWith("/view-url") || pathname.endsWith("/preview-url")) {
        contentRequests.push(pathname)
        return { data: { view_url: "https://files.example.test/authorized.zip" } }
      }
    })
    await page.goto("/resource-packs/catalog")
    await expect(page.getByTestId("resource-file-card")).toHaveCount(4)
    for (let index = 0; index < 3; index++) {
      const card = page.locator(`[data-file-id="file-${index}"]`)
      await expect(card).toContainText("内容已锁定")
      for (const button of await card.getByRole("button").all()) {
        await expect(button).toBeDisabled()
        // Attempting a DOM click must not bypass the disabled card.
        await button.evaluate((element: HTMLButtonElement) => element.click())
      }
      await expect(card.getByRole("img")).toBeVisible()
    }
    expect(thumbnails).toEqual(["/api/resource-pack-files/file-0/thumbnail-url"])
    expect(contentRequests).toEqual([])
    expect(await page.evaluate(() => window.openedResources)).toEqual([])
    const authorized = page.locator('[data-file-id="file-3"]')
    await expect(authorized).not.toContainText("内容已锁定")
    await authorized.getByRole("button").first().click()
    const expectedUrl = fileType === 1 ? "/video-preview/resource-pack-files/file-3"
      : fileType === 2 ? "/resource-pack-files/file-3/preview" : "https://files.example.test/authorized.zip"
    await expect.poll(() => page.evaluate(() => window.openedResources)).toEqual([expectedUrl])
    expect(contentRequests).toEqual(fileType === 3 ? ["/api/resource-pack-files/file-3/view-url"] : [])
  })
}

for (const preview of [
  { route: "/resource-pack-files/report/preview", endpoint: "/api/resource-pack-files/report/preview-url" },
  { route: "/video-preview/resource-pack-files/video", endpoint: "/api/resource-pack-files/video/view-url" },
]) {
  test(`直接打开 ${preview.route} 时服务端拒绝访问，显示双语权限提示`, async ({ page }) => {
    let contentRequests = 0
    await seedAuthenticatedCandidate(page)
    await installCandidateApiMocks(page, ({ pathname }) => {
      if (pathname === preview.endpoint) {
        contentRequests++
        return { status: 403, message: "permission denied" }
      }
    })
    await page.goto(preview.route)
    const notice = page.getByTestId("resource-access-notice")
    await expect(notice).toContainText("完整内容需要访问权限")
    await expect(page.locator("iframe")).toHaveCount(0)
    expect(contentRequests).toBe(1)
    await switchToEnglish(page)
    await expect(notice).toContainText("Access required for full content")
    await notice.getByRole("link").click()
    await expect(page).toHaveURL(/\/membership$/)
  })
}

test("文件访问权限过期时保持 403 并锁定卡片", async ({ page }) => {
  await seedAuthenticatedCandidate(page)
  await recordOpenedResources(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/resource-packs/catalog/files") return { data: { files: [
      { file_id: "expired", title: "Expired access", file_type: 3, can_access: true, thumbnail_url: thumbnail },
    ] } }
    if (pathname === "/api/resource-pack-files/expired/view-url") return { status: 403, message: "permission denied" }
  })
  await page.goto("/resource-packs/catalog")
  const card = page.getByTestId("resource-file-card")
  await card.getByRole("button").first().click()
  await expect(card).toContainText("内容已锁定")
  await expect(card.getByRole("button").first()).toBeDisabled()
  await expect(page.getByTestId("resource-access-notice")).toBeVisible()
  expect(await page.evaluate(() => window.openedResources)).toEqual([])
})

test("首页锁定的 Insights 进入合集浏览，已授权资料保留预览入口", async ({ page }) => {
  const cursors: string[] = []
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname, url }) => {
    if (pathname === "/api/user/me") return { data: member }
    if (pathname === "/api/pipeline") return { data: { list: [] } }
    if (pathname === "/api/mall/bundles") return { data: { bundles: [] } }
    if (pathname === "/api/resource-packs") {
      cursors.push(url.searchParams.get("page_token") || "")
      return { data: url.searchParams.has("page_token") ? { packs: [pack] }
        : { packs: [{ pack_id: "webinars", title: "Webinars", can_access: false }], next_cursor: "insights-page" } }
    }
    if (pathname === "/api/resource-packs/catalog/files") return { data: { files: [
      { file_id: "locked", title: "Locked insight", file_type: 2, can_access: false },
      { file_id: "allowed", title: "Allowed insight", file_type: 2, can_access: true },
    ] } }
  })
  await page.goto("/dashboard")
  const locked = page.getByRole("link").filter({ hasText: "Locked insight" })
  await expect(locked).toHaveAttribute("href", "/resource-packs/catalog")
  await expect(locked).toContainText("内容已锁定")
  await expect(locked).not.toHaveAttribute("target", "_blank")
  const authorized = page.getByRole("link").filter({ hasText: "Allowed insight" })
  await expect(authorized).toHaveAttribute("href", "/resource-pack-files/allowed/preview")
  await expect(authorized).toHaveAttribute("target", "_blank")
  expect(cursors).toEqual(["", "insights-page"])
  await locked.click()
  await expect(page.getByTestId("resource-access-notice")).toBeVisible()
})
