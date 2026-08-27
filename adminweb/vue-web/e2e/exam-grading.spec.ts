import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

test("essay grading submits scores without accepting a web-supplied grader identity", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let submittedBody: Record<string, unknown> | undefined
  let submitted = false

  page.on("request", (request) => {
    if (request.method() === "POST" && new URL(request.url()).pathname === "/api/exams/exam-essay-1/essay-grade") {
      submittedBody = request.postDataJSON()
    }
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
    if (method === "POST" && pathname === "/api/exams/exam-essay-1/essay-grade") {
      submitted = true
      return { data: { success: true, final_total_score: 90 } }
    }
    return undefined
  })

  await page.goto("/exam-grading")
  await page.getByText("Ada Lovelace", { exact: true }).click()
  const dialog = page.getByRole("dialog", { name: "主观题阅卷详情" })
  await expect(dialog.getByText("A complete regression answer.", { exact: true })).toBeVisible()
  await dialog.getByLabel("本题得分").fill("18")
  await dialog.getByLabel("单题评语").fill("Clear and complete")
  await dialog.getByLabel("通过", { exact: true }).check()
  await dialog.getByLabel(/我已复核全部作答/).check()
  await dialog.getByRole("button", { name: "提交评分" }).click()

  await expect(page.getByText("主观题评分已提交", { exact: true })).toBeVisible()
  expect(submittedBody).toEqual({
    is_passed: true,
    overall_comment: "",
    items: [{ question_seq: 1, score: 18, comment: "Clear and complete" }],
  })
  expect(submittedBody).not.toHaveProperty("grader_id")
  expect(submittedBody).not.toHaveProperty("grader_name")
})
