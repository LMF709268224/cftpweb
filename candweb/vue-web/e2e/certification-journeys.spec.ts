import { expect, test } from "@playwright/test"
import {
  candidateUser,
  installCandidateApiMocks,
  installStripeCheckoutMock,
  seedAuthenticatedCandidate,
} from "./support/candidate"

const bundleID = "bundle-regression"
const pipelineID = "pipeline-regression"
const pipelineInstanceID = "pipeline-instance-regression"
const courseID = "course-regression"
const lessonID = "lesson-regression"
const quizID = "quiz-regression"

const bundle = {
  bundle_id: bundleID,
  pipeline_id: pipelineID,
  name: "CFtP Regression Certification",
  description: "认证购买完整流程回归数据",
  bundle_item_types: ["pipeline"],
  is_pipeline_bundle: true,
  stages: [],
  eligibility: { can_purchase: true, can_unlock: false, blockers: [] },
  purchase_state: {
    eligibility: { can_purchase: true, can_unlock: false, blockers: [] },
    exemption_options: { stages: [] },
    payment_preview: {
      subtotal: 10000,
      discount_total: 0,
      tax_total: 0,
      total: 10000,
      currency: "USD",
    },
  },
}

function pipelineList() {
  return {
    list: [{
      pipeline_cc_ulid: pipelineID,
      pipeline_ulid: pipelineInstanceID,
      pipeline_name: "CFtP Regression Certification",
      current_stage_name: "Regression Stage",
      progress_available: true,
      progress: 50,
      status: "LEARNING",
      started_at: "2026-08-06T00:00:00Z",
    }],
  }
}

function pipelineRuntime() {
  return {
    instance: { pipeline_ulid: pipelineInstanceID },
    config: {
      pipeline_cc_ulid: pipelineID,
      name: "CFtP Regression Certification",
      description: "认证学习完整流程回归数据",
      stages: [{
        stage_id: "stage-regression",
        name: "Regression Stage",
        sort_order: 1,
        runtime_status: "LEARNING",
        units: [{
          unit_id: "unit-regression",
          course_unit_ulid: "course-unit-regression",
          glms_course_id: courseID,
          name: "Regression Learning Course",
          runtime_status: "LEARNING",
        }],
      }],
    },
    next_step: {
      action: "continue_learning",
      course_id: courseID,
      course_unit_ulid: "course-unit-regression",
      lesson_id: lessonID,
      status: "LEARNING",
    },
    pipeline_status: "LEARNING",
    current_stage_name: "Regression Stage",
    current_stage_status: "LEARNING",
    current_unit_status: "LEARNING",
  }
}

test.beforeEach(async ({ page }) => {
  await seedAuthenticatedCandidate(page)
})

test("认证从商城下单、Stripe 支付到已购认证完整闭环", async ({ page }) => {
  let paid = false
  let purchaseBody: unknown

  await installStripeCheckoutMock(page)
  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/mall/bundles" && method === "GET") return { data: { bundles: [bundle] } }
    if (pathname === `/api/mall/bundles/${bundleID}` && method === "GET") return { data: bundle }
    if (pathname === `/api/mall/bundles/${bundleID}/pricing-detail`) return { data: {} }
    if (pathname === "/api/user/me") return { data: candidateUser }
    if (pathname === "/api/user/profile" && method === "PUT") return { data: { success: true } }
    if (pathname === `/api/mall/bundles/${bundleID}/purchase` && method === "POST") {
      purchaseBody = body
      return {
        data: {
          bundle_order_ulid: "order-regression",
          bundle_pay_order_ulid: "pay-order-regression",
          order_status: "WAIT_PAYMENT",
        },
      }
    }
    if (pathname === "/api/mall/payments/preview" && method === "POST") {
      return { data: bundle.purchase_state.payment_preview }
    }
    if (pathname === "/api/mall/payments/initiate" && method === "POST") {
      return { data: { payment_key: "cs_test_playwright_secret" } }
    }
    if (pathname === "/api/pipeline") return { data: paid ? pipelineList() : { list: [] } }
    return undefined
  })

  await page.goto("/certifications", { waitUntil: "domcontentloaded" })
  await page.locator(`[data-testid="certification-card"][data-bundle-id="${bundleID}"]`).click()

  await expect(page).toHaveURL(new RegExp(`/checkout/${bundleID}$`))
  await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
  await page.getByTestId("checkout-agreement").check()
  await page.getByTestId("checkout-next").click()

  await expect(page.getByTestId("checkout-step-review")).toBeVisible()
  await page.getByTestId("checkout-confirm-pay").click()
  await expect(page.getByTestId("checkout-step-payment")).toBeVisible()
  await expect(page.getByTestId("fake-stripe-complete")).toBeVisible()

  paid = true
  await page.getByTestId("fake-stripe-complete").click()
  await expect(page).toHaveURL(/\/checkout\/success\/order-regression/)
  await page.getByRole("link", { name: "查看我的认证" }).click()

  await expect(page.locator(
    `[data-testid="owned-certification-details"][data-pipeline-config-id="${pipelineID}"]`,
  )).toBeVisible()
  expect(purchaseBody).toEqual({
    payment_mode: "FULL_PIPELINE",
    selected_exemptions_json: JSON.stringify({ [pipelineID]: { stages: [] } }),
  })
})

test("已购认证进入课程、完成课件并通过测验完整闭环", async ({ page }) => {
  let lessonCompleted = false
  let quizPassed = false
  const submittedBodies: unknown[] = []

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === "/api/pipeline") return { data: pipelineList() }
    if (pathname === `/api/mall/pipelines/${pipelineID}/runtime`) return { data: pipelineRuntime() }
    if (pathname === `/api/mall/courses/${courseID}`) {
      return { data: { course_id: courseID, title: "Regression Learning Course", duration_min: 30 } }
    }
    if (pathname === `/api/mall/courses/${courseID}/thumbnail-url`) return { data: {} }
    if (pathname === `/api/pipeline/courses/${courseID}/complete`) {
      return {
        data: {
          complete_course: {
            course: { course_id: courseID, title: "Regression Learning Course", duration_min: 30 },
            chapters: [{
              chapter: { chapter_id: "chapter-regression", title: "Regression Chapter" },
              lessons: [{
                lesson: {
                  lesson_id: lessonID,
                  title: "Regression Lesson",
                  lesson_type: "article",
                  body: "<p>Regression lesson content</p>",
                },
                quizzes: [],
              }],
              quizzes: [],
            }],
            materials: [],
            quizzes: [{
              quiz: { quiz_id: quizID, title: "Regression Quiz", quiz_type: 1 },
            }],
          },
          quiz_progress: quizPassed ? { [quizID]: { is_passed: true } } : {},
        },
      }
    }
    if (pathname === "/api/progress") {
      return { data: { records: lessonCompleted ? [{ material_id: lessonID }] : [] } }
    }
    if (pathname === `/api/progress/courses/${courseID}/sync` && method === "POST") {
      return {
        data: {
          progress_percentage: lessonCompleted ? (quizPassed ? 100 : 50) : 0,
          completed_lessons_count: lessonCompleted ? 1 : 0,
          passed_quizzes_count: quizPassed ? 1 : 0,
        },
      }
    }
    if (pathname === `/api/pipeline/lessons/${lessonID}/complete` && method === "POST") {
      lessonCompleted = true
      return { data: { success: true } }
    }
    if (pathname === `/api/quizzes/${quizID}/take` && method === "POST") {
      return { data: { attempt_id: "attempt-regression" } }
    }
    if (pathname === "/api/quizzes/attempts/attempt-regression/paper") {
      return {
        data: {
          attempt_id: "attempt-regression",
          title: "Regression Quiz",
          remaining_seconds: 600,
          questions: [{
            question_id: "question-regression",
            question_text: "Which option completes the regression journey?",
            question_type: 1,
            points: 10,
            options: [
              { option_id: "option-correct", option_text: "Complete the lesson and quiz" },
              { option_id: "option-wrong", option_text: "Skip all learning" },
            ],
          }],
        },
      }
    }
    if (pathname === "/api/quizzes/attempts/attempt-regression/draft" && method === "POST") {
      return { data: { success: true } }
    }
    if (pathname === "/api/quizzes/attempts/attempt-regression/submit" && method === "POST") {
      submittedBodies.push(body)
      quizPassed = true
      return { data: { score: 10, max_score: 10, pass_status: 1, is_passed: true } }
    }
    return undefined
  })

  await page.goto("/my-certifications", { waitUntil: "domcontentloaded" })
  await page.locator(
    `[data-testid="owned-certification-details"][data-pipeline-config-id="${pipelineID}"]`,
  ).click()
  await page.locator(`[data-testid="course-unit-link"][data-course-id="${courseID}"]`).click()

  await expect(page.getByRole("heading", { name: "Regression Learning Course" }).first()).toBeVisible()
  await page.getByTestId("complete-lesson").click()
  await expect(page.locator(`[data-testid="course-lesson"][data-lesson-id="${lessonID}"]`)).toHaveAttribute("data-completed", "true")

  await page.locator('[data-testid="certification-flow-step"][data-step-id="quiz"]').first().click()
  await page.getByTestId("open-quiz-list").click()
  await page.locator(`[data-testid="start-quiz"][data-quiz-id="${quizID}"]`).click()
  await page.locator('[data-testid="quiz-option"][data-option-id="option-correct"]').click()
  await page.getByTestId("quiz-submit").click()
  await page.getByTestId("quiz-submit-confirm").click()

  await expect(page.getByTestId("quiz-result")).toBeVisible()
  await expect(page.getByText("10", { exact: true }).first()).toBeVisible()
  expect(submittedBodies).toEqual([{
    submissions: [{
      question_id: "question-regression",
      selected_option_ids: ["option-correct"],
    }],
  }])
})
