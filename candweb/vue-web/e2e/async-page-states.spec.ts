import { expect, test } from "@playwright/test"
import { installCandidateApiMocks, seedAuthenticatedCandidate } from "./support/candidate"

type AsyncPageCase = {
  name: string
  path: string
  endpoint: string
  errorTitle: string
  successTitle: string
  successData: unknown
}

const asyncPageCases: AsyncPageCase[] = [
  {
    name: "商城",
    path: "/certifications",
    endpoint: "/api/mall/bundles",
    errorTitle: "商城信息加载失败",
    successTitle: "暂无可购买商品",
    successData: { bundles: [] },
  },
  {
    name: "我的认证",
    path: "/my-certifications",
    endpoint: "/api/pipeline",
    errorTitle: "认证信息加载失败",
    successTitle: "还没有购买认证",
    successData: { list: [] },
  },
  {
    name: "资源包",
    path: "/resource-packs",
    endpoint: "/api/resource-packs",
    errorTitle: "资源包加载失败",
    successTitle: "暂无可访问资源包",
    successData: { packs: [], next_page_token: "" },
  },
  {
    name: "资源包详情",
    path: "/resource-packs/pack-regression",
    endpoint: "/api/resource-packs/pack-regression/files",
    errorTitle: "资源文件加载失败",
    successTitle: "这个资源包暂无文件",
    successData: { files: [], next_page_token: "" },
  },
  {
    name: "考试结果",
    path: "/exams/result?examId=exam-regression",
    endpoint: "/api/exams/exam-regression/result",
    errorTitle: "考试结果加载失败",
    successTitle: "暂无评分明细",
    successData: { has_result: false },
  },
  {
    name: "测验",
    path: "/quizzes?attemptId=attempt-regression",
    endpoint: "/api/quizzes/attempts/attempt-regression/paper",
    errorTitle: "测验加载失败",
    successTitle: "重试测验",
    successData: { title: "重试测验", questions: [], remaining_seconds: 0 },
  },
]

for (const pageCase of asyncPageCases) {
  test(`${pageCase.name}加载失败后可以重试`, async ({ page }) => {
    let requestCount = 0

    await seedAuthenticatedCandidate(page)
    await installCandidateApiMocks(page, ({ pathname }) => {
      if (pathname !== pageCase.endpoint) return undefined

      requestCount += 1
      if (requestCount === 1) return { status: 503, message: "temporarily unavailable" }
      return { data: pageCase.successData }
    })
    await page.goto(pageCase.path, { waitUntil: "domcontentloaded" })

    await expect(page.getByRole("heading", { name: pageCase.errorTitle })).toBeVisible()
    await page.getByRole("button", { name: "重新加载" }).click()
    await expect(page.getByRole("heading", { name: pageCase.successTitle })).toBeVisible()
    expect(requestCount).toBe(2)
  })
}
