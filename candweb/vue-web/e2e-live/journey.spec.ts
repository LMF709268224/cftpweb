import { expect, test, type Locator, type Page } from "@playwright/test"

test.setTimeout(600_000)

function requiredEnvironment(name: string) {
  const value = process.env[name]?.trim()
  if (!value) throw new Error(`${name} is required for the live business journey`)
  return value
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

async function completeStripeTestPayment(page: Page) {
  const email = process.env.E2E_STRIPE_TEST_EMAIL?.trim() || "candidate-regression@example.test"
  const fields = [
    {
      selectors: ['input[name="email"]', 'input[type="email"]'],
      value: email,
      description: "email",
    },
    {
      selectors: ['input[name="cardNumber"]', 'input[autocomplete="cc-number"]'],
      value: "4242424242424242",
      description: "card number",
    },
    {
      selectors: ['input[name="cardExpiry"]', 'input[autocomplete="cc-exp"]'],
      value: "1230",
      description: "expiry",
    },
    {
      selectors: ['input[name="cardCvc"]', 'input[autocomplete="cc-csc"]'],
      value: "212",
      description: "CVC",
    },
    {
      selectors: ['input[name="billingName"]', 'input[autocomplete="name"]'],
      value: "Candidate Regression",
      description: "billing name",
    },
  ]

  for (const field of fields) {
    const locator = await visibleFrameLocator(page, field.selectors, field.description)
    await locator.fill(field.value)
  }

  const submit = await visibleFrameLocator(
    page,
    ['button[type="submit"]'],
    "payment submit button",
  )
  await expect(submit, "Stripe payment submit button").toBeVisible()
  await submit.click()
}

test.describe("candidate live certification and learning journeys", () => {
  test.skip(
    process.env.E2E_LIVE_JOURNEY_ENABLED !== "true",
    "Set E2E_LIVE_JOURNEY_ENABLED=true only for an approved destructive test-data run.",
  )

  test("purchase a certification through Stripe Test Mode", async ({ page }) => {
    const bundleID = requiredEnvironment("E2E_LIVE_PURCHASE_BUNDLE_ID")
    await page.goto("/certifications", { waitUntil: "domcontentloaded" })

    const card = page.locator(
      `[data-testid="certification-card"][data-bundle-id="${bundleID}"]`,
    )
    await expect(card, "configured test certification must be purchasable").toBeVisible()
    await card.click()

    await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
    const requiredInputs = page.getByTestId("checkout-step-registration").locator("input[required], select[required]")
    const missingFields: string[] = []
    for (let index = 0; index < await requiredInputs.count(); index += 1) {
      const field = requiredInputs.nth(index)
      if (await field.getAttribute("type") === "checkbox") continue
      if (!String(await field.inputValue()).trim()) {
        missingFields.push(await field.getAttribute("name") || await field.getAttribute("id") || `field-${index + 1}`)
      }
    }
    expect(
      missingFields,
      "Dedicated journey candidate profile must be complete before running a paid journey",
    ).toEqual([])

    await page.getByTestId("checkout-agreement").check()
    await page.getByTestId("checkout-next").click()
    await expect(page.getByTestId("checkout-step-review")).toBeVisible()
    await page.getByTestId("checkout-confirm-pay").click()
    await expect(page.getByTestId("checkout-step-payment")).toBeVisible()

    await completeStripeTestPayment(page)
    await expect(page).toHaveURL(/\/checkout\/success\/[^/?#]+/, { timeout: 180_000 })
    await expect(page.locator('a[href="/my-certifications"]')).toBeVisible()
  })

  test("complete a lesson and pass the configured course quiz", async ({ page }) => {
    const pipelineID = requiredEnvironment("E2E_LIVE_LEARNING_PIPELINE_ID")
    const courseID = requiredEnvironment("E2E_LIVE_LEARNING_COURSE_ID")
    const lessonID = requiredEnvironment("E2E_LIVE_LEARNING_LESSON_ID")
    const quizID = requiredEnvironment("E2E_LIVE_LEARNING_QUIZ_ID")

    await page.goto("/my-certifications", { waitUntil: "domcontentloaded" })
    const certification = page.locator(
      `[data-testid="owned-certification-details"][data-pipeline-config-id="${pipelineID}"]`,
    )
    await expect(certification, "configured certification must belong to the journey candidate").toBeVisible()
    await certification.click()

    const course = page.locator(`[data-testid="course-unit-link"][data-course-id="${courseID}"]`)
    await expect(course, "configured learning course must be available").toBeVisible()
    await course.click()

    const lesson = page.locator(`[data-testid="course-lesson"][data-lesson-id="${lessonID}"]`)
    await expect(lesson, "configured lesson must exist").toBeVisible()
    await expect(
      lesson,
      "configured lesson must be reset to incomplete before rerunning the journey",
    ).toHaveAttribute("data-completed", "false")
    await lesson.click()
    await page.getByTestId("complete-lesson").click()
    await expect(lesson).toHaveAttribute("data-completed", "true")

    await page.locator('[data-testid="certification-flow-step"][data-step-id="quiz"]').first().click()
    const openQuizList = page.getByTestId("open-quiz-list")
    if (await openQuizList.isVisible().catch(() => false)) await openQuizList.click()
    const startQuiz = page.locator(`[data-testid="start-quiz"][data-quiz-id="${quizID}"]`)
    await expect(
      startQuiz,
      "configured quiz must be reset to incomplete before rerunning the journey",
    ).toBeEnabled()
    await startQuiz.click()

    const questions = page.locator('[data-testid="quiz-option"][data-question-id]')
    const questionIDs = await questions.evaluateAll((nodes) =>
      [...new Set(nodes.map((node) => node.getAttribute("data-question-id")).filter(Boolean))],
    )
    expect(questionIDs.length, "configured quiz must contain questions").toBeGreaterThan(0)
    for (const questionID of questionIDs) {
      await page.locator(`[data-testid="quiz-option"][data-question-id="${questionID}"]`).first().click()
    }

    await page.getByTestId("quiz-submit").click()
    await page.getByTestId("quiz-submit-confirm").click()
    await expect(page.getByTestId("quiz-result")).toBeVisible()
    await expect(page.getByText(/通过|Passed/i).first()).toBeVisible()
  })
})
