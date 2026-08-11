import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const pendingApplication = {
  app_ulid: "app-1",
  candidate_ulid: "candidate-1",
  candidate_name: "Regression Candidate",
  cred_def_ulid: "credential-1",
  cred_def_name: "Regression Credential",
  status: "Pending",
  created_at: "2026-08-11T00:00:00Z",
  files: [],
}

async function installApplicationMocks(
  page: Parameters<typeof seedAuthenticatedAdmin>[0],
  onAudit?: (postData: string | null) => void,
) {
  return installAdminApiMocks(page, ({ method, pathname, postData }) => {
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
    if (method === "POST" && pathname === "/api/applications/audit") {
      onAudit?.(postData)
      return { data: { app_ulid: "app-1", status: "Approved" } }
    }
    return undefined
  })
}

async function openApplicationAudit(page: Parameters<typeof seedAuthenticatedAdmin>[0]) {
  await page.goto("/applications")
  await expect(page.getByText("Regression Credential").first()).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  await page.getByRole("button", { name: /审核操作/ }).click()
  await expect(page.getByText("支持通过、打回重提、最终拒绝")).toBeVisible()
}

test("approving an application submits the selected application contract", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let auditBody: Record<string, unknown> | null = null
  await installApplicationMocks(page, (postData) => {
    auditBody = JSON.parse(postData || "null") as Record<string, unknown> | null
  })
  await openApplicationAudit(page)

  await page.getByRole("button", { name: "通过", exact: true }).click()

  await expect.poll(() => auditBody).not.toBeNull()
  expect(auditBody).toEqual({
    application_id: "app-1",
    approved: true,
    reject_reason: "",
    require_resubmit: false,
  })
  await expect(page.getByText("审核已提交")).toBeVisible()
})

test("rejecting an application without a remark is blocked before the API call", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  let auditRequests = 0
  await installApplicationMocks(page, () => {
    auditRequests += 1
  })
  await openApplicationAudit(page)

  await page.getByRole("button", { name: "最终拒绝" }).click()

  await expect(page.getByText("拒绝或要求补交时需要填写审核备注")).toBeVisible()
  expect(auditRequests).toBe(0)
})
