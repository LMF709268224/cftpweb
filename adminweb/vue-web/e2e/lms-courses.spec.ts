import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const course = {
  course_ulid: "course-1",
  title: "Regression Course",
  description: "Read-only LMS course",
  category_tips: "Automation",
  course_gpath: "/courses/regression-course",
  duration_min: 90,
  status: "Active",
  is_published: true,
  version: 4,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installLmsCourseReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/lms/courses") {
      return { data: { courses: [course], has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/lms/courses/course-1/detail") {
      return { data: { course_detail: { course, chapter_count: 2, lesson_count: 3, quiz_count: 1, material_count: 1 } } }
    }
    if (method === "GET" && pathname === "/api/lms/courses/course-1/complete") {
      return {
        data: {
          complete_course: {
            course,
            materials: [{ material_ulid: "material-1", title: "Regression Material" }],
            chapters: [
              {
                chapter: { chapter_ulid: "chapter-1", title: "Regression Chapter", sort_order: 1 },
                lessons: [{ lesson: { lesson_ulid: "lesson-1", title: "Regression Lesson", sort_order: 1, lesson_type: 2, body: "Lesson body", meta_json: "{}" } }],
                quizzes: [{
                  quiz: { quiz_ulid: "quiz-1", title: "Regression Quiz", description: "Chapter review", passing_score: 70, time_limit: 30, randomize_questions: true, quiz_type: 1 },
                  questions: [{
                    question: { question_ulid: "question-1", question_text: "Regression question", question_type: 1, points: 10, sort_order: 1, is_required: true, explanation: "Regression explanation", media_items_json: "[]" },
                    options: [{ option_ulid: "option-1", option_text: "Correct", is_correct: true, sort_order: 1 }],
                  }],
                }],
              },
              { chapter: { chapter_ulid: "chapter-2", title: "Review Chapter", sort_order: 2 }, lessons: [{ lesson: { lesson_ulid: "lesson-2", title: "Review Lesson", sort_order: 1, lesson_type: 2, body: "Review body" } }, { lesson: { lesson_ulid: "lesson-3", title: "Final Lesson", sort_order: 2, lesson_type: 2, body: "Final body" } }] },
            ],
            quizzes: [],
          },
        },
      }
    }
    return undefined
  })
}

test("LMS course list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installLmsCourseReadMocks(page, requests)

  await page.goto("/lms")

  await expect(page.getByText("Regression Course", { exact: true })).toBeVisible()
  await expect(page.getByText("Automation", { exact: true })).toBeVisible()
  const courseRow = page.getByRole("button", { name: "查看详情" }).first().locator("../..")
  await expect(courseRow.getByText("已发布", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/lms/courses")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("LMS course detail reads counts and complete tree without editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installLmsCourseReadMocks(page, requests)
  await page.goto("/lms")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  const dialog = page.getByRole("dialog")
  await expect(dialog.getByRole("heading", { name: "课程顶层数据" })).toBeVisible()
  await expect(dialog.getByText("course-1", { exact: true }).first()).toBeVisible()
  await expect(dialog.getByText("Regression Course", { exact: true })).toBeVisible()
  await expect(dialog.getByText("3", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/lms/courses/course-1/detail")
  expect(requests).toContain("GET /api/lms/courses/course-1/complete")
  expect(requests.some((request) => request.includes("/publish") || request.includes("/import"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("course detail exposes import-ready JSON with a GPath warning", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installLmsCourseReadMocks(page, requests)
  await page.goto("/lms")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  const dialog = page.getByRole("dialog")
  await dialog.getByRole("tab", { name: "查看 JSON" }).click()

  await expect(dialog.getByText("复制为新课程前必须修改 course_gpath", { exact: true })).toBeVisible()
  await expect(dialog.getByText("course_gpath: /courses/regression-course", { exact: true })).toBeVisible()
  await expect(dialog.getByText(/当前导入入口支持课程、章节、课时和章节测验/)).toBeVisible()
  const jsonText = await dialog.locator("pre").textContent()
  const exported = JSON.parse(jsonText || "{}")

  expect(exported.course_gpath).toBe("/courses/regression-course")
  expect(exported.chapters).toHaveLength(2)
  expect(exported.chapters[0].lessons[0]).toMatchObject({ title: "Regression Lesson", lesson_type: 2, body: "Lesson body" })
  expect(exported.quizzes[0]).toMatchObject({ chapter_title: "Regression Chapter", title: "Regression Quiz" })
  expect(exported.quizzes[0].questions[0].options[0]).toEqual({ option_text: "Correct", is_correct: true, sort_order: 1 })
  expect(JSON.stringify(exported)).not.toContain("course_ulid")
  expect(JSON.stringify(exported)).not.toContain("chapter_ulid")
  expect(JSON.stringify(exported)).not.toContain("lesson_ulid")
  expect(JSON.stringify(exported)).not.toContain("quiz_ulid")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)

  await page.setViewportSize({ width: 390, height: 844 })
  await expect(dialog.getByRole("tab", { name: "详情" })).toBeVisible()
  await expect(dialog.getByRole("tab", { name: "查看 JSON" })).toBeVisible()
  await expect(dialog.getByRole("button", { name: "关闭" })).toBeVisible()
  await expect(dialog.getByRole("button", { name: "复制 JSON" })).toBeVisible()
})

test("course import stops without retry and keeps the failed draft ID", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let chapterImportCalls = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/lms/courses") {
      return { data: { courses: [], has_more: false, next_cursor: "" } }
    }
    if (method === "POST" && pathname === "/api/lms/courses") {
      return { data: { course_ulid: "draft-course-1" } }
    }
    if (method === "POST" && pathname === "/api/lms/courses/draft-course-1/chapters/import") {
      chapterImportCalls += 1
      if (chapterImportCalls === 1) return { data: { chapter_ulid: "chapter-1", lesson_count: 1 } }
      return { status: 504, errorCode: "SERVICE_UNAVAILABLE", message: "chapter import timed out" }
    }
    return undefined
  })

  await page.goto("/lms")
  await page.getByRole("button", { name: "导入 JSON" }).click()
  await page.getByPlaceholder("也可以直接粘贴 JSON").fill(JSON.stringify({
    title: "Chunked Course",
    course_gpath: "/courses/chunked-course",
    chapters: [
      { title: "Chapter 1", lessons: [{ title: "Lesson 1", lesson_type: "video" }] },
      { title: "Chapter 2", lessons: [{ title: "Lesson 2", lesson_type: "video" }] },
    ],
    quizzes: [],
  }))
  await page.getByRole("button", { name: "开始导入" }).click()

  const dialog = page.getByRole("dialog")
  await expect(dialog.getByText("导入已停止", { exact: true })).toBeVisible()
  await expect(dialog.getByText("已保留未完成的课程草稿。请关闭窗口后删除该草稿，再重新导入。", { exact: true })).toBeVisible()
  await expect(dialog.getByText("草稿课程 ID: draft-course-1", { exact: true })).toBeVisible()
  await expect(dialog.getByRole("button", { name: "开始导入" })).toBeDisabled()

  expect(chapterImportCalls).toBe(2)
  expect(requests.filter((request) => request === "POST /api/lms/courses")).toHaveLength(1)
  expect(requests).not.toContain("GET /api/lms/courses/draft-course-1/complete")
})
