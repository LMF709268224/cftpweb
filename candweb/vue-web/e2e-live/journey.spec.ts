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
  await fillOptionalStripeField(
    page,
    ['input[name="postalCode"]', 'input[autocomplete="postal-code"]'],
    "10001",
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
  const orders = await requestData<{ orders?: any[] }>(
    request,
    baseURL,
    "/api/orders?page_size=50",
  )
  const order = (orders?.orders || []).find((item) =>
    String(item?.biz_ref_ulid || "").trim() === orderID
      || String(item?.order_id || "").trim() === orderID,
  )
  if (!order) return `order=${orderID}; common-order=not-found`
  return [
    `order=${orderID}`,
    `order-status=${String(order?.order_status || "unknown")}`,
    `payment-status=${String(order?.payment_status || "unknown")}`,
  ].join("; ")
}

type CertificationSelection = {
  bundle: any | null
  diagnostic: string
}

async function findPurchasableCertification(
  request: APIRequestContext,
  baseURL: string,
  catalog: any[],
): Promise<CertificationSelection> {
  const counts = {
    catalog: catalog.length,
    details: 0,
    pipeline: 0,
    purchasable: 0,
    unlockable: 0,
    paid: 0,
    withCourse: 0,
    recoveredPendingOrders: 0,
  }

  for (const summary of catalog) {
    const id = bundleID(summary)
    if (!id) continue

    let detail = await requestData<any>(
      request,
      baseURL,
      `/api/mall/bundles/${encodeURIComponent(id)}`,
    )
    counts.details += 1
    let hasPipeline = Boolean(bundlePipelineID(detail))
    let eligibility = purchaseEligibility(detail)
    let canPurchase = eligibility?.can_purchase === true
    let canUnlock = eligibility?.can_unlock === true
    let isPaid = bundlePaidAmount(detail) > 0
    let hasCourse = bundleHasCourse(detail)
    const activeOrder = activeBundleOrder(detail)

    if (
      hasPipeline
      && isPaid
      && hasCourse
      && !canPurchase
      && activeOrder?.action === "purchase"
      && activeOrder?.can_cancel === true
      && String(activeOrder?.order_id || "").trim()
    ) {
      await cancelPendingBundleOrder(
        request,
        baseURL,
        String(activeOrder.order_id).trim(),
      )
      counts.recoveredPendingOrders += 1
      detail = await requestData<any>(
        request,
        baseURL,
        `/api/mall/bundles/${encodeURIComponent(id)}`,
      )
      eligibility = purchaseEligibility(detail)
      hasPipeline = Boolean(bundlePipelineID(detail))
      canPurchase = eligibility?.can_purchase === true
      canUnlock = eligibility?.can_unlock === true
      isPaid = bundlePaidAmount(detail) > 0
      hasCourse = bundleHasCourse(detail)
    }

    if (hasPipeline) counts.pipeline += 1
    if (canPurchase) counts.purchasable += 1
    if (canUnlock) counts.unlockable += 1
    if (isPaid) counts.paid += 1
    if (hasCourse) counts.withCourse += 1

    if (hasPipeline && canPurchase && isPaid && hasCourse) {
      return {
        bundle: detail,
        diagnostic: "",
      }
    }
  }

  return {
    bundle: null,
    diagnostic: [
      "No direct paid certification purchase is currently available for the configured candidate.",
      `catalog=${counts.catalog}`,
      `details=${counts.details}`,
      `pipeline=${counts.pipeline}`,
      `purchasable=${counts.purchasable}`,
      `unlockable=${counts.unlockable}`,
      `paid=${counts.paid}`,
      `with-course=${counts.withCourse}`,
      `recovered-pending-orders=${counts.recoveredPendingOrders}`,
    ].join("; "),
  }
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
    const selection = await findPurchasableCertification(
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
    const paymentOrderID = String(activeBundleOrder(paymentBundle)?.order_id || "").trim()
    expect(paymentOrderID, "checkout must expose the created bundle order").not.toBe("")

    let purchaseCompleted = false
    try {
      await completeStripeTestPayment(page)
      try {
        await expect.poll(
          () => candidateHasPipeline(
            request,
            environment.baseURL,
            pipelineID,
          ),
          {
            message: "Stripe payment should create the purchased candidate pipeline",
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
          `Stripe payment did not create the candidate pipeline; ${diagnostic}; page=${new URL(page.url()).pathname}`,
        )
      }
      purchaseCompleted = true
    } finally {
      if (!purchaseCompleted && !await candidateHasPipeline(
        request,
        environment.baseURL,
        pipelineID,
      )) {
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

    if (!/\/checkout\/success\/[^/?#]+/.test(new URL(page.url()).pathname)) {
      test.info().annotations.push({
        type: "payment-return",
        description: "Payment completed and the pipeline was created, but the browser did not enter the checkout success route.",
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
