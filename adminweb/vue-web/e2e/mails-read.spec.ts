import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const mailTemplate = {
  path: "certification/regression",
  name: "Regression Mail",
  subject_template: "Certificate for {{.name}}",
  description: "Regression email template",
  version: 2,
  updated_at: "2026-08-11T00:00:00Z",
}

const sentMail = {
  mail_ulid: "mail-1",
  subject: "Regression certificate",
  to_email: "candidate@example.test",
  status: "SENT",
  created_at: "2026-08-11T01:00:00Z",
}

async function installMailReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/user/list") {
      return { data: { users: [{ id: "candidate-1", name: "Regression Candidate", email: "candidate@example.test" }] } }
    }
    if (method === "GET" && pathname === "/api/mails/stats") {
      return { data: { total: 1, sent: 1, failed: 0 } }
    }
    if (method === "GET" && pathname === "/api/mails/templates") {
      return { data: { templates: [mailTemplate], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/mails/templates/detail") {
      return { data: { ...mailTemplate, html_body: "<p>Read-only mail body</p>", parameter_schema: '{"type":"object"}' } }
    }
    if (method === "GET" && pathname === "/api/mails/sent") {
      return { data: { mails: [sentMail], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/mails") {
      return { data: { ...sentMail, html_body: "<p>Read-only delivered mail</p>", template_path: "certification/regression" } }
    }
    if (method === "GET" && pathname === "/api/mails/status") {
      return { data: { mail_ulid: "mail-1", status: "SENT", provider_message_id: "provider-1" } }
    }
    return undefined
  })
}

test("HTML mail can be previewed without sending", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMailReadMocks(page, requests)
  await page.goto("/mails")

  await page.getByLabel("邮件主题").fill("HTML preview")
  await page.getByLabel("邮件正文 HTML / Text").fill("<h1>Rendered HTML preview</h1>")
  await page.getByRole("button", { name: "预览", exact: true }).click()

  const dialog = page.getByRole("dialog", { name: "邮件预览" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("邮件主题: HTML preview", { exact: true })).toBeVisible()
  const iframe = dialog.locator("iframe")
  await expect(iframe).toHaveAttribute("srcdoc", "<h1>Rendered HTML preview</h1>")
  await expect(iframe).toHaveAttribute("sandbox", "")

  expect(requests).not.toContain("POST /api/mails/send")
  expect(requests).not.toContain("POST /api/mails/templates/render")
})

test("mail template detail is read without editing it", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMailReadMocks(page, requests)
  await page.goto("/mails")

  await page.getByRole("button", { name: "模板管理" }).click()
  await expect(page.getByText("Regression Mail", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "模板详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("Certificate for {{.name}}", { exact: true })).toBeVisible()
  await expect(dialog.getByText("<p>Read-only mail body</p>", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/mails/templates/detail")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("sent mail detail and delivery status are read without cancellation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installMailReadMocks(page, requests)
  await page.goto("/mails")

  await page.getByRole("button", { name: "发送记录" }).click()
  await expect(page.getByText("Regression certificate", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  const dialog = page.getByRole("dialog", { name: "邮件详情" })
  await expect(dialog).toBeVisible()
  await dialog.getByText("状态详情", { exact: true }).first().click()
  await expect(dialog.getByText(/provider-1/)).toBeVisible()
  await dialog.getByText("原始 JSON", { exact: true }).first().click()
  await expect(dialog.getByText(/Read-only delivered mail/)).toBeVisible()

  expect(requests).toContain("GET /api/mails")
  expect(requests).toContain("GET /api/mails/status")
  expect(requests.some((request) => request.includes("/cancel"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
