import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const webhookSummary = {
  id: 101,
  msg_fp: "webhook-fingerprint-1",
  event_type: "result_created",
  event_timestamp: "2026-08-11T00:00:00Z",
  exam_ulid: "exam-1",
  confirmation_number: "CONF-001",
  processed_status: "PROCESSED",
  created_at: "2026-08-11T00:01:00Z",
}

async function installWebhookReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/audit/webhooks") {
      return {
        data: {
          webhook_messages: [webhookSummary],
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/audit/webhooks/detail") {
      return {
        data: {
          ...webhookSummary,
          payload_json: JSON.stringify({ result: "pass" }),
          processed_at: "2026-08-11T00:02:00Z",
        },
      }
    }
    return undefined
  })
}

test("webhook list renders the returned read-only processing result", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installWebhookReadMocks(page, requests)

  await page.goto("/audit/webhooks")

  await expect(page.getByText("考试结果回调", { exact: true })).toBeVisible()
  await expect(page.getByText("确认编号：CONF-001", { exact: true })).toBeVisible()
  await expect(page.getByText("已处理", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/audit/webhooks")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("webhook detail displays payload metadata without replaying it", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installWebhookReadMocks(page, requests)
  await page.goto("/audit/webhooks")

  await page.getByRole("button", { name: "查看详情" }).click()

  const detailDialog = page.getByRole("dialog", { name: "Webhook 详情" })
  await expect(detailDialog).toBeVisible()
  await expect(detailDialog.getByText("CONF-001", { exact: true }).first()).toBeVisible()
  await expect(detailDialog.getByText("exam-1", { exact: true }).first()).toBeVisible()
  expect(requests).toContain("GET /api/audit/webhooks/detail")
  expect(requests.some((request) => request.includes("/reprocess"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
