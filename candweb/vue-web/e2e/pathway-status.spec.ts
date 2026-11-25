import { expect, test } from "@playwright/test"
import { installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

test("pathway separates learning, booking and navigation without inventing missing status", async ({ page }) => {
  await seedAuthenticatedCandidate(page)
  const units = [
    { unit_id: "zero", enrollment_status: "learning", learning_progress_percentage: 0, has_exam: true, runtime_status: "COURSE_UNIT_STATUS_WAITING_STUDY" },
    { unit_id: "booked", enrollment_status: "learning", learning_progress_percentage: 40, has_exam: true, runtime_status: "COURSE_UNIT_STATUS_EXAM_SCHEDULED", current_exam_id: "current-exam", current_exam: { status: "SCHEDULED", appointment_start_time: "2026-12-01T08:00:00Z" } },
    { unit_id: "completed", enrollment_status: "completed", has_exam: true, runtime_status: "COURSE_UNIT_STATUS_EXAM_OPEN", current_exam_id: "open-exam", current_exam: { status: "OPEN" } },
    { unit_id: "missing", has_exam: true, runtime_status: "COURSE_UNIT_STATUS_EXAM_SCHEDULED", current_exam_id: "unavailable-exam" },
    { unit_id: "no-exam", enrollment_status: "learning", learning_progress_percentage: 15, runtime_status: "COURSE_UNIT_STATUS_WAITING_STUDY" },
    { unit_id: "done", enrollment_status: "completed", has_exam: true, runtime_status: "COURSE_UNIT_STATUS_COMPLETED", current_exam_id: "done-exam", current_exam: { status: "DONE", appointment_start_time: "2026-10-30T08:00:00Z" } },
  ].map(unit => ({ ...unit, glms_course_id: unit.unit_id, name: unit.unit_id, has_learning_access: true }))
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/mall/pipelines/status-demo/runtime") return { data: {
      instance: { pipeline_ulid: "instance" },
      config: { name: "Status Demo", stages: [{ stage_id: "stage", name: "Foundation", runtime_status: "STAGE_STATUS_RUNNING", units }] },
      next_step: { action: "LEARN", course_id: "zero" },
    } }
    if (pathname.startsWith("/api/mall/courses/")) return { data: { course: { title: pathname.split("/").pop() } } }
  })
  await page.goto("/certifications/status-demo")
  const row = (id: string) => page.locator(`[data-testid="course-unit-row"][data-course-id="${id}"]`)
  await expect(row("zero").getByTestId("unit-learning-status")).toHaveText("学习进度：0%")
  await expect(row("zero").getByTestId("unit-exam-status")).toHaveText("暂未预约考试")
  await expect(row("booked").getByTestId("unit-learning-status")).toHaveText("学习：进行中（40%）")
  await expect(row("booked").getByTestId("unit-exam-status")).toContainText("已预约考试")
  await expect(row("booked").getByTestId("unit-exam-status")).toContainText("2026")
  await expect(row("completed").getByTestId("unit-learning-status")).toHaveText("学习：已完成")
  await expect(row("completed").getByTestId("unit-exam-status")).toHaveText("暂未预约考试")
  await expect(row("missing").getByTestId("unit-learning-status")).toHaveText("学习状态暂不可用")
  await expect(row("missing").getByTestId("unit-exam-status")).toHaveText("考试预约状态暂不可用")
  await expect(row("no-exam").getByTestId("unit-exam-status")).toHaveCount(0)
  await expect(row("no-exam").getByTestId("course-unit-link")).toHaveText("继续学习")
  await expect(row("done").getByTestId("unit-exam-status")).toHaveText("考试已完成")

  const badge = row("booked").getByTestId("unit-learning-status")
  expect(await badge.evaluate(el => ({ link: !!el.closest("a,button"), background: getComputedStyle(el).backgroundColor, cursor: getComputedStyle(el).cursor })))
    .toEqual({ link: false, background: "rgb(241, 245, 249)", cursor: "default" })
  await badge.click()
  await expect(page).toHaveURL(/\/certifications\/status-demo$/)
  await expect(row("booked").getByTestId("course-unit-link")).toHaveAttribute("href", "/certifications/status-demo/learn/booked")

  await page.evaluate(() => { localStorage.setItem("app_lang", "en"); window.dispatchEvent(new Event("lang_change")) })
  await expect(row("zero").getByTestId("unit-learning-status")).toHaveText("Learning progress: 0%")
  await expect(row("zero").getByTestId("unit-exam-status")).toHaveText("Exam Not Booked")
  await expect(row("booked").getByTestId("unit-learning-status")).toHaveText("Learning: In Progress (40%)")
  await expect(row("booked").getByTestId("unit-exam-status")).toContainText("Exam Booked")
  await expect(row("completed").getByTestId("unit-learning-status")).toHaveText("Learning: Completed")
  await page.screenshot({ path: "test-results/pathway-status-desktop.png", fullPage: true })
  await page.setViewportSize({ width: 390, height: 844 })
  const booked = row("booked")
  await booked.scrollIntoViewIfNeeded()
  expect(await booked.evaluate(el => el.scrollWidth <= el.clientWidth)).toBe(true)
  await page.screenshot({ path: "test-results/pathway-status-mobile.png", fullPage: true })
})
