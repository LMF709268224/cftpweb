import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const credentialSummary = {
  cred_def_ulid: "credential-1",
  name: "Regression Credential",
  category: "Certification",
}

const credentialDetail = {
  ...credentialSummary,
  description: "Read-only regression definition",
  respath: "/gcc/credential/regression",
  acquisition_method: "Pass the regression assessment",
  file_constraints: [
    {
      name: "Evidence PDF",
      type: 2,
      is_required: true,
    },
  ],
}

async function installCredentialReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/credentials/definitions") {
      return { data: { definitions: [credentialSummary] } }
    }
    if (method === "GET" && pathname === "/api/credentials/definitions/credential-1") {
      return { data: credentialDetail }
    }
    return undefined
  })
}

test("credential definition list renders returned read-only data", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installCredentialReadMocks(page, requests)

  await page.goto("/credentials")

  await expect(page.getByText("Regression Credential").first()).toBeVisible()
  await expect(page.getByText("认证资格").first()).toBeVisible()
  expect(requests).toContain("GET /api/credentials/definitions")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("credential definition detail displays metadata without a mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installCredentialReadMocks(page, requests)
  await page.goto("/credentials")

  await page.getByRole("button", { name: "查看详情" }).click()
  const detailDialog = page.getByRole("dialog", { name: "Regression Credential" })
  await expect(detailDialog.getByText("/gcc/credential/regression", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Read-only regression definition", { exact: true })).toBeVisible()
  await expect(detailDialog.getByText("Evidence PDF", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/credentials/definitions/credential-1")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
