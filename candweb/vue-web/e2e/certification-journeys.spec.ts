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

test("免考材料待上传时显示本地化原因并进入对应资格申请", async ({ page }) => {
  const blockedBundleID = "bundle-exemption-documents-pending"
  const blockedUnitID = "unit-exemption-documents-pending"
  const blockedQualificationID = "qualification-exemption-documents-pending"
  const backendDescription = "exemption review permission is active, but required documents have not been uploaded"
  const pendingBundle = {
    ...bundle,
    bundle_id: blockedBundleID,
    name: "Pending Exemption Documents",
    stages: [{
      stage_id: "stage-exemption-documents-pending",
      name: "L1",
      units: [{
        unit_id: blockedUnitID,
        name: "L1A Finance",
        allow_exemption: true,
        exemption_quals: [blockedQualificationID],
      }],
    }],
    eligibility: {
      can_purchase: false,
      can_unlock: false,
      blockers: [{
        blocker_type: "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
        description: backendDescription,
        details: [blockedUnitID],
      }],
    },
    purchase_state: {
      ...bundle.purchase_state,
      eligibility: {
        can_purchase: false,
        can_unlock: false,
        blockers: [{
          blocker_type: "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
          description: backendDescription,
          details: [blockedUnitID],
        }],
      },
    },
  }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/mall/bundles" && method === "GET") return { data: { bundles: [pendingBundle] } }
    if (pathname === `/api/mall/bundles/${blockedBundleID}` && method === "GET") return { data: pendingBundle }
    return undefined
  })

  await page.goto("/certifications", { waitUntil: "domcontentloaded" })
  const card = page.locator(`[data-testid="certification-card"][data-bundle-id="${blockedBundleID}"]`)
  await expect(card.getByText("请先上传免考材料", { exact: true })).toBeVisible()
  await expect(card.getByText("免考材料尚未上传。请先前往资格申请上传材料，完成审核后再购买认证。", { exact: true })).toBeVisible()
  await expect(card.getByText("去上传材料", { exact: true })).toBeVisible()
  await expect(card).not.toContainText(backendDescription)

  await page.evaluate(() => {
    localStorage.setItem("app_lang", "en")
    window.dispatchEvent(new Event("lang_change"))
  })
  const englishCard = page.locator(`[data-testid="certification-card"][data-bundle-id="${blockedBundleID}"]`)
  await expect(englishCard.getByText("Exemption documents required", { exact: true })).toBeVisible()
  await expect(englishCard.getByText("Upload Documents", { exact: true })).toBeVisible()
  await expect(englishCard).not.toContainText(backendDescription)

  await englishCard.click()
  await expect(page).toHaveURL(new RegExp(`/credentials\\?qual_ids=${blockedQualificationID}$`))
})

test("商城同时存在审核中和待上传免考时进入全部对应资格", async ({ page }) => {
  const blockedBundleID = "bundle-mixed-exemption-review"
  const underReviewUnitID = "unit-exemption-under-review"
  const pendingUploadUnitID = "unit-exemption-pending-upload"
  const underReviewQualificationID = "qualification-exemption-under-review"
  const pendingUploadQualificationID = "qualification-exemption-pending-upload"
  const mixedBundle = {
    ...bundle,
    bundle_id: blockedBundleID,
    name: "Mixed Exemption Review",
    stages: [{
      stage_id: "stage-mixed-exemption-review",
      name: "L1",
      units: [
        {
          unit_id: underReviewUnitID,
          name: "L1A Finance",
          allow_exemption: true,
          exemption_quals: [underReviewQualificationID],
        },
        {
          unit_id: pendingUploadUnitID,
          name: "L1B Fintech",
          allow_exemption: true,
          exemption_quals: [pendingUploadQualificationID],
        },
      ],
    }],
    eligibility: {
      can_purchase: false,
      can_unlock: false,
      blockers: [
        {
          blocker_type: "EXEMPTION_UNDER_REVIEW",
          details: [underReviewUnitID],
        },
        {
          blocker_type: "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
          details: [pendingUploadUnitID],
        },
      ],
    },
    purchase_state: {
      ...bundle.purchase_state,
      eligibility: {
        can_purchase: false,
        can_unlock: false,
        blockers: [
          {
            blocker_type: "EXEMPTION_UNDER_REVIEW",
            details: [underReviewUnitID],
          },
          {
            blocker_type: "EXEMPTION_DOCUMENTS_PENDING_UPLOAD",
            details: [pendingUploadUnitID],
          },
        ],
      },
    },
  }

  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === "/api/mall/bundles" && method === "GET") return { data: { bundles: [mixedBundle] } }
    if (pathname === `/api/mall/bundles/${blockedBundleID}` && method === "GET") return { data: mixedBundle }
    return undefined
  })

  await page.goto("/certifications", { waitUntil: "domcontentloaded" })
  const card = page.locator(`[data-testid="certification-card"][data-bundle-id="${blockedBundleID}"]`)
  await expect(card.getByText("免考资格审核中", { exact: true })).toBeVisible()
  await card.click()
  await expect(page).toHaveURL(new RegExp(
    `/credentials\\?qual_ids=${underReviewQualificationID},${pendingUploadQualificationID}$`,
  ))
})

test("已持有有效资格的课程自动免考且不可取消", async ({ page }) => {
  const stageID = "stage-auto-exemption"
  const unitID = "unit-auto-exemption"
  const includedUnits = [
    { unit_id: "unit-foundation", name: "CFtP Foundation Course", amount: 220000 },
    { unit_id: "unit-finance", name: "L1A Finance", amount: 75000 },
    { unit_id: "unit-fintech", name: "L1B Fintech", amount: 75000 },
  ]
  const qualificationID = "qualification-auto-exemption"
  const automaticExemptionBundle = {
    ...bundle,
    stages: [{
      stage_id: stageID,
      name: "Automatic Exemption Stage",
      sort_order: 1,
      units: [
        { unit_id: unitID, name: "CFtA Course" },
        ...includedUnits.map(({ unit_id, name }) => ({ unit_id, name })),
      ],
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
          pricing_detail_json: JSON.stringify({
            units: [
              { unit_id: unitID, access: { amount: 10000, currency: "USD" } },
              ...includedUnits.map(({ unit_id, amount }) => ({
                unit_id,
                access: { amount, currency: "USD" },
              })),
            ],
            memberships: [],
          }),
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

  await expect(page.locator(`[data-testid="checkout-exemption-apply"][data-unit-id="${unitID}"]`)).toHaveCount(0)
  await expect(page.locator(`[data-testid="checkout-exemption-waive"][data-unit-id="${unitID}"]`)).toHaveCount(0)
  await expect(page.getByText("系统自动免考", { exact: true })).toBeVisible()
  await expect(page.getByTestId("checkout-included-items")).toBeVisible()
  await expect(page.getByTestId("checkout-included-item")).toHaveCount(3)
  for (const unit of includedUnits) {
    const item = page.locator(`[data-testid="checkout-included-item"][data-item-id="${unit.unit_id}"]`)
    await expect(item).toContainText(unit.name)
    await expect(item).toContainText((unit.amount / 100).toLocaleString("en-US", { minimumFractionDigits: 2 }))
  }
  await expect(page.locator(".checkout-total")).toContainText("基础总额")
  await expect(page.locator(".checkout-total")).toContainText("3,700.00")

  await page.getByTestId("checkout-selection-next").click()
  await waitForCheckoutProfile(page)
  await page.getByTestId("checkout-agreement").check()
  await page.getByTestId("checkout-next").click()
  await page.getByTestId("checkout-confirm-pay").click()

  await expect(page).toHaveURL(/\/checkout\/success\/order-auto-exemption/)
  expect(purchaseBody).toEqual({
    payment_mode: "FULL_PIPELINE",
    selected_exemptions_json: JSON.stringify({
      [pipelineID]: {
        stages: [{
          stage_cc_ulid: stageID,
          exempted_unit_cc_ulids: [unitID],
          waived_unit_cc_ulids: [],
        }],
      },
    }),
  })
})

test("系统资格、自动免考和按原价购买以管线维度提交完整决定", async ({ page }) => {
  const systemStageID = "stage-system-decision"
  const visibleStageID = "stage-visible-decisions"
  const systemUnitID = "unit-system-decision"
  const grantedUnitID = "unit-granted-decision"
  const waivedUnitID = "unit-waived-decision"
  const systemQualificationID = "qualification-system-decision"
  const grantedQualificationID = "qualification-granted-decision"
  const waivedQualificationID = "qualification-waived-decision"
  let pricingSelections: unknown

  const decisionBundle = {
    ...bundle,
    stages: [
      {
        stage_id: systemStageID,
        name: "System Stage",
        units: [{ unit_id: systemUnitID, name: "System Course" }],
      },
      {
        stage_id: visibleStageID,
        name: "Visible Stage",
        sort_order: 1,
        units: [
          { unit_id: grantedUnitID, name: "Granted Course" },
          { unit_id: waivedUnitID, name: "Full Price Course" },
        ],
      },
    ],
    purchase_state: {
      ...bundle.purchase_state,
      exemption_options: {
        stages: [
          {
            stage_id: systemStageID,
            stage_name: "System Stage",
            units: [{
              unit_id: systemUnitID,
              unit_name: "System Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: systemQualificationID, name: "System Qualification" }],
            }],
          },
          {
            stage_id: visibleStageID,
            stage_name: "Visible Stage",
            sort_order: 1,
            units: [
              {
                unit_id: grantedUnitID,
                unit_name: "Granted Course",
                allow_exemption: true,
                qualified: true,
                exemption_quals: [{
                  qual_id: grantedQualificationID,
                  name: "Granted Qualification",
                  eligible: true,
                  credential_status: "CREDENTIAL_STATUS_ACTIVE",
                }],
              },
              {
                unit_id: waivedUnitID,
                unit_name: "Full Price Course",
                allow_exemption: true,
                qualified: false,
                exemption_quals: [{ qual_id: waivedQualificationID, name: "Waived Qualification" }],
              },
            ],
          },
        ],
      },
    },
  }
  const expectedSelections = {
    [pipelineID]: {
      stages: [
        {
          stage_cc_ulid: systemStageID,
          exempted_unit_cc_ulids: [],
          waived_unit_cc_ulids: [systemUnitID],
        },
        {
          stage_cc_ulid: visibleStageID,
          exempted_unit_cc_ulids: [grantedUnitID],
          waived_unit_cc_ulids: [waivedUnitID],
        },
      ],
    },
  }

  await installCandidateApiMocks(page, ({ pathname, url, method }) => {
    if (pathname === `/api/mall/bundles/${bundleID}` && method === "GET") return { data: decisionBundle }
    if (pathname === `/api/mall/bundles/${bundleID}/pricing-detail`) {
      pricingSelections = JSON.parse(url.searchParams.get("selected_exemptions_json") || "{}")
      return {
        data: {
          can_checkout: JSON.stringify(pricingSelections) === JSON.stringify(expectedSelections),
          checkout_blocker_reason: "EXEMPTIONS_UNCONFIRMED_WAIVER",
        },
      }
    }
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            {
              cred_def_ulid: systemQualificationID,
              name: "System Qualification",
              respath: "/gcreds/core/system/internal",
            },
            {
              cred_def_ulid: grantedQualificationID,
              name: "Granted Qualification",
              respath: "/gcreds/core/granted",
            },
            {
              cred_def_ulid: waivedQualificationID,
              name: "Waived Qualification",
              respath: "/gcreds/core/waived",
            },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/credentials/application-orders/latest") {
      return {
        data: {
          found: true,
          application_order_ulid: "existing-decision-order",
          order_status: "UPLOAD_READY",
          items: [{ qual_id: grantedQualificationID, item_status: "APPROVED" }],
        },
      }
    }
    if (pathname === "/api/user/me") return { data: candidateUser }
    return undefined
  })

  await page.goto(`/checkout/${bundleID}`, { waitUntil: "domcontentloaded" })
  await expect(page.getByText("System Course", { exact: true })).toHaveCount(0)
  await expect(page.getByText("系统自动免考", { exact: true }).first()).toBeVisible()
  await page.locator(`[data-testid="checkout-exemption-waive"][data-unit-id="${waivedUnitID}"]`).click()
  await page.getByTestId("checkout-selection-next").click()

  await expect(page.getByTestId("checkout-step-registration")).toBeVisible()
  expect(pricingSelections).toEqual(expectedSelections)
})

test("资格审核轮询刷新后继续隐藏系统管理资格", async ({ page }) => {
  const pollingBundleID = "bundle-system-filter-polling"
  const visibleQualificationID = "qualification-review-polling"
  const systemQualificationID = "qualification-system-polling"
  let applicationStatus = "APPLICATION_STATUS_PENDING"
  let bundleRequests = 0
  const pollingBundle = {
    ...bundle,
    bundle_id: pollingBundleID,
    stages: [{
      stage_id: "stage-review-polling",
      name: "Review Polling Stage",
      sort_order: 1,
      units: [
        { unit_id: "unit-review-polling", name: "Reviewed Course" },
        { unit_id: "unit-system-polling", name: "System Only Course" },
      ],
    }],
    purchase_state: {
      ...bundle.purchase_state,
      exemption_options: {
        stages: [{
          stage_id: "stage-review-polling",
          stage_name: "Review Polling Stage",
          units: [
            {
              unit_id: "unit-review-polling",
              unit_name: "Reviewed Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: visibleQualificationID, name: "Reviewed Qualification" }],
            },
            {
              unit_id: "unit-system-polling",
              unit_name: "System Only Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: systemQualificationID, name: "System Managed Qualification" }],
            },
          ],
        }],
      },
    },
  }

  await page.clock.install()
  await installCandidateApiMocks(page, ({ pathname, method }) => {
    if (pathname === `/api/mall/bundles/${pollingBundleID}` && method === "GET") {
      bundleRequests += 1
      return { data: pollingBundle }
    }
    if (pathname === `/api/mall/bundles/${pollingBundleID}/pricing-detail`) {
      return { data: { units: [], memberships: [] } }
    }
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            { cred_def_ulid: visibleQualificationID, name: "Reviewed Qualification", respath: "/gcreds/core/reviewed" },
            { cred_def_ulid: systemQualificationID, name: "System Managed Qualification", respath: "/gcreds/core/system/internal" },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") {
      return { data: { applications: [{ app_ulid: "application-review-polling", status: applicationStatus }] } }
    }
    return undefined
  })

  await page.goto(`/checkout/${pollingBundleID}`, { waitUntil: "domcontentloaded" })
  await expect(page.getByText("Reviewed Course", { exact: true })).toBeVisible()
  await expect(page.getByText("System Only Course", { exact: true })).toHaveCount(0)

  applicationStatus = "APPLICATION_STATUS_APPROVED"
  await page.clock.fastForward(30_000)
  await expect.poll(() => bundleRequests).toBeGreaterThan(1)
  await expect(page.getByText("System Only Course", { exact: true })).toHaveCount(0)
})

test("结账资格申请提供官方模板预览与下载", async ({ page }) => {
  const stageID = "stage-template-application"
  const unitID = "unit-template-application"
  const qualificationID = "qualification-template-application"
  const systemUnitID = "unit-system-qualification"
  const systemQualificationID = "qualification-system-only"
  let uploadRequest: any
  let submitRequest: any
  let pricingSelections: any
  const checkoutBundle = {
    ...bundle,
    stages: [{
      stage_id: stageID,
      name: "Template Application Stage",
      sort_order: 1,
      units: [
        { unit_id: unitID, name: "Template Application Course" },
        { unit_id: systemUnitID, name: "System Only Course" },
      ],
    }],
    purchase_state: {
      ...bundle.purchase_state,
      exemption_options: {
        stages: [{
          index: 0,
          stage_id: stageID,
          stage_name: "Template Application Stage",
          units: [
            {
              unit_id: unitID,
              unit_name: "Template Application Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: qualificationID, name: "Work Experience Qualification" }],
            },
            {
              unit_id: systemUnitID,
              unit_name: "System Only Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: systemQualificationID, name: "System Managed Qualification" }],
            },
          ],
        }],
      },
    },
  }

  await page.route("https://uploads.example/**", async (route) => {
    await route.fulfill({ status: 200, body: "" })
  })
  await installCandidateApiMocks(page, ({ pathname, url, method, body }) => {
    if (pathname === `/api/mall/bundles/${bundleID}` && method === "GET") return { data: checkoutBundle }
    if (pathname === `/api/mall/bundles/${bundleID}/pricing-detail`) {
      pricingSelections = JSON.parse(url.searchParams.get("selected_exemptions_json") || "{}")
      return { data: { units: [], memberships: [] } }
    }
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            {
              cred_def_ulid: qualificationID,
              name: "Work Experience Qualification",
              description: "Upload signed work experience evidence.",
              respath: "/gcreds/core/work-experience",
              file_constraints: [{
                name: "Employment Certificate",
                display_name: "雇佣证明",
                type: 2,
                is_required: true,
              }],
              attachments: [{
                attachment_id: "checkout-template-1",
                name: "工作经验证明模板",
                file_name: "work-experience-template.docx",
                file_size: 4096,
                download_url: "https://downloads.example/work-experience-template.docx",
              }],
            },
            {
              cred_def_ulid: systemQualificationID,
              name: "System Managed Qualification",
              respath: "/gcc/credential/system/internal-certification",
              file_constraints: [],
            },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/credentials/application-orders/latest") {
      return {
        data: {
          found: true,
          application_order_ulid: "active-template-order",
          order_status: "UPLOAD_READY",
          items: [{ qual_id: qualificationID, item_status: "PENDING" }],
        },
      }
    }
    if (pathname === "/api/credentials/upload-url" && method === "POST") {
      uploadRequest = body
      return {
        data: {
          upload_url: "https://uploads.example/employment-certificate.pdf",
          file_key: "credentials/employment-certificate.pdf",
          signed_headers: {},
        },
      }
    }
    if (pathname === "/api/credentials/submit" && method === "POST") {
      submitRequest = body
      return { data: { app_ulid: "application-1", status: "PENDING" } }
    }
    return undefined
  })

  await page.goto(`/checkout/${bundleID}`, { waitUntil: "domcontentloaded" })
  await expect(page.getByText("System Only Course", { exact: true })).toHaveCount(0)
  await expect.poll(() => pricingSelections).toEqual({
    [pipelineID]: {
      stages: [{
        stage_cc_ulid: stageID,
        exempted_unit_cc_ulids: [],
        waived_unit_cc_ulids: [systemUnitID],
      }],
    },
  })
  await expect(page.locator(`[data-testid="checkout-exemption-apply"][data-unit-id="${unitID}"]`)).toHaveCount(0)

  await expect(page.getByText("模板与参考文件", { exact: true })).toBeVisible()
  await expect(page.getByRole("link", { name: "预览模板" })).toHaveAttribute(
    "href",
    "https://downloads.example/work-experience-template.docx",
  )
  await expect(page.getByRole("link", { name: "下载模板" })).toHaveAttribute("download", "work-experience-template.docx")
  await expect(page.getByText("雇佣证明", { exact: true })).toBeVisible()
  await expect(page.locator(`[id="qualification-file-${unitID}-Employment Certificate"]`)).toHaveCount(1)

  await page.locator(`[id="qualification-file-${unitID}-Employment Certificate"]`).setInputFiles({
    name: "employment-certificate.pdf",
    mimeType: "application/pdf",
    buffer: Buffer.from("pdf-content"),
  })
  await expect(page.getByText("employment-certificate.pdf 上传成功", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "提交申请", exact: true }).click()

  expect(uploadRequest.file_usage).toBe("Employment Certificate")
  expect(submitRequest.files).toHaveLength(1)
  expect(submitRequest.files[0].file_usage).toBe("Employment Certificate")
})

test("资格申请页恢复活跃审核订单并阻止追加其他资格", async ({ page }) => {
  const includedQualificationID = "qualification-active-order-included"
  const excludedQualificationID = "qualification-active-order-excluded"

  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            {
              cred_def_ulid: includedQualificationID,
              name: "Included Qualification",
              category: "Exemption",
              respath: "/gcreds/core/included",
              file_constraints: [],
            },
            {
              cred_def_ulid: excludedQualificationID,
              name: "Excluded Qualification",
              category: "Exemption",
              respath: "/gcreds/core/excluded",
              file_constraints: [],
            },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/credentials/application-orders/latest") {
      return {
        data: {
          found: true,
          application_order_ulid: "active-qualification-order",
          order_status: "UPLOAD_READY",
          items: [{ qual_id: includedQualificationID, item_status: "PENDING" }],
        },
      }
    }
    return undefined
  })

  await page.goto("/credentials", { waitUntil: "domcontentloaded" })

  await expect(page.getByText("资格审核订单", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "上传证明材料", exact: true })).toBeEnabled()
  await expect(page.getByRole("button", { name: "本轮订单未包含，暂不可追加", exact: true })).toBeDisabled()
  await page.getByRole("button", { name: "上传证明材料", exact: true }).click()
  await expect(page.locator(".credentials-apply-dialog").getByText("Included Qualification", { exact: true })).toBeVisible()
})

test("资格审核订单结束后不能再次创建资格申请", async ({ page }) => {
  const qualificationID = "qualification-resolved-order"

  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [{
            cred_def_ulid: qualificationID,
            name: "Rejected Qualification",
            category: "Exemption",
            respath: "/gcreds/core/rejected",
            file_constraints: [],
          }],
        },
      }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/credentials/application-orders/latest") {
      return {
        data: {
          found: true,
          application_order_ulid: "resolved-qualification-order",
          order_status: "RESOLVED",
          items: [{ qual_id: qualificationID, item_status: "REJECTED" }],
        },
      }
    }
    return undefined
  })

  await page.goto("/credentials", { waitUntil: "domcontentloaded" })

  await expect(page.getByText("资格审核订单已结束，不能再新增资格申请。", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "被拒绝", exact: true })).toBeDisabled()
})

test("免考选择完成后才合并创建所选资格订单", async ({ page }) => {
  const selectionBundleID = "bundle-explicit-exemption-decisions"
  const stageID = "stage-explicit-exemption-decisions"
  const applyUnitID = "unit-apply-exemption"
  const waiveUnitID = "unit-waive-exemption"
  const applyQualificationID = "qualification-apply-exemption"
  const waiveQualificationID = "qualification-waive-exemption"
  let applicationOrderBody: any

  const selectionBundle = {
    ...bundle,
    bundle_id: selectionBundleID,
    stages: [{
      stage_id: stageID,
      name: "Explicit Exemption Decisions",
      sort_order: 1,
      units: [
        { unit_id: applyUnitID, name: "Apply Exemption Course" },
        { unit_id: waiveUnitID, name: "Full Price Course" },
      ],
    }],
    purchase_state: {
      ...bundle.purchase_state,
      exemption_options: {
        stages: [{
          stage_id: stageID,
          stage_name: "Explicit Exemption Decisions",
          units: [
            {
              unit_id: applyUnitID,
              unit_name: "Apply Exemption Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: applyQualificationID, name: "Apply Qualification" }],
            },
            {
              unit_id: waiveUnitID,
              unit_name: "Full Price Course",
              allow_exemption: true,
              qualified: false,
              exemption_quals: [{ qual_id: waiveQualificationID, name: "Waive Qualification" }],
            },
          ],
        }],
      },
    },
  }

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === `/api/mall/bundles/${selectionBundleID}` && method === "GET") return { data: selectionBundle }
    if (pathname === `/api/mall/bundles/${selectionBundleID}/pricing-detail`) return { data: { can_checkout: true } }
    if (pathname === "/api/credentials/definitions") {
      return {
        data: {
          definitions: [
            { cred_def_ulid: applyQualificationID, name: "Apply Qualification", respath: "/gcreds/core/apply" },
            { cred_def_ulid: waiveQualificationID, name: "Waive Qualification", respath: "/gcreds/core/waive" },
          ],
        },
      }
    }
    if (pathname === "/api/credentials/applications") return { data: { applications: [] } }
    if (pathname === "/api/credentials/upload-permission") return { data: { granted: false } }
    if (pathname === "/api/credentials/application-orders" && method === "POST") {
      applicationOrderBody = body
      return {
        data: {
          application_order_ulid: "application-order-explicit-selection",
          order_status: "WAIT_REVIEW_FEE_PAYMENT",
        },
      }
    }
    if (pathname === "/api/mall/payments/preview" && method === "POST") {
      return { data: bundle.purchase_state.payment_preview }
    }
    return undefined
  })

  await page.goto(`/checkout/${selectionBundleID}`, { waitUntil: "domcontentloaded" })
  await page.locator(`[data-testid="checkout-exemption-apply"][data-unit-id="${applyUnitID}"]`).click()
  await page.locator(`[data-testid="checkout-exemption-waive"][data-unit-id="${waiveUnitID}"]`).click()

  expect(applicationOrderBody).toBeUndefined()
  await expect(page.getByText("这里仅展示申请要求和官方模板。请先完成所有免考选择并支付资格审核费，付款成功后才能上传证明材料。", { exact: true })).toBeVisible()

  await page.getByTestId("checkout-apply-selected-exemptions").click()

  await expect.poll(() => applicationOrderBody).toEqual({
    pipeline_cc_ulid: pipelineID,
    bundle_ulid: selectionBundleID,
    qual_ulids: [applyQualificationID],
  })
  await expect(page.getByTestId("checkout-step-payment")).toBeVisible()
})

test("分阶段购买先完成免考声明再创建阶段订单", async ({ page }) => {
  const stageID = "stage-by-stage-exemption"
  const stageInstanceID = "stage-instance-by-stage-exemption"
  const exemptedUnitID = "unit-stage-exempted"
  const waivedUnitID = "unit-stage-waived"
  const exemptedQualificationID = "qualification-stage-exempted"
  const waivedQualificationID = "qualification-stage-waived"
  let stageOrderBody: any

  const runtime = {
    instance: { pipeline_ulid: pipelineInstanceID },
    config: {
      pipeline_cc_ulid: pipelineID,
      name: "By-stage Exemption Certification",
      stages: [{
        stage_id: stageID,
        name: "By-stage Exemption Stage",
        sort_order: 1,
        runtime_status: "STAGE_STATUS_WAIT_CANDIDATE",
        units: [
          {
            unit_id: exemptedUnitID,
            name: "Eligible Exemption Course",
            allow_exemption: true,
            exemption_quals: [{
              qual_id: exemptedQualificationID,
              name: "Eligible Stage Qualification",
              eligible: true,
              credential_status: "CREDENTIAL_STATUS_ACTIVE",
            }],
          },
          {
            unit_id: waivedUnitID,
            name: "Waived Exemption Course",
            allow_exemption: true,
            exemption_quals: [{
              qual_id: waivedQualificationID,
              name: "Unavailable Stage Qualification",
            }],
          },
        ],
      }],
    },
    next_step: {
      action: "wait_candidate",
      stage_id: stageInstanceID,
      status: "STAGE_STATUS_WAIT_CANDIDATE",
    },
    pipeline_status: "PIPELINE_STATUS_LEARNING",
    current_stage_name: "By-stage Exemption Stage",
    current_stage_status: "STAGE_STATUS_WAIT_CANDIDATE",
  }

  await installCandidateApiMocks(page, ({ pathname, method, body }) => {
    if (pathname === `/api/mall/pipelines/${pipelineID}/runtime`) return { data: runtime }
    if (pathname === "/api/credentials/qualifications") {
      return {
        data: {
          qualifications: [{
            qual_id: exemptedQualificationID,
            eligible: true,
            credential_status: "CREDENTIAL_STATUS_ACTIVE",
          }],
        },
      }
    }
    if (pathname === `/api/mall/pipelines/${pipelineID}/stages/${stageID}/purchase` && method === "POST") {
      stageOrderBody = body
      return {
        data: {
          stage_order_ulid: "stage-order-explicit-selection",
          order_status: "COMPLETED",
        },
      }
    }
    return undefined
  })

  await page.goto(`/certifications/${pipelineID}`, { waitUntil: "domcontentloaded" })
  await page.getByRole("button", { name: "解锁当前阶段", exact: true }).click()

  await expect(page.getByText("选择当前阶段免考", { exact: true })).toBeVisible()
  expect(stageOrderBody).toBeUndefined()

  await page.getByRole("button", { name: "放弃免考，原价购买", exact: true }).last().click()
  await page.getByRole("button", { name: "确认并继续", exact: true }).click()

  await expect.poll(() => stageOrderBody).toEqual({
    pipeline_ulid: pipelineInstanceID,
    stage_ulid: stageInstanceID,
    selected_exemptions_json: JSON.stringify({
      stages: [{
        stage_cc_ulid: stageID,
        exempted_unit_cc_ulids: [exemptedUnitID],
        waived_unit_cc_ulids: [waivedUnitID],
      }],
    }),
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
    selected_exemptions_json: JSON.stringify({ [pipelineID]: { stages: [] } }),
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
  await expect(page.getByTestId("checkout-cancel-and-edit")).toHaveCount(0)

  await page.getByTestId("checkout-progress-step-1").click()
  let editDialog = page.getByRole("dialog", { name: "返回修改报名信息？" })
  await expect(editDialog.getByText("待支付订单已经创建，价格和报名选择已锁定，不能直接返回修改。如需调整，请先取消当前订单。", { exact: true })).toBeVisible()
  await editDialog.getByRole("button", { name: "继续支付" }).click()
  await expect(editDialog).toHaveCount(0)

  await page.getByTestId("checkout-progress-step-3").click()
  editDialog = page.getByRole("dialog", { name: "返回修改报名信息？" })
  await expect(editDialog).toBeVisible()
  await editDialog.getByRole("button", { name: "继续支付" }).click()

  await page.getByTestId("checkout-progress-step-2").click()
  editDialog = page.getByRole("dialog", { name: "返回修改报名信息？" })
  await editDialog.getByTestId("checkout-cancel-and-edit").click()

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
