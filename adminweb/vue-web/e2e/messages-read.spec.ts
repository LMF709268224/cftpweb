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

async function installMessageReadMocks(page: Page, requests: string[], sentStatuses: string[] = []) {
  return installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/user/list") {
      return { data: { users: [
        { id: "candidate-1", name: "CFtP Candidate" },
        { id: "candidate-2", name: "CFtA Candidate" },
        { id: "candidate-3", name: "Candidate Without Certificate" },
      ] } }
    }
    if (method === "GET" && pathname === "/api/credentials/definitions") {
      return { data: { definitions: [
        { cred_def_ulid: "cftp-level-1", name: "CFtP Level 1" },
        { cred_def_ulid: "cfta", name: "CFtA" },
      ] } }
    }
    if (method === "GET" && pathname === "/api/credentials") {
      return { data: { credentials: [
        { candidate_ulid: "candidate-1", cred_def_ulid: "cftp-level-1", source: "pdf_cert" },
        { candidate_ulid: "candidate-2", cred_def_ulid: "cfta", source: "pdf_cert" },
        { candidate_ulid: "candidate-3", cred_def_ulid: "cftp-level-1", source: "application" },
      ], has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/messages/templates") {
      return { data: { templates: [messageTemplate], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/messages/templates/detail") {
      return { data: { ...messageTemplate, title_tpl: "Regression {{.name}}", content_tpl: "Read-only notification", parameter_schema: '{"type":"object"}' } }
    }
    if (method === "GET" && pathname === "/api/messages/sent") {
      sentStatuses.push(url.searchParams.get("status") ?? "")
      return { data: { messages: [sentMessage], total: 1, has_more: false, next_cursor: "" } }
    }
    return undefined
  })
}

test("message recipients can be filtered by earned certificate", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMessageReadMocks(page, requests)
  await page.goto("/messages")

  const certificateFilter = page.getByRole("combobox", { name: "按已获证书筛选" })
  await certificateFilter.selectOption({ label: "CFtP Level 1（1）" })
  await expect(page.getByText("CFtP Candidate", { exact: true })).toBeVisible()
  await expect(page.getByText("CFtA Candidate", { exact: true })).toBeHidden()
  await page.getByRole("button", { name: "全选" }).click()
  await expect(page.getByText("已选择 1 个用户。", { exact: true })).toBeVisible()

  await certificateFilter.selectOption({ label: "未获得证书（1）" })
  await expect(page.getByText("Candidate Without Certificate", { exact: true })).toBeVisible()
  await expect(page.getByText("已选择 0 个用户。", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/credentials/definitions")
  expect(requests).toContain("GET /api/credentials")
})

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

test("sent message status filters use the gmsg enum values", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  const sentStatuses: string[] = []
  await installMessageReadMocks(page, requests, sentStatuses)
  await page.goto("/messages")

  await page.getByRole("button", { name: "发送记录" }).click()
  const statusFilter = page.getByRole("combobox")
  await statusFilter.selectOption({ label: "未读" })
  await expect.poll(() => sentStatuses.at(-1)).toBe("0")
  await statusFilter.selectOption({ label: "已读" })
  await expect.poll(() => sentStatuses.at(-1)).toBe("1")
})
