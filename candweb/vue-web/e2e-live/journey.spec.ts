import { expect, test, type APIRequestContext, type Locator, type Page } from "@playwright/test"
import { liveEnvironment } from "./support/live"

test.setTimeout(600_000)

type APIEnvelope<T> = {
  code?: number
  message?: string
  data?: T
}

async function requestData<T>(request: APIRequestContext, baseURL: string, path: string) {
  const response = await request.get(new URL(path, baseURL).toString(), {
    headers: { Accept: "application/json" },
  })
  const payload = await response.json().catch(() => null) as APIEnvelope<T> | null
  expect(response.status(), `GET ${path}`).toBeGreaterThanOrEqual(200)
  expect(response.status(), `GET ${path}`).toBeLessThan(300)
  expect(payload?.code, `GET ${path} response code`).toBe(200)
  return payload?.data as T
}

async function postData<T>(
  request: APIRequestContext,
  baseURL: string,
  path: string,
  data: unknown,
) {
  const response = await request.post(new URL(path, baseURL).toString(), {
    data,
    headers: { Accept: "application/json" },
  })
  const payload = await response.json().catch(() => null) as APIEnvelope<T> | null
  expect(response.status(), `POST ${path}`).toBeGreaterThanOrEqual(200)
  expect(response.status(), `POST ${path}`).toBeLessThan(300)
  expect(payload?.code, `POST ${path} response code`).toBe(200)
  return payload?.data as T
}

async function missingRequiredProfileFields(page: Page) {
  const form = page.getByTestId("checkout-step-registration")
  const fields = form.locator("input[required], select[required]")
  const missing: string[] = []
  for (let index = 0; index < await fields.count(); index += 1) {
    const field = fields.nth(index)
    if (await field.getAttribute("type") === "checkbox") continue
    if (!String(await field.inputValue()).trim()) {
      missing.push(await field.getAttribute("id") || `field-${index + 1}`)
    }
  }
  return missing
}

async function visibleFrameLocator(
  page: Page,
  selectors: string[],
  description: string,
  timeout = 60_000,
): Promise<Locator> {
  const deadline = Date.now() + timeout
  while (Date.now() < deadline) {
    for (const frame of page.frames()) {
      for (const selector of selectors) {
        const locator = frame.locator(selector).first()
        if (await locator.isVisible().catch(() => false)) return locator
      }
    }
    await page.waitForTimeout(500)
  }
  throw new Error(`Stripe ${description} field was not found`)
}

async function fillOptionalStripeField(page: Page, selectors: string[], value: string) {
  for (const frame of page.frames()) {
    for (const selector of selectors) {
      const locator = frame.locator(selector).first()
      if (await locator.isVisible().catch(() => false)) {
        await locator.fill(value)
        return
      }
    }
  }
}

async function selectOptionalStripeField(page: Page, selectors: string[], value: string) {
  for (const frame of page.frames()) {
    for (const selector of selectors) {
      const locator = frame.locator(selector).first()
      if (await locator.isVisible().catch(() => false)) {
        await locator.selectOption(value)
        return
      }
    }
  }
}

async function completeStripeTestPayment(page: Page) {
  await fillOptionalStripeField(
    page,
    ['input[name="email"]', 'input[type="email"]'],
    process.env.E2E_STRIPE_TEST_EMAIL?.trim() || "candidate-regression@example.test",
  )
  await (await visibleFrameLocator(
    page,
    ['input[name="cardNumber"]', 'input[autocomplete="cc-number"]'],
    "card number",
  )).fill("4242424242424242")
  await (await visibleFrameLocator(
    page,
    ['input[name="cardExpiry"]', 'input[autocomplete="cc-exp"]'],
    "expiry",
  )).fill("1230")
  await (await visibleFrameLocator(
    page,
    ['input[name="cardCvc"]', 'input[autocomplete="cc-csc"]'],
    "CVC",
  )).fill("212")
  await fillOptionalStripeField(
    page,
    ['input[name="billingName"]', 'input[autocomplete="name"]'],
    "Candidate Regression",
  )
  await selectOptionalStripeField(
    page,
    ['select[name="country"]', 'select[name="billingCountry"]'],
    "US",
  )
  await fillOptionalStripeField(
    page,
    ['input[name="addressLine1"]', 'input[autocomplete="address-line1"]'],
    "1 Regression Street",
  )
  await (await visibleFrameLocator(
    page,
    [
      'input[name="postalCode"]',
      'input[name="billingPostalCode"]',
      'input[autocomplete="postal-code"]',
      'input[aria-label="ZIP"]',
      'input[placeholder="ZIP"]',
    ],
    "postal code",
    15_000,
  )).fill("10001")

  const submit = await visibleFrameLocator(
    page,
    ['button[type="submit"]'],
    "payment submit button",
  )
  await submit.click()
}

function purchaseEligibility(bundle: any) {
  return bundle?.purchase_state?.eligibility || bundle?.eligibility || {}
}

function bundleID(bundle: any) {
  return String(bundle?.bundle_id || bundle?.bundle_ulid || "").trim()
}

function bundlePipelineID(bundle: any) {
  return String(bundle?.pipeline_id || bundle?.pipeline_cc_ulid || "").trim()
}

function pipelineConfigID(pipeline: any) {
  return String(pipeline?.pipeline_cc_ulid || "").trim()
}

function bundlePaidAmount(bundle: any) {
  const preview = bundle?.purchase_state?.payment_preview || bundle?.payment_preview || {}
  const previewTotal = Number(preview?.total || 0)
  if (previewTotal > 0) return previewTotal
  return Math.max(
    Number(bundle?.display_amount_min || 0),
    Number(bundle?.display_amount_max || 0),
  )
}

function bundleHasCourse(bundle: any) {
  return (bundle?.stages || []).some((stage: any) =>
    (stage?.units || []).some((unit: any) => Boolean(unit?.glms_course_id)),
  )
}

function activeBundleOrder(bundle: any) {
  return bundle?.purchase_state?.active_order || bundle?.active_order || null
}

type CandidateOrder = {
  order_id?: string
  biz_type?: string
  biz_ref_ulid?: string
  order_status?: string
  payment_status?: string
  product_name?: string
  meta?: {
    product_name?: string
  }
}

const actionableOrderStatuses = new Set([
  "WAIT_PAYMENT",
  "PENDING",
  "PENDING_PAYMENT",
  "WAIT_PIPELINE_PAYMENT",
  "WAIT_STAGE_PAYMENT",
  "WAIT_RETAKE_PAYMENT",
  "WAIT_UNLOCK_PAYMENT",
  "WAIT_BUNDLE_PAYMENT",
  "WAIT_REVIEW_FEE_PAYMENT",
])

const paidPaymentStatuses = new Set(["PAID", "SUCCESS", "COMPLETED"])

function normalizedStatus(value: unknown) {
  return String(value || "").trim().toUpperCase()
}

function normalizedName(value: unknown) {
  return String(value || "").trim().replace(/\s+/g, " ").toLowerCase()
}

function bundleName(bundle: any) {
  return String(bundle?.name || bundle?.bundle_name || bundle?.title || "").trim()
}

function orderProductName(order: CandidateOrder | null | undefined) {
  return String(order?.product_name || order?.meta?.product_name || "").trim()
}

function isPaidOrder(order: CandidateOrder | null | undefined) {
  return paidPaymentStatuses.has(normalizedStatus(order?.payment_status))
    || paidPaymentStatuses.has(normalizedStatus(order?.order_status))
}

function isActionableOrder(order: CandidateOrder | null | undefined) {
  return Boolean(
    order
    && normalizedStatus(order.biz_type) === "BUNDLE_PURCHASE"
    && order.biz_ref_ulid
    && !isPaidOrder(order)
    && actionableOrderStatuses.has(normalizedStatus(order.order_status)),
  )
}

async function candidateOrders(request: APIRequestContext, baseURL: string) {
  const result = await requestData<{ orders?: CandidateOrder[] }>(
    request,
    baseURL,
    "/api/orders?page_size=50",
  )
  return result?.orders || []
}

function commonOrderForBusinessOrder(orders: CandidateOrder[], businessOrderID: string) {
  return orders.find((order) =>
    String(order?.biz_ref_ulid || "").trim() === businessOrderID
      || String(order?.order_id || "").trim() === businessOrderID,
  ) || null
}

async function commonOrderPaymentCompleted(
  request: APIRequestContext,
  baseURL: string,
  businessOrderID: string,
) {
  const order = commonOrderForBusinessOrder(
    await candidateOrders(request, baseURL),
    businessOrderID,
  )
  return isPaidOrder(order)
}

async function cancelPendingBundleOrder(
  request: APIRequestContext,
  baseURL: string,
  orderID: string,
) {
  const cancelled = await postData<{
    success?: boolean
    biz_ref_ulid?: string
    status?: string
  }>(
    request,
    baseURL,
    "/api/orders/cancel",
    {
      biz_type: "BUNDLE_PURCHASE",
      biz_ref_ulid: orderID,
    },
  )
  expect(cancelled?.success, `cancel pending bundle order ${orderID}`).toBe(true)
}

async function cancelBundleOrderIfPossible(
  request: APIRequestContext,
  baseURL: string,
  bundleID: string,
  expectedOrderID: string,
) {
  const detail = await requestData<any>(
    request,
    baseURL,
    `/api/mall/bundles/${encodeURIComponent(bundleID)}`,
  )
  const activeOrder = activeBundleOrder(detail)
  const activeOrderID = String(activeOrder?.order_id || "").trim()
  if (activeOrderID !== expectedOrderID || activeOrder?.can_cancel !== true) {
    return false
  }
  await cancelPendingBundleOrder(request, baseURL, expectedOrderID)
  return true
}

async function candidateHasPipeline(
  request: APIRequestContext,
  baseURL: string,
  pipelineID: string,
) {
  const pipelines = await requestData<{ list?: any[] }>(
    request,
    baseURL,
    "/api/pipeline",
  )
  return (pipelines?.list || []).some((pipeline) => pipelineConfigID(pipeline) === pipelineID)
}

async function bundleOrderDiagnostic(
  request: APIRequestContext,
  baseURL: string,
  orderID: string,
) {
  const order = commonOrderForBusinessOrder(
    await candidateOrders(request, baseURL),
    orderID,
  )
  if (!order) return `order=${orderID}; common-order=not-found`
  return [
    `order=${orderID}`,
    `common-order=${String(order?.order_id || "unknown")}`,
    `order-status=${String(order?.order_status || "unknown")}`,
    `payment-status=${String(order?.payment_status || "unknown")}`,
  ].join("; ")
}

type CertificationSelectionMode = "resume" | "purchase" | "owned"

type CertificationSelection = {
  bundle: any | null
  mode: CertificationSelectionMode | null
  order: CandidateOrder | null
  diagnostic: string
}

async function findCertificationJourney(
  request: APIRequestContext,
  baseURL: string,
  catalog: any[],
): Promise<CertificationSelection> {
  const orders = await candidateOrders(request, baseURL)
  const pipelines = await requestData<{ list?: any[] }>(
    request,
    baseURL,
    "/api/pipeline",
  )
  const ownedPipelineIDs = new Set(
    (pipelines?.list || []).map(pipelineConfigID).filter(Boolean),
  )
  const counts = {
    catalog: catalog.length,
    details: 0,
    pipeline: 0,
    purchasable: 0,
    unlockable: 0,
    paid: 0,
    withCourse: 0,
    resumablePendingOrders: 0,
    owned: 0,
  }
  let purchaseSelection: CertificationSelection | null = null
  let ownedSelection: CertificationSelection | null = null

  for (const summary of catalog) {
    const id = bundleID(summary)
    if (!id) continue

    const detail = await requestData<any>(
      request,
      baseURL,
      `/api/mall/bundles/${encodeURIComponent(id)}`,
    )
    counts.details += 1
    const pipelineID = bundlePipelineID(detail)
    const hasPipeline = Boolean(pipelineID)
    const eligibility = purchaseEligibility(detail)
    const canPurchase = eligibility?.can_purchase === true
    const canUnlock = eligibility?.can_unlock === true
    const isPaid = bundlePaidAmount(detail) > 0
    const hasCourse = bundleHasCourse(detail)
    const activeOrder = activeBundleOrder(detail)
    const activeOrderID = String(activeOrder?.order_id || "").trim()
    const commonOrder = commonOrderForBusinessOrder(orders, activeOrderID)
    const activeOrderMatchesBundle = Boolean(
      commonOrder
      && normalizedName(orderProductName(commonOrder))
      && normalizedName(orderProductName(commonOrder)) === normalizedName(bundleName(detail)),
    )

    if (hasPipeline) counts.pipeline += 1
    if (canPurchase) counts.purchasable += 1
    if (canUnlock) counts.unlockable += 1
    if (isPaid) counts.paid += 1
    if (hasCourse) counts.withCourse += 1
    if (ownedPipelineIDs.has(pipelineID)) counts.owned += 1

    if (
      hasPipeline
      && isPaid
      && hasCourse
      && activeOrder?.action === "purchase"
      && activeOrderMatchesBundle
      && isActionableOrder(commonOrder)
    ) {
      counts.resumablePendingOrders += 1
      return {
        bundle: detail,
        mode: "resume",
        order: commonOrder,
        diagnostic: "",
      }
    }

    if (!purchaseSelection && hasPipeline && canPurchase && isPaid && hasCourse) {
      purchaseSelection = {
        bundle: detail,
        mode: "purchase",
        order: null,
        diagnostic: "",
      }
    }

    if (!ownedSelection && hasPipeline && isPaid && hasCourse && ownedPipelineIDs.has(pipelineID)) {
      ownedSelection = {
        bundle: detail,
        mode: "owned",
        order: null,
        diagnostic: "",
      }
    }
  }

  if (purchaseSelection) return purchaseSelection
  if (ownedSelection) return ownedSelection

  return {
    bundle: null,
    mode: null,
    order: null,
    diagnostic: [
      "No resumable, directly purchasable, or already owned paid certification with a course is available.",
      `catalog=${counts.catalog}`,
      `details=${counts.details}`,
      `pipeline=${counts.pipeline}`,
      `purchasable=${counts.purchasable}`,
      `unlockable=${counts.unlockable}`,
      `paid=${counts.paid}`,
      `with-course=${counts.withCourse}`,
      `resumable-pending-orders=${counts.resumablePendingOrders}`,
      `owned=${counts.owned}`,
    ].join("; "),
  }
}

function reportJourneyStage(stage: string, detail: string) {
  const description = detail.trim()
  console.log(`[candidate-live-journey] ${stage}: ${description}`)
  test.info().annotations.push({
    type: `journey-${stage}`,
    description,
  })
}

async function completeReadyLessons(page: Page) {
  const lessons = page.locator('[data-testid="course-lesson"][data-lesson-id]')
  await expect(
    page.getByTestId("course-learning-loading"),
    "the selected certification course must finish loading",
  ).toBeHidden({ timeout: 60_000 })
  await expect(
    page.getByTestId("course-unavailable"),
    "the selected certification course must be available",
  ).toBeHidden()
  await expect(
    lessons.first(),
    "the selected certification course must contain at least one lesson after loading",
  ).toBeVisible({ timeout: 60_000 })
  await expect(
    page.getByTestId("course-lessons-empty"),
    "the selected certification course must not be empty",
  ).toBeHidden()

  const total = await lessons.count()
  expect(total, "the selected certification course must contain at least one lesson").toBeGreaterThan(0)

  const completedBefore = await page.locator(
    '[data-testid="course-lesson"][data-completed="true"]',
  ).count()
  let completedNow = 0

  for (let pass = 0; pass < total; pass += 1) {
    let progressed = false
    for (let index = 0; index < total; index += 1) {
      const lesson = lessons.nth(index)
      if (await lesson.getAttribute("data-completed") === "true") continue
      if (await lesson.getAttribute("data-has-pending-quizzes") === "true") continue

      await lesson.click()
      const complete = page.getByTestId("complete-lesson")
      await expect(complete, "an incomplete lesson without pending quizzes must be completable").toBeEnabled()
      await complete.click()
      await expect(lesson, "completed lesson state must refresh in the course list").toHaveAttribute(
        "data-completed",
        "true",
        { timeout: 30_000 },
      )
      completedNow += 1
      progressed = true
    }
    if (!progressed) break
  }

  return {
    total,
    completedBefore,
    completedNow,
    remaining: await page.locator(
      '[data-testid="course-lesson"][data-completed="false"]',
    ).count(),
  }
}

async function openQuizTasks(page: Page) {
  const quizStep = page.locator(
    '[data-testid="certification-flow-step"][data-step-id="quiz"]',
  ).first()
  await expect(quizStep, "the selected certification must expose a quiz step").toBeEnabled()
  await quizStep.click()

  const openQuizList = page.getByTestId("open-quiz-list")
  if (await openQuizList.isVisible().catch(() => false)) await openQuizList.click()
  await expect(
    page.getByTestId("start-quiz").first(),
    "the selected certification must contain at least one configured quiz",
  ).toBeVisible()
}

async function firstEnabledQuizButton(page: Page) {
  const buttons = page.getByTestId("start-quiz").filter({ visible: true })
  for (let index = 0; index < await buttons.count(); index += 1) {
    const button = buttons.nth(index)
    if (await button.isEnabled().catch(() => false)) return button
  }
  return null
}

async function chooseQuizAnswers(
  page: Page,
  correctAnswers?: Map<string, string[]>,
) {
  const options = page.locator('[data-testid="quiz-option"][data-question-id]')
  await expect.poll(
    () => options.count(),
    {
      message: "started quiz must render at least one question",
      timeout: 30_000,
      intervals: [500, 1_000, 2_000],
    },
  ).toBeGreaterThan(0)

  const questionIDs = await options.evaluateAll((nodes) =>
    [...new Set(nodes.map((node) => node.getAttribute("data-question-id")).filter(Boolean))],
  )
  for (const questionID of questionIDs) {
    const configuredAnswers = correctAnswers?.get(String(questionID)) || []
    if (correctAnswers) {
      expect(
        configuredAnswers.length,
        `detailed grading must expose at least one correct option for question ${questionID}`,
      ).toBeGreaterThan(0)
    }
    const optionIDs = configuredAnswers.length > 0
      ? configuredAnswers
      : [
          String(
            (await page.locator(
              `[data-testid="quiz-option"][data-question-id="${questionID}"]`,
            ).first().getAttribute("data-option-id")) || "",
          ),
        ]
    for (const optionID of optionIDs) {
      expect(optionID, `question ${questionID} must expose an option ID`).not.toBe("")
      await page.locator(
        `[data-testid="quiz-option"][data-question-id="${questionID}"][data-option-id="${optionID}"]`,
      ).click()
    }
  }
}

async function submitQuiz(page: Page) {
  await page.getByTestId("quiz-submit").click()
  await page.getByTestId("quiz-submit-confirm").click()
  const result = page.getByTestId("quiz-result")
  await expect(result).toBeVisible()
  return await result.getAttribute("data-passed") === "true"
}

async function correctAnswersFromQuizDetail(page: Page) {
  const correctOptions = page.locator(
    '[data-testid="quiz-detail-option"][data-correct="true"]',
  )
  if (!await correctOptions.first().isVisible().catch(() => false)) {
    const detailButton = page.getByTestId("quiz-detail")
    await expect(
      detailButton,
      "failed quiz must either show grading automatically or expose the detail action",
    ).toBeVisible({ timeout: 10_000 })
    await detailButton.click()
  }
  await expect(correctOptions.first(), "failed quiz must expose detailed correct answers").toBeVisible()
  const answers = await correctOptions.evaluateAll((nodes) => {
    const result: Record<string, string[]> = {}
    for (const node of nodes) {
      const questionID = node.getAttribute("data-question-id") || ""
      const optionID = node.getAttribute("data-option-id") || ""
      if (!questionID || !optionID) continue
      result[questionID] = [...(result[questionID] || []), optionID]
    }
    return result
  })
  return new Map(Object.entries(answers))
}

async function returnToLearning(page: Page) {
  await page.getByTestId("quiz-return").click()
  await expect(page).toHaveURL(
    /\/certifications\/[^/?#]+\/learn\/[^/?#]+/,
    { timeout: 30_000 },
  )
}

async function completePendingQuizzes(page: Page) {
  await openQuizTasks(page)
  const total = await page.getByTestId("start-quiz").count()
  let completedNow = 0

  for (let index = 0; index < total; index += 1) {
    const startQuiz = await firstEnabledQuizButton(page)
    if (!startQuiz) break
    const quizID = String(await startQuiz.getAttribute("data-quiz-id") || "").trim()
    expect(quizID, "an available quiz must expose its quiz ID").not.toBe("")
    await startQuiz.click()
    await chooseQuizAnswers(page)

    let passed = await submitQuiz(page)
    if (!passed) {
      const correctAnswers = await correctAnswersFromQuizDetail(page)
      await returnToLearning(page)
      await openQuizTasks(page)
      const retry = page.locator(
        `[data-testid="start-quiz"][data-quiz-id="${quizID}"]`,
      ).filter({ visible: true }).first()
      await expect(retry, `failed quiz ${quizID} must allow a retry`).toBeEnabled()
      await retry.click()
      await chooseQuizAnswers(page, correctAnswers)
      passed = await submitQuiz(page)
    }

    expect(passed, `quiz ${quizID} must pass before the formal exam can open`).toBe(true)
    completedNow += 1
    await returnToLearning(page)
    await openQuizTasks(page)
  }

  expect(
    await firstEnabledQuizButton(page),
    "all configured quizzes must be completed before continuing to the formal exam",
  ).toBeNull()
  const quizStep = page.locator(
    '[data-testid="certification-flow-step"][data-step-id="quiz"]',
  ).first()
  await expect(quizStep, "the quiz flow step must be marked completed").toHaveAttribute(
    "data-status",
    "done",
    { timeout: 30_000 },
  )

  return {
    total,
    completedBefore: total - completedNow,
    completedNow,
  }
}

function runtimeCourseUnitID(runtime: any, courseID: string) {
  const nextStepID = String(runtime?.next_step?.course_unit_ulid || "").trim()
  if (nextStepID) return nextStepID
  for (const stage of runtime?.config?.stages || []) {
    for (const unit of stage?.units || []) {
      const unitCourseID = String(
        unit?.glms_course_id || unit?.course_id || unit?.course_ulid || unit?.courseUlid || "",
      ).trim()
      if (unitCourseID === courseID) {
        return String(unit?.course_unit_ulid || unit?.unit_ulid || "").trim()
      }
    }
  }
  return ""
}

async function missingExamSignupFields(page: Page) {
  const form = page.getByTestId("exam-signup-form")
  const fields = form.locator("input[required], select[required]")
  const missing: string[] = []
  for (let index = 0; index < await fields.count(); index += 1) {
    const field = fields.nth(index)
    if (!String(await field.inputValue()).trim()) {
      missing.push(await field.getAttribute("data-testid") || `field-${index + 1}`)
    }
  }

  const birthdateMarker = page.getByTestId("exam-signup-birthdate")
  const birthdateInput = await birthdateMarker.locator("input").count() > 0
    ? birthdateMarker.locator("input").first()
    : birthdateMarker
  if (!String(await birthdateInput.inputValue()).trim()) missing.push("exam-signup-birthdate")
  if (!String(await page.locator("#exam-signup-work-phone").inputValue()).trim()) {
    missing.push("exam-signup-work-phone")
  }
  return missing
}

async function candidateExamForUnit(
  request: APIRequestContext,
  baseURL: string,
  courseUnitID: string,
) {
  const result = await requestData<{ exams?: any[] }>(
    request,
    baseURL,
    "/api/exams?page_size=100",
  )
  return (result?.exams || []).find((exam) =>
    String(exam?.course_unit_ulid || "").trim() === courseUnitID,
  ) || null
}

test.describe("candidate live certification purchase, learning, quiz, and exam journey", () => {
  test.skip(
    process.env.E2E_LIVE_JOURNEY_ENABLED !== "true",
    "Set E2E_LIVE_JOURNEY_ENABLED=true only for an approved test-data run.",
  )

  test("buy or resume a certification and progress through learning, quizzes, and exam signup", async ({ page }) => {
    const environment = liveEnvironment()
    const request = page.context().request
    const catalog = await requestData<{ bundles?: any[] }>(
      request,
      environment.baseURL,
      "/api/mall/bundles",
    )
    const catalogBundles = catalog?.bundles || []

    await page.goto("/certifications", { waitUntil: "domcontentloaded" })
    const renderedCards = page.locator(
      '[data-testid="certification-card"][data-bundle-id]',
    )
    if (catalogBundles.length > 0) {
      await expect.poll(
        () => renderedCards.count(),
        {
          message: [
            "The deployed candidate portal did not render the certification-card test markers.",
            "Deploy candweb from commit f2bc67c or later before running the live paid journey.",
          ].join(" "),
          timeout: 30_000,
        },
      ).toBeGreaterThan(0)
    }

    const renderedBundleIDs = new Set(
      await renderedCards.evaluateAll((nodes) =>
        nodes
          .map((node) => node.getAttribute("data-bundle-id")?.trim() || "")
          .filter(Boolean),
      ),
    )
    const renderedCatalog = catalogBundles.filter((bundle) =>
      renderedBundleIDs.has(bundleID(bundle)),
    )
    const selection = await findCertificationJourney(
      request,
      environment.baseURL,
      renderedCatalog,
    )

    expect(
      selection.bundle,
      [
        selection.diagnostic,
        `api-catalog=${catalogBundles.length}`,
        `rendered-cards=${renderedBundleIDs.size}`,
      ].filter(Boolean).join("; "),
    ).toBeTruthy()

    const bundle = selection.bundle
    const selectedBundleID = bundleID(bundle)
    const pipelineID = bundlePipelineID(bundle)
    const mode = selection.mode
    let paymentOrderID = String(selection.order?.biz_ref_ulid || "").trim()
    let createdNewOrder = false

    test.info().annotations.push({
      type: "journey-mode",
      description: mode === "resume"
        ? `Continue pending order ${paymentOrderID}`
        : mode === "purchase"
          ? `Create and pay for bundle ${selectedBundleID}`
          : `Reuse owned pipeline ${pipelineID}`,
    })

    if (mode === "purchase") {
      const card = page.locator(
        `[data-testid="certification-card"][data-bundle-id="${selectedBundleID}"]`,
      )
      await expect(card, "selected purchasable certification card").toBeVisible()
      await card.click()

      const selectionStep = page.getByTestId("checkout-step-selection")
      if (await selectionStep.isVisible().catch(() => false)) {
        const selectedExemptions = selectionStep.locator(
          '[data-testid="checkout-exemption-toggle"]:checked',
        )
        while (await selectedExemptions.count() > 0) {
          await selectedExemptions.first().uncheck({ force: true })
        }
        const selectionNext = page.getByTestId("checkout-selection-next")
        await expect(selectionNext).toBeEnabled()
        await selectionNext.click()
      }
      await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
      await expect.poll(
        () => missingRequiredProfileFields(page),
        {
          message: "The existing candidate profile must be complete before running the purchase journey",
          timeout: 30_000,
        },
      ).toEqual([])
      expect(
        await page.locator("#exam-signup-work-phone").inputValue(),
        "The existing candidate profile must be complete before running the purchase journey",
      ).toMatch(/\d/)

      const agreement = page.getByTestId("checkout-agreement")
      if (await agreement.isVisible().catch(() => false)) {
        await agreement.check()
      }
      const checkoutNext = page.getByTestId("checkout-next")
      await expect(checkoutNext).toBeEnabled()
      await checkoutNext.click()
      await expect(page.getByTestId("checkout-step-review")).toBeVisible()
      await page.getByTestId("checkout-confirm-pay").click()
      await expect(page.getByTestId("checkout-step-payment")).toBeVisible()

      const paymentBundle = await requestData<any>(
        request,
        environment.baseURL,
        `/api/mall/bundles/${encodeURIComponent(selectedBundleID)}`,
      )
      paymentOrderID = String(activeBundleOrder(paymentBundle)?.order_id || "").trim()
      expect(paymentOrderID, "checkout must expose the created bundle order").not.toBe("")
      createdNewOrder = true
    } else if (mode === "resume") {
      expect(paymentOrderID, "resumable order must expose its business order ID").not.toBe("")
      const productName = orderProductName(selection.order) || bundleName(bundle)
      await page.goto("/orders", { waitUntil: "domcontentloaded" })
      const matchingRows = page.locator(".order-row").filter({ hasText: productName })
      const continuePayment = matchingRows
        .getByRole("button", { name: /继续支付|continue payment/i })
        .first()
      await expect(
        continuePayment,
        `order ${paymentOrderID} must expose Continue Payment in the candidate order list`,
      ).toBeVisible()
      await continuePayment.click()
    }

    let purchaseCompleted = false
    if (mode !== "owned") {
      try {
        await completeStripeTestPayment(page)
        try {
          await expect.poll(
            () => commonOrderPaymentCompleted(
              request,
              environment.baseURL,
              paymentOrderID,
            ),
            {
              message: "Stripe payment should mark the common order as paid or completed",
              timeout: 180_000,
              intervals: [1_000, 2_000, 5_000],
            },
          ).toBe(true)
          await expect.poll(
            () => candidateHasPipeline(
              request,
              environment.baseURL,
              pipelineID,
            ),
            {
              message: "Stripe payment should create or retain the candidate pipeline",
              timeout: 180_000,
              intervals: [1_000, 2_000, 5_000],
            },
          ).toBe(true)
        } catch {
          const diagnostic = await bundleOrderDiagnostic(
            request,
            environment.baseURL,
            paymentOrderID,
          )
          throw new Error(
            `Stripe payment did not complete the order and candidate pipeline; ${diagnostic}; page=${new URL(page.url()).pathname}`,
          )
        }
        purchaseCompleted = true
      } finally {
        if (
          createdNewOrder
          && !purchaseCompleted
          && !await commonOrderPaymentCompleted(
            request,
            environment.baseURL,
            paymentOrderID,
          )
        ) {
          try {
            await cancelBundleOrderIfPossible(
              request,
              environment.baseURL,
              selectedBundleID,
              paymentOrderID,
            )
          } catch (error) {
            test.info().annotations.push({
              type: "pending-order-cleanup",
              description: error instanceof Error ? error.message : String(error),
            })
          }
        }
      }
    }

    if (
      mode === "purchase"
      && !/\/checkout\/success\/[^/?#]+/.test(new URL(page.url()).pathname)
    ) {
      test.info().annotations.push({
        type: "payment-return",
        description: "Payment completed, but the browser did not enter the checkout success route.",
      })
    }

    await page.goto("/my-certifications", { waitUntil: "domcontentloaded" })
    const certification = page.locator(
      `[data-testid="owned-certification-details"][data-pipeline-config-id="${pipelineID}"]`,
    )
    await expect(certification, "newly purchased certification").toBeVisible()
    await certification.click()

    const runtime = await requestData<{ next_step?: { course_id?: string } }>(
      request,
      environment.baseURL,
      `/api/mall/pipelines/${encodeURIComponent(pipelineID)}/runtime`,
    )
    const nextCourseID = String(runtime?.next_step?.course_id || "").trim()
    const course = nextCourseID
      ? page.locator(`[data-testid="course-unit-link"][data-course-id="${nextCourseID}"]`)
      : page.locator('[data-testid="course-unit-link"][data-course-id]').first()
    await expect(course, "purchased certification must expose a learning course").toBeVisible()
    await course.click()
    await expect(
      page,
      "opening the selected certification course must enter the learning page",
    ).toHaveURL(/\/certifications\/[^/?#]+\/learn\/[^/?#]+(?:\/lessons\/[^/?#]+)?(?:\?|$)/, {
      timeout: 30_000,
    })

    reportJourneyStage(
      "course-opened",
      `pipeline=${pipelineID}; course=${nextCourseID || "first-visible-course"}`,
    )

    const firstLessonPass = await test.step(
      "complete lessons that are not blocked by quizzes",
      () => completeReadyLessons(page),
    )
    reportJourneyStage(
      "lessons-before-quizzes",
      `total=${firstLessonPass.total}; already-complete=${firstLessonPass.completedBefore}; completed-now=${firstLessonPass.completedNow}; blocked-or-remaining=${firstLessonPass.remaining}`,
    )

    const quizProgress = await test.step(
      "complete every configured quiz and retry failed attempts with published grading",
      () => completePendingQuizzes(page),
    )
    reportJourneyStage(
      "quizzes",
      `total=${quizProgress.total}; already-complete=${quizProgress.completedBefore}; completed-now=${quizProgress.completedNow}`,
    )

    const finalLessonPass = await test.step(
      "finish lessons that were blocked by lesson quizzes",
      () => completeReadyLessons(page),
    )
    expect(
      finalLessonPass.remaining,
      "all course lessons must be completed before formal exam signup",
    ).toBe(0)
    reportJourneyStage(
      "lessons-complete",
      `total=${finalLessonPass.total}; completed-now-after-quizzes=${finalLessonPass.completedNow}`,
    )

    const examActions = new Set([
      "signup_exam",
      "schedule_exam",
      "view_exam_schedule",
      "view_exam_result",
      "apply_retake",
      "view_certificate",
      "completed",
      "issuing_certificate",
      "final_qualification",
    ])
    const completedExamActions = new Set(
      [...examActions].filter((action) => action !== "signup_exam"),
    )
    await expect.poll(
      async () => {
        const nextRuntime = await requestData<any>(
          request,
          environment.baseURL,
          `/api/mall/pipelines/${encodeURIComponent(pipelineID)}/runtime`,
        )
        return examActions.has(String(nextRuntime?.next_step?.action || "").trim())
      },
      {
        message: "completed learning and quizzes must advance the pipeline to an exam or later state",
        timeout: 90_000,
        intervals: [1_000, 2_000, 5_000],
      },
    ).toBe(true)

    let examRuntime = await requestData<any>(
      request,
      environment.baseURL,
      `/api/mall/pipelines/${encodeURIComponent(pipelineID)}/runtime`,
    )
    let examAction = String(examRuntime?.next_step?.action || "").trim()
    const courseID = nextCourseID || String(
      new URL(page.url()).pathname.split("/learn/")[1] || "",
    ).split("/")[0]
    const courseUnitID = runtimeCourseUnitID(examRuntime, courseID)
    reportJourneyStage(
      "exam-ready",
      `action=${examAction}; course-unit=${courseUnitID || "not-required-for-terminal-state"}`,
    )

    if (examAction === "signup_exam") {
      expect(courseUnitID, "formal exam signup requires a runtime course unit ID").not.toBe("")
      const examStep = page.locator(
        '[data-testid="certification-flow-step"][data-step-id="exam"]',
      ).first()
      await expect(examStep, "formal exam step must be available after learning and quizzes").toBeEnabled({
        timeout: 30_000,
      })
      await examStep.click()
      const signupLink = page.getByTestId("exam-signup-link").first()
      await expect(signupLink, "formal exam step must expose the signup action").toBeVisible({
        timeout: 30_000,
      })
      await signupLink.click()
      await expect(page).toHaveURL(/\/exams\/signup(?:\?|$)/)
      await expect(page.getByTestId("exam-signup-form")).toBeVisible()
      await expect.poll(
        () => missingExamSignupFields(page),
        {
          message: "the existing candidate profile must contain all formal exam signup fields",
          timeout: 30_000,
        },
      ).toEqual([])
      await page.getByTestId("exam-signup-submit").click()
      await expect(page).toHaveURL(/\/exams(?:\?|$)/, { timeout: 60_000 })
      await expect.poll(
        () => candidateExamForUnit(request, environment.baseURL, courseUnitID),
        {
          message: "formal exam signup must create a candidate exam record",
          timeout: 60_000,
          intervals: [1_000, 2_000, 5_000],
        },
      ).not.toBeNull()
      reportJourneyStage("exam-signup", `registered course-unit=${courseUnitID}`)

      await expect.poll(
        async () => {
          const nextRuntime = await requestData<any>(
            request,
            environment.baseURL,
            `/api/mall/pipelines/${encodeURIComponent(pipelineID)}/runtime`,
          )
          return completedExamActions.has(
            String(nextRuntime?.next_step?.action || "").trim(),
          )
        },
        {
          message: "formal exam signup must advance the pipeline to an exam or certification state",
          timeout: 60_000,
          intervals: [1_000, 2_000, 5_000],
        },
      ).toBe(true)
      examRuntime = await requestData<any>(
        request,
        environment.baseURL,
        `/api/mall/pipelines/${encodeURIComponent(pipelineID)}/runtime`,
      )
      examAction = String(examRuntime?.next_step?.action || "").trim()
    } else if (courseUnitID) {
      await expect.poll(
        () => candidateExamForUnit(request, environment.baseURL, courseUnitID),
        {
          message: `exam state ${examAction} must have a candidate exam record`,
          timeout: 30_000,
          intervals: [1_000, 2_000, 5_000],
        },
      ).not.toBeNull()
    }

    expect(
      [...completedExamActions],
      `formal exam signup must advance beyond signup_exam; current action=${examAction}`,
    ).toContain(examAction)
    reportJourneyStage(
      "journey-complete",
      `pipeline=${pipelineID}; final-action=${examAction}; formal exam execution itself remains with the third-party exam provider`,
    )
  })
})
