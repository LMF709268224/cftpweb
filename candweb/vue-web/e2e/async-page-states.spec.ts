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
  {
    name: "资格申请",
    path: "/credentials",
    endpoint: "/api/credentials/definitions",
    errorTitle: "资格申请加载失败",
    successTitle: "暂无申请",
    successData: { definitions: [] },
  },
  {
    name: "会员",
    path: "/membership",
    endpoint: "/api/membership/plans",
    errorTitle: "会员信息加载失败",
    successTitle: "暂无有效会员",
    successData: { memberships: [] },
  },
  {
    name: "课程时间线",
    path: "/certifications/pipeline-regression/timeline",
    endpoint: "/api/mall/pipelines/pipeline-regression/timeline",
    errorTitle: "课程时间线加载失败",
    successTitle: "暂无状态流转记录",
    successData: { logs: [] },
  },
  {
    name: "认证详情",
    path: "/certifications/pipeline-regression",
    endpoint: "/api/mall/pipelines/pipeline-regression/runtime",
    errorTitle: "认证详情加载失败",
    successTitle: "重试后认证详情",
    successData: {
      config: {
        pipeline_cc_ulid: "pipeline-regression",
        name: "重试后认证详情",
        stages: [],
      },
    },
  },
  {
    name: "课程学习",
    path: "/certifications/pipeline-regression/learn/course-regression",
    endpoint: "/api/pipeline/courses/course-regression/complete",
    errorTitle: "课程内容加载失败",
    successTitle: "重试后课程",
    successData: {
      complete_course: {
        course: {
          course_id: "course-regression",
          title: "重试后课程",
        },
        chapters: [],
        materials: [],
      },
    },
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

test("首页统计全部加载失败后可以重试", async ({ page }) => {
  const responses: Record<string, unknown> = {
    "/api/pipeline": { list: [] },
    "/api/certificates": { certificates: [] },
    "/api/exams": { exams: [] },
    "/api/resource-packs": { packs: [] },
    "/api/orders": { orders: [] },
  }
  const requestCounts = new Map<string, number>()

  await seedAuthenticatedCandidate(page)
  await installCandidateApiMocks(page, ({ pathname }) => {
    if (!(pathname in responses)) return undefined

    const requestCount = (requestCounts.get(pathname) || 0) + 1
    requestCounts.set(pathname, requestCount)
    if (requestCount === 1) return { status: 503, message: "temporarily unavailable" }
    return { data: responses[pathname] }
  })
  await page.goto("/dashboard", { waitUntil: "domcontentloaded" })

  await expect(page.getByRole("heading", { name: "首页统计加载失败" })).toBeVisible()
  await page.getByRole("button", { name: "重新加载" }).click()
  await expect(page.getByRole("heading", { name: "已购买认证" })).toBeVisible()

  for (const endpoint of Object.keys(responses)) {
    expect(requestCounts.get(endpoint)).toBe(2)
  }
})
