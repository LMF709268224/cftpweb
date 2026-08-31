import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const gradingFilterOptions = {
  data: {
    options: [
      { program_code: "CFTP", exam_code: "ESSAY-1", exam_form: "Form A" },
      { program_code: "CFTP", exam_code: "ESSAY-2", exam_form: "Form B" },
      { program_code: "CFTE", exam_code: "ESSAY-1", exam_form: "Online" },
    ],
  },
}

test("essay grading exports a workbook and imports professor grades after preview", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const mutations: string[] = []
  let submitted = false

  page.on("request", (request) => {
    if (request.method() === "POST") mutations.push(new URL(request.url()).pathname)
  })
  await installAdminApiMocks(page, ({ method, pathname }) => {
    if (method === "GET" && pathname === "/api/exams/grading-filter-options") {
      return gradingFilterOptions
    }
    if (method === "GET" && pathname === "/api/exams/pending-grading") {
      return {
        data: {
          items: submitted ? [] : [{
            exam_ulid: "exam-essay-1",
            candidate_first_name: "Ada",
            candidate_last_name: "Lovelace",
            candidate_email: "ada@example.test",
            program_code: "CFTP",
            exam_code: "ESSAY-1",
            objective_score: 72,
            essay_count: 1,
          }],
          total: submitted ? 0 : 1,
          has_more: false,
        },
      }
    }
    if (method === "GET" && pathname === "/api/exams/exam-essay-1/essay-details") {
      return {
        data: {
          exam_ulid: "exam-essay-1",
          objective_score: 72,
          essays: [{
            question_seq: 1,
            question_name: "Risk analysis",
            section_name: "Essay",
            candidate_response: "A complete regression answer.",
            max_score: 20,
          }],
        },
      }
    }
    if (method === "POST" && pathname === "/api/exams/exam-essay-1/essay-grade/import/preview") {
      return {
        data: {
          exam_ulid: "exam-essay-1",
          grader_name: "Professor Lee",
          grader_id: "01ARZ3NDEKTSV4RRFFQ69G5FAV",
          is_passed: true,
          objective_score: 72,
          essay_score: 18,
          final_score: 90,
          overall_comment: "Meets the standard.",
          items: [{ question_seq: 1, question_name: "Risk analysis", max_score: 20, score: 18, comment: "Clear analysis" }],
        },
      }
    }
    if (method === "POST" && pathname === "/api/exams/exam-essay-1/essay-grade/import") {
      submitted = true
      return { data: { success: true, final_total_score: 90 } }
    }
    return undefined
  })

  await page.goto("/exam-grading")
  await page.getByText("Ada Lovelace", { exact: true }).click()
  const dialog = page.getByRole("dialog", { name: "主观题阅卷详情" })
  await expect(dialog.getByText("A complete regression answer.", { exact: true })).toBeVisible()
  await expect(dialog.getByRole("link", { name: "导出阅卷表" })).toHaveAttribute("href", "/api/exams/exam-essay-1/essay-grade/export")

  await dialog.getByLabel("选择评分文件").setInputFiles({
    name: "essay-grading-exam-essay-1.xlsx",
    mimeType: "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
    buffer: Buffer.from("workbook"),
  })
  await expect(dialog.getByText("Professor Lee", { exact: true })).toBeVisible()
  await expect(dialog.getByText("18 / 20", { exact: true })).toBeVisible()
  await dialog.getByLabel(/我已核对批改教授/).check()
  await dialog.getByRole("button", { name: "确认导入并提交" }).click()

  await expect(page.getByText("教授评分已导入并提交", { exact: true })).toBeVisible()
  expect(mutations).toEqual([
    "/api/exams/exam-essay-1/essay-grade/import/preview",
    "/api/exams/exam-essay-1/essay-grade/import",
  ])
})

test("essay grading history filters and displays the submitted professor result read-only", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let historyQuery = ""
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    if (method === "GET" && pathname === "/api/exams/grading-filter-options") {
      return gradingFilterOptions
    }
    if (method === "GET" && pathname === "/api/exams/pending-grading") {
      return { data: { items: [], total: 0, has_more: false } }
    }
    if (method === "GET" && pathname === "/api/exams/graded-essay") {
      historyQuery = url.search
      return {
        data: {
          items: [{
            exam_ulid: "exam-graded-1",
            candidate_first_name: "Grace",
            candidate_last_name: "Hopper",
            candidate_email: "grace@example.test",
            program_code: "CFTP",
            exam_code: "ESSAY-2",
            grader_id: "professor-1",
            grader_name: "Professor Lee",
            graded_at: "2026-08-28T02:00:00Z",
            final_score: 88,
            is_passed: true,
            essay_count: 1,
          }],
          has_more: false,
        },
      }
    }
    if (method === "GET" && pathname === "/api/exams/exam-graded-1/essay-details") {
      return {
        data: {
          exam_ulid: "exam-graded-1",
          objective_score: 70,
          total_score: 88,
          is_passed: true,
          overall_comment: "Meets the certification standard.",
          essays: [{
            question_seq: 1,
            question_name: "Risk analysis",
            section_name: "Essay",
            candidate_response: "A complete historical response.",
            max_score: 20,
            score: 18,
            grader_name: "Professor Lee",
            grader_comment: "Clear analysis.",
            graded_at: "2026-08-28T02:00:00Z",
          }],
        },
      }
    }
    return undefined
  })

  await page.goto("/exam-grading")
  await page.getByLabel("考试科目（Program）").selectOption("CFTP")
  await expect(page.getByLabel("考试代码（Exam Code）").getByRole("option", { name: "ESSAY-2 (Form B)" })).toHaveCount(1)
  await page.getByLabel("考试代码（Exam Code）").selectOption("ESSAY-2")
  await page.getByRole("tab", { name: "历史记录" }).click()
  await expect(page.getByText("Grace Hopper", { exact: true })).toBeVisible()
  expect(historyQuery).toContain("program_code=CFTP")
  expect(historyQuery).toContain("exam_code=ESSAY-2")

  await page.getByPlaceholder("批改教授姓名").fill("Professor Lee")
  await page.getByLabel("最终判定").selectOption("true")
  await expect.poll(() => historyQuery).toContain("grader_name=Professor+Lee")
  expect(historyQuery).toContain("is_passed=true")

  await page.getByText("Grace Hopper", { exact: true }).click()
  const dialog = page.getByRole("dialog", { name: "主观题阅卷历史详情" })
  await expect(dialog.getByText("Meets the certification standard.", { exact: true })).toBeVisible()
  await expect(dialog.getByText("18 / 20", { exact: true })).toBeVisible()
  await expect(dialog.getByText("Clear analysis.", { exact: true })).toBeVisible()
  await expect(dialog.getByText("Professor Lee", { exact: true })).toBeVisible()
  await expect(dialog.getByRole("link", { name: "导出阅卷表" })).toHaveCount(0)
  await expect(dialog.getByLabel("选择评分文件")).toHaveCount(0)
})
