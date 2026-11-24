import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

async function setup(page: Page, status: string | undefined = "Active", reorderFails = false) {
  const course = { course_ulid: "course-1", title: "Authoring Course", status, version: 4, course_gpath: "/course/authoring", duration_min: 60 }
  const chapter = { chapter_ulid: "chapter-1", course_ulid: "course-1", title: "Chapter Alpha", sort_order: 1, version: 3 }
  const lessons = [
    { lesson_ulid: "lesson-1", chapter_ulid: "chapter-1", title: "Lesson Alpha", sort_order: 1, version: 7, lesson_type: 2, body: "<p>Existing <strong>body</strong></p>" },
    { lesson_ulid: "lesson-2", chapter_ulid: "chapter-1", title: "Lesson Beta", sort_order: 2, version: 2, lesson_type: 2, body: "<p>Second body</p>" },
  ]
  const writes: { method: string; path: string; body: Record<string, any> }[] = []
  page.on("request", (request) => {
    if (!["POST", "PATCH", "PUT", "DELETE"].includes(request.method())) return
    const path = new URL(request.url()).pathname
    const body = request.postDataJSON()
    writes.push({ method: request.method(), path, body })
    if (path === "/api/lms/lessons/lesson-1/content") Object.assign(lessons[0]!, { title: body.title, ...("body" in body ? { body: body.body } : {}), version: 8 })
    if (path.endsWith("/reorder") && !reorderFails) {
      for (const item of body.items) {
        const lesson = lessons.find((row) => row.lesson_ulid === item.entity_ulid)
        if (lesson) Object.assign(lesson, { sort_order: item.sort_order, version: lesson.version + 1 })
      }
    }
  })
  await installAdminApiMocks(page, ({ pathname, method }) => {
    if (method === "PATCH" && pathname.endsWith("/content")) return { data: { success: true } }
    if (method === "POST" && pathname.endsWith("/reorder")) return reorderFails ? { status: 409, message: "Version conflict", delayMs: 100 } : { data: { success: true } }
    if (pathname === "/api/lms/courses") return { data: { courses: [course], has_more: false, next_cursor: "" } }
    if (pathname === "/api/lms/courses/course-1/detail") return { data: { course_detail: { course } } }
    if (pathname === "/api/lms/courses/course-1/complete") return { data: { complete_course: { course, chapters: [{ chapter, lessons: lessons.map((lesson) => ({ lesson })), quizzes: [] }], quizzes: [], materials: [] } } }
    if (pathname === "/api/lms/courses/course-1/chapters") return { data: { chapters: [chapter] } }
    if (pathname === "/api/lms/chapters/chapter-1/lessons") return { data: { lessons } }
    return undefined
  })
  await page.goto("/lms")
  await page.getByRole("button", { name: /^(编辑|Edit)$/ }).first().click()
  await expect(page.getByTestId("course-outline").getByRole("button", { name: "Lesson Alpha", exact: true })).toBeVisible()
  return { writes, lessons }
}

for (const lang of ["en", "zh"] as const) {
  test(`published minor fixes preserve structure and submit only content (${lang})`, async ({ page }, testInfo) => {
    if (lang === "zh") await page.setViewportSize({ width: 390, height: 844 })
    await seedAuthenticatedAdmin(page, lang)
    const { writes } = await setup(page)
    await page.getByTestId("course-outline").getByRole("button", { name: "Lesson Alpha", exact: true }).click()
    const dialog = page.getByRole("dialog")
    await expect(dialog.locator("select").first()).toBeDisabled()
    await expect(dialog.locator('input[type="number"]')).toBeDisabled()
    await expect(dialog.locator("select").nth(1)).toBeDisabled()
    await expect(dialog.getByRole("button", { name: /^(上传课时文件|Upload Lesson File)$/ })).toHaveCount(0)
    const editor = dialog.getByTestId("course-body-editor").getByRole("textbox", { name: lang === "zh" ? "课时正文编辑器" : "Lesson body editor" })
    await editor.fill("Corrected content")
    await editor.press("ControlOrMeta+a")
    await dialog.getByRole("button", { name: lang === "zh" ? "加粗" : "Bold", exact: true }).click()
    const preview = dialog.getByRole("region", { name: lang === "zh" ? "正文预览" : "Body preview" })
    await expect(preview.locator("strong")).toHaveText("Corrected content")
    await page.screenshot({ path: testInfo.outputPath(`course-editor-${lang}.png`), fullPage: true })
    await dialog.getByRole("button", { name: lang === "zh" ? "保存课时" : "Save Lesson", exact: true }).click()
    await expect(dialog).toHaveCount(0)
    expect(writes).toHaveLength(1)
    expect(writes[0]).toMatchObject({ method: "PATCH", path: "/api/lms/lessons/lesson-1/content", body: { title: "Lesson Alpha", version: 7 } })
    expect(Object.keys(writes[0]!.body).sort()).toEqual(["body", "title", "version"])
    expect(writes[0]!.body.body).toContain("<strong>Corrected content</strong>")
  })
}

test("source preview removes executable markup and switching lessons protects unsaved work", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page)
  await page.getByTestId("course-outline").getByRole("button", { name: "Lesson Alpha", exact: true }).click()
  const dialog = page.getByRole("dialog")
  await dialog.getByRole("button", { name: "HTML source", exact: true }).click()
  await dialog.getByRole("textbox", { name: "Lesson HTML source" }).fill('<p>Safe text</p><img src="x" onerror="window.badPreview=true"><script>window.badPreview=true</script><iframe srcdoc="bad"></iframe>')
  const preview = dialog.getByRole("region", { name: "Body preview" })
  await expect(preview.getByText("Safe text")).toBeVisible()
  expect(await preview.locator("script, iframe, [onerror]").count()).toBe(0)
  expect(await page.evaluate(() => (window as any).badPreview)).toBeUndefined()
  page.once("dialog", (confirmation) => confirmation.dismiss())
  await dialog.getByTestId("course-outline").getByRole("button", { name: "Lesson Beta", exact: true }).click()
  await expect(dialog.getByRole("textbox", { name: "Lesson HTML source" })).toHaveValue(/Safe text/)
  page.once("dialog", (confirmation) => confirmation.accept())
  await dialog.getByTestId("course-outline").getByRole("button", { name: "Lesson Beta", exact: true }).click()
  await expect(dialog.getByRole("region", { name: "Body preview" })).toContainText("Second body")
  expect(writes).toHaveLength(0)
})

for (const status of ["Draft", "Active"]) {
  test(`missing lesson body is never submitted as empty during title corrections (${status})`, async ({ page }) => {
    await seedAuthenticatedAdmin(page, "en")
    const { writes, lessons } = await setup(page, status)
    Reflect.deleteProperty(lessons[0]!, "body")
    const outline = page.getByTestId("course-outline")
    await Promise.all([
      page.waitForResponse("**/api/lms/courses/course-1/complete"),
      outline.getByRole("button", { name: "Reload outline" }).click(),
    ])
    await outline.getByRole("button", { name: "Lesson Alpha", exact: true }).click()
    const dialog = page.getByRole("dialog")
    await expect(dialog.getByTestId("course-body-editor")).toHaveCount(0)
    await dialog.locator("input").first().fill("Corrected title")
    if (status === "Draft") {
      await dialog.locator('input[type="number"]').fill("3")
      await dialog.getByRole("button", { name: "Save Lesson", exact: true }).click()
      await expect(page.locator("[data-sonner-toast]").filter({ hasText: "Required course data is unavailable" })).toBeVisible()
      expect(writes).toHaveLength(0)
      await dialog.locator('input[type="number"]').fill("1")
    }
    await dialog.getByRole("button", { name: "Save Lesson", exact: true }).click()
    await expect(dialog).toHaveCount(0)
    expect(writes).toEqual([{ method: "PATCH", path: "/api/lms/lessons/lesson-1/content", body: { title: "Corrected title", version: 7 } }])
  })
}

test("draft outline reorder sends actual entity versions and reloads confirmed order", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page, "Draft")
  const outline = page.getByTestId("course-outline")
  const firstLesson = outline.getByRole("button", { name: "Lesson Alpha", exact: true }).locator("..");
  const secondLesson = outline.getByRole("button", { name: "Lesson Beta", exact: true }).locator("..");
  await firstLesson.locator("[draggable]").dragTo(secondLesson)
  await expect.poll(() => writes.length).toBe(1)
  expect(writes[0]).toMatchObject({ method: "POST", path: "/api/lms/chapters/chapter-1/lessons/reorder", body: { items: [{ entity_ulid: "lesson-2", sort_order: 1, version: 2 }, { entity_ulid: "lesson-1", sort_order: 2, version: 7 }] } })
  await expect(outline.getByRole("button", { name: /^Lesson / }).first()).toHaveText("Lesson Beta")
  await outline.getByRole("button", { name: "Move down", exact: true }).nth(1).click()
  await expect.poll(() => writes.length).toBe(2)
  expect(writes[1]!.body.items).toEqual([{ entity_ulid: "lesson-1", sort_order: 1, version: 8 }, { entity_ulid: "lesson-2", sort_order: 2, version: 3 }])
})

test("failed reorders do not claim success or change the returned order", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  await setup(page, "Draft", true)
  const outline = page.getByTestId("course-outline")
  await outline.getByRole("button", { name: "Move down", exact: true }).nth(1).click()
  await expect(outline.getByRole("alert")).toContainText("Could not save order")
  await expect(outline.getByRole("button", { name: /^Lesson / }).first()).toHaveText("Lesson Alpha")
  await expect(outline.getByRole("button", { name: "Move down", exact: true }).nth(1)).toBeDisabled()
  await expect(page.getByText("Order saved", { exact: true })).toHaveCount(0)
})

test("missing course status does not grant draft editing or publishing", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page, "")
  await expect(page.getByRole("button", { name: "Publish Course", exact: true })).toHaveCount(0)
  await expect(page.getByRole("button", { name: "New Lesson", exact: true })).toHaveCount(0)
  await expect(page.getByRole("button", { name: "Save Course", exact: true })).toBeDisabled()
  await page.getByTestId("course-outline").getByRole("button", { name: "Lesson Alpha", exact: true }).click()
  await expect(page.getByRole("dialog").getByRole("button", { name: "Save Lesson", exact: true })).toHaveCount(0)
  expect(writes).toHaveLength(0)
})

test("outline refresh does not silently replace the editing version after concurrent changes", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes, lessons } = await setup(page)
  await page.getByTestId("course-outline").getByRole("button", { name: "Lesson Alpha", exact: true }).click()
  const dialog = page.getByRole("dialog")
  await dialog.getByRole("textbox", { name: "Lesson body editor" }).fill("My correction")
  lessons[0]!.version = 8
  await dialog.getByRole("button", { name: "Reload outline", exact: true }).click()
  await page.route("**/api/lms/lessons/lesson-1/content", (route) => route.fulfill({ status: 409, json: { code: 409, error_code: "PRECONDITION_FAILED", message: "lesson version conflict; reload before saving" } }))
  await dialog.getByRole("button", { name: "Save Lesson", exact: true }).click()
  await expect.poll(() => writes.length).toBe(1)
  expect(writes[0]!.body.version).toBe(7)
  await expect(dialog.getByRole("textbox", { name: "Lesson body editor" })).toHaveText("My correction")
  await expect(dialog).toBeVisible()
})

test("an incomplete course tree is not treated as an empty editable outline", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page, "Draft")
  await page.route("**/api/lms/courses/course-1/complete", (route) => route.fulfill({ json: { code: 200, data: { complete_course: { course: { course_ulid: "course-1", status: "Draft" } } } } }))
  const outline = page.getByTestId("course-outline")
  await outline.getByRole("button", { name: "Reload outline" }).click()
  await expect(outline.getByRole("alert")).toContainText("complete course outline is unavailable")
  await expect(outline.getByText("No chapters", { exact: true })).toHaveCount(0)
  await expect(outline.locator("[draggable]")).toHaveCount(0)
  expect(writes).toHaveLength(0)
})

test("published chapter correction retains the original sort and version", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page)
  await page.route("**/api/lms/chapters/chapter-1", (route) => route.fulfill({ json: { code: 200, data: { success: true } } }))
  await page.getByTestId("course-outline").getByRole("button", { name: "Chapter Alpha", exact: true }).click()
  const dialog = page.getByRole("dialog")
  await expect(dialog.locator('input[type="number"]')).toBeDisabled()
  await dialog.locator("input").first().fill("Corrected chapter")
  await dialog.getByRole("button", { name: "Save Chapter", exact: true }).click()
  await expect(dialog).toHaveCount(0)
  expect(writes).toEqual([{ method: "PUT", path: "/api/lms/chapters/chapter-1", body: { course_id: "course-1", title: "Corrected chapter", sort_order: 1, version: 3 } }])
})

test("creating a draft version opens the actual returned course and does not migrate students", async ({ page }) => {
  await seedAuthenticatedAdmin(page, "en")
  const { writes } = await setup(page)
  const created = { course_ulid: "new-draft", title: "Authoring Course (Copy)", status: "Draft", course_gpath: "/course/authoring", version: 1 }
  await page.route("**/api/lms/courses", (route) => route.request().method() === "POST" ? route.fulfill({ json: { code: 200, data: created } }) : route.fallback())
  await page.route("**/api/lms/courses/new-draft/detail", (route) => route.fulfill({ json: { code: 200, data: { course_detail: { course: created } } } }))
  await page.route("**/api/lms/courses/new-draft/complete", (route) => route.fulfill({ json: { code: 200, data: { complete_course: { course: created, chapters: [], quizzes: [], materials: [] } } } }))
  await page.goto("/lms")
  await page.getByRole("button", { name: "Create draft version", exact: true }).click()
  await expect(page.getByText("new-draft", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "Publish Course", exact: true })).toBeVisible()
  expect(writes).toEqual([{ method: "POST", path: "/api/lms/courses", body: { from_course_id: "course-1", title: "Authoring Course (Copy)" } }])
})
