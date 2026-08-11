import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const resourcePack = {
  pack_id: "pack-1",
  title: "Regression Resources",
  description: "Read-only resource pack",
  category: "automation",
  icon: "book-open",
  respath: "/resources/regression",
  thumbnail_object_key: "resources/regression.png",
  status: "Active",
  version: 3,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installResourcePackReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/lms/resource-packs") {
      return { data: { packs: [resourcePack], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/lms/resource-packs/pack-1") {
      return { data: { ...resourcePack, description: "Read-only resource pack detail" } }
    }
    return undefined
  })
}

test("resource pack list renders the returned read-only summary", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installResourcePackReadMocks(page, requests)

  await page.goto("/resource-packs")

  await expect(page.getByText("Regression Resources", { exact: true }).first()).toBeVisible()
  await expect(page.getByText("Read-only resource pack", { exact: true })).toBeVisible()
  await expect(page.getByText("已上架", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/lms/resource-packs")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("resource pack detail is read without publishing or editing", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installResourcePackReadMocks(page, requests)
  await page.goto("/resource-packs")

  await page.getByRole("button", { name: "查看详情" }).click()
  await expect(page.getByRole("heading", { name: "资源包详情" })).toBeVisible()
  const dialog = page.getByLabel("资源包详情")
  await expect(dialog.getByText("Read-only resource pack detail", { exact: true })).toBeVisible()
  await expect(dialog.getByText("resources/regression.png", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/lms/resource-packs/pack-1")
  expect(requests.some((request) => request.includes("/publish") || request.includes("/revert-to-draft"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
