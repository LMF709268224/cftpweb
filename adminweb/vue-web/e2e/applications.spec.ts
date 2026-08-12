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

async function installApplicationReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/applications") {
      return {
        data: {
          applications: [pendingApplication],
          total: 1,
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
  expect(requests).toContain("GET /api/applications")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
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
