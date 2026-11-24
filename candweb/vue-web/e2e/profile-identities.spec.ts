import { expect, test, type Page } from "@playwright/test"
import { candidateUser, installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

async function installProfile(page: Page, status: Record<string, unknown>) {
  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (pathname === "/api/user/me") return { data: { ...candidateUser, account_status: status } }
    if (pathname === "/api/pipeline") return { data: { list: [] } }
    if (pathname === "/api/resource-packs") return { data: { packs: [] } }
    if (pathname === "/api/mall/bundles") return { data: { bundles: [] } }
  })
}

const certification = (codes: string[]) => ({
  available: true, is_candidate: codes.length > 0, purchase_count: codes.length,
  programs: codes.map((code, index) => ({
    pipeline_ulid: `instance-${index}`, pipeline_config_ulid: `config-${index}`,
    certification_code: code, name: `${code} Certification Pathway`, status: "PIPELINE_STATUS_RUNNING",
  })),
})
const qualification = (codes: string[]) => ({
  available: true, has_qualification: codes.length > 0, credential_count: codes.length,
  charterholder_programs: codes,
})

for (const scenario of [
  { candidates: ["CFtA"], holders: [], zh: "CFtA 考生", en: "CFtA Candidate" },
  { candidates: ["CFtP", "CFtA", "CFtP"], holders: [], zh: "CFtP 考生", en: "CFtP Candidate" },
  { candidates: ["CFtA", "CFtP"], holders: ["CFtA"], zh: "CFtA 持证人", en: "CFtA Charterholder" },
  { candidates: ["CFtP", "CFtA"], holders: ["CFtP", "CFtA"], zh: "CFtP 持证人", en: "CFtP Charterholder" },
]) {
  test(`认证身份按优先级只显示 ${scenario.en}`, async ({ page }) => {
    await installProfile(page, {
      certification: certification(scenario.candidates),
      qualification: qualification(scenario.holders),
    })
    await page.goto("/dashboard")
    const badges = page.getByTestId("dashboard-account-status")
    await expect(badges.getByRole("listitem")).toHaveText([scenario.zh])
    await page.getByRole("button", { name: "中文 / EN" }).click()
    await expect(badges.getByRole("listitem")).toHaveText([scenario.en])
  })
}

test("会员身份可以独立显示，特许会员不会被当作 CFtP 持证人", async ({ page }) => {
  await installProfile(page, {
    membership: { available: true, is_member: true, display_code: "charterholder_member", expires_at: "2099-01-01T00:00:00Z" },
    certification: certification([]), qualification: qualification([]),
  })
  await page.goto("/dashboard")
  await expect(page.getByTestId("dashboard-account-status").getByRole("listitem")).toHaveText(["特许会员"])
})

test("会员在首页停留期间到期后隐藏，认证身份保留", async ({ page }) => {
  await page.clock.install({ time: new Date("2026-11-24T00:00:00Z") })
  await installProfile(page, {
    membership: { available: true, is_member: true, display_code: "affiliate_member", expires_at: "2026-11-24T00:00:30Z" },
    qualification: qualification(["CFtP"]),
  })
  await page.goto("/dashboard")
  const badges = page.getByTestId("dashboard-account-status")
  await expect(badges.getByRole("listitem")).toHaveText(["CFtP 持证人", "附属会员"])
  await page.getByRole("button", { name: "中文 / EN" }).click()
  await expect(badges.getByRole("listitem")).toHaveText(["CFtP Charterholder", "Affiliate Member"])
  await page.clock.fastForward(61_000)
  await expect(badges.getByRole("listitem")).toHaveText(["CFtP Charterholder"])
})

test("已过期会员、取消或完成的管线以及 Fellow 不产生身份标签", async ({ page }) => {
  const programs = certification(["CFtA", "CFtP"]).programs
  programs[0]!.status = "PIPELINE_STATUS_CANCELLED"
  programs[1]!.status = "PIPELINE_STATUS_COMPLETED"
  await installProfile(page, {
    membership: { available: true, is_member: true, display_code: "affiliate_member", expires_at: "2020-01-01T00:00:00Z" },
    certification: { ...certification([]), is_candidate: true, programs },
    qualification: { ...qualification([]), has_qualification: true, credential_count: 4, charterholder_programs: ["Industry Fellow"] },
  })
  await page.goto("/dashboard")
  await expect(page.getByRole("heading", { name: "欢迎回来，Regression" })).toBeVisible()
  await expect(page.getByTestId("dashboard-account-status")).toHaveCount(0)
})

test("身份只按服务确认的数据展示，不从名称或不可用的数据推断", async ({ page }) => {
  await installProfile(page, {
    certification: { ...certification([]), is_candidate: true, programs: [{ name: "CFtP", status: "PIPELINE_STATUS_RUNNING" }] },
    qualification: { ...qualification(["CFtP"]), available: false },
    membership: { available: false, is_member: true, display_code: "affiliate_member" },
  })
  await page.goto("/dashboard")
  await expect(page.getByRole("heading", { name: "欢迎回来，Regression" })).toBeVisible()
  await expect(page.getByTestId("dashboard-account-status")).toHaveCount(0)
})
