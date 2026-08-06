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

function bundlePaymentTotal(bundle: any) {
  const preview = bundle?.purchase_state?.payment_preview || bundle?.payment_preview || {}
  return Number(preview?.total || 0)
}

function bundleHasCourse(bundle: any) {
  return (bundle?.stages || []).some((stage: any) =>
    (stage?.units || []).some((unit: any) => Boolean(unit?.glms_course_id)),
  )
}

async function findPurchasableCertification(
  request: APIRequestContext,
  baseURL: string,
  catalog: any[],
) {
  for (const summary of catalog) {
    const id = bundleID(summary)
    if (!id) continue

    const detail = await requestData<any>(
      request,
      baseURL,
      `/api/mall/bundles/${encodeURIComponent(id)}`,
    )
    if (
      bundlePipelineID(detail)
      && purchaseEligibility(detail)?.can_purchase === true
      && bundlePaymentTotal(detail) > 0
      && bundleHasCourse(detail)
    ) {
      return detail
    }
  }
  return null
}

test.describe("candidate live certification purchase and learning journey", () => {
  test.skip(
    process.env.E2E_LIVE_JOURNEY_ENABLED !== "true",
    "Set E2E_LIVE_JOURNEY_ENABLED=true only for an approved test-data run.",
  )

  test("buy an available certification and continue into its learning flow", async ({ page }) => {
    const environment = liveEnvironment()
    const request = page.context().request
    const catalog = await requestData<{ bundles?: any[] }>(
      request,
      environment.baseURL,
      "/api/mall/bundles",
    )
    const bundle = await findPurchasableCertification(
      request,
      environment.baseURL,
      catalog?.bundles || [],
    )

    expect(
      bundle,
      "The configured candidate must have at least one currently purchasable certification",
    ).toBeTruthy()

    const selectedBundleID = bundleID(bundle)
    const pipelineID = bundlePipelineID(bundle)
    await page.goto("/certifications", { waitUntil: "domcontentloaded" })
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

    await page.getByTestId("checkout-agreement").check()
    await page.getByTestId("checkout-next").click()
    await expect(page.getByTestId("checkout-step-review")).toBeVisible()
    await page.getByTestId("checkout-confirm-pay").click()
    await expect(page.getByTestId("checkout-step-payment")).toBeVisible()

    await completeStripeTestPayment(page)
    await expect(page).toHaveURL(/\/checkout\/success\/[^/?#]+/, { timeout: 180_000 })

    await expect.poll(async () => {
      const pipelines = await requestData<{ list?: any[] }>(
        request,
        environment.baseURL,
        "/api/pipeline",
      )
      return (pipelines?.list || []).some((pipeline) => pipelineConfigID(pipeline) === pipelineID)
    }, {
      message: "Stripe webhook should create the purchased candidate pipeline",
      timeout: 120_000,
      intervals: [1_000, 2_000, 5_000],
    }).toBe(true)

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

    const incompleteLesson = page.locator(
      '[data-testid="course-lesson"][data-completed="false"]',
    ).first()
    await expect(
      incompleteLesson,
      "The purchased certification must have an incomplete lesson for this run",
    ).toBeVisible()
    await incompleteLesson.click()
    await expect(page.getByTestId("complete-lesson")).toBeEnabled()
    await page.getByTestId("complete-lesson").click()
    await expect(incompleteLesson).toHaveAttribute("data-completed", "true")

    const quizStep = page.locator(
      '[data-testid="certification-flow-step"][data-step-id="quiz"]',
    ).first()
    if (!await quizStep.isEnabled().catch(() => false)) {
      test.info().annotations.push({
        type: "learning",
        description: "The selected certification has no currently available quiz after the completed lesson.",
      })
      return
    }

    await quizStep.click()
    const openQuizList = page.getByTestId("open-quiz-list")
    if (await openQuizList.isVisible().catch(() => false)) await openQuizList.click()
    const startQuiz = page.getByTestId("start-quiz").filter({ visible: true }).first()
    if (!await startQuiz.isEnabled().catch(() => false)) {
      test.info().annotations.push({
        type: "learning",
        description: "The selected certification has no incomplete quiz available for submission.",
      })
      return
    }
    await startQuiz.click()

    const questionIDs = await page.locator(
      '[data-testid="quiz-option"][data-question-id]',
    ).evaluateAll((nodes) =>
      [...new Set(nodes.map((node) => node.getAttribute("data-question-id")).filter(Boolean))],
    )
    expect(questionIDs.length, "available quiz must contain questions").toBeGreaterThan(0)
    for (const questionID of questionIDs) {
      await page.locator(
        `[data-testid="quiz-option"][data-question-id="${questionID}"]`,
      ).first().click()
    }

    await page.getByTestId("quiz-submit").click()
    await page.getByTestId("quiz-submit-confirm").click()
    await expect(page.getByTestId("quiz-result")).toBeVisible()
  })
})
