import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const auditSummary = {
  audit_ulid: "audit-1",
  created_at: "2026-08-11T00:00:00Z",
  source_service: "gcreds",
  action: "READ",
  status: "SUCCESS",
  summary_text: "Viewed credential application",
  operator_id: "admin-1",
  operator_name: "Regression Admin",
  resource_type: "credential_application",
  resource_id: "application-1",
  resource_display_name: "Regression Application",
}

async function installAuditReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/audit/logs") {
      return {
        data: {
          items: [auditSummary],
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/audit/logs/audit-1") {
      return {
        data: {
          summary: auditSummary,
          details: JSON.stringify({ field: "status", operation: "read" }),
        },
      }
    }
    return undefined
  })
}

test("audit list renders the returned read-only operation summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAuditReadMocks(page, requests)

  await page.goto("/audit/logs")

  await expect(page.getByText("Viewed credential application", { exact: true })).toBeVisible()
  await expect(page.getByText("Regression Admin", { exact: true }).last()).toBeVisible()
  await expect(page.getByText("SUCCESS", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/audit/logs")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("audit detail displays the nested summary without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAuditReadMocks(page, requests)
  await page.goto("/audit/logs")

  await page.getByText("Viewed credential application", { exact: true }).click()

  const detailDialog = page.getByRole("dialog", { name: "审计详情" })
  await expect(detailDialog).toBeVisible()
  await expect(detailDialog.getByText("Regression Application", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText(/"operation": "read"/)).toBeVisible()
  expect(requests).toContain("GET /api/audit/logs/audit-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
