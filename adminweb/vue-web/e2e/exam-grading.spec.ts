import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("essay grading exports a workbook and imports professor grades after preview", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const mutations: string[] = []
  let submitted = false

  page.on("request", (request) => {
    if (request.method() === "POST") mutations.push(new URL(request.url()).pathname)
  })
  await installAdminApiMocks(page, ({ method, pathname }) => {
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
