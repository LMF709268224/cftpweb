import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const examSummary = {
  exam_ulid: "exam-1",
  exam_code: "REG-101",
  program_code: "REG",
  exam_status: "DONE",
  result_status: "AVAILABLE",
  total_score: 88.5,
  is_passed: true,
  candidate_ulid: "candidate-1",
  candidate_first_name: "Regression",
  candidate_last_name: "Candidate",
  candidate_email: "candidate@example.test",
  confirmation_number: "CONF-001",
  appointment_start_time: "2026-08-11T00:00:00Z",
}

async function installExamReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/exams") {
      return { data: { exams: [examSummary], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/exams/exam-1") {
      return {
        data: {
          ...examSummary,
          pipeline_ulid: "pipeline-1",
          course_unit_ulid: "course-unit-1",
          certification_name: "Regression Certification",
          site_name: "Regression Test Centre",
        },
      }
    }
    if (method === "GET" && pathname === "/api/exams/exam-1/result") {
      return { data: { exam_ulid: "exam-1", total_score: 88.5, is_passed: true, score_details_json: '{"theory":88.5}' } }
    }
    if (method === "GET" && pathname === "/api/exams/exam-1/transitions") {
      return {
        data: {
          exam_ulid: "exam-1",
          transitions: [{
            msg_fp: "event-1",
            exam_ulid: "exam-1",
            event_type: "result_created",
            status_type: "RESULT",
            from_status: "NONE",
            to_status: "AVAILABLE",
            transitioned_at: "2026-08-11T01:00:00Z",
          }],
        },
      }
    }
    return undefined
  })
}

test("exam list renders candidate, appointment, and result data", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installExamReadMocks(page, requests)

  await page.goto("/exams")

  await expect(page.getByText("REG-101", { exact: true }).last()).toBeVisible()
  await expect(page.getByText("Regression Candidate", { exact: true }).last()).toBeVisible()
  await expect(page.getByText("CONF-001", { exact: true }).last()).toBeVisible()
  await expect(page.getByText("成绩已出", { exact: true }).last()).toBeVisible()
  expect(requests).toContain("GET /api/exams")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("exam detail reads metadata, result, and transitions without synchronizing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installExamReadMocks(page, requests)
  await page.goto("/exams")

  await page.getByRole("button", { name: "查看详情" }).last().click()

  const dialog = page.getByRole("dialog", { name: "考试详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("Regression Certification", { exact: true })).toBeVisible()
  await expect(dialog.getByText("candidate@example.test", { exact: true })).toBeVisible()
  await expect(dialog.getByText("88.5", { exact: true }).first()).toBeVisible()
  await expect(dialog.getByText("成绩已生成", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/exams/exam-1")
  expect(requests).toContain("GET /api/exams/exam-1/result")
  expect(requests).toContain("GET /api/exams/exam-1/transitions")
  expect(requests.some((request) => request.includes("sync-result"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
