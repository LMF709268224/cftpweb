import { expect, test, type Page } from "@playwright/test"
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

async function waitForCheckoutProfile(page: Page) {
  const form = page.getByTestId("checkout-step-registration")
  await expect.poll(async () => {
    const fields = form.locator("input[required], select[required]")
    const missing: string[] = []
    for (let index = 0; index < await fields.count(); index += 1) {
      const field = fields.nth(index)
      if (!await field.isVisible()) continue
      if (await field.getAttribute("type") === "checkbox") continue
      if (!String(await field.inputValue()).trim()) {
        missing.push(await field.getAttribute("id") || `field-${index + 1}`)
      }
    }
    return missing
  }, {
    message: "candidate profile should finish loading before checkout submission",
  }).toEqual([])
}

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

test("商城和结账页展示并拦截层级互斥资格", async ({ page }) => {
  const blockerCases = [
    {
      id: "bundle-forbidden-qualification",
      type: "FORBIDDEN_QUALIFICATION",
      message: "已持有更高阶认证，不能报读",
    },
    {
      id: "bundle-conflict-in-progress",
      type: "CONFLICT_PIPELINE_IN_PROGRESS",
      message: "有互斥认证正在进行",
    },
    {
      id: "bundle-conflict-unavailable",
      type: "CONFLICT_CHECK_UNAVAILABLE",
      message: "暂时无法核验互斥认证状态",
    },
  ]
  const blockedBundles = blockerCases.map((blocker) => ({
    ...bundle,
    bundle_id: blocker.id,
    name: `Blocked ${blocker.id}`,
    eligibility: { can_purchase: false, can_unlock: false, blockers: [{ blocker_type: blocker.type }] },
    purchase_state: {
      ...bundle.purchase_state,
      eligibility: { can_purchase: false, can_unlock: false, blockers: [{ blocker_type: blocker.type }] },
    },
  }))

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/mall/bundles" && method === "GET") return { data: { bundles: blockedBundles } }
    const blockedBundle = blockedBundles.find((item) => pathname === `/api/mall/bundles/${item.bundle_id}`)
    if (blockedBundle && method === "GET") return { data: blockedBundle }
    if (pathname.endsWith("/pricing-detail")) return { data: {} }
    return undefined
  })

  await page.goto("/certifications", { waitUntil: "domcontentloaded" })
  for (const blocker of blockerCases) {
    const card = page.locator(`[data-testid="certification-card"][data-bundle-id="${blocker.id}"]`)
    await expect(card.getByText(blocker.message, { exact: true })).toBeVisible()
  }

  await page.locator(`[data-testid="certification-card"][data-bundle-id="${blockerCases[0].id}"]`).click()
  await expect(page).toHaveURL(/\/certifications$/)

  await page.goto(`/checkout/${blockerCases[1].id}`, { waitUntil: "domcontentloaded" })
  const blockerPanel = page.getByTestId("checkout-eligibility-blockers")
  await expect(blockerPanel).toContainText("你有互斥认证正在进行，请先完成或取消后再购买")
  await expect(page.getByTestId("checkout-next")).toBeDisabled()
})

test("已持有有效资格的课程自动免考且不可取消", async ({ page }) => {
  const stageID = "stage-auto-exemption"
  const unitID = "unit-auto-exemption"
  const qualificationID = "qualification-auto-exemption"
  const automaticExemptionBundle = {
    ...bundle,
    stages: [{
      stage_id: stageID,
      name: "Automatic Exemption Stage",
      sort_order: 1,
      units: [{ unit_id: unitID, name: "CFtA Course" }],
    }],
    purchase_state: {
      ...bundle.purchase_state,
      exemption_options: {
        stages: [{
          index: 0,
          stage_id: stageID,
          stage_name: "Automatic Exemption Stage",
          units: [{
            unit_id: unitID,
            unit_name: "CFtA Course",
            allow_exemption: true,
            qualified: true,
            exemption_quals: [{
              qual_id: qualificationID,
              name: "CFtA Certification",
              eligible: true,
              credential_status: "CREDENTIAL_STATUS_ACTIVE",
            }],
          }],
        }],
      },
    },
  }
  let purchaseBody: unknown

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === `/api/mall/bundles/${bundleID}` && method === "GET") return { data: automaticExemptionBundle }
    if (pathname === `/api/mall/bundles/${bundleID}/pricing-detail`) {
      return {
        data: {
          units: [{ unit_id: unitID, access: { amount: 10000, currency: "USD" } }],
          memberships: [],
        },
      }
    }
    if (pathname === "/api/credentials/definitions") {
      return { data: { definitions: [{ cred_def_ulid: qualificationID, name: "CFtA Certification" }] } }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/user/me") return { data: candidateUser }
    if (pathname === "/api/user/profile" && method === "PUT") return { data: { success: true } }
    if (pathname === `/api/mall/bundles/${bundleID}/purchase` && method === "POST") {
      purchaseBody = body
      return {
        data: {
          bundle_order_ulid: "order-auto-exemption",
          order_status: "COMPLETED",
        },
      }
    }
    return undefined
  })

  await page.goto(`/checkout/${bundleID}`, { waitUntil: "domcontentloaded" })

  const automaticToggle = page.locator(`[data-testid="checkout-exemption-toggle"][data-unit-id="${unitID}"]`)
  await expect(automaticToggle).toBeChecked()
  await expect(automaticToggle).toBeDisabled()
  await expect(page.getByText("系统自动免考", { exact: true })).toBeVisible()

  await page.getByTestId("checkout-selection-next").click()
  await waitForCheckoutProfile(page)
  await page.getByTestId("checkout-agreement").check()
  await page.getByTestId("checkout-next").click()
  await page.getByTestId("checkout-confirm-pay").click()

  await expect(page).toHaveURL(/\/checkout\/success\/order-auto-exemption/)
  expect(purchaseBody).toEqual({
    payment_mode: "FULL_PIPELINE",
    selected_exemptions_json: JSON.stringify({}),
  })
})

test("课程视频预览通过签名地址全屏播放", async ({ page }) => {
  let requestedSignedURL = false
  const signedURL = "https://iframe.videodelivery.net/signed-lesson-token"

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === `/api/pipeline/lessons/${lessonID}/video-play-url` && method === "GET") {
      requestedSignedURL = true
      return { data: { url: signedURL, expires_at: "2026-08-17T12:00:00Z" } }
    }
    return undefined
  })

  await page.goto(`/video-preview/lessons/${lessonID}`, { waitUntil: "domcontentloaded" })

  await expect(page.getByTestId("video-preview-frame")).toHaveAttribute("src", signedURL)
  expect(requestedSignedURL).toBe(true)
})

test("Token 外部课件使用考生凭证打开并保留手动完成流程", async ({ page }) => {
  const tokenLessonID = "lesson-token-courseware"
  let accessURLRequested = false
  let lessonCompleted = false

  await page.context().route("https://partner.example/**", async (route) => {
    await route.fulfill({ status: 200, contentType: "text/html", body: "<title>Partner courseware</title>" })
  })
  await installCandidateApiMocks(page, ({ pathname, method }) => {
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
              chapter: { chapter_id: "chapter-token", title: "External Content" },
              lessons: [{
                lesson: {
                  lesson_id: tokenLessonID,
                  title: "Partner Token Lesson",
                  lesson_type: 8,
                  external_courseware_ulid: "courseware-1",
                },
                quizzes: [],
              }],
              quizzes: [],
            }],
            materials: [],
            quizzes: [],
          },
          quiz_progress: {},
        },
      }
    }
    if (pathname === "/api/progress") {
      return { data: { records: lessonCompleted ? [{ material_id: tokenLessonID }] : [] } }
    }
    if (pathname === `/api/progress/courses/${courseID}/sync` && method === "POST") {
      return { data: { progress_percentage: lessonCompleted ? 100 : 0 } }
    }
    if (pathname === `/api/pipeline/lessons/${tokenLessonID}/access-url` && method === "POST") {
      accessURLRequested = true
      return {
        data: {
          access_url: "https://partner.example/learn/candidate-access-value",
          courseware_name: "Partner Academy",
        },
      }
    }
    if (pathname === `/api/pipeline/lessons/${tokenLessonID}/complete` && method === "POST") {
      lessonCompleted = true
      return { data: { success: true } }
    }
    return undefined
  })

  await page.goto(`/certifications/${pipelineID}/learn/${courseID}`, { waitUntil: "domcontentloaded" })
  const popupPromise = page.waitForEvent("popup")
  await page.getByRole("button", { name: /打开外部课件/ }).click()
  const popup = await popupPromise
  await expect.poll(() => popup.url()).toBe("https://partner.example/learn/candidate-access-value")
  expect(accessURLRequested).toBe(true)

  await page.getByTestId("complete-lesson").click()
  await expect(page.locator(`[data-testid="course-lesson"][data-lesson-id="${tokenLessonID}"]`)).toHaveAttribute("data-completed", "true")
  expect(lessonCompleted).toBe(true)
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
  await waitForCheckoutProfile(page)
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
    selected_exemptions_json: JSON.stringify({}),
  })
})

test("支付步骤说明锁定原因并允许取消订单后返回修改", async ({ page }) => {
  let cancelBody: unknown

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === `/api/mall/bundles/${bundleID}` && method === "GET") return { data: bundle }
    if (pathname === `/api/mall/bundles/${bundleID}/pricing-detail`) return { data: {} }
    if (pathname === "/api/user/me") return { data: candidateUser }
    if (pathname === "/api/user/profile" && method === "PUT") return { data: { success: true } }
    if (pathname === `/api/mall/bundles/${bundleID}/purchase` && method === "POST") {
      return { data: { bundle_order_ulid: "order-cancel-regression", order_status: "WAIT_PAYMENT" } }
    }
    if (pathname === "/api/mall/payments/preview" && method === "POST") {
      return { data: bundle.purchase_state.payment_preview }
    }
    if (pathname === "/api/orders/cancel" && method === "POST") {
      cancelBody = body
      return { data: { success: true, order_status: "CANCELLED" } }
    }
    return undefined
  })

  await page.goto(`/checkout/${bundleID}`, { waitUntil: "domcontentloaded" })
  await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
  await waitForCheckoutProfile(page)
  await page.getByTestId("checkout-agreement").check()
  await page.getByTestId("checkout-next").click()
  await page.getByTestId("checkout-confirm-pay").click()

  await expect(page.getByTestId("checkout-step-payment")).toBeVisible()
  await expect(page.getByText("待支付订单已经创建，价格和报名选择已锁定，不能直接返回修改。如需调整，请先取消当前订单。", { exact: true })).toBeVisible()
  page.once("dialog", dialog => dialog.accept())
  await page.getByTestId("checkout-cancel-and-edit").click()

  await expect(page.getByTestId("checkout-step-review")).toBeVisible()
  await page.getByRole("button", { name: "返回" }).click()
  await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
  expect(cancelBody).toEqual({
    biz_type: "BUNDLE_PURCHASE",
    biz_ref_ulid: "order-cancel-regression",
  })
})

test("已购认证进入课程、完成课件并通过测验完整闭环", async ({ page }) => {
  let lessonCompleted = false
  let quizPassed = false
  let quizStartRequests = 0
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
      quizStartRequests += 1
      if (quizStartRequests === 1) {
        return { status: 503, message: "quiz service temporarily unavailable" }
      }
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
  await expect(page.getByTestId("supplementary-content-nav")).toHaveCount(0)
  await page.getByTestId("complete-lesson").click()
  await expect(page.locator(`[data-testid="course-lesson"][data-lesson-id="${lessonID}"]`)).toHaveAttribute("data-completed", "true")

  await page.locator('[data-testid="certification-flow-step"][data-step-id="quiz"]').first().click()
  await page.getByTestId("open-quiz-list").click()
  await page.locator(`[data-testid="start-quiz"][data-quiz-id="${quizID}"]`).click()
  await expect(page.getByText("测验服务暂时繁忙，请稍后重试。", { exact: true })).toBeVisible()
  await expect(page.getByText("操作失败", { exact: true })).toHaveCount(0)
  await page.locator(`[data-testid="start-quiz"][data-quiz-id="${quizID}"]`).click()
  await page.locator('[data-testid="quiz-option"][data-option-id="option-correct"]').click()
  await page.getByTestId("quiz-submit").click()
  await page.getByTestId("quiz-submit-confirm").click()

  await expect(page.getByTestId("quiz-result")).toBeVisible()
  await expect(page.getByText("10", { exact: true }).first()).toBeVisible()
  await page.getByTestId("quiz-return").click()
  await expect(page).toHaveURL(
    new RegExp(`/certifications/${pipelineID}/learn/${courseID}`),
  )
  await page.locator(
    '[data-testid="certification-flow-step"][data-step-id="quiz"]',
  ).first().click()
  await page.locator(
    '[data-testid="certification-flow-step"][data-step-id="lesson"]',
  ).first().click()
  await expect(
    page.locator(`[data-testid="course-lesson"][data-lesson-id="${lessonID}"]`),
  ).toBeVisible()
  expect(submittedBodies).toEqual([{
    submissions: [{
      question_id: "question-regression",
      selected_option_ids: ["option-correct"],
    }],
  }])
  expect(quizStartRequests).toBe(2)
})

test("同阶段课程可分别进入各自的考试报名", async ({ page }) => {
  const courseAID = "course-parallel-a"
  const courseBID = "course-parallel-b"
  const courseAUnitID = "course-unit-parallel-a"
  const courseBUnitID = "course-unit-parallel-b"
  const lessonBID = "lesson-parallel-b"

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === `/api/mall/pipelines/${pipelineID}/runtime`) {
      return {
        data: {
          instance: { pipeline_ulid: pipelineInstanceID },
          config: {
            pipeline_cc_ulid: pipelineID,
            name: "Parallel Exam Certification",
            stages: [{
              stage_id: "stage-parallel",
              name: "Parallel Stage",
              runtime_status: "STAGE_STATUS_RUNNING",
              units: [
                {
                  unit_id: "unit-parallel-a",
                  course_unit_ulid: courseAUnitID,
                  glms_course_id: courseAID,
                  name: "Parallel Course A",
                  program: "PROGRAM-A",
                  runtime_status: "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
                },
                {
                  unit_id: "unit-parallel-b",
                  course_unit_ulid: courseBUnitID,
                  glms_course_id: courseBID,
                  name: "Parallel Course B",
                  program: "PROGRAM-B",
                  runtime_status: "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
                },
              ],
            }],
          },
          next_step: {
            action: "signup_exam",
            course_id: courseAID,
            course_unit_ulid: courseAUnitID,
            status: "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
          },
          pipeline_status: "PIPELINE_STATUS_RUNNING",
          current_stage_status: "STAGE_STATUS_RUNNING",
          current_unit_status: "COURSE_UNIT_STATUS_WAITING_SIGNUP_EXAM",
        },
      }
    }
    if (pathname === `/api/pipeline/courses/${courseBID}/complete`) {
      return {
        data: {
          complete_course: {
            course: { course_id: courseBID, title: "Parallel Course B", duration_min: 30 },
            chapters: [{
              chapter: { chapter_id: "chapter-parallel-b", title: "Parallel Chapter B" },
              lessons: [{
                lesson: {
                  lesson_id: lessonBID,
                  title: "Parallel Lesson B",
                  lesson_type: "article",
                  body: "<p>Completed course content</p>",
                },
                quizzes: [],
              }],
              quizzes: [],
            }],
            materials: [],
            quizzes: [],
          },
          quiz_progress: {},
        },
      }
    }
    if (pathname === "/api/progress") return { data: { records: [{ material_id: lessonBID }] } }
    if (pathname === `/api/progress/courses/${courseBID}/sync` && method === "POST") {
      return { data: { progress_percentage: 100, completed_lessons_count: 1, passed_quizzes_count: 0 } }
    }
    if (pathname === "/api/exams") return { data: { exams: [], total: 0 } }
    return undefined
  })

  await page.goto(`/certifications/${pipelineID}/learn/${courseBID}`, { waitUntil: "domcontentloaded" })
  await page.locator('[data-testid="certification-flow-step"][data-step-id="exam"]').first().click()

  const signupLink = page.getByTestId("exam-signup-link")
  await expect(signupLink).toBeVisible()
  await signupLink.click()
  await expect(page).toHaveURL(
    `http://127.0.0.1:4173/exams/signup?unitId=${courseBUnitID}&pipelineId=${pipelineID}&courseId=${courseBID}`,
  )
})
