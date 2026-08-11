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
