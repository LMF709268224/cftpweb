import { expect, test } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const definition = {
  cred_def_ulid: "credential-definition-regression",
  name: "Regression Qualification",
  category: "Professional",
}

test("permission definitions and qualification result are displayed without mutation", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname === "/api/credentials/definitions") return { data: { definitions: [definition] } }
    if (pathname === "/api/permissions/check") {
      return { data: { credential_status: "ACTIVE", eligible: true, evidence_count: 2 } }
    }
    return undefined
  })

  await page.goto("/permissions")
  await expect(page.getByText("Regression Qualification", { exact: true }).first()).toBeVisible()
  await page.getByPlaceholder("输入考生 ID").fill("candidate-regression")
  await page.getByRole("button", { name: "检查权限" }).click()

  await expect(page.getByText("candidate-regression", { exact: true }).first()).toBeVisible()
  await expect(page.getByRole("textbox", { name: "credential_status" })).toHaveValue("ACTIVE")
  await expect(page.getByRole("textbox", { name: "eligible" })).toHaveValue("true")
  expect(requests.some((request) => request.includes("candidate_ulid=candidate-regression") && request.includes("cred_def_ulid=credential-definition-regression"))).toBe(true)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("permission check ignores a response for a candidate changed while loading", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installAdminApiMocks(page, ({ method, pathname, url }) => {
    requests.push(`${method} ${pathname}${url.search}`)
    if (pathname === "/api/credentials/definitions") return { data: { definitions: [definition] } }
    if (pathname === "/api/permissions/check") {
      const candidate = url.searchParams.get("candidate_ulid")
      return candidate === "candidate-old"
        ? { data: { credential_status: "STALE" }, delayMs: 500 }
        : { data: { credential_status: "ACTIVE", candidate_ulid: candidate } }
    }
    return undefined
  })
  await page.goto("/permissions")
  const candidate = page.getByPlaceholder("输入考生 ID")
  const staleRequest = page.waitForRequest((request) => new URL(request.url()).searchParams.get("candidate_ulid") === "candidate-old")
  const staleResponse = page.waitForResponse((response) => new URL(response.url()).searchParams.get("candidate_ulid") === "candidate-old")
  await candidate.fill("candidate-old")
  await page.getByRole("button", { name: "检查权限" }).click()
  await staleRequest
  await candidate.fill("candidate-current")
  await staleResponse
  await expect(page.getByText("权限详情", { exact: true })).toHaveCount(0)

  await page.getByRole("button", { name: "检查权限" }).click()
  await expect(page.getByRole("textbox", { name: "credential_status" })).toHaveValue("ACTIVE")
  await expect(page.getByRole("textbox", { name: "candidate_ulid" })).toHaveValue("candidate-current")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("permission definitions recover after their initial read fails", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  let definitionReads = 0
  await installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (pathname !== "/api/credentials/definitions") return undefined
    definitionReads += 1
    if (definitionReads === 1) return { status: 503, errorCode: "DEFINITIONS_UNAVAILABLE", message: "Definitions unavailable" }
    return { data: { definitions: [definition] } }
  })

  await page.goto("/permissions")
  await expect(page.getByText("资格定义加载失败", { exact: true })).toBeVisible()
  await page.getByRole("button", { name: "刷新资格定义", exact: true }).click()

  await expect(page.getByText("Regression Qualification", { exact: true }).first()).toBeVisible()
  expect(definitionReads).toBe(2)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
