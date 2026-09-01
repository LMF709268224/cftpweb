import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const pendingApplication = {
  app_ulid: "app-1",
  candidate_ulid: "candidate-1",
  candidate_name: "Regression Candidate",
  cred_def_ulid: "credential-1",
  cred_def_name: "Regression Credential",
  status: "PENDING",
  created_at: "2026-08-11T00:00:00Z",
  files: [
    {
      file_hash: "sha256-regression",
      file_name: "evidence.pdf",
      file_ext: "pdf",
      file_size: 2048,
      file_usage: "evidence",
    },
  ],
}

const pendingUploadApplication = {
  ...pendingApplication,
  app_ulid: "app-pending-upload",
  cred_def_name: "Upload Pending Credential",
  status: "PendingUpload",
}

async function installApplicationReadMocks(page: Page, requests: string[], applicationItems = [pendingApplication], queriedStatuses?: string[]) {
  return installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/applications") {
      queriedStatuses?.push(url.searchParams.get("status") || "")
      return {
        data: {
          applications: applicationItems,
          total: applicationItems.length,
          status_subtotals: [
            { status: "", count: 6, count_label: "6", exact: true },
            { status: "Pending", count: 1, count_label: "1", exact: true },
            { status: "PendingUpload", count: 1, count_label: "1", exact: true },
            { status: "Approved", count: 2, count_label: "2", exact: true },
            { status: "Rejected", count: 1, count_label: "1", exact: true },
            { status: "Reupload", count: 1, count_label: "1", exact: true },
          ],
          has_more: false,
          next_cursor: "",
        },
      }
    }
    if (method === "GET" && pathname === "/api/applications/app-1") {
      return { data: pendingApplication }
    }
    return undefined
  })
}

test("application list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installApplicationReadMocks(page, requests)

  await page.goto("/applications")

  await expect(page.getByText("Regression Credential").first()).toBeVisible()
  await expect(page.getByText(/Regression Candidate/).first()).toBeVisible()
  await expect(page.getByText(/app-1/).first()).toBeVisible()
  const statusSubtotals = page.getByTestId("application-status-subtotal")
  await expect(statusSubtotals).toHaveCount(6)
  await expect(statusSubtotals.filter({ hasText: "全部" })).toContainText("6")
  await expect(statusSubtotals.filter({ hasText: "待审核" })).toContainText("1")
  await expect(statusSubtotals.filter({ hasText: "等待上传" })).toContainText("1")
  await expect(statusSubtotals.filter({ hasText: "已通过" })).toContainText("2")
  await expect(statusSubtotals.filter({ hasText: "已拒绝" })).toContainText("1")
  await expect(statusSubtotals.filter({ hasText: "需补交" })).toContainText("1")
  expect(requests).toContain("GET /api/applications")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("application list distinguishes pending uploads from pending review", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  const queriedStatuses: string[] = []
  await installApplicationReadMocks(page, requests, [pendingUploadApplication], queriedStatuses)

  await page.goto("/applications")
  await page.getByRole("combobox").selectOption("PendingUpload")

  await expect(page.getByText("Upload Pending Credential")).toBeVisible()
  await expect(page.locator("span").filter({ hasText: /^等待上传$/ })).toBeVisible()
  expect(queriedStatuses).toContain("PendingUpload")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Upload Pending Credential" })
  await expect(detailDialog.getByRole("button", { name: /审核操作/ })).toHaveCount(0)
})

test("application detail displays metadata and evidence without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installApplicationReadMocks(page, requests)
  await page.goto("/applications")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Credential" })
  await expect(detailDialog.getByRole("heading", { name: "Regression Credential" })).toBeVisible()
  await expect(detailDialog.getByText("app-1", { exact: true }).first()).toBeVisible()
  await detailDialog.getByRole("button", { name: /申请材料/ }).click()
  await expect(detailDialog.getByText("evidence.pdf")).toBeVisible()

	await detailDialog.getByRole("button", { name: /审核操作/ }).click()
	const expectedExpiry = new Date()
	expectedExpiry.setFullYear(expectedExpiry.getFullYear() + 2)
	const expectedExpiryValue = [
		expectedExpiry.getFullYear(),
		String(expectedExpiry.getMonth() + 1).padStart(2, "0"),
		String(expectedExpiry.getDate()).padStart(2, "0"),
	].join("-")
	await expect(detailDialog.getByLabel("资格有效期截止日")).toHaveValue(expectedExpiryValue)

  expect(requests).toContain("GET /api/applications/app-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("application approval requires a user remark", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let auditRequest: Record<string, unknown> | null = null
  await installApplicationReadMocks(page, requests)
  await page.route("**/api/applications/audit", async (route) => {
    auditRequest = route.request().postDataJSON()
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ code: 200, error_code: "OK", message: "OK", data: { app_ulid: "app-1" } }),
    })
  })

  await page.goto("/applications")
  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Credential" })
  await detailDialog.getByRole("button", { name: /审核操作/ }).click()
  await detailDialog.getByRole("button", { name: "通过" }).click()

  await expect(page.getByText("请填写审核备注")).toBeVisible()
  expect(auditRequest).toBeNull()

  await detailDialog.locator("textarea").fill("Approved after review")
  await detailDialog.getByRole("button", { name: "通过" }).click()

  await expect.poll(() => auditRequest).not.toBeNull()
  expect(auditRequest).toMatchObject({ approved: true, reject_reason: "Approved after review", require_resubmit: false })
  await expect(page.getByText("审核已提交")).toBeVisible()
})
