import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const pipelineSummary = {
  pipeline_ulid: "pipeline-1",
  pipeline_gpath: "/pipelines/regression",
  name: "Regression Pipeline",
  description: "Read-only pipeline summary",
  category_tips: "Automation",
  status: "Active",
  version: 3,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installPipelineReadMocks(page: Page, requests: string[], detailStatus = "Active") {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/pipelines") {
      return { data: { pipelines: [pipelineSummary], has_more: false, next_cursor: "", prev_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/pipelines/pipeline-1") {
      return {
        data: {
          ...pipelineSummary,
          status: detailStatus,
          description: "Read-only pipeline detail",
          version: 7,
          stages: [],
          prerequisite_quals: [],
          final_audit_quals: [],
          award_certs: [],
          forbidden_quals: [],
          conflict_pipeline_gpaths: [],
        },
      }
    }
    if (method === "GET" && pathname === "/api/lms/courses") {
      return { data: { courses: [], has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/credentials/definitions") {
      return { data: { definitions: [] } }
    }
    if (method === "GET" && pathname === "/api/pdf-templates") {
      return { data: { templates: [] } }
    }
    return undefined
  })
}

test("pipeline list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPipelineReadMocks(page, requests)

  await page.goto("/pipelines")

  await expect(page.getByText("Regression Pipeline", { exact: true })).toBeVisible()
  await expect(page.getByText("Read-only pipeline summary", { exact: true })).toBeVisible()
  await expect(page.getByText("Automation", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/pipelines")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("pipeline detail reads the selected configuration without editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPipelineReadMocks(page, requests)
  await page.goto("/pipelines")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  const dialog = page.getByRole("dialog", { name: "Regression Pipeline" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("pipeline-1", { exact: true })).toBeVisible()
  await expect(dialog.getByText("v7", { exact: true })).toBeVisible()
  await expect(dialog.getByRole("button", { name: /前置准入资格/ })).toBeVisible()
  await expect(dialog.getByRole("button", { name: /发证前终审材料/ })).toBeVisible()
  await expect(dialog.getByRole("button", { name: /最终颁发证书/ })).toBeVisible()
  await expect(dialog.getByRole("button", { name: /禁止报读资格/ })).toBeVisible()
  await expect(dialog.getByRole("button", { name: /互斥认证管线/ })).toBeVisible()

  expect(requests).toContain("GET /api/pipelines/pipeline-1")
  expect(requests.some((request) => request.includes("/publish") || request.includes("/deprecate") || request.startsWith("DELETE "))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("draft pipeline accepts a conflicting pipeline GPath outside the current page", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPipelineReadMocks(page, requests, "Draft")
  await page.goto("/pipelines")

  await page.getByRole("button", { name: "查看详情" }).first().click()
  const dialog = page.getByRole("dialog", { name: "Regression Pipeline" })
  await dialog.getByRole("button", { name: /互斥认证管线/ }).click()
  const input = dialog.getByPlaceholder("输入或选择互斥认证管线 GPath")
  await input.fill("/pipelines/not-in-current-page")
  await dialog.getByRole("button", { name: "新增", exact: true }).click()

  await expect(dialog.getByText("/pipelines/not-in-current-page", { exact: true }).first()).toBeVisible()
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
