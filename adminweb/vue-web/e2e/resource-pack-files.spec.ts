import { expect, test, type Page } from "@playwright/test"
import { installAdminApiMocks, seedAuthenticatedAdmin } from "./support/admin"

const resourcePack = { pack_id: "pack-1", title: "Regression Resources", status: "Active", version: 3 }
const resourceFile = {
  file_id: "file-1",
  pack_id: "pack-1",
  title: "Regression Guide",
  description: "Read-only PDF guide",
  file_type: 2,
  file_name: "regression.pdf",
  file_size: 2048,
  file_hash: "sha256-regression",
  file_object_key: "resources/regression.pdf",
  sort_order: 1,
  version: 2,
  created_at: "2026-08-11T00:00:00Z",
  updated_at: "2026-08-11T01:00:00Z",
}

async function installResourceFileReadMocks(page: Page, requests: string[]) {
  return installAdminApiMocks(page, ({ method, pathname }) => {
    requests.push(`${method} ${pathname}`)
    if (method === "GET" && pathname === "/api/lms/resource-packs") {
      return { data: { packs: [resourcePack], has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/lms/resource-pack-files") {
      return { data: { files: [resourceFile], total: 1, has_more: false, next_cursor: "" } }
    }
    if (method === "GET" && pathname === "/api/lms/resource-pack-files/file-1") {
      return { data: { ...resourceFile, description: "Read-only PDF guide detail" } }
    }
    return undefined
  })
}

test("resource file list renders its owner and file metadata", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installResourceFileReadMocks(page, requests)

  await page.goto("/resource-pack-files")

  await expect(page.getByText("Regression Guide", { exact: true }).first()).toBeVisible()
  await expect(page.getByRole("button", { name: /Regression Guide.*Regression Resources/ })).toBeVisible()
  await expect(page.getByText("PDF 文档", { exact: true })).toBeVisible()
  expect(requests).toContain("GET /api/lms/resource-packs")
  expect(requests).toContain("GET /api/lms/resource-pack-files")
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})

test("resource file detail is read without upload, edit, or delete", async ({ page }) => {
  await seedAuthenticatedAdmin(page)
  const requests: string[] = []
  await installResourceFileReadMocks(page, requests)
  await page.goto("/resource-pack-files")

  await page.getByRole("button", { name: "查看详情" }).click()
  await expect(page.getByRole("heading", { name: "资源文件详情" })).toBeVisible()
  const dialog = page.getByLabel("资源文件详情")
  await expect(dialog.getByText("Read-only PDF guide detail", { exact: true })).toBeVisible()
  await expect(dialog.getByText("resources/regression.pdf", { exact: true })).toBeVisible()
  await expect(dialog.getByText("sha256-regression", { exact: true })).toBeVisible()

  expect(requests).toContain("GET /api/lms/resource-pack-files/file-1")
  expect(requests.some((request) => request.includes("upload-url"))).toBe(false)
  expect(requests.every((request) => request.startsWith("GET "))).toBe(true)
})
