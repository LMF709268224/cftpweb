import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const credential = {
  cred_ulid: "credential-regression",
  candidate_ulid: "candidate-regression",
  candidate_name: "Regression Candidate",
  cred_def_ulid: "definition-regression",
  cred_def_name: "Regression Qualification",
  source: 2,
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
  await expect(page.getByRole("link", { name: "查看申请记录" })).toHaveAttribute("href", "/applications")
  await expect(page.getByRole("cell", { name: "已颁发证书" })).toBeVisible()
  await page.getByRole("button", { name: "查看详情" }).click()
  await expect(page.getByRole("dialog").getByText("Read-only regression detail", { exact: true })).toBeVisible()
  await expect(page.getByRole("button", { name: "标记资格过期" })).toHaveCount(0)

  expect(requests).toContain("GET /api/credentials?is_current=true&page_size=10&source=2")
  expect(requests).toContain("GET /api/credentials/credential-regression")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("credential source tabs keep their own count and reset pagination", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  const applicationCredential = {
    ...credential,
    cred_ulid: "application-credential",
    cred_def_name: "Application Qualification",
    source: 1,
  }
  await installAdminApiMocks(page, ({ pathname, url }) => {
    if (pathname !== "/api/credentials") return undefined
    requests.push(url.search)
    if (url.searchParams.get("source") === "1") {
      return { data: { credentials: [applicationCredential], total: 1 } }
    }
    return { data: { credentials: [credential], total: 11, next_cursor: "cert-next", prev_cursor: "" } }
  })

  await page.goto("/permissions")
  await expect(page.getByRole("cell", { name: "已颁发证书" })).toBeVisible()
  await page.getByRole("button", { name: "下一页" }).click()
  await expect.poll(() => requests.some((search) => search.includes("cursor=cert-next"))).toBe(true)

  await page.getByRole("tab", { name: "申请获批资格" }).click()
  await expect(page.getByRole("cell", { name: "申请获批资格" })).toBeVisible()
  await expect(page.getByText("Application Qualification", { exact: true }).first()).toBeVisible()
  expect(requests.at(-1)).toBe("?is_current=true&page_size=10&source=1")
  await expect(page.getByText("共 1 条 · 第 1 / 1 页")).toBeVisible()
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
