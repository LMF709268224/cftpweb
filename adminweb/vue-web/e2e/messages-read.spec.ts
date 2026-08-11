import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const messageTemplate = {
  path: "system/regression",
  description: "Regression notice",
  version: 3,
  updated_at: "2026-08-11T00:00:00Z",
}

const sentMessage = {
  message_id: "message-1",
  user_id: "candidate-1",
  title: "Regression notification",
  content: "Read-only notification body",
  status: 2,
  template_path: "system/regression",
  created_at: "2026-08-11T01:00:00Z",
}

async function installMessageReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/user/list") {
      return { data: { users: [{ id: "candidate-1", name: "Regression Candidate" }] } }
    }
    if (method === "GET" && pathname === "/api/messages/templates") {
      return { data: { templates: [messageTemplate], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/messages/templates/detail") {
      return { data: { ...messageTemplate, title_tpl: "Regression {{.name}}", content_tpl: "Read-only notification", parameter_schema: '{"type":"object"}' } }
    }
    if (method === "GET" && pathname === "/api/messages/sent") {
      return { data: { messages: [sentMessage], total: 1, has_more: false, next_cursor: "" } }
    }
    return undefined
  })
}

test("message template detail is read without editing it", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMessageReadMocks(page, requests)
  await page.goto("/messages")

  await page.getByRole("button", { name: "模板管理" }).click()
  await expect(page.getByText("system/regression", { exact: true }).first()).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "模板详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("Regression {{.name}}", { exact: true })).toBeVisible()
  await expect(dialog.getByText("Read-only notification", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/messages/templates/detail")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("sent message history and detail reuse read-only list data", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMessageReadMocks(page, requests)
  await page.goto("/messages")

  await page.getByRole("button", { name: "发送记录" }).click()
  await expect(page.getByText("Regression notification", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "站内信详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("Read-only notification body", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/messages/sent")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
