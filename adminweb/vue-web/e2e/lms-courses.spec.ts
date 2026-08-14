import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const course = {
  course_ulid: "course-1",
  title: "Regression Course",
  description: "Read-only LMS course",
  category_tips: "Automation",
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
              { chapter: { chapter_ulid: "chapter-1", title: "Regression Chapter" }, lessons: [{ lesson: { lesson_ulid: "lesson-1", title: "Regression Lesson" } }] },
              { chapter: { chapter_ulid: "chapter-2", title: "Review Chapter" }, lessons: [{ lesson: { lesson_ulid: "lesson-2", title: "Review Lesson" } }, { lesson: { lesson_ulid: "lesson-3", title: "Final Lesson" } }] },
            ],
            quizzes: [{ quiz: { quiz_ulid: "quiz-1", title: "Regression Quiz" } }],
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
