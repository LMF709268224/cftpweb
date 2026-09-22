import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const credential = {
  cred_ulid: "credential-regression",
  candidate_ulid: "candidate-regression",
  candidate_name: "Regression Candidate",
  cred_def_ulid: "definition-regression",
  cred_def_name: "Regression Qualification",
  status: "ACTIVE",
  version: 1,
  is_current: true,
}

test("candidate credential list and detail are displayed without mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname === "/api/credentials") {
      return { data: { credentials: [credential], total: 1, next_cursor: "", prev_cursor: "" } }
    }
    if (pathname === "/api/credentials/credential-regression") {
      return { data: { ...credential, audit_remark: "Read-only regression detail" } }
    }
    return undefined
  })

  await page.goto("/permissions")
  await expect(page.getByText("Regression Qualification", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("Regression Candidate", { exact: true }).first()).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  await expect(page.getByRole("dialog").getByText("Read-only regression detail", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/credentials?is_current=true&page_size=10")
  expect(requests).toContain("GET /api/credentials/credential-regression")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("a closed credential detail ignores its late response", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  const currentCredential = {
    ...credential,
    cred_ulid: "credential-current",
    candidate_ulid: "candidate-current",
    candidate_name: "Current Candidate",
  }
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname === "/api/credentials") {
      return { data: { credentials: [credential, currentCredential], total: 2 } }
    }
    if (pathname === "/api/credentials/credential-regression") {
      return { data: { ...credential, audit_remark: "Stale detail" } }
    }
    if (pathname === "/api/credentials/credential-current") {
      return { data: { ...currentCredential, audit_remark: "Current detail" } }
    }
    return undefined
  })
  let releaseStaleResponse!: () => void
  const staleResponseGate = new Promise<void>((resolve) => {
    releaseStaleResponse = resolve
  })
  await page.route("**/api/credentials/credential-regression", async (route) => {
    await staleResponseGate
    await route.fallback()
  })

  await page.goto("/permissions")
  await expect(page.getByText("Current Candidate", { exact: true })).toBeVisible()
  const staleResponse = page.waitForResponse((response) => new URL(response.url()).pathname === "/api/credentials/credential-regression")
  await page.getByRole("button", { name: "查看详情" }).first().click()
  await page.getByRole("button", { name: "关闭" }).click()
  await page.getByRole("button", { name: "查看详情" }).nth(1).click()
  const detail = page.getByRole("dialog")
  await expect(detail.getByText("Current detail", { exact: true })).toBeVisible()
  releaseStaleResponse()
  await staleResponse
  await expect(detail.getByText("Current detail", { exact: true })).toBeVisible()
  await expect(detail.getByText("Stale detail", { exact: true })).toHaveCount(0)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("candidate credentials recover after their initial read fails", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let credentialReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/credentials") return undefined
    credentialReads += 1
    if (credentialReads === 1) return { status: 503, errorCode: "CREDENTIALS_UNAVAILABLE", message: "Credentials unavailable" }
    return { data: { credentials: [credential], total: 1 } }
  })

  await page.goto("/permissions")
  await expect(page.getByText("暂无考生资格记录", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "刷新资格列表", exact: true }).click()

  await expect(page.getByText("Regression Qualification", { exact: true }).first()).toBeVisible()
  expect(credentialReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
