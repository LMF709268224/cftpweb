import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const pipelineSummary = {
  pipeline_ulid: "pipeline-1",
  pipeline_cc_ulid: "pipeline-config-1",
  candidate_ulid: "candidate-1",
  current_stage_ulid: "stage-1",
  status: "1",
  started_at: "2026-08-11T00:00:00Z",
}

const transitionSummary = {
  transition_ulid: "transition-1",
  entity_type: "PIPELINE",
  entity_ulid: "pipeline-1",
  from_status: "1",
  to_status: "2",
  event_type: "STATUS_CHANGED",
  created_at: "2026-08-11T01:00:00Z",
}

async function installProgReadMocks(page: Page, requests: string[], progQueries: string[] = []) {
  return installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/pipelines") {
      return {
        data: {
          pipelines: [
            { pipeline_ulid: "pipeline-config-1", name: "Regression Pipeline", version: 1 },
            { pipeline_ulid: "pipeline-config-2", name: "CFtA Certification", version: 2 },
          ],
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/prog/pipelines") {
      progQueries.push(url.search)
      return { data: { pipelines: [pipelineSummary], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/prog/pipelines/pipeline-1") {
      return {
        data: {
          pipeline: pipelineSummary,
          stages: [{
            stage: {
              stage_ulid: "stage-1",
              name: "Regression Stage",
              status: "1",
            },
            course_units: [{ course_unit_ulid: "course-unit-1", name: "Regression Course", status: "2" }],
          }],
        },
      }
    }
    if (method === "GET" && pathname === "/api/prog/pipelines/pipeline-1/logs") {
      return { data: { logs: [transitionSummary], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/prog/pipelines/logs/transition-1") {
      return { data: { summary: transitionSummary, reason_message: "Read-only regression transition" } }
    }
    return undefined
  })
}

test("Prog list renders the read-only pipeline summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installProgReadMocks(page, requests)

  await page.goto("/prog")

  await expect(page.getByText("Regression Pipeline", { exact: true })).toBeVisible()
  await expect(page.getByText("candidate-1", { exact: true }).first()).toBeVisible()
  const pipelineRow = page.getByRole("button", { name: "查看详情" }).locator("..")
  await expect(pipelineRow.getByText("运行中", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/prog/pipelines")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("Prog list filters certification instances by certificate type", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  const progQueries: string[] = []
  await installProgReadMocks(page, requests, progQueries)

  await page.goto("/prog")
  await page.getByLabel("证书类型").selectOption("pipeline-config-2")

  await expect.poll(() => progQueries.some((query) => new URLSearchParams(query).get("pipeline_cc_ulid") === "pipeline-config-2")).toBe(true)
})

test("Prog detail reads stages and transition detail without an operator action", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installProgReadMocks(page, requests)
  await page.goto("/prog")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "认证详情" })
  await expect(detailDialog).toBeVisible()
  await detailDialog.getByRole("button", { name: /^阶段 1$/ }).click()
  await expect(detailDialog.getByText("Regression Stage", { exact: true }).first()).toBeVisible()
  await detailDialog.getByRole("button", { name: /^课程单元 1$/ }).click()
  await expect(detailDialog.getByText("course-unit-1", { exact: true }).first()).toBeVisible()
  await detailDialog.getByRole("button", { name: /^状态日志 1$/ }).click()
  await expect(detailDialog.getByText("transition-1", { exact: true })).toBeVisible()
  await detailDialog.getByRole("button", { name: "查看详情" }).click()
  await expect(page.getByRole("dialog", { name: "日志详情" })).toBeVisible()

  expect(requests).toContain("GET /api/prog/pipelines/pipeline-1")
  expect(requests).toContain("GET /api/prog/pipelines/pipeline-1/logs")
  expect(requests).toContain("GET /api/prog/pipelines/logs/transition-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
