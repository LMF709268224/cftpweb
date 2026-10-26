import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const pdfRequestSummary = {
  request_ulid: "pdf-request-1",
  business_unit: "gprog",
  candidate_ulid: "candidate-1",
  candidate_name: "Regression Candidate",
  degree_no: "CERT-2026-001",
  status: 3,
  created_at: "2026-08-11T00:00:00Z",
}

async function installPdfRequestReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/pdf-requests") {
      return {
        data: {
          requests: [pdfRequestSummary],
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/pdf-requests/pdf-request-1/detail") {
      return {
        data: {
          ...pdfRequestSummary,
          cred_def_ulid: "credential-definition-1",
          template_ulid: "template-1",
          template_params: JSON.stringify({ name: "Regression Candidate", qualification: "CFTP" }),
          status: 4,
          error_msg: "Renderer unavailable",
          pdf_file_hash: "sha256-regression",
        },
      }
    }
    return undefined
  })
}

test("PDF request list renders the returned read-only generation status", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPdfRequestReadMocks(page, requests)

  await page.goto("/pdf-requests")

  await expect(page.getByText("CERT-2026-001", { exact: true })).toBeVisible()
  await expect(page.getByText("Regression Candidate", { exact: true })).toBeVisible()
  await expect(page.getByText("pdf-request-1", { exact: true })).toBeVisible()
  await expect(page.getByText("成功", { exact: true }).first()).toBeVisible()
  expect(requests).toContain("GET /api/pdf-requests")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("PDF request detail displays certificate identifiers without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installPdfRequestReadMocks(page, requests)
  await page.goto("/pdf-requests")

  await page.getByRole("button", { name: "查看详情" }).click()

  const dialog = page.getByRole("dialog", { name: "流水详情" })
  await expect(dialog).toBeVisible()
  await expect(dialog.getByText("Renderer unavailable")).toBeVisible()
  await expect(dialog.getByText("CFTP", { exact: true })).toBeVisible()
  await dialog.getByText("技术字段").click()
  await expect(dialog.getByText("credential-definition-1", { exact: true })).toBeVisible()
  await expect(dialog.getByText("sha256-regression", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/pdf-requests/pdf-request-1/detail")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
